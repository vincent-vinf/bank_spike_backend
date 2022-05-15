package orm

import (
	"time"
)

type Spike struct {
	ID            string
	CommodityID   string
	Quantity      int
	Withholding   int
	PurchaseLimit int
	AccessRule    string
	StartTime     time.Time
	EndTime       time.Time
}
