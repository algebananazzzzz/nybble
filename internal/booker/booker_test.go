package booker

import (
	"testing"

	"github.com/algebananazzzzz/nybble/internal/api"
)

func TestBuildOrderMapsItemFields(t *testing.T) {
	id := Identity{OpenID: "ou_x", UnionID: "on_y", TenantEmpID: "t#1"}
	pickup := Pickup{Code: 1185, Name: "Pickup Point"}
	it := api.Item{
		SkuCode: "sku1", Name: "Gyushi - Mala Gyudon", LabelMealCode: "IOTE_0001",
		BuildingCode: "BLDG00000001", MealDate: "2026-06-10", MealType: "lunch",
		MealStartTime: "12:00", MealEndTime: "14:00",
	}
	o := BuildOrder(it, id, "Example Tower", pickup)
	if o.SkuCode != "sku1" || o.FoodName != "Gyushi - Mala Gyudon" {
		t.Fatalf("bad dish mapping: %+v", o)
	}
	if o.OpenID != "ou_x" || o.TenantEmpID != "t#1" {
		t.Fatalf("bad identity mapping: %+v", o)
	}
	if o.PickupAddressCode != 1185 || o.BuildingName != "Example Tower" {
		t.Fatalf("bad pickup/building: %+v", o)
	}
}
