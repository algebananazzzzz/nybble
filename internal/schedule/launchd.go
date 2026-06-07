package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/algebananazzzzz/nybble/internal/clock"
)

// weekdayNum maps the config's weekday code to a time.Weekday for the open-time lookup.
var weekdayNum = map[string]time.Weekday{
	"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
}

// LocalFire converts a configured open (weekdayCode + hh:mm in tz) into the local
// weekday/hour/minute that launchd and pmset use (both run in the machine's local time).
// now anchors which upcoming occurrence is resolved.
func LocalFire(weekdayCode string, hh, mm int, tz string, now time.Time) (launchdWeekday, lhh, lmm int, err error) {
	loc, lerr := time.LoadLocation(tz)
	if lerr != nil {
		return 0, 0, 0, fmt.Errorf("bad timezone %q: %w", tz, lerr)
	}
	open := clock.NextOpen(now, weekdayNum[weekdayCode], hh, mm, loc).Local()
	return int(open.Weekday()), open.Hour(), open.Minute(), nil
}

const Label = "com.algebananazzzzz.nybble"

var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

// jobEnvKeys are the NYBBLE_* variables captured from the installing shell into the
// scheduled job. A launchd run has none of the shell's environment, so without these the
// `book` run hits the hard "NYBBLE_API_BASE not set" error and never books.
var jobEnvKeys = []string{"NYBBLE_API_BASE", "NYBBLE_LOGIN_URL"}

// envForJob snapshots the currently-set jobEnvKeys (skipping empty ones).
func envForJob() map[string]string {
	m := map[string]string{}
	for _, k := range jobEnvKeys {
		if v := os.Getenv(k); v != "" {
			m[k] = v
		}
	}
	return m
}

// Plist renders a launchd job that runs `<bin> book` at weekday hh:mm (local time).
// pathEnv is baked into the job's PATH: launchd's default PATH is minimal, so without
// this the run can't find lark-cli and Lark notifications silently degrade (or vanish).
// env carries additional EnvironmentVariables (the NYBBLE_* endpoint config) so a
// scheduled run can resolve the API; keys are emitted sorted for a stable plist.
func Plist(bin, pathEnv string, env map[string]string, weekday, hh, mm int) string {
	var envXML strings.Builder
	fmt.Fprintf(&envXML, "<key>PATH</key><string>%s</string>", xmlEscaper.Replace(pathEnv))
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&envXML, "<key>%s</key><string>%s</string>", xmlEscaper.Replace(k), xmlEscaper.Replace(env[k]))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>%s</string><string>book</string></array>
  <key>EnvironmentVariables</key>
  <dict>%s</dict>
  <key>StartCalendarInterval</key>
  <dict>
    <key>Weekday</key><integer>%d</integer>
    <key>Hour</key><integer>%d</integer>
    <key>Minute</key><integer>%d</integer>
  </dict>
  <key>StandardOutPath</key><string>%s/nybble.log</string>
  <key>StandardErrorPath</key><string>%s/nybble.err</string>
</dict></plist>`, Label, bin, envXML.String(), weekday, hh, mm, logDir(), logDir())
}

// jobPath is the PATH baked into the scheduled job. launchd's default PATH is minimal,
// so we build a small, purposeful one: the standard system + Homebrew dirs, plus the
// real directory of lark-cli (the notifier). lark-cli is a node script, so its own dir
// also carries the node it needs. Resolving the tool (rather than dumping the installer's
// whole $PATH) keeps the plist clean and doesn't leak unrelated environment.
func jobPath() string {
	dirs := []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin", "/opt/homebrew/bin", "/usr/local/bin"}
	for _, tool := range []string{"lark-cli"} {
		if p, err := exec.LookPath(tool); err == nil {
			dirs = append(dirs, filepath.Dir(p))
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		if d != "" && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return strings.Join(out, ":")
}

func logDir() string {
	h, _ := os.UserHomeDir()
	d := filepath.Join(h, "Library", "Logs")
	return d
}

func plistPath() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Library", "LaunchAgents", Label+".plist")
}

// wakeBufferMin is how far ahead of the launchd fire the Mac is woken, so it's fully
// up before the job runs. Total wake lead = leadMin + wakeBufferMin.
const wakeBufferMin = 2

// jobPlist renders the launchd job for an open time of hh:mm, set to FIRE leadMin early
// so the run can notify "starts in N min" then wait to the exact open. Pure (no I/O
// beyond reading the current PATH/endpoint env), so the fire-time arithmetic is testable.
func jobPlist(bin string, launchdWeekday, hh, mm, leadMin int) string {
	lh, lm := subMinutes(hh, mm, leadMin)
	return Plist(bin, jobPath(), envForJob(), launchdWeekday, lh, lm)
}

// InstallJob writes + (re)loads the launchd LaunchAgent. No sudo — it lives in the
// user's ~/Library/LaunchAgents. launchdWeekday is 0=Sun..6=Sat.
func InstallJob(bin string, launchdWeekday, hh, mm, leadMin int) error {
	if err := os.WriteFile(plistPath(), []byte(jobPlist(bin, launchdWeekday, hh, mm, leadMin)), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", plistPath()).Run() // ignore: may not be loaded
	return exec.Command("launchctl", "load", plistPath()).Run()
}

// RemoveJob unloads + deletes the LaunchAgent. No sudo. Idempotent.
func RemoveJob() error {
	_ = exec.Command("launchctl", "unload", plistPath()).Run()
	if err := os.Remove(plistPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WakeCmd builds the `sudo pmset repeat wake` command that wakes the Mac
// leadMin+wakeBufferMin before the open time. Returned unrun so the caller (the TUI)
// can drive it through tea.ExecProcess and surface macOS's own sudo prompt.
func WakeCmd(launchdWeekday, hh, mm, leadMin int) *exec.Cmd {
	day := pmsetDay(launchdWeekday)
	wh, wm := subMinutes(hh, mm, leadMin+wakeBufferMin)
	return exec.Command("sudo", "pmset", "repeat", "wake", day, fmt.Sprintf("%02d:%02d:00", wh, wm))
}

// WakeCancelCmd builds the `sudo pmset repeat cancel` command (also needs sudo).
func WakeCancelCmd() *exec.Cmd {
	return exec.Command("sudo", "pmset", "repeat", "cancel")
}

// Installed reports whether the LaunchAgent plist exists — the schedule's on/off state.
func Installed() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

// subMinutes returns hh:mm minus d minutes, wrapping within a 24h day. It does not roll
// the weekday back, so keep leads small relative to the scheduled time (e.g. a 10:00 job).
func subMinutes(hh, mm, d int) (int, int) {
	t := ((hh*60+mm-d)%(24*60) + 24*60) % (24 * 60)
	return t / 60, t % 60
}

func pmsetDay(launchdWeekday int) string {
	// launchd 0=Sun..6=Sat → pmset letters MTWRFSU
	return map[int]string{0: "U", 1: "M", 2: "T", 3: "W", 4: "R", 5: "F", 6: "S"}[launchdWeekday]
}
