package tui

import (
	"testing"

	"github.com/algebananazzzzz/bytecanteen/internal/config"
)

func TestNotifyOptionsGatedOnLark(t *testing.T) {
	if got := notifyOptions(false); len(got) != 1 || got[0].Value != "off" {
		t.Errorf("lark unavailable: want [off] only, got %d options", len(got))
	}
	got := notifyOptions(true)
	if len(got) != 2 {
		t.Fatalf("lark available: want 2 options, got %d", len(got))
	}
	for i, w := range []string{"lark", "off"} {
		if got[i].Value != w {
			t.Errorf("option[%d] = %q, want %q", i, got[i].Value, w)
		}
	}
}

func TestMigrateChannel(t *testing.T) {
	cases := map[string]string{
		"macos": "off",
		"both":  "lark",
		"lark":  "lark",
		"off":   "off",
		"":      "off",
		"bogus": "off",
	}
	for in, want := range cases {
		if got := migrateChannel(in); got != want {
			t.Errorf("migrateChannel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProbeBuildsFormAndGatesLark(t *testing.T) {
	// Lark unavailable: form built, channel forced Off.
	s := &settings{cfg: withChannel("lark"), hour: "10", phase: phaseProbing}
	m, _ := s.Update(larkProbeMsg{Installed: true, Authed: false, Reason: "bot not configured"})
	ss := m.(*settings)
	if ss.phase != phaseForm || ss.form == nil {
		t.Fatalf("probe should build the form and enter phaseForm, got phase %d form=%v", ss.phase, ss.form != nil)
	}
	if ss.cfg.Notify.Channel != "off" {
		t.Errorf("unavailable lark should force channel=off, got %q", ss.cfg.Notify.Channel)
	}

	// Lark available: channel preserved, form built.
	s2 := &settings{cfg: withChannel("lark"), hour: "10", phase: phaseProbing}
	m2, _ := s2.Update(larkProbeMsg{Installed: true, Authed: true})
	if got := m2.(*settings).cfg.Notify.Channel; got != "lark" {
		t.Errorf("available lark should keep channel=lark, got %q", got)
	}
}

func TestValidateLarkTarget(t *testing.T) {
	bad := map[string]string{
		"empty":     "",
		"spaces":    "   ",
		"no prefix": "abc123",
		"email":     "user@example.com",
	}
	for name, v := range bad {
		if validateLarkTarget(v) == nil {
			t.Errorf("%s (%q): want error, got nil", name, v)
		}
	}
	good := []string{"ou_7ae70037c13a107d25e8b4ed0cd87746", "oc_abc123"}
	for _, v := range good {
		if err := validateLarkTarget(v); err != nil {
			t.Errorf("%q: want ok, got %v", v, err)
		}
	}
}

func TestNormalizeRunDay(t *testing.T) {
	for _, d := range []string{"mon", "tue", "wed", "thu", "fri"} {
		if got := normalizeRunDay(d); got != d {
			t.Errorf("normalizeRunDay(%q) = %q, want unchanged", d, got)
		}
	}
	for _, d := range []string{"sat", "sun", "", "xxx"} {
		if got := normalizeRunDay(d); got != "thu" {
			t.Errorf("normalizeRunDay(%q) = %q, want thu", d, got)
		}
	}
}

func TestValidateBookDays(t *testing.T) {
	if validateBookDays(nil) == nil {
		t.Error("empty selection should error")
	}
	if err := validateBookDays([]string{"mon"}); err != nil {
		t.Errorf("non-empty should pass, got %v", err)
	}
}

func TestDayMultiOptionsAreMonToFri(t *testing.T) {
	opts := dayMultiOptions([]string{"mon"})
	if len(opts) != 5 {
		t.Fatalf("want 5 day options, got %d", len(opts))
	}
	for i, d := range config.Weekdays {
		if opts[i].Value != d {
			t.Errorf("option[%d] = %q, want %q", i, opts[i].Value, d)
		}
	}
}

func withChannel(ch string) config.Config {
	c := config.Default()
	c.Notify.Channel = ch
	return c
}
