package models

type OrderItem struct {
	ProductID  int64
	Quantity   int
	PriceCents int64
}

type Order struct {
	ID         int64
	UserID     int64
	Items      []OrderItem
	TotalCents int64
	Status     string
}
