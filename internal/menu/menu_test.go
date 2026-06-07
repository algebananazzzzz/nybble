package menu

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/algebananazzzzz/nybble/internal/api"
)

func loadMenu(t *testing.T) *api.MenuResp {
	raw, err := os.ReadFile("../../fixtures/menu-v3.sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var m api.MenuResp
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return &m
}

func TestViewableDaysIncludesBookedAndUnreservable(t *testing.T) {
	cal := &api.CalendarResp{}
	cal.Data.Dates = []api.CalendarDate{
		{Date: "2026-06-04", IsHistory: true, ShouldShowMenu: false},                        // past → excluded
		{Date: "2026-06-10", IsHistory: false, ShouldShowMenu: true, HadReserveLunch: true}, // already booked → still viewable
		{Date: "2026-06-16", IsHistory: false, ShouldShowMenu: false},                       // not released → excluded
		{Date: "2026-06-11", IsHistory: false, ShouldShowMenu: true, CanReserve: false},     // viewable, not yet reservable
	}
	got := ViewableDays(cal)
	if len(got) != 2 || got[0] != "2026-06-10" || got[1] != "2026-06-11" {
		t.Fatalf("want [06-10 06-11], got %v", got)
	}
}

func TestBookableWeekdaysKeepsAlreadyBooked(t *testing.T) {
	cal := &api.CalendarResp{}
	cal.Data.Dates = []api.CalendarDate{
		{Date: "2026-06-06", CanReserve: true, HadReserveLunch: false},  // Sat → excluded (weekend)
		{Date: "2026-06-08", CanReserve: true, HadReserveLunch: true},   // Mon, already booked → kept
		{Date: "2026-06-09", CanReserve: true, HadReserveLunch: false},  // Tue, open
		{Date: "2026-06-15", CanReserve: false, HadReserveLunch: false}, // not released → excluded
	}
	got := BookableWeekdays(cal)
	if len(got) != 2 {
		t.Fatalf("want 2 slots, got %v", got)
	}
	if got[0].Date != "2026-06-08" || !got[0].AlreadyBooked {
		t.Errorf("slot0 = %+v, want 06-08 alreadyBooked", got[0])
	}
	if got[1].Date != "2026-06-09" || got[1].AlreadyBooked {
		t.Errorf("slot1 = %+v, want 06-09 open", got[1])
	}
}

func TestItemsFlattensSites(t *testing.T) {
	items := Items(loadMenu(t))
	if len(items) == 0 {
		t.Fatal("expected items from fixture")
	}
	found := false
	for _, it := range items {
		if it.SkuCode != "" && it.Name != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("items missing skuCode/name")
	}
}

func TestVendorParsesNamePrefix(t *testing.T) {
	cases := map[string]string{
		"Gyushi - Mala Gyudon":                 "Gyushi",
		"Jollibee - 2pc Original Chickenjoy":   "Jollibee",
		"Twyst - Chicken Miso Spaghetti - CS5": "Twyst", // only the first " - " splits
		"Chuan Tai Zhi Mala Tang - Beef Bento": "Chuan Tai Zhi Mala Tang",
		"KFC - Zinger Box":                     "KFC",
		"NoSeparatorDish":                      "", // no vendor prefix
	}
	for name, want := range cases {
		if got := Vendor(name); got != want {
			t.Errorf("Vendor(%q) = %q, want %q", name, got, want)
		}
	}
}
