package run

import (
	"strings"
	"testing"

	"github.com/algebananazzzzz/nybble/internal/api"
)

func TestScanRequiresBuilding(t *testing.T) {
	// No building configured (fresh install): Scan must fail loudly with actionable
	// guidance, not silently return zero dishes/vendors.
	_, _, err := Scan(Deps{})
	if err == nil {
		t.Fatal("Scan with empty building should error")
	}
	if !strings.Contains(err.Error(), "building") {
		t.Fatalf("error should mention the missing building, got: %v", err)
	}
}

func TestDishNamesFromMenus(t *testing.T) {
	menus := []*api.MenuResp{{}}
	menus[0].Data.MenuSites = []api.MenuSite{{Items: []api.Item{
		{Name: "Gyushi - Mala Gyudon"}, {Name: "Jollibee - Chickenjoy"},
	}}}
	names := dishNames(menus)
	if len(names) != 2 || names[0] != "Gyushi - Mala Gyudon" {
		t.Fatalf("bad names: %v", names)
	}
}
