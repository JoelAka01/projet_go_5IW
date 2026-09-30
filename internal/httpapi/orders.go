package httpapi

import (
	"fmt"
	"net/http"

	"ecommerce-cli/internal/models"
)

// handleOrders gère :
//   - GET /orders            : commandes de l'utilisateur connecté (ou de
//     tout le monde si ?all=1 et que l'utilisateur est admin)
//   - POST /orders           : transforme le panier courant en commande
func (a *api) handleOrders(w http.ResponseWriter, r *http.Request) {
	a.requireAuth(func(w http.ResponseWriter, r *http.Request, u *models.User) {
		switch r.Method {
		case http.MethodGet:
			if u.IsAdmin && r.URL.Query().Get("all") == "1" {
				orders, err := a.store.ListAllOrders()
				if err != nil {
					http.Error(w, "erreur interne", http.StatusInternalServerError)
					return
				}
				writeJSON(w, orders)
				return
			}
			orders, err := a.store.ListOrdersByUser(u.ID)
			if err != nil {
				http.Error(w, "erreur interne", http.StatusInternalServerError)
				return
			}
			writeJSON(w, orders)

		case http.MethodPost:
			a.createOrderFromCart(w, u)

		default:
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		}
	})(w, r)
}

// createOrderFromCart lit le panier de l'utilisateur, calcule le total à
// partir des prix actuels des produits, crée la commande puis vide le panier.
func (a *api) createOrderFromCart(w http.ResponseWriter, u *models.User) {
	cart, err := a.store.GetCart(u.ID)
	if err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}
	if len(cart.Items) == 0 {
		http.Error(w, "panier vide", http.StatusBadRequest)
		return
	}

	order := &models.Order{UserID: u.ID, Status: "pending"}
	for _, item := range cart.Items {
		product, err := a.store.GetProduct(item.ProductID)
		if err != nil {
			http.Error(w, fmt.Sprintf("produit %d introuvable", item.ProductID), http.StatusBadRequest)
			return
		}
		order.Items = append(order.Items, models.OrderItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: product.PriceCents,
		})
		order.TotalCents += product.PriceCents * int64(item.Quantity)
	}

	if err := a.store.CreateOrder(order); err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}

	// Vider le panier une fois la commande créée.
	if err := a.store.SaveCart(&models.Cart{UserID: u.ID}); err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}

	writeJSON(w, order)
}
