package httpapi

import (
	"net/http"

	"ecommerce-cli/internal/store"
)

// api regroupe les dépendances partagées par tous les handlers (accès à la
// base de données). Les fonctions handleXxx sont des méthodes de ce type
// plutôt que des fonctions libres afin d'éviter tout état global package-level.
type api struct {
	store *store.Store
}

// NewRouter construit le routeur HTTP de l'application, avec s comme unique
// dépendance vers la persistence.
func NewRouter(s *store.Store) http.Handler {
	a := &api{store: s}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", a.handleRegister)
	mux.HandleFunc("/auth/login", a.handleLogin)
	mux.HandleFunc("/auth/logout", a.handleLogout)
	mux.HandleFunc("/products", a.handleProducts)
	mux.HandleFunc("/products/", a.handleProductByID)
	mux.HandleFunc("/cart", a.handleCart)
	mux.HandleFunc("/orders", a.handleOrders)
	return mux
}
