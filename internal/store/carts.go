package store

import "ecommerce-cli/internal/models"

func (s *Store) GetCart(userID int64) (*models.Cart, error) {
	return nil, nil
}

func (s *Store) SaveCart(c *models.Cart) error {
	return nil
}
