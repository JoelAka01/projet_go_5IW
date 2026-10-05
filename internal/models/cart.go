package models

type CartItem struct {
	ProductID int64
	Quantity  int
}

type Cart struct {
	UserID int64
	Items  []CartItem
}

// CartItemView est une ligne de panier enrichie des informations produit
// (nom, prix) nécessaires à l'affichage, ainsi que du sous-total TTC de la
// ligne (prix unitaire TTC * quantité).
type CartItemView struct {
	ProductID         int64  `json:"product_id"`
	ProductName       string `json:"product_name"`
	Quantity          int    `json:"quantity"`
	UnitPriceCents    int64  `json:"unit_price_cents"`     // prix unitaire HT
	UnitPriceCentsTTC int64  `json:"unit_price_cents_ttc"` // prix unitaire TTC
	SubtotalCentsTTC  int64  `json:"subtotal_cents_ttc"`
}

// CartView est la représentation du panier renvoyée par l'API : les lignes
// enrichies plus le total TTC du panier. Les frais de livraison sont
// toujours gratuits, donc TotalCentsTTC est directement la somme des
// sous-totaux TTC de chaque ligne.
type CartView struct {
	UserID        int64          `json:"user_id"`
	Items         []CartItemView `json:"items"`
	TotalCentsTTC int64          `json:"total_cents_ttc"`
}
