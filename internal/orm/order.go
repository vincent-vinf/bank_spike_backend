package orm

import (
	"time"
)

// ordered 已下单
// cancelled
// paid

const (
	OrderOrdered   = "ordered"
	OrderCancelled = "cancelled"
	OrderPaid      = "paid"
)

type Order struct {
	ID         string
	UserID     string
	SpikeID    string
	Quantity   int
	State      string
	CreateTime time.Time
}
