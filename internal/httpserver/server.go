// Package httpserver expose l'application e-commerce via une API HTTP en
// JSON, construite uniquement avec net/http (aucun framework). Les CLI
// client et admin communiquent avec le serveur via internal/apiclient au
// lieu d'accéder directement à internal/store.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ecommerce-cli/internal/api"
	"ecommerce-cli/internal/auth"
	"ecommerce-cli/internal/models"
	"ecommerce-cli/internal/store"
)

const sessionTTL = 24 * time.Hour

type Server struct {
	store *store.Store
	mux   *http.ServeMux
}

func New(s *store.Store) *Server {
	srv := &Server{store: s, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /auth/login", s.handleLogin)

	s.mux.HandleFunc("GET /products", s.handleListProducts)
	s.mux.HandleFunc("GET /products/search", s.handleSearchProducts)
	s.mux.HandleFunc("POST /products", s.requireAdmin(s.handleCreateProduct))
	s.mux.HandleFunc("DELETE /products/{id}", s.requireAdmin(s.handleDeleteProduct))

	s.mux.HandleFunc("GET /cart", s.requireAuth(s.handleGetCart))
	s.mux.HandleFunc("POST /cart/items", s.requireAuth(s.handleAddCartItem))
	s.mux.HandleFunc("PUT /cart/items/{productID}", s.requireAuth(s.handleSetCartItem))
	s.mux.HandleFunc("DELETE /cart/items/{productID}", s.requireAuth(s.handleRemoveCartItem))
	s.mux.HandleFunc("POST /checkout", s.requireAuth(s.handleCheckout))

	s.mux.HandleFunc("GET /orders", s.requireAuth(s.handleListMyOrders))

	s.mux.HandleFunc("GET /admin/orders", s.requireAdmin(s.handleListAllOrders))
	s.mux.HandleFunc("POST /admin/orders", s.requireAdmin(s.handleCreateOrderForUser))
	s.mux.HandleFunc("PUT /admin/orders/{id}/status", s.requireAdmin(s.handleUpdateOrderStatus))

	s.mux.HandleFunc("GET /admin/users", s.requireAdmin(s.handleListUsers))
	s.mux.HandleFunc("POST /admin/users", s.requireAdmin(s.handleCreateUser))
	s.mux.HandleFunc("PUT /admin/users/{id}", s.requireAdmin(s.handleUpdateUser))
	s.mux.HandleFunc("POST /admin/users/{id}/confirm", s.requireAdmin(s.handleConfirmUser))
	s.mux.HandleFunc("DELETE /admin/users/{id}", s.requireAdmin(s.handleDeleteUser))
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpserver: échec d'encodage JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, api.ErrorResponse{Error: err.Error()})
}

func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("corps de requête invalide: %w", err)
	}
	return nil
}

func userToDTO(u *models.User) api.UserDTO {
	return api.UserDTO{ID: u.ID, Email: u.Email, IsAdmin: u.IsAdmin, Confirmed: u.Confirmed}
}

func productToDTO(p models.Product) api.ProductDTO {
	return api.ProductDTO{
		ID:             p.ID,
		Reference:      p.Reference,
		Name:           p.Name,
		Description:    p.Description,
		Category:       p.Category,
		PriceCents:     p.PriceCents,
		TaxRatePercent: p.TaxRatePercent,
		Stock:          p.Stock,
		PriceCentsTTC:  p.PriceCentsTTC(),
	}
}

func orderToDTO(o models.Order) api.OrderDTO {
	items := make([]api.OrderItemDTO, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, api.OrderItemDTO{ProductID: it.ProductID, Quantity: it.Quantity, PriceCents: it.PriceCents})
	}
	return api.OrderDTO{ID: o.ID, UserID: o.UserID, Items: items, TotalCents: o.TotalCents, Status: o.Status}
}

func pathInt64(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("identifiant invalide: %q", raw)
	}
	return id, nil
}

type userContextKey struct{}

func contextWithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, u)
}

func userFromContext(r *http.Request) *models.User {
	u, _ := r.Context().Value(userContextKey{}).(*models.User)
	return u
}

func (s *Server) requireAuth(next func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, errors.New("authentification requise"))
			return
		}
		user, err := s.store.GetUserBySessionToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, errors.New("session invalide ou expirée"))
			return
		}
		r = r.WithContext(contextWithUser(r.Context(), user))
		next(w, r)
	}
}

func (s *Server) requireAdmin(next func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r)
		if user == nil || !user.IsAdmin {
			writeError(w, http.StatusForbidden, errors.New("accès réservé aux administrateurs"))
			return
		}
		next(w, r)
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimPrefix(h, prefix)
	}
	return ""
}

func (s *Server) createSessionToken(ctx context.Context, userID int64) (string, error) {
	token, err := auth.GenerateSessionToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateSession(ctx, int(userID), token, sessionTTL); err != nil {
		return "", err
	}
	return token, nil
}

// --- auth handlers ---

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req api.AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	u := &models.User{Email: email, PasswordHash: hash}
	if err := s.store.CreateUser(u); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	token, err := s.createSessionToken(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, api.AuthResponse{Token: token, User: userToDTO(u)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req api.AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	u, err := s.store.GetUserByEmail(email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, errors.New("identifiants invalides"))
		return
	}
	token, err := s.createSessionToken(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, api.AuthResponse{Token: token, User: userToDTO(u)})
}

