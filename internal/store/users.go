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
		`SELECT id, email, password_hash, is_admin, confirmed FROM users WHERE email = ?`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.Confirmed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("store: échec de récupération de l'utilisateur: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByID(id int64) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, is_admin, confirmed FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.Confirmed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("store: échec de récupération de l'utilisateur: %w", err)
	}
	return &u, nil
}

func (s *Store) ListUsers() ([]models.User, error) {
	rows, err := s.db.Query(`SELECT id, email, password_hash, is_admin, confirmed FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: échec du listage des utilisateurs: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.Confirmed); err != nil {
			return nil, fmt.Errorf("store: échec de lecture d'un utilisateur: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
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

func (s *Store) UpdateUserEmail(id int64, email string) error {
	res, err := s.db.Exec(`UPDATE users SET email = ? WHERE id = ?`, email, id)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return ErrUserExists
		}
		return fmt.Errorf("store: échec de mise à jour de l'email: %w", err)
	}
	return checkRowAffected(res)
}

func (s *Store) UpdateUserPassword(id int64, passwordHash string) error {
	res, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("store: échec de mise à jour du mot de passe: %w", err)
	}
	return checkRowAffected(res)
}

func (s *Store) SetUserAdminByID(id int64, isAdmin bool) error {
	res, err := s.db.Exec(`UPDATE users SET is_admin = ? WHERE id = ?`, isAdmin, id)
	if err != nil {
		return fmt.Errorf("store: échec de mise à jour du statut admin: %w", err)
	}
	return checkRowAffected(res)
}

func (s *Store) ConfirmUser(id int64) error {
	res, err := s.db.Exec(`UPDATE users SET confirmed = 1, confirm_code = NULL WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: échec de confirmation du compte: %w", err)
	}
	return checkRowAffected(res)
}

func (s *Store) DeleteUser(id int64) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: échec de suppression de l'utilisateur: %w", err)
	}
	return checkRowAffected(res)
}

func checkRowAffected(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: échec de vérification de la mise à jour: %w", err)
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}
