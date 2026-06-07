package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func clearIndex(d *dashboard) int {
	for i, it := range d.items {
		if it.clear {
			return i
		}
	}
	return -1
}

func TestDashboardClearConfirmThenRun(t *testing.T) {
	// Isolate any disk effect if the returned Cmd is executed (it isn't here).
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	d := newDashboard()
	idx := clearIndex(d)
	if idx < 0 {
		t.Fatal("dashboard has no Clear all data item")
	}
	d.cursor = idx

	// First Enter only asks for confirmation — no command yet.
	got, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dd := got.(*dashboard)
	if !dd.confirmClear || cmd != nil {
		t.Fatal("Clear should ask for confirmation before acting")
	}

	// 'y' confirms → returns the clear command and resets the flag.
	got, cmd = dd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y should trigger the clear command")
	}
	if got.(*dashboard).confirmClear {
		t.Fatal("confirm flag should clear after y")
	}
}

func TestDashboardClearCancel(t *testing.T) {
	d := newDashboard()
	d.cursor = clearIndex(d)

	d.Update(tea.KeyMsg{Type: tea.KeyEnter}) // enter confirmation
	got, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if got.(*dashboard).confirmClear {
		t.Fatal("esc should cancel the clear confirmation")
	}
	if cmd != nil {
		t.Fatal("cancelling clear must not run it")
	}
}

func TestDashboardClearDoneShowsNotice(t *testing.T) {
	d := newDashboard()
	got, cmd := d.Update(clearDoneMsg{})
	if cmd == nil {
		t.Fatal("clearDone should refresh state")
	}
	if got.(*dashboard).notice == "" {
		t.Fatal("clearDone should set a notice")
	}
}
