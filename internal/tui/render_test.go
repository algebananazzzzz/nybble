package tui

import (
	"strings"
	"testing"
)

func TestDashboardViewRenders(t *testing.T) {
	m := New()
	m.state = State{LoggedIn: true, FavCount: 5, Building: "Example Tower", NotifyCh: "lark"}
	out := m.View()
	if !strings.Contains(out, "nybble") || !strings.Contains(out, "Favorites & menu") {
		t.Fatalf("dashboard view missing expected content")
	}
}

func TestFavoritesViewRenders(t *testing.T) {
	f := newFavModel()
	// Opens on the Vendors view; tab cycles to Dishes then Deleted.
	if !strings.Contains(f.View(70, 12), "Vendors") {
		t.Fatal("favorites view missing Vendors title")
	}
	f.cycle()
	if !strings.Contains(f.View(70, 12), "Dishes") {
		t.Fatal("dishes view missing title after cycle")
	}
}
