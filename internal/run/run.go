package run

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/algebananazzzzz/nybble/internal/api"
	"github.com/algebananazzzzz/nybble/internal/booker"
	"github.com/algebananazzzzz/nybble/internal/clock"
	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/menu"
	"github.com/algebananazzzzz/nybble/internal/notify"
	"github.com/algebananazzzzz/nybble/internal/selector"
	"github.com/algebananazzzzz/nybble/internal/session"
)

func identityFrom(m map[string]any) (booker.Identity, error) {
	data, _ := m["data"].(map[string]any)
	u, _ := data["user"].(map[string]any)
	if u == nil {
		return booker.Identity{}, fmt.Errorf("no user in user_info")
	}
	s := func(k string) string { v, _ := u[k].(string); return v }
	return booker.Identity{
		OpenID:      s("open_id"),
		UnionID:     s("union_id"),
		TenantEmpID: s("tenant_key") + "#" + s("employee_no"),
	}, nil
}

type Deps struct {
	Cfg       config.Config
	Endpoints config.Endpoints
	Favs      config.Favorites
	Vendors   config.Favorites // ranked vendor fallback (vendors.json)
	Cookies   session.CookieStore
	Notif     notify.Dispatcher
	Now       func() time.Time
	WaitFor   func(time.Time)
}

