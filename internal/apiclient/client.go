// Package apiclient fournit un client HTTP pour consommer l'API exposée par
// internal/httpapi, utilisé par les CLI client et admin. Il gère la session
// (cookie) de façon transparente via un http.CookieJar.
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"

	"ecommerce-cli/internal/models"
)

type Client struct {
	BaseURL string
	http    *http.Client
}

// New crée un client pointant vers baseURL (ex: "http://localhost:8080"),
// avec un cookiejar pour conserver le cookie de session entre les appels.
func New(baseURL string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		BaseURL: baseURL,
		http:    &http.Client{Jar: jar},
	}
}

type User struct {
	ID      int64  `json:"id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

// Register crée un compte et ouvre une session.
func (c *Client) Register(email, password string) (*User, error) {
	return c.authRequest("/auth/register", email, password)
}

// Login authentifie l'utilisateur et ouvre une session.
func (c *Client) Login(email, password string) (*User, error) {
	return c.authRequest("/auth/login", email, password)
}

func (c *Client) authRequest(path, email, password string) (*User, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	var u User
	if err := c.do(http.MethodPost, path, bytes.NewReader(body), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// Logout ferme la session courante.
func (c *Client) Logout() error {
	return c.do(http.MethodPost, "/auth/logout", nil, nil)
}

// ListProducts récupère le catalogue de produits.
func (c *Client) ListProducts() ([]models.Product, error) {
	var products []models.Product
	if err := c.do(http.MethodGet, "/products", nil, &products); err != nil {
		return nil, err
	}
	return products, nil
}

// SearchProducts recherche des produits par nom, description, catégorie,
// prix HT ou prix TTC via un unique terme de recherche.
func (c *Client) SearchProducts(query string) ([]models.Product, error) {
	var products []models.Product
	url := "/products?q=" + neturl.QueryEscape(query)
	if err := c.do(http.MethodGet, url, nil, &products); err != nil {
		return nil, err
	}
	return products, nil
}

// CreateProduct ajoute un produit au catalogue (admin uniquement).
func (c *Client) CreateProduct(p models.Product) (*models.Product, error) {
	body, _ := json.Marshal(p)
	var created models.Product
	if err := c.do(http.MethodPost, "/products", bytes.NewReader(body), &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// DeleteProduct supprime un produit (admin uniquement).
func (c *Client) DeleteProduct(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/products/%d", id), nil, nil)
}

// GetCart récupère le panier de l'utilisateur connecté.
func (c *Client) GetCart() (*models.Cart, error) {
	var cart models.Cart
	if err := c.do(http.MethodGet, "/cart", nil, &cart); err != nil {
		return nil, err
	}
	return &cart, nil
}

// SaveCart remplace le contenu du panier de l'utilisateur connecté.
func (c *Client) SaveCart(cart models.Cart) (*models.Cart, error) {
	body, _ := json.Marshal(cart)
	var saved models.Cart
	if err := c.do(http.MethodPut, "/cart", bytes.NewReader(body), &saved); err != nil {
		return nil, err
	}
	return &saved, nil
}

// ListOrders récupère les commandes de l'utilisateur connecté.
func (c *Client) ListOrders() ([]models.Order, error) {
	var orders []models.Order
	if err := c.do(http.MethodGet, "/orders", nil, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// ListAllOrders récupère toutes les commandes (admin uniquement).
func (c *Client) ListAllOrders() ([]models.Order, error) {
	var orders []models.Order
	if err := c.do(http.MethodGet, "/orders?all=1", nil, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// CreateOrder transforme le panier courant en commande.
func (c *Client) CreateOrder() (*models.Order, error) {
	var order models.Order
	if err := c.do(http.MethodPost, "/orders", nil, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// do exécute une requête HTTP et décode la réponse JSON dans out (si non nil).
func (c *Client) do(method, path string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("apiclient: échec de construction de la requête: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: échec de la requête vers %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apiclient: %s %s: %s (%d)", method, path, string(msg), resp.StatusCode)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
		return fmt.Errorf("apiclient: échec du décodage de la réponse: %w", err)
	}
	return nil
}
