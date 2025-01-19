package filter

// OrderType represents sort type
type OrderType int

const (
	// ASC ascending sort order
	ASC = OrderType(0)
	// DESC descending sort order
	DESC = OrderType(1)
)

func (o OrderType) String() string {
	switch o {
	case ASC:
		return "ASC"
	case DESC:
		return "DESC"
	}
	return ""
}

// OrderCond sort condition
type OrderCond struct {
	// orderID identifier of the parameter being sorted
	orderID int
	Type    OrderType
}

// NewOrder returns an object for sorting data by orderID with sort type t
func NewOrder(orderID int, t OrderType) *OrderCond {
	return &OrderCond{
		orderID: orderID,
		Type:    t,
	}
}
