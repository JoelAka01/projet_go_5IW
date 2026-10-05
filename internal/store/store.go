package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: échec de l'ouverture de la base %q: %w", dbPath, err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: échec de l'activation des clés étrangères: %w", err)
	}

	schema, err := os.ReadFile(schemaPath())
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

func schemaPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations", "schema.sql")
}

func (s *Store) Close() error {
	return s.db.Close()
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