// Book runs one full booking. Booking is always live; forceDry (the `--dry` CLI
// flag) is a manual preview that selects picks without submitting.
func Book(d Deps, forceDry bool) (booker.Result, error) {
	dry := forceDry

	store, err := session.Refresh(d.Cookies, d.Endpoints.APIBase)
	if err != nil {
		_ = d.Notif.NotifyDetailed("Canteen",
			"Auth expired — run `nybble auth`",
			"🔑 Login expired — run `nybble auth` to reconnect me")
		return booker.Result{}, err
	}
	c := session.ClientFor(store, d.Endpoints.APIBase, nil)

	ui, err := c.UserInfo()
	if err != nil {
		_ = d.Notif.NotifyDetailed("Canteen",
			"Couldn't reach the canteen",
			"😵 Couldn't reach the canteen — I'll need a retry")
		return booker.Result{}, err
	}
	id, err := identityFrom(ui)
	if err != nil {
		_ = d.Notif.NotifyDetailed("Canteen",
			"Couldn't read your profile",
			"😵 Couldn't read your profile — try `nybble auth` again")
		return booker.Result{}, err
	}
	// No Lark target configured → DM the user via their union_id (tenant-stable, so
	// it works even though the bot is a different Lark app than the canteen portal).
	d.Notif.DefaultTarget(id.UnionID)

	loc, err := time.LoadLocation(d.Cfg.Schedule.TZ)
	if err != nil {
		_ = d.Notif.NotifyDetailed("Canteen",
			"Bad timezone in config: "+d.Cfg.Schedule.TZ,
			"🔧 Config issue — bad timezone: "+d.Cfg.Schedule.TZ)
		return booker.Result{}, fmt.Errorf("load timezone %q: %w", d.Cfg.Schedule.TZ, err)
	}
	open := clock.NextOpen(d.Now(), weekday(d.Cfg.Schedule.Weekday), d.Cfg.Schedule.Hour, d.Cfg.Schedule.Minute, loc)
	// Wait for the window to open BEFORE reading menus (pre-open stock is 0).
	// The scheduler fires launchd ~5 min early, so we sit here until the exact open.
	// Notify #1 (warning) at the start of the wait, #2 (fire) the instant it opens.
	// A manual run far from the window skips the wait and just announces "booking now".
	if w := time.Until(open); w > 0 && w < 2*time.Hour {
		mins := int(w.Round(time.Minute).Minutes())
		if mins < 1 {
			mins = 1
		}
		_ = d.Notif.NotifyDetailed("Canteen",
			fmt.Sprintf("Booking starts in %d min", mins),
			fmt.Sprintf("⏰ Lunch booking starts in %d min — warming up…", mins))
		d.WaitFor(open)
	}
	_ = d.Notif.NotifyDetailed("Canteen",
		"Booking now…",
		"🔥 Go time — grabbing your lunches!")

	cal, err := c.Calendar(d.Cfg.Building.Code, d.Cfg.MealType)
	if err != nil {
		_ = d.Notif.NotifyDetailed("Canteen",
			"Couldn't load the menu",
			"😵 Couldn't load the menu — I'll need a retry")
		return booker.Result{}, err
	}
	// All upcoming bookable weekdays, including ones already reserved — so the final
	// notification can report the whole week, not just what this run touched. Then
	// keep only the weekdays the user chose to book; the rest are never touched or
	// reported.
	slots := keepBookDays(menu.BookableWeekdays(cal), d.Cfg.BookSet())
	if len(slots) == 0 {
		_ = d.Notif.NotifyDetailed("Canteen",
			"Nothing to book yet",
			"😴 Nothing to book yet — next week's menu isn't open")
		return booker.Result{DryRun: dry}, nil
	}
	var targetDays []string
	for _, s := range slots {
		if !s.AlreadyBooked {
			targetDays = append(targetDays, s.Date)
		}
	}

	pickup := booker.Pickup{Code: d.Cfg.Pickup.Code, Name: d.Cfg.Pickup.Name}

	// rerank tracking: a vendor fallback or any in-stock dish missing from favorites
	// means the menu drifted from the ranking, so the run nudges the user to re-rank.
	rerankNeeded := false
	unrankedSet := map[string]bool{}
	var unranked []string

	// selectFor builds an order for a day from the CURRENT live menu under the
	// two-tier policy (favorite dish first, vendor fallback if none in stock).
	// Re-running it after a sold-out makes Choose skip the now-zero-stock pick and
	// fall to the next favorite (or the vendor fallback) automatically.
	selectFor := func(day string) (api.Order, bool) {
		mr, err := c.Menu(d.Cfg.Building.Code, day, d.Cfg.MealType)
		if err != nil {
			return api.Order{}, false
		}
		p := selector.Choose(menu.Items(mr), d.Favs, d.Vendors)
		if p.FellBack {
			rerankNeeded = true
		}
		for _, name := range p.Unranked {
			if !unrankedSet[name] {
				unrankedSet[name] = true
				unranked = append(unranked, name)
			}
		}
		if p.Item == nil {
			return api.Order{}, false
		}
		return booker.BuildOrder(*p.Item, id, d.Cfg.Building.Name, pickup), true
	}

	// freshDish maps date -> dish booked (or, in dry-run, would-be booked) THIS run.
	freshDish := map[string]string{}
	if dry {
		// Single selection pass, no submit.
		for _, day := range targetDays {
			if o, ok := selectFor(day); ok {
				freshDish[day] = o.FoodName
			}
		}
	} else {
		// Submit, then retry unbooked days with the next available favorite until all
		// booked or the deadline. resp.Code != 200 (logical failure, e.g. not-open-yet)
		// triggers a retry.
		deadline := open.Add(60 * time.Second)
		for attempt := 0; attempt < 8 && d.Now().Before(deadline); attempt++ {
			var orders []api.Order
			for _, day := range targetDays {
				if _, done := freshDish[day]; done {
					continue
				}
				if o, ok := selectFor(day); ok {
					orders = append(orders, o)
				}
			}
			if len(orders) == 0 {
				time.Sleep(150 * time.Millisecond)
				continue
			}
			resp, serr := c.Submit(booker.BuildBatch(orders))
			if serr != nil || resp.Code != 200 {
				time.Sleep(150 * time.Millisecond)
				continue
			}
			for _, s := range resp.Data.SuccessOrders {
				freshDish[s.MealDate] = s.FoodName
			}
			allDone := true
			for _, day := range targetDays {
				if _, ok := freshDish[day]; !ok {
					allDone = false
					break
				}
			}
			if allDone {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
	}

	// bookedDish reads the dish already reserved for a day from its menu (menu/v3
	// carries bookedOrderInfo for days you've booked). Empty on any error.
	bookedDish := func(day string) string {
		mr, err := c.Menu(d.Cfg.Building.Code, day, d.Cfg.MealType)
		if err != nil {
			return ""
		}
		return mr.Data.BookedOrderInfo.FoodName
	}

	// Aggregate the whole week for the notification (and res for stdout/the log).
	plan := make([]weekLunch, 0, len(slots))
	res := booker.Result{DryRun: dry}
	for _, s := range slots {
		switch {
		case s.AlreadyBooked:
			plan = append(plan, weekLunch{date: s.Date, dish: bookedDish(s.Date), state: "booked"})
		case freshDish[s.Date] != "":
			plan = append(plan, weekLunch{date: s.Date, dish: freshDish[s.Date], state: "new"})
			res.Booked = append(res.Booked, s.Date+" "+freshDish[s.Date])
		default:
			plan = append(plan, weekLunch{date: s.Date, state: "miss"})
			res.Failed = append(res.Failed, s.Date+" (unbooked)")
		}
	}
	_ = d.Notif.NotifyDetailed("Canteen", planSummary(plan, dry), planDetail(plan, dry))

	// Nudge a re-rank when the menu drifted: a vendor fallback fired, or in-stock
	// dishes showed up that aren't in the favorites ranking.
	if rerankNeeded || len(unranked) > 0 {
		_ = d.Notif.NotifyDetailed("Canteen",
			rerankSummary(rerankNeeded), rerankDetail(rerankNeeded, unranked))
	}
	return res, nil
}

// rerankSummary is the plain desktop line prompting a favorites re-rank.
func rerankSummary(fellBack bool) string {
	if fellBack {
		return "Menu changed — booked a fallback, re-rank favorites"
	}
	return "Menu changed — new dishes to rank"
}

// rerankDetail lists the in-stock dishes that match no favorite so the user knows
// what to slot into their ranking.
func rerankDetail(fellBack bool, unranked []string) string {
	head := "🆕 Menu changed — open `nybble` to re-rank your favorites."
	if fellBack {
		head = "↩️ No favorite was in stock — I booked a fallback. Re-rank in `nybble`."
	}
	lines := []string{head}
	for _, n := range unranked {
		lines = append(lines, "• "+n)
	}
	return strings.Join(lines, "\n")
}

// weekLunch is one bookable weekday's outcome for the aggregate notification.
// state: "new" (booked this run, dish known), "booked" (already reserved before this
// run, dish unknown), "miss" (open but couldn't book).
type weekLunch struct {
	date  string
	dish  string
	state string
}

// covered counts days that end up reserved (already-booked + freshly booked).
func covered(plan []weekLunch) (have, total, newly int) {
	total = len(plan)
	for _, w := range plan {
		if w.state != "miss" {
			have++
		}
		if w.state == "new" {
			newly++
		}
	}
	return
}

// planSummary is the plain, compact desktop status (no emoji): how much of next week
// is booked, plus how many this run just grabbed.
func planSummary(plan []weekLunch, dry bool) string {
	have, total, newly := covered(plan)
	if dry {
		return fmt.Sprintf("Dry run — next week %d/%d covered", have, total)
	}
	s := fmt.Sprintf("Next week: %d/%d lunches booked", have, total)
	if newly > 0 {
		s += fmt.Sprintf(" (+%d new)", newly)
	}
	return s
}

// planDetail is the cute Lark body: an emoji headline plus every weekday next week —
// freshly booked dishes by name, already-booked days as "(booked)", misses as "(not
// booked)". Unicode emoji + the [赞] Lark sticker both render in Lark.
func planDetail(plan []weekLunch, dry bool) string {
	have, total, _ := covered(plan)
	var head string
	switch {
	case dry:
		head = "👀 Dry run — next week's lunch plan:"
	case have == 0:
		head = "😭 Couldn't book next week — everything sold out"
	case have == total:
		head = fmt.Sprintf("🍱 All set for next week — %d/%d lunches booked! [赞]", have, total)
	default:
		head = fmt.Sprintf("🍱 Next week — %d/%d lunches booked", have, total)
	}
	lines := []string{head}
	for _, w := range plan {
		label := prettyDate(w.date)
		switch {
		case w.state == "new":
			lines = append(lines, "✅ "+label+"  "+w.dish+"  (just booked)")
		case w.state == "booked" && w.dish != "":
			lines = append(lines, "✅ "+label+"  "+w.dish)
		case w.state == "booked":
			lines = append(lines, "✅ "+label+"  (booked)")
		default:
			lines = append(lines, "❌ "+label+"  (not booked)")
		}
	}
	return strings.Join(lines, "\n")
}

// prettyDate turns "2026-06-10" into "Wed Jun 10"; on a parse error it returns the input.
func prettyDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("Mon Jan 2")
}

// weekdayCode maps a YYYY-MM-DD date to a lowercase 3-letter weekday code
// (mon..sun); "" on a parse error.
func weekdayCode(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return [...]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}[t.Weekday()]
}

