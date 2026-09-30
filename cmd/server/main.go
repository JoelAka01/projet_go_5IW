package main

import (
	"log"
	"net/http"
	"os"

	"ecommerce-cli/internal/httpapi"
	"ecommerce-cli/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "ecommerce.db"
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("server: échec d'ouverture de la base %q: %v", dbPath, err)
	}
	defer s.Close()

	router := httpapi.NewRouter(s)

	log.Printf("server: écoute sur %s (base: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server: échec du serveur HTTP: %v", err)
	}
}
