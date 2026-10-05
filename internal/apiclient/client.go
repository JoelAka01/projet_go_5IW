// Package apiclient fournit un client HTTP (net/http côté transport) utilisé
// par les CLI client et admin pour communiquer avec cmd/server, au lieu
// d'accéder directement à la base de données.
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"ecommerce-cli/internal/api"
)

// APIError représente une erreur retournée par le serveur HTTP.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("serveur: %s (code %d)", e.Message, e.StatusCode)
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) do(method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("apiclient: échec d'encodage de la requête: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("apiclient: échec de création de la requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: échec de la requête HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp api.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error == "" {
			errResp.Error = resp.Status
		}
		return &APIError{StatusCode: resp.StatusCode, Message: errResp.Error}
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("apiclient: échec de décodage de la réponse: %w", err)
	}
	return nil
}

func (c *Client) Register(email, password string) (*api.AuthResponse, error) {
	var out api.AuthResponse
	if err := c.do(http.MethodPost, "/auth/register", api.AuthRequest{Email: email, Password: password}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Login(email, password string) (*api.AuthResponse, error) {
	var out api.AuthResponse
	if err := c.do(http.MethodPost, "/auth/login", api.AuthRequest{Email: email, Password: password}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListProducts() ([]api.ProductDTO, error) {
	var out []api.ProductDTO
	if err := c.do(http.MethodGet, "/products", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SearchProducts(query string) ([]api.ProductDTO, error) {
	var out []api.ProductDTO
	path := "/products/search?q=" + url.QueryEscape(query)
	if err := c.do(http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateProduct(req api.CreateProductRequest) (*api.ProductDTO, error) {
	var out api.ProductDTO
	if err := c.do(http.MethodPost, "/products", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteProduct(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/products/%d", id), nil, nil)
}

func (c *Client) GetCart() (*api.CartDTO, error) {
	var out api.CartDTO
	if err := c.do(http.MethodGet, "/cart", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AddCartItem(productID int64, quantity int) error {
	return c.do(http.MethodPost, "/cart/items", api.AddCartItemRequest{ProductID: productID, Quantity: quantity}, nil)
}

func (c *Client) SetCartItemQuantity(productID int64, quantity int) error {
	path := fmt.Sprintf("/cart/items/%d", productID)
	return c.do(http.MethodPut, path, api.SetCartItemRequest{Quantity: quantity}, nil)
}

func (c *Client) RemoveCartItem(productID int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/cart/items/%d", productID), nil, nil)
}

func (c *Client) Checkout(card api.CardDetailsDTO) (*api.CheckoutResponse, error) {
	var out api.CheckoutResponse
	if err := c.do(http.MethodPost, "/checkout", api.CheckoutRequest{Card: card}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListMyOrders() ([]api.OrderDTO, error) {
	var out []api.OrderDTO
	if err := c.do(http.MethodGet, "/orders", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListAllOrders() ([]api.OrderDTO, error) {
	var out []api.OrderDTO
	if err := c.do(http.MethodGet, "/admin/orders", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateOrderStatus(orderID int64, status string) error {
	path := fmt.Sprintf("/admin/orders/%d/status", orderID)
	return c.do(http.MethodPut, path, api.UpdateOrderStatusRequest{Status: status}, nil)
}

func (c *Client) CreateOrderForUser(req api.CreateOrderRequest) (*api.OrderDTO, error) {
	var out api.OrderDTO
	if err := c.do(http.MethodPost, "/admin/orders", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListUsers() ([]api.UserDTO, error) {
	var out []api.UserDTO
	if err := c.do(http.MethodGet, "/admin/users", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateUser(req api.CreateUserRequest) (*api.UserDTO, error) {
	var out api.UserDTO
	if err := c.do(http.MethodPost, "/admin/users", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateUser(id int64, req api.UpdateUserRequest) (*api.UserDTO, error) {
	var out api.UserDTO
	if err := c.do(http.MethodPut, fmt.Sprintf("/admin/users/%d", id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ConfirmUser(id int64) error {
	return c.do(http.MethodPost, fmt.Sprintf("/admin/users/%d/confirm", id), nil, nil)
}

func (c *Client) DeleteUser(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/admin/users/%d", id), nil, nil)
}
