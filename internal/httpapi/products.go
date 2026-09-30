package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ecommerce-cli/internal/models"
	"ecommerce-cli/internal/store"
)

// handleProducts gère GET /products (liste, avec recherche optionnelle via
// ?q=...) et POST /products (création, réservée aux administrateurs).
func (a *api) handleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
			products, err := a.store.SearchProducts(q)
			if err != nil {
				http.Error(w, "erreur interne", http.StatusInternalServerError)
				return
			}
			writeJSON(w, products)
			return
		}
		products, err := a.store.ListProducts()
		if err != nil {
			http.Error(w, "erreur interne", http.StatusInternalServerError)
			return
		}
		writeJSON(w, products)

	case http.MethodPost:
		a.requireAuth(func(w http.ResponseWriter, r *http.Request, u *models.User) {
			if !u.IsAdmin {
				http.Error(w, "accès refusé", http.StatusForbidden)
				return
			}
			var p models.Product
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, "corps de requête invalide", http.StatusBadRequest)
				return
			}
			if err := a.store.CreateProduct(&p); err != nil {
				http.Error(w, "erreur interne", http.StatusInternalServerError)
				return
			}
			writeJSON(w, p)
		})(w, r)

	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// handleProductByID gère GET /products/{id} et DELETE /products/{id}
// (suppression réservée aux administrateurs).
func (a *api) handleProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/products/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, err := a.store.GetProduct(id)
		if err != nil {
			if errors.Is(err, store.ErrProductNotFound) {
				http.Error(w, "produit introuvable", http.StatusNotFound)
				return
			}
			http.Error(w, "erreur interne", http.StatusInternalServerError)
			return
		}
		writeJSON(w, p)

	case http.MethodDelete:
		a.requireAuth(func(w http.ResponseWriter, r *http.Request, u *models.User) {
			if !u.IsAdmin {
				http.Error(w, "accès refusé", http.StatusForbidden)
				return
			}
			if err := a.store.DeleteProduct(id); err != nil {
				http.Error(w, "erreur interne", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})(w, r)

	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