// keepBookDays filters lunch slots to the weekdays the user chose to book. The set
// comes from config.BookSet (empty config → all Mon–Fri), so this only ever drops
// days the user explicitly deselected.
func keepBookDays(slots []menu.LunchSlot, set map[string]bool) []menu.LunchSlot {
	var out []menu.LunchSlot
	for _, s := range slots {
		if set[weekdayCode(s.Date)] {
			out = append(out, s)
		}
	}
	return out
}

func weekday(s string) time.Weekday {
	switch strings.ToLower(s) {
	case "mon":
		return time.Monday
	case "tue":
		return time.Tuesday
	case "wed":
		return time.Wednesday
	case "thu":
		return time.Thursday
	case "fri":
		return time.Friday
	case "sat":
		return time.Saturday
	default:
		return time.Sunday
	}
}

// LoadDeps assembles Deps from config/cookies on disk.
func LoadDeps() (Deps, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return Deps{}, err
	}
	cfg, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		cfg = config.Default()
	}
	eps, err := config.LoadEndpoints()
	if err != nil {
		return Deps{}, err
	}
	favs, _ := config.LoadFavorites(filepath.Join(dir, "favorites.json"))
	vendors, _ := config.LoadFavorites(filepath.Join(dir, "vendors.json"))
	cookies, err := session.LoadCookies(filepath.Join(dir, "cookies.json"))
	if err != nil {
		return Deps{}, fmt.Errorf("not logged in: run `nybble auth`")
	}
	disp := notify.Dispatcher{}
	if cfg.Notify.LarkOn() {
		disp.Enabled = true
		disp.Lark = notify.Lark{Target: cfg.Notify.LarkTarget}
	}
	return Deps{
		Cfg: cfg, Endpoints: eps, Favs: favs, Vendors: vendors, Cookies: cookies,
		Notif: disp,
		Now:   time.Now, WaitFor: clock.WaitUntil,
	}, nil
}
