package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAboutViewRenders(t *testing.T) {
	a := newAbout(State{LoggedIn: true, FavCount: 9, Building: "Example Tower", NotifyCh: "lark"})
	out := a.View(80, 20)
	for _, want := range []string{"Endpoints", "Status", "Paths", "Example Tower", "9 dishes"} {
		if !strings.Contains(out, want) {
			t.Errorf("about view missing %q", want)
		}
	}
}

func TestAboutBackReturnsToDashboard(t *testing.T) {
	a := newAbout(State{})
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should navigate back")
	}
	if msg, ok := cmd().(navMsg); !ok || msg.to != scrDashboard {
		t.Fatal("esc should return to the dashboard")
	}
}

func TestDashboardAboutItemNavigates(t *testing.T) {
	d := newDashboard()
	var idx int = -1
	for i, it := range d.items {
		if it.label == "About" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("dashboard has no About item")
	}
	d.cursor = idx
	_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on About should navigate")
	}
	if msg, ok := cmd().(navMsg); !ok || msg.to != scrAbout {
		t.Fatal("About should navigate to scrAbout")
	}
}

func TestDashboardGroupsCoverAllItems(t *testing.T) {
	d := newDashboard()
	groups := d.groups()
	seen := 0
	for _, g := range groups {
		if g.title == "" {
			t.Error("panel group missing a title")
		}
		seen += len(g.idx)
	}
	if seen != len(d.items) {
		t.Errorf("groups cover %d items, want %d", seen, len(d.items))
	}
}
