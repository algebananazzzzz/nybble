package clock

import "time"

// NextOpen returns the next occurrence of weekday at hh:mm in loc strictly after from.
func NextOpen(from time.Time, weekday time.Weekday, hh, mm int, loc *time.Location) time.Time {
	from = from.In(loc)
	d := (int(weekday) - int(from.Weekday()) + 7) % 7
	cand := time.Date(from.Year(), from.Month(), from.Day(), hh, mm, 0, 0, loc).AddDate(0, 0, d)
	if !cand.After(from) {
		cand = cand.AddDate(0, 0, 7)
	}
	return cand
}

// WaitUntil blocks until t (coarse sleep, then a tight spin for the last 2s).
func WaitUntil(t time.Time) {
	for {
		d := time.Until(t)
		if d <= 0 {
			return
		}
		if d > 2*time.Second {
			time.Sleep(d - 2*time.Second)
		} else {
			time.Sleep(2 * time.Millisecond)
		}
	}
}
