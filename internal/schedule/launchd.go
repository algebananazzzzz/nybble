package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const Label = "com.bytecanteen.autobooker"

var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

// jobEnvKeys are the CANTEEN_* variables captured from the installing shell into the
// scheduled job. A launchd run has none of the shell's environment, so without these the
// `book` run hits the hard "CANTEEN_API_BASE not set" error and never books.
var jobEnvKeys = []string{"CANTEEN_API_BASE", "CANTEEN_LOGIN_URL", "CANTEEN_LARK_TARGET"}

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
// env carries additional EnvironmentVariables (the CANTEEN_* endpoint config) so a
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
  <key>StandardOutPath</key><string>%s/canteen.log</string>
  <key>StandardErrorPath</key><string>%s/canteen.err</string>
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

const (
	launchLeadMin = 5 // launchd fires this many min early; the run waits precisely to hh:mm
	wakeLeadMin   = 7 // pmset wakes the Mac this many min early (before launchd fires)
)

// On writes + loads the job and schedules a pmset wake before it.
// weekday: 1=Mon..7=Sun for pmset; launchd uses 0=Sun..6=Sat (caller passes launchd weekday).
func On(bin string, launchdWeekday, hh, mm int) error {
	// Fire launchd ~5 min early; the run notifies "starts in N min" then waits to hh:mm.
	lh, lm := subMinutes(hh, mm, launchLeadMin)
	if err := os.WriteFile(plistPath(), []byte(Plist(bin, jobPath(), envForJob(), launchdWeekday, lh, lm)), 0o644); err != nil {
		return err
	}
	if err := exec.Command("launchctl", "unload", plistPath()).Run(); err != nil {
		// ignore: may not be loaded yet
	}
	if err := exec.Command("launchctl", "load", plistPath()).Run(); err != nil {
		return err
	}
	// pmset wake ~7 min before open (requires sudo; print guidance if it fails)
	day := pmsetDay(launchdWeekday)
	wh, wm := subMinutes(hh, mm, wakeLeadMin)
	cmd := exec.Command("sudo", "pmset", "repeat", "wake", day, fmt.Sprintf("%02d:%02d:00", wh, wm))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// subMinutes returns hh:mm minus d minutes, wrapping within a 24h day. It does not roll
// the weekday back, so keep leads small relative to the scheduled time (e.g. a 10:00 job).
func subMinutes(hh, mm, d int) (int, int) {
	t := ((hh*60+mm-d)%(24*60) + 24*60) % (24 * 60)
	return t / 60, t % 60
}

func Off() error {
	_ = exec.Command("launchctl", "unload", plistPath()).Run()
	_ = os.Remove(plistPath())
	return exec.Command("sudo", "pmset", "repeat", "cancel").Run()
}

func pmsetDay(launchdWeekday int) string {
	// launchd 0=Sun..6=Sat → pmset letters MTWRFSU
	return map[int]string{0: "U", 1: "M", 2: "T", 3: "W", 4: "R", 5: "F", 6: "S"}[launchdWeekday]
}
