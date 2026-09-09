package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// newTestStore crée une base SQLite temporaire et isolée (t.TempDir garantit
// un fichier propre par test, supprimé automatiquement après) et y insère un
// utilisateur de test. Elle retourne le *Store ainsi que l'ID de cet
// utilisateur, pour que les tests suivants disposent d'un userID valide pour
// CreateSession sans avoir à réécrire cette insertion à chaque fois.
func newTestStore(t *testing.T) (*Store, int) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() a retourné une erreur inattendue: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() a retourné une erreur inattendue: %v", err)
		}
	})

	res, err := s.db.Exec(
		`INSERT INTO users (email, password_hash) VALUES (?, ?)`,
		"test@example.com", "fake-hash",
	)
	if err != nil {
		t.Fatalf("insertion de l'utilisateur de test a échoué: %v", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("récupération de l'ID de l'utilisateur de test a échoué: %v", err)
	}

	return s, int(userID)
}

func TestCreateSessionAndRetrieve(t *testing.T) {
	ctx := context.Background()
	s, userID := newTestStore(t)
	token := "test-token-valide"

	if err := s.CreateSession(ctx, userID, token, time.Hour); err != nil {
		t.Fatalf("CreateSession() a retourné une erreur inattendue: %v", err)
	}

	user, err := s.GetUserBySessionToken(ctx, token)

	// TODO: complète ici — vérifie que err est nil, que user n'est pas nil,
	// et que user.ID correspond bien à userID.
	_ = user
	_ = err
}

func TestGetUserBySessionToken_NotFound(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)

	user, err := s.GetUserBySessionToken(ctx, "token-jamais-cree")

	// TODO: complète ici — que doit renvoyer l'erreur ? (errors.Is(err, ErrSessionNotFound))
	// vérifie aussi que user est nil.
	_ = user
	_ = err
}

func TestGetUserBySessionToken_Expired(t *testing.T) {
	ctx := context.Background()
	s, userID := newTestStore(t)
	token := "test-token-expire"

	if err := s.CreateSession(ctx, userID, token, -time.Hour); err != nil {
		t.Fatalf("CreateSession() a retourné une erreur inattendue: %v", err)
	}

	user, err := s.GetUserBySessionToken(ctx, token)

	// TODO: complète ici — que doit renvoyer l'erreur ? (errors.Is(err, ErrSessionExpired))
	// vérifie aussi que user est nil.
	_ = user
	_ = err
}

func TestDeleteSession(t *testing.T) {
	ctx := context.Background()
	s, userID := newTestStore(t)
	token := "test-token-a-supprimer"

	if err := s.CreateSession(ctx, userID, token, time.Hour); err != nil {
		t.Fatalf("CreateSession() a retourné une erreur inattendue: %v", err)
	}

	if err := s.DeleteSession(ctx, token); err != nil {
		t.Fatalf("DeleteSession() a retourné une erreur inattendue: %v", err)
	}

	user, err := s.GetUserBySessionToken(ctx, token)

	// TODO: complète ici — après suppression, vérifie que GetUserBySessionToken
	// renvoie une erreur (errors.Is(err, ErrSessionNotFound)) et que user est nil.
	_ = user
	_ = err
}
