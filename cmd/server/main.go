// Commande du serveur HTTP : expose l'API utilisée par les CLI client et
// admin. Implémentée uniquement avec net/http (aucun framework).
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ecommerce-cli/internal/httpserver"
	"ecommerce-cli/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "ecommerce.db"
	}

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	s, err := store.New(dbPath)
	if err != nil {
		fmt.Println("Erreur d'ouverture de la base:", err)
		os.Exit(1)
	}
	defer s.Close()

	srv := httpserver.New(s)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Serveur HTTP démarré sur %s (DB_PATH=%s)\n", addr, dbPath)
	if err := httpSrv.ListenAndServe(); err != nil {
		fmt.Println("Erreur du serveur HTTP:", err)
		os.Exit(1)
	}
}
