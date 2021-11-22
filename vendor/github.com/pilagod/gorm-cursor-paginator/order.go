package paginator

// Order type for order
type Order string

// Orders
const (
	ASC  Order = "ASC"
	DESC Order = "DESC"
)

func flip(order Order) Order {
	if order == ASC {
		return DESC
	}
	return ASC
}
