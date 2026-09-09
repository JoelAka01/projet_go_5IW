package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ecommerce-cli/internal/models"
)

// Erreurs sentinelles exportées : elles permettent à l'appelant (ex: le
// middleware d'authentification HTTP) de distinguer par errors.Is un token
// absent d'un token expiré, sans dépendre du message d'erreur ni exposer
// sql.ErrNoRows, qui est un détail d'implémentation propre à cette couche.
var (
	ErrSessionNotFound = errors.New("store: session introuvable")
	ErrSessionExpired  = errors.New("store: session expirée")
)

// CreateSession enregistre une nouvelle session pour userID, valable jusqu'à
// now + ttl.
//
// Pourquoi context.Context en premier paramètre : c'est la convention Go
// pour propager une annulation ou un délai (fermeture de la connexion
// cliente, timeout global de la requête) jusqu'à la requête SQL en cours,
// via ExecContext plutôt qu'Exec qui ignore tout contexte.
func (s *Store) CreateSession(ctx context.Context, userID int, token string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)

	// Requête préparée avec paramètres positionnels (?) : le token et l'ID
	// utilisateur ne sont jamais concaténés dans le texte SQL, ce qui
	// élimine tout risque d'injection SQL.
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("store: échec de création de la session: %w", err)
	}
	return nil
}

// GetUserBySessionToken retrouve l'utilisateur associé à un token de session
// encore valide.
//
// La jointure sessions/users et la vérification expires_at > now sont faites
// en une seule requête plutôt qu'en deux allers-retours séparés : cela évite
// une race condition où la session expirerait entre les deux requêtes, et
// réduit le nombre de round-trips vers la base.
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
			// sql.ErrNoRows ne dit pas si le token est absent ou expiré (les
			// deux conditions sont dans le même WHERE) ; on ne laisse jamais
			// remonter cette erreur générique à l'appelant et on la
			// transforme systématiquement en erreur métier explicite.
			return nil, s.classifyMissingSession(ctx, token)
		}
		return nil, fmt.Errorf("store: échec de récupération de l'utilisateur par session: %w", err)
	}

	return &u, nil
}

// classifyMissingSession distingue un token totalement absent d'un token
// présent mais expiré. Comme la contrainte ON DELETE CASCADE supprime les
// sessions d'un utilisateur effacé, un token présent dans la table sessions
// mais exclu du JOIN précédent ne peut l'être qu'à cause de son expiration.
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

// DeleteSession supprime une session (déconnexion). Supprimer un token qui
// n'existe pas ou plus n'est pas traité comme une erreur : l'état recherché
// (aucune session valide sous ce token) est déjà atteint.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("store: échec de suppression de la session: %w", err)
	}
	return nil
}
