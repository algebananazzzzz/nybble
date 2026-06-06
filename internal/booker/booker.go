package booker

import (
	"fmt"

	"github.com/algebananazzzzz/bytecanteen/internal/api"
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

// Submit sends the batch unless dry. In dry mode it returns the would-be payload
// description without calling the API.
func Submit(c *api.Client, batch api.SubmitReq, dry bool) (Result, error) {
	if dry {
		r := Result{DryRun: true}
		for _, o := range batch.Orders {
			r.Booked = append(r.Booked, fmt.Sprintf("%s %s (DRY)", o.MealDate, o.FoodName))
		}
		return r, nil
	}
	resp, err := c.Submit(batch)
	if err != nil {
		return Result{}, err
	}
	var r Result
	for _, s := range resp.Data.SuccessOrders {
		r.Booked = append(r.Booked, fmt.Sprintf("%s %s", s.MealDate, s.FoodName))
	}
	for _, f := range resp.Data.FailOrders {
		r.Failed = append(r.Failed, fmt.Sprintf("%s %s: %s", f.MealDate, f.FoodName, f.FailedReason))
	}
	return r, nil
}
