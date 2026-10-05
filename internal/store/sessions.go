package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ecommerce-cli/internal/models"
)

var (
	ErrSessionNotFound = errors.New("store: session introuvable")
	ErrSessionExpired  = errors.New("store: session expirée")
)

func (s *Store) CreateSession(ctx context.Context, userID int, token string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("store: échec de création de la session: %w", err)
	}
	return nil
}

func (s *Store) GetUserBySessionToken(ctx context.Context, token string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT users.id, users.email, users.password_hash, users.is_admin
		 FROM sessions
		 JOIN users ON users.id = sessions.user_id
		 WHERE sessions.token = ? AND sessions.expires_at > ?`,
		token, time.Now(),
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsAdmin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {

			return nil, s.classifyMissingSession(ctx, token)
		}
		return nil, fmt.Errorf("store: échec de récupération de l'utilisateur par session: %w", err)
	}

	return &u, nil
}

func (s *Store) classifyMissingSession(ctx context.Context, token string) error {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM sessions WHERE token = ?)`, token,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("store: échec de vérification de la session: %w", err)
	}
	if !exists {
		return ErrSessionNotFound
	}
	return ErrSessionExpired
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("store: échec de suppression de la session: %w", err)
	}
	return nil
}
