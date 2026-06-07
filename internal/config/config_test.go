package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultLeadMin(t *testing.T) {
	if got := Default().Schedule.LeadMin; got != DefaultLeadMin {
		t.Errorf("default LeadMin = %d, want %d", got, DefaultLeadMin)
	}
}

// A legacy config.json predating LeadMin unmarshals it to 0; Lead() must fall back to
// the default so a scheduled run keeps its sensible heads-up lead.
func TestScheduleLeadFallsBackWhenZero(t *testing.T) {
	if got := (Schedule{}).Lead(); got != DefaultLeadMin {
		t.Errorf("zero LeadMin should fall back to %d, got %d", DefaultLeadMin, got)
	}
	if got := (Schedule{LeadMin: 9}).Lead(); got != 9 {
		t.Errorf("set LeadMin should pass through, got %d", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Default()
	c.BookDays = []string{"mon", "wed", "fri"}
	c.Building.Code = "BLDG00000001"
	if err := Save(filepath.Join(dir, "config.json"), c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.BookDays, []string{"mon", "wed", "fri"}) || got.Building.Code != "BLDG00000001" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestDefaultBooksAllWeekdays(t *testing.T) {
	if !reflect.DeepEqual(Default().BookDays, Weekdays) {
		t.Fatalf("default should book all weekdays, got %v", Default().BookDays)
	}
}

func TestBookSetFallsBackToFullWeekWhenEmpty(t *testing.T) {
	// A pre-BookDays config.json unmarshals to an empty slice; it must still book.
	set := Config{}.BookSet()
	for _, d := range Weekdays {
		if !set[d] {
			t.Errorf("empty BookDays should fall back to all weekdays, missing %q", d)
		}
	}
	if len(set) != len(Weekdays) {
		t.Errorf("fallback set size = %d, want %d", len(set), len(Weekdays))
	}
}
