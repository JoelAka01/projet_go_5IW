// Package api définit les structures JSON partagées entre le serveur HTTP
// (internal/httpserver) et le client HTTP utilisé par les CLI (internal/apiclient).
package api

// ErrorResponse est le format d'erreur renvoyé par le serveur.
type ErrorResponse struct {
	Error string `json:"error"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserDTO struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
	Confirmed bool   `json:"confirmed"`
}

type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

type ProductDTO struct {
	ID             int64  `json:"id"`
	Reference      string `json:"reference"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	PriceCents     int64  `json:"price_cents"`
	TaxRatePercent int    `json:"tax_rate_percent"`
	Stock          int    `json:"stock"`
	PriceCentsTTC  int64  `json:"price_cents_ttc"`
}

type CreateProductRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	PriceCents     int64  `json:"price_cents"`
	TaxRatePercent int    `json:"tax_rate_percent"`
	Stock          int    `json:"stock"`
}

type CartItemDTO struct {
	ProductID         int64  `json:"product_id"`
	ProductName       string `json:"product_name"`
	Quantity          int    `json:"quantity"`
	UnitPriceCentsTTC int64  `json:"unit_price_cents_ttc"`
	SubtotalCentsTTC  int64  `json:"subtotal_cents_ttc"`
}

type CartDTO struct {
	Reference     string        `json:"reference"`
	Items         []CartItemDTO `json:"items"`
	TotalCentsTTC int64         `json:"total_cents_ttc"`
}

type AddCartItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type SetCartItemRequest struct {
	Quantity int `json:"quantity"`
}

type CardDetailsDTO struct {
	Number   string `json:"number"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
	CVC      string `json:"cvc"`
}

type CheckoutRequest struct {
	Card CardDetailsDTO `json:"card"`
}

type OrderItemDTO struct {
	ProductID  int64 `json:"product_id"`
	Quantity   int   `json:"quantity"`
	PriceCents int64 `json:"price_cents"`
}

type OrderDTO struct {
	ID         int64          `json:"id"`
	UserID     int64          `json:"user_id"`
	Items      []OrderItemDTO `json:"items"`
	TotalCents int64          `json:"total_cents"`
	Status     string         `json:"status"`
}

type CheckoutResponse struct {
	Order         OrderDTO `json:"order"`
	CartReference string   `json:"cart_reference"`
	CardLast4     string   `json:"card_last4"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type CreateOrderItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type CreateOrderRequest struct {
	Email string                   `json:"email"`
	Items []CreateOrderItemRequest `json:"items"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
}
