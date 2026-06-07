package auth

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/algebananazzzzz/nybble/internal/api"
	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/menu"
	"github.com/algebananazzzzz/nybble/internal/session"
)

// parseMenuLocation pulls the pickup point (and, as a fallback, building + meal type)
// out of a menu/v3 response. The pickup lives in lastUsedAddress and is present even
// on days with no service, so it's reliable to read from any menu fetch.
func parseMenuLocation(m *api.MenuResp) (config.NamedCode, config.Pickup, string) {
	var b config.NamedCode
	var p config.Pickup
	meal := ""
	for _, s := range m.Data.MenuSites {
		for _, it := range s.Items {
			if b.Code == "" {
				b.Code = it.BuildingCode
			}
			if meal == "" {
				meal = it.MealType
			}
			if p.Name == "" {
				p.Name = it.PickupAddress
			}
			if p.Code == 0 {
				p.Code = it.PickupAddressID
			}
		}
	}
	if b.Code == "" {
		b.Code = m.Data.BookedOrderInfo.BuildingCode
	}
	b.Name = m.Data.BookedOrderInfo.MealBuilding
	if la := m.Data.LastUsedAddress; la.PickupAddressCode != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(la.PickupAddressCode)); err == nil {
			p.Code = n
		}
		if la.PickupAddressName != "" {
			p.Name = la.PickupAddressName
		}
	}
	if meal == "" {
		meal = m.Data.BookedOrderInfo.TimeCode
	}
	return b, p, meal
}

// detectLocation reads the user's building straight from the API (their saved
// CURRENT_BUILDING_KEY preference) and the pickup point from a menu fetch, then
// persists both to config.json. It needs only a valid session — no menu-opening or
// browser-request scraping — so it works the moment login completes. Best-effort: a
// failure is returned for the caller to surface, never fatal.
func detectLocation(apiBase string, store session.CookieStore) error {
	c := session.ClientFor(store, apiBase, nil)

	code, name, err := c.CurrentBuilding()
	if err != nil {
		return fmt.Errorf("read current building: %w", err)
	}
	if code == "" {
		return fmt.Errorf("no building selected in the canteen app")
	}
	b := config.NamedCode{Code: code, Name: name}
	if b.Name == "" {
		b.Name = code
	}

	meal := "lunch"
	var p config.Pickup
	// Pickup rides in lastUsedAddress on every menu/v3 response (even no-service days),
	// so fetch the first viewable day and read it from there.
	if cal, cerr := c.Calendar(code, meal); cerr == nil {
		for _, day := range menu.ViewableDays(cal) {
			mr, merr := c.Menu(code, day, meal)
			if merr != nil {
				continue
			}
			_, pp, mm := parseMenuLocation(mr)
			if pp.Code != 0 || pp.Name != "" {
				p = pp
			}
			if mm != "" {
				meal = mm
			}
			break
		}
	}
	return saveLocation(b, p, meal)
}

// saveLocation merges the detected building/pickup/meal into config.json, preserving
// the user's other settings (schedule, notify, book days).
func saveLocation(b config.NamedCode, p config.Pickup, meal string) error {
	dir, err := config.ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	cfg, err := config.Load(path)
	if err != nil {
		cfg = config.Default()
	}
	cfg.Building = b
	if p.Code != 0 || p.Name != "" {
		cfg.Pickup = p
	}
	if meal != "" {
		cfg.MealType = meal
	}
	return config.Save(path, cfg)
}
