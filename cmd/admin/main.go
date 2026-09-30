package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ecommerce-cli/internal/apiclient"
	"ecommerce-cli/internal/models"
)

// NOTE: ui/admin/model.go est prévu pour un écran Bubble Tea, mais cette
// dépendance n'est pas encore présente dans go.mod. Ce main.go propose donc
// un menu texte fonctionnel qui consomme la même API HTTP, en attendant
// l'implémentation de la TUI.
func main() {
	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	c := apiclient.New(baseURL)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Email admin: ")
	scanner.Scan()
	email := strings.TrimSpace(scanner.Text())
	fmt.Print("Mot de passe: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	user, err := c.Login(email, password)
	if err != nil {
		fmt.Println("Erreur de connexion:", err)
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
			listProducts(c)
		case "2":
			addProduct(c, scanner)
		case "3":
			deleteProduct(c, scanner)
		case "4":
			listAllOrders(c)
		case "5":
			_ = c.Logout()
			return
		default:
			fmt.Println("Choix invalide")
		}
	}
}

func listProducts(c *apiclient.Client) {
	products, err := c.ListProducts()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	for _, p := range products {
		fmt.Printf("#%d %s - %.2f€ (stock: %d)\n", p.ID, p.Name, float64(p.PriceCents)/100, p.Stock)
	}
}

func addProduct(c *apiclient.Client, scanner *bufio.Scanner) {
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

	p := models.Product{
		Name:        name,
		Description: description,
		Category:    category,
		PriceCents:  int64(priceEuros * 100),
		Stock:       stock,
	}
	created, err := c.CreateProduct(p)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Produit créé: #%d %s\n", created.ID, created.Name)
}

func deleteProduct(c *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID du produit à supprimer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}
	if err := c.DeleteProduct(id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Produit supprimé.")
}

func listAllOrders(c *apiclient.Client) {
	orders, err := c.ListAllOrders()
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
