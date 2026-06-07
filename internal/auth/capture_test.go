package auth

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/algebananazzzzz/nybble/internal/api"
)

// parseMenuLocation must pull building + pickup + meal type out of a real menu/v3
// response shape — the data that lets `nybble auth` auto-configure the deployment.
func TestParseMenuLocationFromFixture(t *testing.T) {
	raw, err := os.ReadFile("../../fixtures/menu-v3.sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var m api.MenuResp
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}

	b, p, meal := parseMenuLocation(&m)

	if b.Code != "BLDG00000001" {
		t.Errorf("building code = %q, want BLDG00000001", b.Code)
	}
	if b.Name != "Example Tower" {
		t.Errorf("building name = %q, want Example Tower", b.Name)
	}
	if p.Code != 1185 {
		t.Errorf("pickup code = %d, want 1185", p.Code)
	}
	if p.Name != "Pickup Point" {
		t.Errorf("pickup name = %q, want Pickup Point", p.Name)
	}
	if meal != "lunch" {
		t.Errorf("meal type = %q, want lunch", meal)
	}
}

// When bookedOrderInfo is absent (a user who hasn't reserved), the item rows alone
// must still yield the building code and a name fallback.
func TestParseMenuLocationItemsOnly(t *testing.T) {
	m := &api.MenuResp{}
	m.Data.MenuSites = []api.MenuSite{{Items: []api.Item{
		{BuildingCode: "BLDG42", MealType: "lunch", PickupAddress: "Counter A", PickupAddressID: 7},
	}}}

	b, p, meal := parseMenuLocation(m)

	if b.Code != "BLDG42" {
		t.Errorf("building code = %q, want BLDG42", b.Code)
	}
	if p.Code != 7 || p.Name != "Counter A" {
		t.Errorf("pickup = %+v, want {7 Counter A}", p)
	}
	if meal != "lunch" {
		t.Errorf("meal = %q", meal)
	}
}

// Pickup must be read from lastUsedAddress even when a day has no menu items (e.g. a
// weekend), since that's the common case at detection time.
func TestParseMenuLocationPickupWithoutItems(t *testing.T) {
	m := &api.MenuResp{}
	m.Data.LastUsedAddress.PickupAddressCode = "1185"
	m.Data.LastUsedAddress.PickupAddressName = "Level 26"

	_, p, _ := parseMenuLocation(m)

	if p.Code != 1185 || p.Name != "Level 26" {
		t.Fatalf("pickup = %+v, want {1185 Level 26}", p)
	}
}
