package store

import (
	"database/sql"
	"fmt"
	"os"

	// Le driver s'enregistre auprès de database/sql via son seul effet de
	// bord (init()) ; il n'est jamais référencé directement, d'où l'alias
	// vide. C'est un driver SQLite pur Go (aucune dépendance CGO), ce qui
	// garde la compilation/déploiement simples (cf. Dockerfile sans gcc).
	_ "modernc.org/sqlite"
)

// Store encapsule la connexion SQLite. Le champ db est volontairement non
// exporté : toutes les requêtes SQL doivent passer par les méthodes de ce
// package plutôt que par un accès direct à *sql.DB depuis l'extérieur, ce
// qui centralise la gestion des erreurs et le mapping vers les types
// métier (internal/models).
type Store struct {
	db *sql.DB
}

// New ouvre (ou crée) le fichier de base SQLite situé à dbPath et applique
// le schéma décrit dans migrations/schema.sql.
//
// Pourquoi activer PRAGMA foreign_keys = ON explicitement : SQLite désactive
// les contraintes de clé étrangère par défaut pour des raisons historiques
// de compatibilité. Sans ce pragma, le "ON DELETE CASCADE" du schéma
// (suppression des sessions quand un utilisateur est supprimé) ne serait
// jamais appliqué.
//
// Pourquoi exécuter migrations/schema.sql directement plutôt qu'un outil de
// migration dédié : les instructions du schéma utilisent "CREATE TABLE IF
// NOT EXISTS", ce qui rend l'opération idempotente et suffisante pour une
// application CLI sans framework.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: échec de l'ouverture de la base %q: %w", dbPath, err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: échec de l'activation des clés étrangères: %w", err)
	}

	schema, err := os.ReadFile("migrations/schema.sql")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("store: échec de lecture du schéma: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: échec de l'exécution du schéma: %w", err)
	}

	return &Store{db: db}, nil
}

// Close ferme la connexion à la base de données.
func (s *Store) Close() error {
	return s.db.Close()
}
