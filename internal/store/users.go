package store

import (
	"database/sql"
	"errors"
	"fmt"

	"ecommerce-cli/internal/models"
)

var ErrUserNotFound = errors.New("store: utilisateur introuvable")

var ErrUserExists = errors.New("store: email déjà utilisé")

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, is_admin FROM users WHERE email = ?`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("store: échec de récupération de l'utilisateur: %w", err)
	}
	return &u, nil
}

func (s *Store) CreateUser(u *models.User) error {
	res, err := s.db.Exec(
		`INSERT INTO users (email, password_hash, is_admin, confirmed) VALUES (?, ?, ?, 1)`,
		u.Email, u.PasswordHash, u.IsAdmin,
	)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return ErrUserExists
		}
		return fmt.Errorf("store: échec de création de l'utilisateur: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("store: échec de récupération de l'id créé: %w", err)
	}
	u.ID = id
	return nil
}

func (s *Store) SetUserAdmin(email string, isAdmin bool) error {
	_, err := s.db.Exec(`UPDATE users SET is_admin = ? WHERE email = ?`, isAdmin, email)
	if err != nil {
		return fmt.Errorf("store: échec de mise à jour du statut admin: %w", err)
	}
	return nil
}
