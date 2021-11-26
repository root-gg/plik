package paginator

// Order type for order
type Order string

// Orders
const (
	ASC  Order = "ASC"
	DESC Order = "DESC"
)

func (o *Order) flip() Order {
	if *o == ASC {
		return DESC
	}
	return ASC
}

func (o *Order) validate() error {
	if *o != ASC && *o != DESC {
		return ErrInvalidOrder
	}
	return nil
}
