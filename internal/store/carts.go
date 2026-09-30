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
