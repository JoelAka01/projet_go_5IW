package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ecommerce-cli/internal/models"
)

var ErrProductNotFound = errors.New("store: produit introuvable")

func (s *Store) ListProducts() ([]models.Product, error) {
	rows, err := s.db.Query(`SELECT id, name, description, category, price_cents, tax_rate_percent, stock FROM products ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: échec du listage des produits: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.PriceCents, &p.TaxRatePercent, &p.Stock); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'un produit: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: échec de parcours des produits: %w", err)
	}
	return products, nil
}

func (s *Store) SearchProducts(query string) ([]models.Product, error) {
	like := "%" + strings.ToLower(query) + "%"

	priceCents := int64(-1)
	if amount, err := strconv.ParseFloat(strings.TrimSpace(query), 64); err == nil {
		priceCents = int64(amount*100 + 0.5)
	}

	rows, err := s.db.Query(
		`SELECT id, name, description, category, price_cents, tax_rate_percent, stock
		 FROM products
		 WHERE LOWER(name) LIKE ?
		    OR LOWER(description) LIKE ?
		    OR LOWER(category) LIKE ?
		    OR price_cents = ?
		    OR (price_cents * (100 + tax_rate_percent) / 100) = ?
		 ORDER BY id`,
		like, like, like, priceCents, priceCents,
	)
	if err != nil {
		return nil, fmt.Errorf("store: échec de la recherche de produits: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.PriceCents, &p.TaxRatePercent, &p.Stock); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'un produit: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: échec de parcours des produits: %w", err)
	}
	return products, nil
}

func (s *Store) GetProduct(id int64) (*models.Product, error) {
	var p models.Product
	err := s.db.QueryRow(
		`SELECT id, name, description, category, price_cents, tax_rate_percent, stock FROM products WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.PriceCents, &p.TaxRatePercent, &p.Stock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("store: échec de récupération du produit: %w", err)
	}
	return &p, nil
}

func (s *Store) CreateProduct(p *models.Product) error {
	if p.TaxRatePercent == 0 {
		p.TaxRatePercent = 20
	}
	res, err := s.db.Exec(
		`INSERT INTO products (name, description, category, price_cents, tax_rate_percent, stock) VALUES (?, ?, ?, ?, ?, ?)`,
		p.Name, p.Description, p.Category, p.PriceCents, p.TaxRatePercent, p.Stock,
	)
	if err != nil {
		return fmt.Errorf("store: échec de création du produit: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("store: échec de récupération de l'id créé: %w", err)
	}
	p.ID = id
	return nil
}

func (s *Store) UpdateProductStock(id int64, delta int) error {
	_, err := s.db.Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, delta, id)
	if err != nil {
		return fmt.Errorf("store: échec de mise à jour du stock: %w", err)
	}
	return nil
}

func (s *Store) DeleteProduct(id int64) error {
	_, err := s.db.Exec(`DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: échec de suppression du produit: %w", err)
	}
	return nil
}
