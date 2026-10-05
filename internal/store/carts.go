package store

import (
	"fmt"

	"ecommerce-cli/internal/models"
)

func (s *Store) GetCart(userID int64) (*models.Cart, error) {
	rows, err := s.db.Query(
		`SELECT product_id, quantity FROM cart_items WHERE user_id = ? ORDER BY product_id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: échec de récupération du panier: %w", err)
	}
	defer rows.Close()

	cart := &models.Cart{UserID: userID}
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'un article du panier: %w", err)
		}
		cart.Items = append(cart.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: échec de parcours du panier: %w", err)
	}
	return cart, nil
}

// SaveCart remplace intégralement le contenu du panier de c.UserID par
// c.Items, dans une transaction : un panier n'est jamais laissé dans un état
// partiellement mis à jour si une erreur survient en cours de route.
func (s *Store) SaveCart(c *models.Cart) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: échec de démarrage de la transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM cart_items WHERE user_id = ?`, c.UserID); err != nil {
		return fmt.Errorf("store: échec de nettoyage du panier: %w", err)
	}

	for _, item := range c.Items {
		if item.Quantity <= 0 {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO cart_items (user_id, product_id, quantity) VALUES (?, ?, ?)`,
			c.UserID, item.ProductID, item.Quantity,
		); err != nil {
			return fmt.Errorf("store: échec d'ajout d'un article au panier: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: échec de validation de la transaction: %w", err)
	}
	return nil
}

// AddCartItem ajoute delta unités du produit productID au panier de userID.
// Si le produit est déjà présent, la quantité est incrémentée ; sinon une
// nouvelle ligne est créée. delta doit être strictement positif.
func (s *Store) AddCartItem(userID, productID int64, delta int) error {
	if delta <= 0 {
		return fmt.Errorf("store: la quantité à ajouter doit être positive")
	}
	_, err := s.db.Exec(
		`INSERT INTO cart_items (user_id, product_id, quantity) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, product_id) DO UPDATE SET quantity = quantity + excluded.quantity`,
		userID, productID, delta,
	)
	if err != nil {
		return fmt.Errorf("store: échec d'ajout d'un article au panier: %w", err)
	}
	return nil
}

// SetCartItemQuantity fixe la quantité du produit productID dans le panier
// de userID. Une quantité <= 0 supprime la ligne du panier.
func (s *Store) SetCartItemQuantity(userID, productID int64, quantity int) error {
	if quantity <= 0 {
		return s.RemoveCartItem(userID, productID)
	}
	_, err := s.db.Exec(
		`INSERT INTO cart_items (user_id, product_id, quantity) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, product_id) DO UPDATE SET quantity = excluded.quantity`,
		userID, productID, quantity,
	)
	if err != nil {
		return fmt.Errorf("store: échec de mise à jour de la quantité: %w", err)
	}
	return nil
}

// RemoveCartItem supprime le produit productID du panier de userID.
func (s *Store) RemoveCartItem(userID, productID int64) error {
	if _, err := s.db.Exec(
		`DELETE FROM cart_items WHERE user_id = ? AND product_id = ?`,
		userID, productID,
	); err != nil {
		return fmt.Errorf("store: échec de suppression d'un article du panier: %w", err)
	}
	return nil
}

// GetCartView récupère le panier de userID enrichi des informations produit
// (nom, prix HT/TTC) et calcule le total TTC du panier. Les lignes dont le
// produit référencé n'existe plus sont ignorées.
func (s *Store) GetCartView(userID int64) (*models.CartView, error) {
	rows, err := s.db.Query(
		`SELECT p.id, p.name, ci.quantity, p.price_cents, p.tax_rate_percent
		 FROM cart_items ci
		 JOIN products p ON p.id = ci.product_id
		 WHERE ci.user_id = ?
		 ORDER BY p.id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: échec de récupération du panier détaillé: %w", err)
	}
	defer rows.Close()

	view := &models.CartView{UserID: userID}
	for rows.Next() {
		var (
			item           models.CartItemView
			priceCents     int64
			taxRatePercent int64
		)
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Quantity, &priceCents, &taxRatePercent); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'un article du panier: %w", err)
		}
		item.UnitPriceCents = priceCents
		item.UnitPriceCentsTTC = priceCents * (100 + taxRatePercent) / 100
		item.SubtotalCentsTTC = item.UnitPriceCentsTTC * int64(item.Quantity)
		view.TotalCentsTTC += item.SubtotalCentsTTC
		view.Items = append(view.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: échec de parcours du panier détaillé: %w", err)
	}
	return view, nil
}
