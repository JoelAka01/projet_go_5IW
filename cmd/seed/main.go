package main

import (
	"fmt"
	"os"

	"ecommerce-cli/internal/models"
	"ecommerce-cli/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "ecommerce.db"
	}

	s, err := store.New(dbPath)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	defer s.Close()

	if len(os.Args) > 1 && os.Args[1] == "make-admin" {
		email := os.Args[2]
		if err := s.SetUserAdmin(email, true); err != nil {
			fmt.Println("erreur:", err)
			os.Exit(1)
		}
		fmt.Println("OK: admin =", email)
		return
	}

	products := []models.Product{
		{Name: "T-shirt", Description: "T-shirt en coton bio", Category: "Vêtements", PriceCents: 1999, TaxRatePercent: 20, Stock: 50},
		{Name: "Mug", Description: "Mug céramique 300ml", Category: "Accessoires", PriceCents: 999, TaxRatePercent: 20, Stock: 100},
		{Name: "Casquette", Description: "Casquette ajustable", Category: "Vêtements", PriceCents: 1499, TaxRatePercent: 20, Stock: 30},
	}
	for _, p := range products {
		if err := s.CreateProduct(&p); err != nil {
			fmt.Println("erreur produit:", err)
			continue
		}
		fmt.Printf("Produit créé: #%d %s\n", p.ID, p.Name)
	}
}
