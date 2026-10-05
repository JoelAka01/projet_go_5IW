// Application d'administration en ligne de commande, séparée de
// l'application cliente. Communique exclusivement avec cmd/server via une
// API HTTP (internal/apiclient).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ecommerce-cli/internal/api"
	"ecommerce-cli/internal/apiclient"
)

func main() {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "http://localhost:8080"
	}

	client := apiclient.New(serverAddr)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Email admin: ")
	scanner.Scan()
	email := strings.TrimSpace(strings.ToLower(scanner.Text()))
	fmt.Print("Mot de passe: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	resp, err := client.Login(email, password)
	if err != nil {
		fmt.Println("Identifiants invalides.")
		os.Exit(1)
	}
	if !resp.User.IsAdmin {
		fmt.Println("Ce compte n'est pas administrateur.")
		os.Exit(1)
	}
	client.SetToken(resp.Token)
	fmt.Printf("Connecté en tant qu'admin (%s)\n", resp.User.Email)

	for {
		fmt.Println("\n--- Menu admin ---")
		fmt.Println("1. Lister les produits")
		fmt.Println("2. Ajouter un produit")
		fmt.Println("3. Supprimer un produit")
		fmt.Println("4. Voir toutes les commandes")
		fmt.Println("5. Changer le statut d'une commande")
		fmt.Println("6. Créer une commande pour un client")
		fmt.Println("7. Lister les utilisateurs")
		fmt.Println("8. Créer un utilisateur")
		fmt.Println("9. Modifier un utilisateur")
		fmt.Println("10. Confirmer un compte")
		fmt.Println("11. Supprimer un utilisateur")
		fmt.Println("12. Quitter")
		fmt.Print("> ")

		if !scanner.Scan() {
			return
		}
		switch strings.TrimSpace(scanner.Text()) {
		case "1":
			listProducts(client)
		case "2":
			addProduct(client, scanner)
		case "3":
			deleteProduct(client, scanner)
		case "4":
			listAllOrders(client)
		case "5":
			changeOrderStatus(client, scanner)
		case "6":
			createOrderForUser(client, scanner)
		case "7":
			listUsers(client)
		case "8":
			createUser(client, scanner)
		case "9":
			updateUser(client, scanner)
		case "10":
			confirmUser(client, scanner)
		case "11":
			deleteUser(client, scanner)
		case "12":
			return
		default:
			fmt.Println("Choix invalide")
		}
	}
}

func listProducts(client *apiclient.Client) {
	products, err := client.ListProducts()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	for _, p := range products {
		fmt.Printf("#%d [%s] %s - %.2f€ (stock: %d)\n", p.ID, p.Reference, p.Name, float64(p.PriceCents)/100, p.Stock)
	}
}

func addProduct(client *apiclient.Client, scanner *bufio.Scanner) {
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

	p, err := client.CreateProduct(api.CreateProductRequest{
		Name:        name,
		Description: description,
		Category:    category,
		PriceCents:  int64(priceEuros * 100),
		Stock:       stock,
	})
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Produit créé: #%d [%s] %s\n", p.ID, p.Reference, p.Name)
}

func deleteProduct(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID du produit à supprimer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}
	if err := client.DeleteProduct(id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Produit supprimé.")
}

func listAllOrders(client *apiclient.Client) {
	orders, err := client.ListAllOrders()
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

func changeOrderStatus(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID de la commande: ")
	scanner.Scan()
	orderID, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	fmt.Print("Nouveau statut (pending, paid, shipped, delivered, cancelled): ")
	scanner.Scan()
	status := strings.TrimSpace(scanner.Text())

	if err := client.UpdateOrderStatus(orderID, status); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Statut mis à jour.")
}

func createOrderForUser(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("Email du client: ")
	scanner.Scan()
	email := strings.TrimSpace(strings.ToLower(scanner.Text()))

	var items []api.CreateOrderItemRequest
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

		fmt.Print("Quantité: ")
		scanner.Scan()
		quantity, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil || quantity <= 0 {
			fmt.Println("Quantité invalide")
			continue
		}

		items = append(items, api.CreateOrderItemRequest{ProductID: productID, Quantity: quantity})
		fmt.Printf("Produit #%d ajouté x%d\n", productID, quantity)
	}

	if len(items) == 0 {
		fmt.Println("Aucun produit ajouté, commande annulée.")
		return
	}

	order, err := client.CreateOrderForUser(api.CreateOrderRequest{Email: email, Items: items})
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Commande #%d créée pour %s - total: %.2f€\n", order.ID, email, float64(order.TotalCents)/100)
}

func listUsers(client *apiclient.Client) {
	users, err := client.ListUsers()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	for _, u := range users {
		fmt.Printf("#%d %s - admin: %t - confirmé: %t\n", u.ID, u.Email, u.IsAdmin, u.Confirmed)
	}
}

func createUser(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("Email: ")
	scanner.Scan()
	email := strings.TrimSpace(strings.ToLower(scanner.Text()))

	fmt.Print("Mot de passe: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	fmt.Print("Administrateur ? (o/N): ")
	scanner.Scan()
	isAdmin := strings.EqualFold(strings.TrimSpace(scanner.Text()), "o")

	u, err := client.CreateUser(api.CreateUserRequest{Email: email, Password: password, IsAdmin: isAdmin})
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Utilisateur créé: #%d %s\n", u.ID, u.Email)
}

func updateUser(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID de l'utilisateur à modifier: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	var req api.UpdateUserRequest

	fmt.Print("Nouvel email (vide pour ne pas changer): ")
	scanner.Scan()
	if email := strings.TrimSpace(strings.ToLower(scanner.Text())); email != "" {
		req.Email = &email
	}

	fmt.Print("Nouveau mot de passe (vide pour ne pas changer): ")
	scanner.Scan()
	if password := strings.TrimSpace(scanner.Text()); password != "" {
		req.Password = &password
	}

	fmt.Print("Changer le statut admin ? (o/n, vide pour ne pas changer): ")
	scanner.Scan()
	switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
	case "o":
		isAdmin := true
		req.IsAdmin = &isAdmin
	case "n":
		isAdmin := false
		req.IsAdmin = &isAdmin
	}

	if _, err := client.UpdateUser(id, req); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Utilisateur mis à jour.")
}

func confirmUser(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID de l'utilisateur à confirmer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}
	if err := client.ConfirmUser(id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Compte confirmé.")
}

func deleteUser(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID de l'utilisateur à supprimer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}
	if err := client.DeleteUser(id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Utilisateur supprimé.")
}
