package run

import (
	"testing"

	"github.com/algebananazzzzz/bytecanteen/internal/api"
)

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
