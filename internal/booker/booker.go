package booker

import (
	"github.com/algebananazzzzz/nybble/internal/api"
)

type Identity struct {
	OpenID      string
	UnionID     string
	TenantEmpID string
}
type Pickup struct {
	Code int
	Name string
}

func BuildOrder(it api.Item, id Identity, buildingName string, p Pickup) api.Order {
	return api.Order{
		OpenID: id.OpenID, UnionID: id.UnionID, TenantEmpID: id.TenantEmpID,
		BuildingCode: it.BuildingCode, BuildingName: buildingName,
		MealDate: it.MealDate, MealType: it.MealType,
		MealStartTime: it.MealStartTime, MealEndTime: it.MealEndTime,
		SkuCode: it.SkuCode, FoodName: it.Name, LabelMealCode: it.LabelMealCode,
		PickupAddressCode: p.Code, PickupAddressName: p.Name,
	}
}

func BuildBatch(orders []api.Order) api.SubmitReq {
	return api.SubmitReq{Orders: orders}
}

// Result summarizes a submit attempt for notification.
type Result struct {
	Booked []string // "2026-06-10 Mala Gyudon"
	Failed []string // "2026-06-11 Chickenjoy: sold out"
	DryRun bool
}
