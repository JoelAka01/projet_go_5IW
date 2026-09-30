package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ecommerce-cli/internal/auth"
	"ecommerce-cli/internal/models"
	"ecommerce-cli/internal/store"
)

// sessionTTL définit la durée de vie d'une session avant expiration.
const sessionTTL = 7 * 24 * time.Hour

const sessionCookieName = "session_token"

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID      int64  `json:"id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

func (a *api) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corps de requête invalide", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email et mot de passe requis", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}

	u := &models.User{Email: req.Email, PasswordHash: hash}
	if err := a.store.CreateUser(u); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			http.Error(w, "email déjà utilisé", http.StatusConflict)
			return
		}
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}

	a.writeSession(w, r, u)
}

func (a *api) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corps de requête invalide", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	u, err := a.store.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "identifiants invalides", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		http.Error(w, "identifiants invalides", http.StatusUnauthorized)
		return
	}

	a.writeSession(w, r, u)
}

func (a *api) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = a.store.DeleteSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

// writeSession crée une session pour u, la pose en cookie HttpOnly et
// renvoie l'utilisateur en JSON.
func (a *api) writeSession(w http.ResponseWriter, r *http.Request, u *models.User) {
	token, err := auth.GenerateSessionToken()
	if err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}
	if err := a.store.CreateSession(r.Context(), int(u.ID), token, sessionTTL); err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(sessionTTL),
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResponse{ID: u.ID, Email: u.Email, IsAdmin: u.IsAdmin})
}

// currentUser récupère l'utilisateur associé au cookie de session de la
// requête, ou renvoie une erreur si absent/invalide/expiré.
func (a *api) currentUser(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}
	return a.store.GetUserBySessionToken(r.Context(), cookie.Value)
}

// requireAuth est un middleware qui rejette la requête avec 401 si aucun
// utilisateur n'est authentifié, et transmet l'utilisateur trouvé au handler.
func (a *api) requireAuth(next func(w http.ResponseWriter, r *http.Request, u *models.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := a.currentUser(r)
		if err != nil {
			http.Error(w, "non authentifié", http.StatusUnauthorized)
			return
		}
		next(w, r, u)
	}
}
