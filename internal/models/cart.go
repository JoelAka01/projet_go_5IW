package models

type CartItem struct {
	ProductID int64
	Quantity  int
}

type Cart struct {
	UserID int64
	Items  []CartItem
}
