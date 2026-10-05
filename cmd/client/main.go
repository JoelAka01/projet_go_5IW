package main

import (
	"bufio"
	"errors"
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

	var user *models.User
	for user == nil {
		user = authenticate(s, scanner)
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
			listProducts(s)
		case "2":
			showCart(s, user)
		case "3":
			addToCart(s, user, scanner)
		case "4":
			updateCartItem(s, user, scanner)
		case "5":
			removeCartItem(s, user, scanner)
		case "6":
			checkout(s, user)
		case "7":
			listOrders(s, user)
		case "8":
			searchProducts(s, scanner)
		case "9":
			return
		default:
			fmt.Println("Choix invalide")
		}
	}
}

func authenticate(s *store.Store, scanner *bufio.Scanner) *models.User {
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

	if choice == "2" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			fmt.Println("Erreur:", err)
			return nil
		}
		u := &models.User{Email: email, PasswordHash: hash}
		if err := s.CreateUser(u); err != nil {
			if errors.Is(err, store.ErrUserExists) {
				fmt.Println("Email deja utilise.")
			} else {
				fmt.Println("Erreur:", err)
			}
			return nil
		}
		return u
	}

	u, err := s.GetUserByEmail(email)
	if err != nil {
		fmt.Println("Identifiants invalides.")
		return nil
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		fmt.Println("Identifiants invalides.")
		return nil
	}
	return u
}

func listProducts(s *store.Store) {
	products, err := s.ListProducts()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(products) == 0 {
		fmt.Println("Aucun produit disponible.")
		return
	}
	for _, p := range products {
		fmt.Printf("#%d %s - %.2f euros (stock: %d)\n", p.ID, p.Name, float64(p.PriceCents)/100, p.Stock)
	}
}

func searchProducts(s *store.Store, scanner *bufio.Scanner) {
	fmt.Print("Recherche (nom, description, categorie ou prix): ")
	if !scanner.Scan() {
		return
	}
	query := strings.TrimSpace(scanner.Text())
	if query == "" {
		fmt.Println("Recherche vide.")
		return
	}

	products, err := s.SearchProducts(query)
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
			p.ID, p.Name, p.Category, float64(p.PriceCents)/100, float64(p.PriceCentsTTC())/100, p.Description, p.Stock)
	}
}

func showCart(s *store.Store, user *models.User) {
	cart, err := s.GetCartView(user.ID)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
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

func addToCart(s *store.Store, user *models.User, scanner *bufio.Scanner) {
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

	if err := s.AddCartItem(user.ID, id, qty); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Ajoute au panier.")
}

func updateCartItem(s *store.Store, user *models.User, scanner *bufio.Scanner) {
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

	if err := s.SetCartItemQuantity(user.ID, id, qty); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Panier mis a jour.")
}

func removeCartItem(s *store.Store, user *models.User, scanner *bufio.Scanner) {
	fmt.Print("ID du produit a retirer: ")
	scanner.Scan()
	id, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		fmt.Println("ID invalide")
		return
	}

	if err := s.RemoveCartItem(user.ID, id); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Produit retire du panier.")
}

func checkout(s *store.Store, user *models.User) {
	cart, err := s.GetCart(user.ID)
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if len(cart.Items) == 0 {
		fmt.Println("Panier vide.")
		return
	}

	order := &models.Order{UserID: user.ID, Status: "pending"}
	for _, item := range cart.Items {
		product, err := s.GetProduct(item.ProductID)
		if err != nil {
			fmt.Printf("Produit %d introuvable\n", item.ProductID)
			return
		}
		order.Items = append(order.Items, models.OrderItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: product.PriceCents,
		})
		order.TotalCents += product.PriceCents * int64(item.Quantity)
	}

	if err := s.CreateOrder(order); err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	if err := s.SaveCart(&models.Cart{UserID: user.ID}); err != nil {
		fmt.Println("Erreur:", err)
		return
	}

	fmt.Printf("Commande #%d creee, total: %.2f euros\n", order.ID, float64(order.TotalCents)/100)
}

func listOrders(s *store.Store, user *models.User) {
	orders, err := s.ListOrdersByUser(user.ID)
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
