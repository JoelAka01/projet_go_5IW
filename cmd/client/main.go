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

// NOTE: ui/client/model.go est prévu pour un écran Bubble Tea, mais cette
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

	var user *apiclient.User
	for user == nil {
		user = authenticate(c, scanner)
	}
	fmt.Printf("Connecté en tant que %s\n", user.Email)

	for {
		fmt.Println("\n--- Menu client ---")
		fmt.Println("1. Voir le catalogue")
		fmt.Println("2. Voir mon panier")
		fmt.Println("3. Ajouter un produit au panier")
		fmt.Println("4. Valider la commande (checkout)")
		fmt.Println("5. Voir mes commandes")
		fmt.Println("6. Rechercher un produit")
		fmt.Println("7. Quitter")
		fmt.Print("> ")

		if !scanner.Scan() {
			return
		}
		switch strings.TrimSpace(scanner.Text()) {
		case "1":
			listProducts(c)
		case "2":
			showCart(c)
		case "3":
			addToCart(c, scanner)
		case "4":
			checkout(c)
		case "5":
			listOrders(c)
		case "6":
			searchProducts(c, scanner)
		case "7":
			_ = c.Logout()
			return
		default:
			fmt.Println("Choix invalide")
		}
	}
}

func authenticate(c *apiclient.Client, scanner *bufio.Scanner) *apiclient.User {
	fmt.Println("1. Se connecter  2. Créer un compte")
	fmt.Print("> ")
	if !scanner.Scan() {
		os.Exit(0)
	}
	choice := strings.TrimSpace(scanner.Text())

	fmt.Print("Email: ")
	scanner.Scan()
	email := strings.TrimSpace(scanner.Text())

	fmt.Print("Mot de passe: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	var (
		user *apiclient.User
		err  error
	)
	if choice == "2" {
		user, err = c.Register(email, password)
	} else {
		user, err = c.Login(email, password)
	}
	if err != nil {
		fmt.Println("Erreur:", err)
		return nil
	}
	return user
}

func listProducts(c *apiclient.Client) {
	products, err := c.ListProducts()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(products) == 0 {
		fmt.Println("Aucun produit disponible.")
		return
	}
	for _, p := range products {
		fmt.Printf("#%d %s - %.2f€ (stock: %d)\n", p.ID, p.Name, float64(p.PriceCents)/100, p.Stock)
	}
}

// searchProducts demande un terme de recherche à l'utilisateur et affiche
// les produits dont le nom, la description, la catégorie, le prix HT ou le
// prix TTC correspondent.
func searchProducts(c *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("Recherche (nom, description, catégorie ou prix): ")
	if !scanner.Scan() {
		return
	}
	query := strings.TrimSpace(scanner.Text())
	if query == "" {
		fmt.Println("Recherche vide.")
		return
	}

	products, err := c.SearchProducts(query)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(products) == 0 {
		fmt.Println("Aucun produit trouvé.")
		return
	}
	for _, p := range products {
		fmt.Printf("#%d %s [%s] - %.2f€ HT / %.2f€ TTC - %s (stock: %d)\n",
			p.ID, p.Name, p.Category, float64(p.PriceCents)/100, float64(p.PriceCentsTTC())/100, p.Description, p.Stock)
	}
}

func showCart(c *apiclient.Client) {
	cart, err := c.GetCart()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(cart.Items) == 0 {
		fmt.Println("Panier vide.")
		return
	}
	for _, item := range cart.Items {
		fmt.Printf("Produit #%d x%d\n", item.ProductID, item.Quantity)
	}
}

func addToCart(c *apiclient.Client, scanner *bufio.Scanner) {
	fmt.Print("ID du produit: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	fmt.Print("Quantité: ")
	scanner.Scan()
	qty, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || qty <= 0 {
		fmt.Println("Quantité invalide")
		return
	}

	cart, err := c.GetCart()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}

	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == id {
			cart.Items[i].Quantity += qty
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, models.CartItem{ProductID: id, Quantity: qty})
	}

	if _, err := c.SaveCart(*cart); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Ajouté au panier.")
}

func checkout(c *apiclient.Client) {
	order, err := c.CreateOrder()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Printf("Commande #%d créée, total: %.2f€\n", order.ID, float64(order.TotalCents)/100)
}

func listOrders(c *apiclient.Client) {
	orders, err := c.ListOrders()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(orders) == 0 {
		fmt.Println("Aucune commande.")
		return
	}
	for _, o := range orders {
		fmt.Printf("Commande #%d - %.2f€ - %s\n", o.ID, float64(o.TotalCents)/100, o.Status)
	}
}
