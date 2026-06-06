package run

import (
	"fmt"
	"path/filepath"

	"github.com/algebananazzzzz/bytecanteen/internal/api"
	"github.com/algebananazzzzz/bytecanteen/internal/catalog"
	"github.com/algebananazzzzz/bytecanteen/internal/config"
	"github.com/algebananazzzzz/bytecanteen/internal/menu"
	"github.com/algebananazzzzz/bytecanteen/internal/session"
)

func dishNames(menus []*api.MenuResp) []string {
	var out []string
	for _, m := range menus {
		for _, it := range menu.Items(m) {
			if it.Name != "" {
				out = append(out, it.Name)
			}
		}
	}
	return out
}

// vendorNames returns the distinct vendors across the scanned menus, parsed from
// the dish-name prefixes ("Vendor - Dish"), in first-seen order.
func vendorNames(menus []*api.MenuResp) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range menus {
		for _, it := range menu.Items(m) {
			v := menu.Vendor(it.Name)
			if v != "" && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// mergeVendors appends newly-seen vendors to vendors.json (preserving the user's
// existing ranking) and returns the merged list.
func mergeVendors(dir string, found []string) (config.Favorites, error) {
	vpath := filepath.Join(dir, "vendors.json")
	vendors, _ := config.LoadFavorites(vpath)
	have := map[string]bool{}
	for _, v := range vendors {
		have[v] = true
	}
	for _, v := range found {
		if !have[v] {
			vendors = append(vendors, v)
			have[v] = true
		}
	}
	if err := config.SaveFavorites(vpath, vendors); err != nil {
		return nil, err
	}
	return vendors, nil
}

// Menu fetches upcoming-week menus, prints them, and updates catalog.json.
func Menu(d Deps) error {
	store, err := session.Refresh(d.Cookies, d.Endpoints.APIBase)
	if err != nil {
		return err
	}
	c := session.ClientFor(store, d.Endpoints.APIBase, nil)
	cal, err := c.Calendar(d.Cfg.Building.Code, d.Cfg.MealType)
	if err != nil {
		return err
	}
	// Scrape every browsable day (weekday menus are visible before their booking
	// window opens and after you've reserved) so the catalog gets the full dish names.
	var menus []*api.MenuResp
	for _, day := range menu.ViewableDays(cal) {
		mr, err := c.Menu(d.Cfg.Building.Code, day, d.Cfg.MealType)
		if err != nil {
			continue
		}
		items := menu.Items(mr)
		if len(items) == 0 {
			continue // skip days with no lunch service (e.g. weekends)
		}
		menus = append(menus, mr)
		fmt.Printf("\n%s:\n", day)
		for _, it := range items {
			fmt.Printf("  [%d left] %s\n", it.CurrentStock, it.Name)
		}
	}
	dir, _ := config.ConfigDir()
	cpath := filepath.Join(dir, "catalog.json")
	cat, _ := catalog.Load(cpath)
	cat = cat.Add(dishNames(menus))
	return catalog.Save(cpath, cat)
}

// Scan fetches the upcoming-week menus and merges their dish names into
// catalog.json and their vendor labels into vendors.json, returning the updated
// dish catalog and vendor ranking. Unlike Menu it prints nothing — it backs the
// TUI's on-demand "rescan" action.
func Scan(d Deps) (catalog.Catalog, config.Favorites, error) {
	store, err := session.Refresh(d.Cookies, d.Endpoints.APIBase)
	if err != nil {
		return nil, nil, err
	}
	c := session.ClientFor(store, d.Endpoints.APIBase, nil)
	cal, err := c.Calendar(d.Cfg.Building.Code, d.Cfg.MealType)
	if err != nil {
		return nil, nil, err
	}
	var menus []*api.MenuResp
	for _, day := range menu.ViewableDays(cal) {
		mr, err := c.Menu(d.Cfg.Building.Code, day, d.Cfg.MealType)
		if err != nil || len(menu.Items(mr)) == 0 {
			continue
		}
		menus = append(menus, mr)
	}
	dir, _ := config.ConfigDir()
	cpath := filepath.Join(dir, "catalog.json")
	cat, _ := catalog.Load(cpath)
	cat = cat.Add(dishNames(menus))
	if err := catalog.Save(cpath, cat); err != nil {
		return nil, nil, err
	}
	vendors, err := mergeVendors(dir, vendorNames(menus))
	if err != nil {
		return nil, nil, err
	}
	return cat, vendors, nil
}
