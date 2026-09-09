package store

import "ecommerce-cli/internal/models"

func (s *Store) CreateOrder(o *models.Order) error {
	return nil
}

func (s *Store) ListOrdersByUser(userID int64) ([]models.Order, error) {
	return nil, nil
}
