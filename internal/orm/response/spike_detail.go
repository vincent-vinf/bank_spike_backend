package response

import "time"

type SpikeDetail struct {
	ID             string
	CommodityID    string
	CommodityName  string
	CommodityPrice float64
	Quantity       int
	Withholding    int
	PurchaseLimit  int
	AccessRule     string
	Status         string
	OrderStatus    bool
	StartTime      time.Time
	EndTime        time.Time
}
