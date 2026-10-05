package store

import (
	"database/sql"
	"errors"
	"fmt"

	"ecommerce-cli/internal/models"
)

var ErrOrderNotFound = errors.New("store: commande introuvable")

var ErrInvalidOrderStatus = errors.New("store: statut de commande invalide")

var ValidOrderStatuses = []string{"pending", "paid", "shipped", "delivered", "cancelled"}

func isValidOrderStatus(status string) bool {
	for _, s := range ValidOrderStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func (s *Store) CreateOrder(o *models.Order) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: échec de démarrage de la transaction: %w", err)
	}
	defer tx.Rollback()

	if o.Status == "" {
		o.Status = "pending"
	}

	res, err := tx.Exec(
		`INSERT INTO orders (user_id, total_cents, status) VALUES (?, ?, ?)`,
		o.UserID, o.TotalCents, o.Status,
	)
	if err != nil {
		return fmt.Errorf("store: échec de création de la commande: %w", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("store: échec de récupération de l'id de la commande: %w", err)
	}
	o.ID = orderID

	for _, item := range o.Items {
		if _, err := tx.Exec(
			`INSERT INTO order_items (order_id, product_id, quantity, price_cents) VALUES (?, ?, ?, ?)`,
			orderID, item.ProductID, item.Quantity, item.PriceCents,
		); err != nil {
			return fmt.Errorf("store: échec d'ajout d'une ligne de commande: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: échec de validation de la transaction: %w", err)
	}
	return nil
}

func (s *Store) ListOrdersByUser(userID int64) ([]models.Order, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, total_cents, status FROM orders WHERE user_id = ? ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: échec du listage des commandes: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalCents, &o.Status); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'une commande: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: échec de parcours des commandes: %w", err)
	}

	for i := range orders {
		items, err := s.listOrderItems(orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}
	return orders, nil
}

func (s *Store) ListAllOrders() ([]models.Order, error) {
	rows, err := s.db.Query(`SELECT id, user_id, total_cents, status FROM orders ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: échec du listage des commandes: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalCents, &o.Status); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'une commande: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (s *Store) GetOrder(orderID int64) (*models.Order, error) {
	var o models.Order
	err := s.db.QueryRow(
		`SELECT id, user_id, total_cents, status FROM orders WHERE id = ?`,
		orderID,
	).Scan(&o.ID, &o.UserID, &o.TotalCents, &o.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("store: échec de récupération de la commande: %w", err)
	}
	items, err := s.listOrderItems(o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (s *Store) UpdateOrderStatus(orderID int64, status string) error {
	if !isValidOrderStatus(status) {
		return ErrInvalidOrderStatus
	}
	res, err := s.db.Exec(`UPDATE orders SET status = ? WHERE id = ?`, status, orderID)
	if err != nil {
		return fmt.Errorf("store: échec de mise à jour du statut: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: échec de vérification de la mise à jour: %w", err)
	}
	if affected == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (s *Store) listOrderItems(orderID int64) ([]models.OrderItem, error) {
	rows, err := s.db.Query(
		`SELECT product_id, quantity, price_cents FROM order_items WHERE order_id = ?`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: échec du listage des lignes de commande: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.PriceCents); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'une ligne de commande: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
