package httpapi

import (
	"encoding/json"
	"net/http"

	"ecommerce-cli/internal/models"
)

// handleCart gère GET /cart (contenu du panier de l'utilisateur connecté) et
// PUT /cart (remplacement complet du panier).
func (a *api) handleCart(w http.ResponseWriter, r *http.Request) {
	a.requireAuth(func(w http.ResponseWriter, r *http.Request, u *models.User) {
		switch r.Method {
		case http.MethodGet:
			cart, err := a.store.GetCart(u.ID)
			if err != nil {
				http.Error(w, "erreur interne", http.StatusInternalServerError)
				return
			}
			writeJSON(w, cart)

		case http.MethodPut:
			var cart models.Cart
			if err := json.NewDecoder(r.Body).Decode(&cart); err != nil {
				http.Error(w, "corps de requête invalide", http.StatusBadRequest)
				return
			}
			cart.UserID = u.ID
			if err := a.store.SaveCart(&cart); err != nil {
				http.Error(w, "erreur interne", http.StatusInternalServerError)
				return
			}
			writeJSON(w, cart)

		default:
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		}
	})(w, r)
}
