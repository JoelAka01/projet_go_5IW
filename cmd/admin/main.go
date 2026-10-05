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
		fmt.Println("5. Changer le statut d'une commande")
		fmt.Println("6. Créer une commande pour un client")
		fmt.Println("7. Quitter")
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
			changeOrderStatus(s, scanner)
		case "6":
			createOrderForUser(s, scanner)
		case "7":
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

func changeOrderStatus(s *store.Store, scanner *bufio.Scanner) {
	fmt.Print("ID de la commande: ")
	scanner.Scan()
	orderID, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	order, err := s.GetOrder(orderID)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Commande #%d - utilisateur #%d - statut actuel: %s\n", order.ID, order.UserID, order.Status)
	fmt.Printf("Statuts possibles: %s\n", strings.Join(store.ValidOrderStatuses, ", "))
	fmt.Print("Nouveau statut: ")
	scanner.Scan()
	status := strings.TrimSpace(scanner.Text())

	if err := s.UpdateOrderStatus(orderID, status); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Statut mis à jour.")
}

func createOrderForUser(s *store.Store, scanner *bufio.Scanner) {
	fmt.Print("Email du client: ")
	scanner.Scan()
	email := strings.TrimSpace(strings.ToLower(scanner.Text()))

	user, err := s.GetUserByEmail(email)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}

	var items []models.OrderItem
	var totalCents int64
	for {
		fmt.Print("ID du produit à ajouter (vide pour terminer): ")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			break
		}
		productID, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			fmt.Println("ID invalide")
			continue
		}
		product, err := s.GetProduct(productID)
		if err != nil {
			fmt.Println("Erreur:", err)
			continue
		}

		fmt.Print("Quantité: ")
		scanner.Scan()
		quantity, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil || quantity <= 0 {
			fmt.Println("Quantité invalide")
			continue
		}

		items = append(items, models.OrderItem{
			ProductID:  product.ID,
			Quantity:   quantity,
			PriceCents: product.PriceCentsTTC(),
		})
		totalCents += product.PriceCentsTTC() * int64(quantity)
		fmt.Printf("Produit ajouté: #%d %s x%d\n", product.ID, product.Name, quantity)
	}

	if len(items) == 0 {
		fmt.Println("Aucun produit ajouté, commande annulée.")
		return
	}

	order := &models.Order{
		UserID:     user.ID,
		Items:      items,
		TotalCents: totalCents,
	}
	if err := s.CreateOrder(order); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Commande #%d créée pour %s - total: %.2f€\n", order.ID, user.Email, float64(order.TotalCents)/100)
}
