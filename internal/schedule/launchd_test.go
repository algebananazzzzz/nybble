package schedule

import (
	"strings"
	"testing"
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
		"CANTEEN_API_BASE":  "https://host.example.com/app",
		"CANTEEN_LOGIN_URL": "https://host.example.com/login?a=1&b=2",
	}
	p := Plist("/usr/local/bin/canteen", "/usr/bin", env, 4, 9, 59)
	if !strings.Contains(p, "<key>CANTEEN_API_BASE</key><string>https://host.example.com/app</string>") {
		t.Errorf("missing CANTEEN_API_BASE env in plist:\n%s", p)
	}
	// Query-string ampersands must be XML-escaped or the plist won't parse.
	if !strings.Contains(p, "https://host.example.com/login?a=1&amp;b=2") {
		t.Errorf("login URL not XML-escaped:\n%s", p)
	}
}

func TestEnvForJobSkipsUnset(t *testing.T) {
	t.Setenv("CANTEEN_API_BASE", "https://host.example.com/app")
	t.Setenv("CANTEEN_LOGIN_URL", "")
	got := envForJob()
	if got["CANTEEN_API_BASE"] != "https://host.example.com/app" {
		t.Errorf("CANTEEN_API_BASE = %q", got["CANTEEN_API_BASE"])
	}
	if _, ok := got["CANTEEN_LOGIN_URL"]; ok {
		t.Error("empty CANTEEN_LOGIN_URL should be skipped")
	}
}

func TestJobPathIncludesHomebrew(t *testing.T) {
	if !strings.Contains(jobPath(), "/opt/homebrew/bin") {
		t.Fatalf("jobPath missing homebrew: %q", jobPath())
	}
}
