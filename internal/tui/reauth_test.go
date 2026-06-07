package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestReauthDoesNotAutoLaunch(t *testing.T) {
	r, cmd := newReauth()
	if cmd != nil {
		t.Fatal("reauth must not auto-launch the browser on entry")
	}
	if r.phase != raIdle {
		t.Fatalf("reauth should start idle, got phase %d", r.phase)
	}
}

func TestReauthLaunchesOnEnter(t *testing.T) {
	// Enter only launches when endpoints are configured (otherwise there's nothing to
	// open). The returned Cmd is not executed here, so no browser actually starts.
	t.Setenv("NYBBLE_API_BASE", "https://host/app")
	t.Setenv("NYBBLE_LOGIN_URL", "https://host/login")

	r, _ := newReauth()
	got, cmd := r.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got.(*reauth).phase != raOpening {
		t.Fatalf("Enter should move to raOpening, got %d", got.(*reauth).phase)
	}
	if cmd == nil {
		t.Fatal("Enter should return a launch command")
	}
}

func TestNextAuthStep(t *testing.T) {
	const (
		to    = pollTimeout
		grace = buildingGrace
	)
	cases := []struct {
		name       string
		loggedIn   bool
		building   string
		elapsed    time.Duration
		sinceLogin time.Duration
		want       authStep
	}{
		{"polling, nothing yet", false, "", 5 * time.Second, 0, stepWait},
		{"logged in, no building, within grace", true, "", 5 * time.Second, 3 * time.Second, stepWait},
		{"logged in + building", true, "BLDG1", 5 * time.Second, 1 * time.Second, stepFinalize},
		{"logged in, building never came → finalize past grace", true, "", 30 * time.Second, grace, stepFinalize},
		{"never logged in → fail at timeout", false, "", to, 0, stepFail},
		{"logged-in near timeout still waits out grace", true, "", to, 2 * time.Second, stepWait},
	}
	for _, c := range cases {
		if got := nextAuthStep(c.loggedIn, c.building, c.elapsed, c.sinceLogin, to, grace); got != c.want {
			t.Errorf("%s: nextAuthStep = %d, want %d", c.name, got, c.want)
		}
	}
}
