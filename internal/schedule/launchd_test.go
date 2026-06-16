package schedule

import (
	"strings"
	"testing"
	"time"
)

func TestPlistContainsWeekdayAndBinary(t *testing.T) {
	p := Plist("/usr/local/bin/canteen", "/opt/homebrew/bin:/usr/bin", nil, 4, 9, 59) // Thu 09:59 launch
	if !strings.Contains(p, "<integer>4</integer>") {
		t.Fatal("missing weekday 4")
	}
	if !strings.Contains(p, "/usr/local/bin/canteen") || !strings.Contains(p, "<string>book</string>") {
		t.Fatal("missing program args")
	}
	if !strings.Contains(p, "<key>PATH</key><string>/opt/homebrew/bin:/usr/bin</string>") {
		t.Fatal("missing PATH env")
	}
}

func TestPlistInjectsEndpointEnv(t *testing.T) {
	env := map[string]string{
		"NYBBLE_API_BASE":  "https://host.example.com/app",
		"NYBBLE_LOGIN_URL": "https://host.example.com/login?a=1&b=2",
	}
	p := Plist("/usr/local/bin/canteen", "/usr/bin", env, 4, 9, 59)
	if !strings.Contains(p, "<key>NYBBLE_API_BASE</key><string>https://host.example.com/app</string>") {
		t.Errorf("missing NYBBLE_API_BASE env in plist:\n%s", p)
	}
	// Query-string ampersands must be XML-escaped or the plist won't parse.
	if !strings.Contains(p, "https://host.example.com/login?a=1&amp;b=2") {
		t.Errorf("login URL not XML-escaped:\n%s", p)
	}
}

// parseProgram must recover the job binary from a rendered plist — including when the
// path needs XML escaping — so a stale plist (binary moved/renamed) is detectable.
func TestParseProgramRoundTrip(t *testing.T) {
	for _, bin := range []string{
		"/Users/me/.local/bin/nybble",
		"/Users/o'brien/tools & bins/nybble",
	} {
		p := Plist(bin, "/usr/bin", nil, 4, 9, 55)
		if got := parseProgram(p); got != bin {
			t.Errorf("parseProgram = %q, want %q", got, bin)
		}
	}
}

func TestParseProgramEmptyOnGarbage(t *testing.T) {
	if got := parseProgram("not a plist"); got != "" {
		t.Errorf("parseProgram on garbage = %q, want empty", got)
	}
}

func TestEnvForJobSkipsUnset(t *testing.T) {
	t.Setenv("NYBBLE_API_BASE", "https://host.example.com/app")
	t.Setenv("NYBBLE_LOGIN_URL", "")
	got := envForJob()
	if got["NYBBLE_API_BASE"] != "https://host.example.com/app" {
		t.Errorf("NYBBLE_API_BASE = %q", got["NYBBLE_API_BASE"])
	}
	if _, ok := got["NYBBLE_LOGIN_URL"]; ok {
		t.Error("empty NYBBLE_LOGIN_URL should be skipped")
	}
}

func TestJobPathIncludesHomebrew(t *testing.T) {
	if !strings.Contains(jobPath(), "/opt/homebrew/bin") {
		t.Fatalf("jobPath missing homebrew: %q", jobPath())
	}
}

// jobPlist must schedule launchd to FIRE leadMin before the open time: a 10:00 open
// with a 5-min lead fires at 09:55, so the run can announce "starts in N min" and wait.
func TestJobPlistFiresLeadMinutesEarly(t *testing.T) {
	p := jobPlist("/usr/local/bin/nybble", 4, 10, 0, 5) // Thu 10:00 open, 5-min lead
	if !strings.Contains(p, "<key>Hour</key><integer>9</integer>") {
		t.Errorf("fire hour should be 9 (10:00 − 5min):\n%s", p)
	}
	if !strings.Contains(p, "<key>Minute</key><integer>55</integer>") {
		t.Errorf("fire minute should be 55 (10:00 − 5min):\n%s", p)
	}
}

// WakeCmd must wake the Mac leadMin+2 before the open: a 10:00 open with a 5-min lead
// wakes at 09:53 (2 min before launchd fires) on the right pmset weekday letter.
func TestWakeCmdWakesBeforeFire(t *testing.T) {
	args := WakeCmd(4, 10, 0, 5).Args // Thu (launchd 4 → pmset "R")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "pmset repeat wake R 09:53:00") {
		t.Errorf("wake should be Thu 09:53 (10:00 − 5 − 2): %q", joined)
	}
}

func TestWakeCancelCmd(t *testing.T) {
	joined := strings.Join(WakeCancelCmd().Args, " ")
	if !strings.Contains(joined, "pmset repeat cancel") {
		t.Errorf("cancel args = %q", joined)
	}
}

// LocalFire converts a configured open (weekday code + hh:mm in a tz) into the local
// launchd weekday/hour/minute. When the source tz IS local, the conversion is identity,
// so the configured Thursday 10:30 comes back unchanged — a deterministic check that
// doesn't depend on the test machine's timezone.
func TestLocalFireIdentityInLocalZone(t *testing.T) {
	wd, hh, mm, err := LocalFire("thu", 10, 30, "Local", time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if wd != int(time.Thursday) || hh != 10 || mm != 30 {
		t.Errorf("got wd=%d hh=%d mm=%d, want 4 10 30", wd, hh, mm)
	}
}

func TestLocalFireRejectsBadZone(t *testing.T) {
	if _, _, _, err := LocalFire("thu", 10, 0, "Not/AZone", time.Now()); err == nil {
		t.Error("bad timezone should error")
	}
}
