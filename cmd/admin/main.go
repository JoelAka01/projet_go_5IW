package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ecommerce-cli/internal/auth"
	"ecommerce-cli/internal/models"
	"ecommerce-cli/internal/store"
)

// Application 100% CLI : aucun serveur, aucun réseau. Toutes les opérations
// passent directement par internal/store (accès à la base SQLite locale).
func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "ecommerce.db"
	}

	s, err := store.New(dbPath)
	if err != nil {
		fmt.Println("Erreur d'ouverture de la base:", err)
		os.Exit(1)
	}
	defer s.Close()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Email admin: ")
	scanner.Scan()
	email := strings.TrimSpace(strings.ToLower(scanner.Text()))
	fmt.Print("Mot de passe: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	user, err := s.GetUserByEmail(email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, password) {
		fmt.Println("Identifiants invalides.")
		os.Exit(1)
	}
	if !user.IsAdmin {
		fmt.Println("Ce compte n'est pas administrateur.")
		os.Exit(1)
	}
	fmt.Printf("Connecté en tant qu'admin (%s)\n", user.Email)

	for {
		fmt.Println("\n--- Menu admin ---")
		fmt.Println("1. Lister les produits")
		fmt.Println("2. Ajouter un produit")
		fmt.Println("3. Supprimer un produit")
		fmt.Println("4. Voir toutes les commandes")
		fmt.Println("5. Quitter")
		fmt.Print("> ")

		if !scanner.Scan() {
			return
		}
		switch strings.TrimSpace(scanner.Text()) {
		case "1":
			listProducts(s)
		case "2":
			addProduct(s, scanner)
		case "3":
			deleteProduct(s, scanner)
		case "4":
			listAllOrders(s)
		case "5":
			return
		default:
			fmt.Println("Choix invalide")
		}
	}
}

func listProducts(s *store.Store) {
	products, err := s.ListProducts()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	for _, p := range products {
		fmt.Printf("#%d %s - %.2f€ (stock: %d)\n", p.ID, p.Name, float64(p.PriceCents)/100, p.Stock)
	}
}

func addProduct(s *store.Store, scanner *bufio.Scanner) {
	fmt.Print("Nom: ")
	scanner.Scan()
	name := strings.TrimSpace(scanner.Text())

	fmt.Print("Description: ")
	scanner.Scan()
	description := strings.TrimSpace(scanner.Text())

	fmt.Print("Catégorie: ")
	scanner.Scan()
	category := strings.TrimSpace(scanner.Text())

	fmt.Print("Prix HT (en euros): ")
	scanner.Scan()
	priceEuros, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
	if err != nil {
		fmt.Println("Prix invalide")
		return
	}

	fmt.Print("Stock: ")
	scanner.Scan()
	stock, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Println("Stock invalide")
		return
	}

	p := &models.Product{
		Name:        name,
		Description: description,
		Category:    category,
		PriceCents:  int64(priceEuros * 100),
		Stock:       stock,
	}
	if err := s.CreateProduct(p); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Produit créé: #%d %s\n", p.ID, p.Name)
}

func deleteProduct(s *store.Store, scanner *bufio.Scanner) {
	fmt.Print("ID du produit à supprimer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}
	if err := s.DeleteProduct(id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Produit supprimé.")
}

func listAllOrders(s *store.Store) {
	orders, err := s.ListAllOrders()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(orders) == 0 {
		fmt.Println("Aucune commande.")
		return
	}
	for _, o := range orders {
		fmt.Printf("Commande #%d - utilisateur #%d - %.2f€ - %s\n", o.ID, o.UserID, float64(o.TotalCents)/100, o.Status)
	}
}
