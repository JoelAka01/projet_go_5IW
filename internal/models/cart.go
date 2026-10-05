package models

type CartItem struct {
	ProductID int64
	Quantity  int
}

type Cart struct {
	UserID int64
	Items  []CartItem
}

type CartItemView struct {
	ProductID         int64  `json:"product_id"`
	ProductName       string `json:"product_name"`
	Quantity          int    `json:"quantity"`
	UnitPriceCents    int64  `json:"unit_price_cents"`
	UnitPriceCentsTTC int64  `json:"unit_price_cents_ttc"`
	SubtotalCentsTTC  int64  `json:"subtotal_cents_ttc"`
}

type CartView struct {
	UserID        int64          `json:"user_id"`
	Reference     string         `json:"reference"`
	Items         []CartItemView `json:"items"`
	TotalCentsTTC int64          `json:"total_cents_ttc"`
}
