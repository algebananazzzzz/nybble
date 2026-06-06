package clock

import (
	"testing"
	"time"
)

func TestNextOpenIsFutureThursday(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	from := time.Date(2026, 6, 5, 12, 0, 0, 0, loc) // Friday
	got := NextOpen(from, time.Thursday, 10, 0, loc)
	want := time.Date(2026, 6, 11, 10, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
