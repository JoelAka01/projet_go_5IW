package store

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"

	"ecommerce-cli/internal/models"
)

var ErrCartNotFound = errors.New("store: panier introuvable")

const cartReferenceAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func generateCartReference() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("store: échec de génération de la référence du panier: %w", err)
	}
	out := make([]byte, 6)
	for i, b := range buf {
		out[i] = cartReferenceAlphabet[int(b)%len(cartReferenceAlphabet)]
	}
	return "BSK-" + string(out), nil
}

type CartRecord struct {
	ID        int64
	UserID    int64
	Reference string
	Status    string
}

func (s *Store) GetOrCreateOpenCart(userID int64) (*CartRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, user_id, reference, status FROM carts WHERE user_id = ? AND status = 'open'`,
		userID,
	)
	var c CartRecord
	err := row.Scan(&c.ID, &c.UserID, &c.Reference, &c.Status)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("store: échec de récupération du panier ouvert: %w", err)
	}

	reference, err := generateCartReference()
	if err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		`INSERT INTO carts (user_id, reference, status) VALUES (?, ?, 'open')`,
		userID, reference,
	)
	if err != nil {
		return nil, fmt.Errorf("store: échec de création du panier: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("store: échec de récupération de l'id du panier: %w", err)
	}
	return &CartRecord{ID: id, UserID: userID, Reference: reference, Status: "open"}, nil
}

func (s *Store) GetCartByReference(userID int64, reference string) (*CartRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, user_id, reference, status FROM carts WHERE user_id = ? AND reference = ?`,
		userID, reference,
	)
	var c CartRecord
	if err := row.Scan(&c.ID, &c.UserID, &c.Reference, &c.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("store: échec de récupération du panier: %w", err)
	}
	return &c, nil
}

func (s *Store) MarkCartPaid(cartID int64) error {
	if _, err := s.db.Exec(
		`UPDATE carts SET status = 'paid', paid_at = CURRENT_TIMESTAMP WHERE id = ?`,
		cartID,
	); err != nil {
		return fmt.Errorf("store: échec de mise à jour du statut du panier: %w", err)
	}
	return nil
}

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

func (s *Store) RemoveCartItem(userID, productID int64) error {
	if _, err := s.db.Exec(
		`DELETE FROM cart_items WHERE user_id = ? AND product_id = ?`,
		userID, productID,
	); err != nil {
		return fmt.Errorf("store: échec de suppression d'un article du panier: %w", err)
	}
	return nil
}

func (s *Store) GetCartView(userID int64) (*models.CartView, error) {
	cartRecord, err := s.GetOrCreateOpenCart(userID)
	if err != nil {
		return nil, err
	}

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

	view := &models.CartView{UserID: userID, Reference: cartRecord.Reference}
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
