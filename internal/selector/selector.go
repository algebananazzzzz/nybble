package selector

import (
	"strings"

	"github.com/algebananazzzzz/nybble/internal/api"
	"github.com/algebananazzzzz/nybble/internal/menu"
)

func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Pick is the outcome of a selection: the chosen item (nil = nothing bookable),
// whether it came from the vendor fallback, and the in-stock dishes that match no
// favorite — the signal that the menu changed and favorites need re-ranking.
type Pick struct {
	Item     *api.Item
	FellBack bool
	Unranked []string
}

// matchesFav reports whether a dish name matches any favorite, exact or substring
// (both normalized).
func matchesFav(name string, favorites []string) bool {
	n := norm(name)
	for _, fav := range favorites {
		f := norm(fav)
		if n == f || strings.Contains(n, f) {
			return true
		}
	}
	return false
}

// Choose selects the dish to book under a strict two-tier policy:
//
//	Tier 1 (default): the highest-ranked in-stock favorite dish, regardless of
//	vendor — a favorite from a low-ranked vendor still beats a non-favorite from
//	the top vendor. Match precedence: exact (normalized) name, then substring.
//
//	Tier 2 (fallback): when NO favorite dish is in stock, pick from the
//	highest-ranked vendor that has any in-stock dish (unranked vendors last). The
//	vendor comes from the dish name ("Vendor - Dish"), since the API exposes only
//	the floor as a site. Within a vendor, take the LEAST-stocked in-stock dish: the
//	scarcest item is the canteen's best signal for the most-wanted one. With no
//	vendor ranking — or no favorites and no vendors configured at all — every dish
//	ties on vendor, so this reduces to "the least-stocked dish on the menu".
//
// Guarantee: as long as ANY dish is in stock, Choose returns one. It books nothing
// only when the entire menu is sold out. The retry loop in run.Book re-runs Choose
// on a fresh menu after a sold-out, so a scarce pick that loses the race falls
// through to the next-scarcest automatically.
//
// Unranked lists in-stock dishes matching no favorite, so callers can prompt a
// re-rank when the menu changes.
func Choose(items []api.Item, favorites, vendors []string) Pick {
	inStock := func(it api.Item) bool { return it.CurrentStock > 0 }

	var unranked []string
	seenUnranked := map[string]bool{}
	for i := range items {
		it := items[i]
		if inStock(it) && it.Name != "" && !matchesFav(it.Name, favorites) && !seenUnranked[it.Name] {
			seenUnranked[it.Name] = true
			unranked = append(unranked, it.Name)
		}
	}

	// Tier 1: ranked favorite dishes, exact before substring, across all vendors.
	for _, fav := range favorites {
		f := norm(fav)
		for i := range items {
			if inStock(items[i]) && norm(items[i].Name) == f {
				return Pick{Item: &items[i], Unranked: unranked}
			}
		}
		for i := range items {
			if inStock(items[i]) && strings.Contains(norm(items[i].Name), f) {
				return Pick{Item: &items[i], Unranked: unranked}
			}
		}
	}

	// Tier 2: vendor fallback. rank(name) is the position of the dish's vendor
	// (lower = better), len(vendors) for any vendor not in the ranking — so with no
	// vendors configured every dish ranks equal and the stock tie-break alone decides.
	rank := func(name string) int {
		v := norm(menu.Vendor(name))
		for i, ranked := range vendors {
			if norm(ranked) == v {
				return i
			}
		}
		return len(vendors)
	}

	best := -1
	for i := range items {
		if !inStock(items[i]) {
			continue
		}
		if best < 0 {
			best = i
			continue
		}
		ri, rb := rank(items[i].Name), rank(items[best].Name)
		switch {
		case ri < rb:
			best = i
		case ri == rb && items[i].CurrentStock < items[best].CurrentStock:
			best = i
		}
	}
	if best < 0 {
		return Pick{Unranked: unranked} // entire menu sold out
	}
	return Pick{Item: &items[best], FellBack: true, Unranked: unranked}
}
