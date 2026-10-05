// Application cliente en ligne de commande. Communique exclusivement avec
// cmd/server via une API HTTP (internal/apiclient), elle n'accède jamais
// directement à la base de données.
package main

import (
	"bufio"
	"errors"
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

	var user *api.UserDTO
	for user == nil {
		user = authenticate(client, scanner)
	}
	fmt.Printf("Connecte en tant que %s\n", user.Email)

	for {
		fmt.Println("\n--- Menu client ---")
		fmt.Println("1. Voir le catalogue")
		fmt.Println("2. Voir mon panier")
		fmt.Println("3. Ajouter un produit au panier")
		fmt.Println("4. Modifier la quantite d'un produit du panier")
		fmt.Println("5. Retirer un produit du panier")
		fmt.Println("6. Valider la commande (checkout)")
		fmt.Println("7. Voir mes commandes")
		fmt.Println("8. Rechercher un produit")
		fmt.Println("9. Quitter")
		fmt.Print("> ")

		if !scanner.Scan() {
			return
		}
		switch strings.TrimSpace(scanner.Text()) {
		case "1":
			listProducts(client)
		case "2":
			showCart(client)
		case "3":
			addToCart(client, scanner)
		case "4":
			updateCartItem(client, scanner)
		case "5":
			removeCartItem(client, scanner)
		case "6":
			checkout(client, scanner)
		case "7":
			listOrders(client)
		case "8":
			searchProducts(client, scanner)
		case "9":
			return
		default:
			fmt.Println("Choix invalide")
		}
	}
}

func authenticate(client *apiclient.Client, scanner *bufio.Scanner) *api.UserDTO {
	fmt.Println("1. Se connecter  2. Creer un compte")
	fmt.Print("> ")
	if !scanner.Scan() {
		os.Exit(0)
	}
	choice := strings.TrimSpace(scanner.Text())

	fmt.Print("Email: ")
	scanner.Scan()
	email := strings.TrimSpace(strings.ToLower(scanner.Text()))

	fmt.Print("Mot de passe: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	var (
		resp *api.AuthResponse
		err  error
	)
	if choice == "2" {
		resp, err = client.Register(email, password)
	} else {
		resp, err = client.Login(email, password)
	}
	if err != nil {
		var apiErr *apiclient.APIError
		if errors.As(err, &apiErr) {
			fmt.Println(apiErr.Message)
		} else {
			fmt.Println("Erreur:", err)
		}
		return nil
	}

	client.SetToken(resp.Token)
	return &resp.User
}

func listProducts(client *apiclient.Client) {
	products, err := client.ListProducts()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(products) == 0 {
		fmt.Println("Aucun produit disponible.")
		return
	}
	for _, p := range products {
		fmt.Printf("#%d [%s] %s - %.2f euros (stock: %d)\n", p.ID, p.Reference, p.Name, float64(p.PriceCents)/100, p.Stock)
	}
}

func searchProducts(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("Recherche (nom, description, categorie ou prix): ")
	if !scanner.Scan() {
		return
	}
	query := strings.TrimSpace(scanner.Text())
	if query == "" {
		fmt.Println("Recherche vide.")
		return
	}

	products, err := client.SearchProducts(query)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(products) == 0 {
		fmt.Println("Aucun produit trouve.")
		return
	}
	for _, p := range products {
		fmt.Printf("#%d %s [%s] - %.2f HT / %.2f TTC - %s (stock: %d)\n",
			p.ID, p.Name, p.Category, float64(p.PriceCents)/100, float64(p.PriceCentsTTC)/100, p.Description, p.Stock)
	}
}

func showCart(client *apiclient.Client) {
	cart, err := client.GetCart()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Panier %s\n", cart.Reference)
	if len(cart.Items) == 0 {
		fmt.Println("Panier vide.")
		return
	}
	for _, item := range cart.Items {
		fmt.Printf("#%d %s x%d - %.2f TTC (unite: %.2f TTC)\n",
			item.ProductID, item.ProductName, item.Quantity,
			float64(item.SubtotalCentsTTC)/100, float64(item.UnitPriceCentsTTC)/100)
	}
	fmt.Printf("Total TTC: %.2f euros (livraison gratuite)\n", float64(cart.TotalCentsTTC)/100)
}

func addToCart(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID du produit: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	fmt.Print("Quantite: ")
	scanner.Scan()
	qty, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || qty <= 0 {
		fmt.Println("Quantite invalide")
		return
	}

	if err := client.AddCartItem(id, qty); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Ajoute au panier.")
}

func updateCartItem(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID du produit: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	fmt.Print("Nouvelle quantite (0 pour retirer): ")
	scanner.Scan()
	qty, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || qty < 0 {
		fmt.Println("Quantite invalide")
		return
	}

	if err := client.SetCartItemQuantity(id, qty); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Panier mis a jour.")
}

func removeCartItem(client *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID du produit a retirer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	if err := client.RemoveCartItem(id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Produit retire du panier.")
}

func checkout(client *apiclient.Client, scanner *bufio.Scanner) {
	card, err := readCardDetails(scanner)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}

	resp, err := client.Checkout(card)
	if err != nil {
		var apiErr *apiclient.APIError
		if errors.As(err, &apiErr) {
			fmt.Println("Paiement refuse:", apiErr.Message)
		} else {
			fmt.Println("Erreur:", err)
		}
		return
	}

	fmt.Printf("Commande #%d creee (panier %s), total: %.2f euros, carte terminant par %s\n",
		resp.Order.ID, resp.CartReference, float64(resp.Order.TotalCents)/100, resp.CardLast4)
}

func readCardDetails(scanner *bufio.Scanner) (api.CardDetailsDTO, error) {
	fmt.Print("Numero de carte: ")
	if !scanner.Scan() {
		return api.CardDetailsDTO{}, fmt.Errorf("saisie annulee")
	}
	number := strings.TrimSpace(scanner.Text())

	fmt.Print("Date d'expiration (MM/AA): ")
	if !scanner.Scan() {
		return api.CardDetailsDTO{}, fmt.Errorf("saisie annulee")
	}
	expiry := strings.TrimSpace(scanner.Text())
	parts := strings.SplitN(expiry, "/", 2)
	if len(parts) != 2 {
		return api.CardDetailsDTO{}, fmt.Errorf("format de date invalide, attendu MM/AA")
	}
	month, errMonth := strconv.Atoi(strings.TrimSpace(parts[0]))
	yearShort, errYear := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errMonth != nil || errYear != nil {
		return api.CardDetailsDTO{}, fmt.Errorf("format de date invalide, attendu MM/AA")
	}
	year := yearShort
	if year < 100 {
		year += 2000
	}

	fmt.Print("CVC: ")
	if !scanner.Scan() {
		return api.CardDetailsDTO{}, fmt.Errorf("saisie annulee")
	}
	cvc := strings.TrimSpace(scanner.Text())

	return api.CardDetailsDTO{
		Number:   number,
		ExpMonth: month,
		ExpYear:  year,
		CVC:      cvc,
	}, nil
}

func listOrders(client *apiclient.Client) {
	orders, err := client.ListMyOrders()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(orders) == 0 {
		fmt.Println("Aucune commande.")
		return
	}
	for _, o := range orders {
		fmt.Printf("Commande #%d - %.2f euros - %s\n", o.ID, float64(o.TotalCents)/100, o.Status)
	}
}