// --- product handlers ---

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.store.ListProducts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]api.ProductDTO, 0, len(products))
	for _, p := range products {
		out = append(out, productToDTO(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	products, err := s.store.SearchProducts(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]api.ProductDTO, 0, len(products))
	for _, p := range products {
		out = append(out, productToDTO(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req api.CreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	p := &models.Product{
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		PriceCents:     req.PriceCents,
		TaxRatePercent: req.TaxRatePercent,
		Stock:          req.Stock,
	}
	if err := s.store.CreateProduct(p); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, productToDTO(*p))
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.DeleteProduct(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// --- cart handlers ---

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	view, err := s.store.GetCartView(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]api.CartItemDTO, 0, len(view.Items))
	for _, it := range view.Items {
		items = append(items, api.CartItemDTO{
			ProductID:         it.ProductID,
			ProductName:       it.ProductName,
			Quantity:          it.Quantity,
			UnitPriceCentsTTC: it.UnitPriceCentsTTC,
			SubtotalCentsTTC:  it.SubtotalCentsTTC,
		})
	}
	writeJSON(w, http.StatusOK, api.CartDTO{Reference: view.Reference, Items: items, TotalCentsTTC: view.TotalCentsTTC})
}

func (s *Server) handleAddCartItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	var req api.AddCartItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.AddCartItem(user.ID, req.ProductID, req.Quantity); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleSetCartItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	productID, err := pathInt64(r, "productID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req api.SetCartItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.SetCartItemQuantity(user.ID, productID, req.Quantity); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleRemoveCartItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	productID, err := pathInt64(r, "productID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.RemoveCartItem(user.ID, productID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)

	var req api.CheckoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	cart, err := s.store.GetCart(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(cart.Items) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("panier vide"))
		return
	}

	cartRecord, err := s.store.GetOrCreateOpenCart(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	order := &models.Order{UserID: user.ID, Status: "pending"}
	for _, item := range cart.Items {
		product, err := s.store.GetProduct(item.ProductID)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("produit %d introuvable", item.ProductID))
			return
		}
		order.Items = append(order.Items, models.OrderItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: product.PriceCents,
		})
		order.TotalCents += product.PriceCents * int64(item.Quantity)
	}

	card := models.CardDetails{
		Number:   req.Card.Number,
		ExpMonth: req.Card.ExpMonth,
		ExpYear:  req.Card.ExpYear,
		CVC:      req.Card.CVC,
	}
	if err := card.Validate(time.Now()); err != nil {
		writeError(w, http.StatusPaymentRequired, err)
		return
	}

	if err := s.store.CreateOrder(order); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.store.RecordPayment(cartRecord.ID, order.ID, order.TotalCents, card.Last4()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.store.MarkCartPaid(cartRecord.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.store.SaveCart(&models.Cart{UserID: user.ID}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := s.store.GetOrCreateOpenCart(user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, api.CheckoutResponse{
		Order:         orderToDTO(*order),
		CartReference: cartRecord.Reference,
		CardLast4:     card.Last4(),
	})
}

// --- order handlers ---

func (s *Server) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	orders, err := s.store.ListOrdersByUser(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]api.OrderDTO, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderToDTO(o))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleListAllOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.store.ListAllOrders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]api.OrderDTO, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderToDTO(o))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req api.UpdateOrderStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.UpdateOrderStatus(orderID, req.Status); err != nil {
		if errors.Is(err, store.ErrInvalidOrderStatus) || errors.Is(err, store.ErrOrderNotFound) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleCreateOrderForUser(w http.ResponseWriter, r *http.Request) {
	var req api.CreateOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := s.store.GetUserByEmail(email)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("la commande doit contenir au moins un produit"))
		return
	}

	order := &models.Order{UserID: user.ID}
	for _, reqItem := range req.Items {
		if reqItem.Quantity <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("quantité invalide pour le produit %d", reqItem.ProductID))
			return
		}
		product, err := s.store.GetProduct(reqItem.ProductID)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("produit %d introuvable", reqItem.ProductID))
			return
		}
		order.Items = append(order.Items, models.OrderItem{
			ProductID:  product.ID,
			Quantity:   reqItem.Quantity,
			PriceCents: product.PriceCentsTTC(),
		})
		order.TotalCents += product.PriceCentsTTC() * int64(reqItem.Quantity)
	}

	if err := s.store.CreateOrder(order); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, orderToDTO(*order))
}

// --- user handlers ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]api.UserDTO, 0, len(users))
	for _, u := range users {
		out = append(out, userToDTO(&u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req api.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	u := &models.User{Email: email, PasswordHash: hash, IsAdmin: req.IsAdmin}
	if err := s.store.CreateUser(u); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, userToDTO(u))
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req api.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*req.Email))
		if email != "" {
			if err := s.store.UpdateUserEmail(id, email); err != nil {
				if errors.Is(err, store.ErrUserExists) {
					writeError(w, http.StatusConflict, err)
					return
				}
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := s.store.UpdateUserPassword(id, hash); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if req.IsAdmin != nil {
		if err := s.store.SetUserAdminByID(id, *req.IsAdmin); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	user, err := s.store.GetUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, userToDTO(user))
}

func (s *Server) handleConfirmUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.ConfirmUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
