package api

// Item is one bookable dish from menu/v3 (data.menuSites[].items[]).
type Item struct {
	ID              int    `json:"id"`
	SkuCode         string `json:"skuCode"`
	Name            string `json:"name"`
	LabelMealCode   string `json:"labelMealCode"`
	CurrentStock    int    `json:"currentStock"`
	TotalStock      int    `json:"totalStock"`
	BuildingCode    string `json:"buildingCode"`
	MealDate        string `json:"mealDate"`
	MealType        string `json:"mealType"`
	MealStartTime   string `json:"mealStartTime"`
	MealEndTime     string `json:"mealEndTime"`
	PickupAddress   string `json:"pickupAddress"`
	PickupAddressID int    `json:"pickupAddressId"`
}

type MenuSite struct {
	SiteID    int    `json:"siteId"`
	SiteLabel string `json:"siteLabel"`
	Items     []Item `json:"items"`
}
type MenuResp struct {
	Code int `json:"code"`
	Data struct {
		MenuSites        []MenuSite `json:"menuSites"`
		HadOrdered       bool       `json:"hadOrdered"`
		OrderLimitedTime string     `json:"orderLimitedTime"`
		LastUsedAddress  struct {
			PickupAddressCode string `json:"pickupAddressCode"`
			PickupAddressName string `json:"pickupAddressName"`
		} `json:"lastUsedAddress"`
		// BookedOrderInfo is present when you've already reserved this day — it names
		// the dish you booked (same "Restaurant - Dish" format as Item.Name).
		BookedOrderInfo struct {
			FoodName string `json:"foodName"`
			MealDate string `json:"mealDate"`
		} `json:"bookedOrderInfo"`
	} `json:"data"`
}

type CalendarDate struct {
	Date             string `json:"date"`
	WeekDayInt       int    `json:"weekDayInt"`
	CanReserve       bool   `json:"canReserve"`
	HadReserveLunch  bool   `json:"hadReserveLunch"`
	HadReserveDinner bool   `json:"hadReserveDinner"`
	IsHistory        bool   `json:"isHistory"`
	ShouldShowMenu   bool   `json:"shouldShowMenu"`
}
type CalendarResp struct {
	Code int `json:"code"`
	Data struct {
		Dates []CalendarDate `json:"dates"`
	} `json:"data"`
}

type Order struct {
	OpenID            string `json:"openId"`
	UnionID           string `json:"unionId"`
	TenantEmpID       string `json:"tenantEmpId"`
	BuildingCode      string `json:"buildingCode"`
	BuildingName      string `json:"buildingName"`
	MealDate          string `json:"mealDate"`
	MealType          string `json:"mealType"`
	MealStartTime     string `json:"mealStartTime"`
	MealEndTime       string `json:"mealEndTime"`
	SkuCode           string `json:"skuCode"`
	FoodName          string `json:"foodName"`
	LabelMealCode     string `json:"labelMealCode"`
	PickupAddressCode int    `json:"pickupAddressCode"`
	PickupAddressName string `json:"pickupAddressName"`
}
type SubmitReq struct {
	Orders []Order `json:"orders"`
}
type SubmitResp struct {
	Code int `json:"code"`
	Data struct {
		SuccessOrders []struct {
			OrderID  string `json:"orderId"`
			FoodName string `json:"foodName"`
			MealDate string `json:"mealDate"`
		} `json:"successOrders"`
		FailOrders []struct {
			FoodName     string `json:"foodName"`
			MealDate     string `json:"mealDate"`
			FailedReason string `json:"failedReason"`
		} `json:"failOrders"`
	} `json:"data"`
	Message string `json:"message"`
}
