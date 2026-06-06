package menu

import (
	"strings"
	"time"

	"github.com/algebananazzzzz/bytecanteen/internal/api"
)

// Items flattens menuSites[].items[] into a single slice.
func Items(m *api.MenuResp) []api.Item {
	var out []api.Item
	for _, s := range m.Data.MenuSites {
		out = append(out, s.Items...)
	}
	return out
}

// Vendor extracts the vendor (stall) from a dish name. Canteen dishes are named
// "Vendor - Dish" (e.g. "Jollibee - 2pc Chickenjoy"); the vendor is the text
// before the first " - ". The API's menu sites only carry the floor/pickup
// location ("Pickup Point"), not the vendor, so the name is the only source.
// Returns "" when the name has no vendor prefix.
func Vendor(dishName string) string {
	if i := strings.Index(dishName, " - "); i >= 0 {
		return strings.TrimSpace(dishName[:i])
	}
	return ""
}

// OpenLunchDays returns dates that are reservable and not already booked for lunch.
func OpenLunchDays(c *api.CalendarResp) []string {
	var out []string
	for _, d := range c.Data.Dates {
		if d.CanReserve && !d.HadReserveLunch {
			out = append(out, d.Date)
		}
	}
	return out
}

// LunchSlot is one bookable weekday in the upcoming window.
type LunchSlot struct {
	Date          string
	AlreadyBooked bool
}

// BookableWeekdays returns the upcoming weekdays whose booking window is open
// (canReserve), each flagged with whether lunch is already reserved. Unlike
// OpenLunchDays it keeps days already booked, so callers can report the whole week
// (e.g. "5/5 booked"), not only what's left to book.
func BookableWeekdays(c *api.CalendarResp) []LunchSlot {
	var out []LunchSlot
	for _, d := range c.Data.Dates {
		if d.CanReserve && IsWeekday(d.Date) {
			out = append(out, LunchSlot{Date: d.Date, AlreadyBooked: d.HadReserveLunch})
		}
	}
	return out
}

// ViewableDays returns dates whose menu is browsable now (shouldShowMenu and not
// history), regardless of whether they are reservable yet or already booked. Used
// to scrape the dish catalog — the menu for a weekday is visible before its booking
// window opens and even after you've reserved it.
func ViewableDays(c *api.CalendarResp) []string {
	var out []string
	for _, d := range c.Data.Dates {
		if d.ShouldShowMenu && !d.IsHistory {
			out = append(out, d.Date)
		}
	}
	return out
}

// IsWeekday reports whether an ISO date (YYYY-MM-DD) is Mon–Fri.
func IsWeekday(date string) bool {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	wd := t.Weekday()
	return wd >= time.Monday && wd <= time.Friday
}
