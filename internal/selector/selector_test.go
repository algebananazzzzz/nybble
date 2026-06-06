package selector

import (
	"testing"

	"github.com/algebananazzzzz/bytecanteen/internal/api"
)

func items() []api.Item {
	return []api.Item{
		{Name: "Gyushi - Mala Gyudon", SkuCode: "a", CurrentStock: 0},
		{Name: "Jollibee - 2pc Chickenjoy", SkuCode: "b", CurrentStock: 5},
		{Name: "Indochilli - Beef Rendang Set", SkuCode: "c", CurrentStock: 3},
	}
}

func TestPicksHighestRankedInStock(t *testing.T) {
	favs := []string{"mala", "chickenjoy", "rendang"}
	got := Choose(items(), favs, nil)
	if got.Item == nil || got.Item.SkuCode != "b" {
		t.Fatalf("mala is sold out → expect chickenjoy (b), got %+v", got.Item)
	}
	if got.FellBack {
		t.Fatal("a favorite matched → should not be a vendor fallback")
	}
}

func TestExactBeatsSubstring(t *testing.T) {
	favs := []string{"Indochilli - Beef Rendang Set"}
	got := Choose(items(), favs, nil)
	if got.Item == nil || got.Item.SkuCode != "c" {
		t.Fatalf("exact name should match, got %+v", got.Item)
	}
}

func TestFavoriteBeatsBetterVendor(t *testing.T) {
	// Favorite dish lives at a low-ranked vendor; a non-favorite sits at the top
	// vendor. Dish ranking is absolute → the favorite wins anyway.
	its := []api.Item{
		{Name: "StarStall - Random Wrap", SkuCode: "x", CurrentStock: 9},
		{Name: "Corner - Beef Rendang", SkuCode: "y", CurrentStock: 2},
	}
	got := Choose(its, []string{"rendang"}, []string{"StarStall", "Corner"})
	if got.Item == nil || got.Item.SkuCode != "y" || got.FellBack {
		t.Fatalf("favorite dish must win over a better vendor, got %+v", got)
	}
}

func TestVendorFallbackPicksTopVendorMostStock(t *testing.T) {
	// No favorites in stock → fall back to the highest-ranked vendor, and within
	// it the most-stocked dish. Vendor is parsed from the dish-name prefix.
	its := []api.Item{
		{Name: "Corner - Soup", SkuCode: "lo", CurrentStock: 9},
		{Name: "StarStall - Wrap", SkuCode: "hi1", CurrentStock: 2},
		{Name: "StarStall - Bowl", SkuCode: "hi2", CurrentStock: 7},
	}
	got := Choose(its, []string{"pizza"}, []string{"StarStall", "Corner"})
	if got.Item == nil || !got.FellBack {
		t.Fatalf("expected vendor fallback, got %+v", got)
	}
	if got.Item.SkuCode != "hi2" {
		t.Fatalf("top vendor's most-stocked dish should win, got %+v", got.Item)
	}
}

func TestNoFavoriteFallsBackToMostStocked(t *testing.T) {
	// Favorites configured but none in stock, no vendor ranking → fall back to the
	// most-stocked dish (chickenjoy, 5).
	got := Choose(items(), []string{"pizza"}, nil)
	if got.Item == nil || !got.FellBack || got.Item.SkuCode != "b" {
		t.Fatalf("want most-stocked fallback (b), got %+v", got)
	}
}

func TestUnconfiguredBooksNothing(t *testing.T) {
	// No favorites AND no vendors → don't grab a random dish.
	if got := Choose(items(), nil, nil); got.Item != nil {
		t.Fatalf("unconfigured user should book nothing, got %+v", got.Item)
	}
}

func TestUnrankedReportsMenuChanges(t *testing.T) {
	favs := []string{"chickenjoy"} // only b matches
	got := Choose(items(), favs, nil)
	// In stock and unmatched: Rendang (c). Mala (a) is sold out → excluded.
	if len(got.Unranked) != 1 || got.Unranked[0] != "Indochilli - Beef Rendang Set" {
		t.Fatalf("unranked = %v, want [rendang]", got.Unranked)
	}
}
