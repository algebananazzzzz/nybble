package run

import (
	"testing"

	"github.com/algebananazzzzz/nybble/internal/menu"
)

func TestWeekdayCode(t *testing.T) {
	cases := map[string]string{
		"2026-06-08": "mon", "2026-06-09": "tue", "2026-06-10": "wed",
		"2026-06-11": "thu", "2026-06-12": "fri", "2026-06-13": "sat",
		"not-a-date": "",
	}
	for date, want := range cases {
		if got := weekdayCode(date); got != want {
			t.Errorf("weekdayCode(%q) = %q, want %q", date, got, want)
		}
	}
}

func TestKeepBookDays(t *testing.T) {
	slots := []menu.LunchSlot{
		{Date: "2026-06-08"}, {Date: "2026-06-09"}, {Date: "2026-06-10"},
		{Date: "2026-06-11"}, {Date: "2026-06-12"},
	}
	set := map[string]bool{"mon": true, "wed": true, "fri": true}
	got := keepBookDays(slots, set)
	if len(got) != 3 {
		t.Fatalf("want 3 kept slots, got %d", len(got))
	}
	for _, s := range got {
		if !set[weekdayCode(s.Date)] {
			t.Errorf("kept unselected day %s (%s)", s.Date, weekdayCode(s.Date))
		}
	}
}

// a full week: two already booked (dish known), one fresh booking, one booked with
// unknown dish (menu fetch failed), one miss.
func sampleWeek() []weekLunch {
	return []weekLunch{
		{date: "2026-06-08", dish: "Pizza Hut - Ham 'n' Shroom", state: "booked"},
		{date: "2026-06-09", dish: "Shake Shack - ShackBurger", state: "booked"},
		{date: "2026-06-10", dish: "Gyushi - Mala Gyudon", state: "new"},
		{date: "2026-06-11", state: "booked"},
		{date: "2026-06-12", state: "miss"},
	}
}

func TestPlanSummary(t *testing.T) {
	if got := planSummary(sampleWeek(), false); got != "Next week: 4/5 lunches booked (+1 new)" {
		t.Errorf("live: got %q", got)
	}
	allBooked := []weekLunch{{state: "booked"}, {state: "booked"}}
	if got := planSummary(allBooked, false); got != "Next week: 2/2 lunches booked" {
		t.Errorf("all booked: got %q", got)
	}
	if got := planSummary(sampleWeek(), true); got != "Dry run — next week 4/5 covered" {
		t.Errorf("dry: got %q", got)
	}
}

func TestPlanDetail(t *testing.T) {
	want := "🍱 Next week — 4/5 lunches booked\n" +
		"✅ Mon Jun 8  Pizza Hut - Ham 'n' Shroom\n" +
		"✅ Tue Jun 9  Shake Shack - ShackBurger\n" +
		"✅ Wed Jun 10  Gyushi - Mala Gyudon  (just booked)\n" +
		"✅ Thu Jun 11  (booked)\n" +
		"❌ Fri Jun 12  (not booked)"
	if got := planDetail(sampleWeek(), false); got != want {
		t.Errorf("planDetail:\n got %q\nwant %q", got, want)
	}
	allBooked := []weekLunch{{date: "2026-06-08", dish: "KFC - Zinger Box", state: "booked"}, {date: "2026-06-09", state: "booked"}}
	if got := planDetail(allBooked, false); got != "🍱 All set for next week — 2/2 lunches booked! [赞]\n✅ Mon Jun 8  KFC - Zinger Box\n✅ Tue Jun 9  (booked)" {
		t.Errorf("all booked: got %q", got)
	}
}

func TestIdentityFromUserInfo(t *testing.T) {
	m := map[string]any{"data": map[string]any{"user": map[string]any{
		"open_id": "ou_a", "union_id": "on_b", "tenant_key": "tk", "employee_no": "999",
	}}}
	id, err := identityFrom(m)
	if err != nil {
		t.Fatal(err)
	}
	if id.OpenID != "ou_a" || id.UnionID != "on_b" || id.TenantEmpID != "tk#999" {
		t.Fatalf("bad identity: %+v", id)
	}
}
