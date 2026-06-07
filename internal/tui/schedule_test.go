package tui

import (
	"strings"
	"testing"

	"github.com/algebananazzzzz/nybble/internal/config"
)

// The Schedule page must build its form and render the timing fields + the Enable
// toggle without panicking.
func TestScheduleScreenRendersForm(t *testing.T) {
	s := &scheduleScreen{cfg: config.Default(), hour: "10", lead: "5"}
	s.buildForm()
	s.form.Init()
	got := s.View(72, 24)
	for _, want := range []string{"Schedule", "Run day", "Notify me", "Enable"} {
		if !strings.Contains(got, want) {
			t.Errorf("schedule view missing %q:\n%s", want, got)
		}
	}
}

// scheduleAction is the submit-diff decision: given the schedule's installed state
// before and after the form, and whether the timing changed, it picks the minimum
// launchd/pmset operation so sudo only fires on a real change.
func TestScheduleActionDiff(t *testing.T) {
	cases := []struct {
		name                     string
		wasOn, nowOn, timingDiff bool
		want                     scheduleOp
	}{
		{"stays off", false, false, false, opNoop},
		{"turn on", false, true, false, opInstall},
		{"turn off", true, false, false, opRemove},
		{"on, unchanged", true, true, false, opNoop},
		{"on, retimed", true, true, true, opReapply},
		{"turn off ignores timing", true, false, true, opRemove},
	}
	for _, c := range cases {
		if got := scheduleAction(c.wasOn, c.nowOn, c.timingDiff); got != c.want {
			t.Errorf("%s: scheduleAction(%v,%v,%v) = %v, want %v",
				c.name, c.wasOn, c.nowOn, c.timingDiff, got, c.want)
		}
	}
}

func TestValidateLead(t *testing.T) {
	bad := []string{"", " ", "0", "-3", "abc", "5x", "9999"}
	for _, v := range bad {
		if validateLead(v) == nil {
			t.Errorf("validateLead(%q) should error", v)
		}
	}
	good := []string{"1", "5", "30", "180"}
	for _, v := range good {
		if err := validateLead(v); err != nil {
			t.Errorf("validateLead(%q) should pass, got %v", v, err)
		}
	}
}
