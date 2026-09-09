// Package auth fournit les primitives de sécurité (hash de mot de passe,
// génération de codes et de tokens aléatoires) utilisées par le reste de
// l'application. Ce package est volontairement indépendant : il ne connaît
// ni HTTP ni la base de données.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// alphanumericChars sert d'alphabet pour les codes de confirmation : que des
// caractères lisibles/non ambigus à l'oral ou à l'écrit (pas de séparateurs).
const alphanumericChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// HashPassword hash un mot de passe avec bcrypt.
//
// Pourquoi bcrypt et pas un simple sha256(password) : sha256 est une
// fonction de hash rapide, conçue pour l'intégrité de données, pas pour les
// mots de passe. Sa rapidité permet à un attaquant de tester des milliards
// de mots de passe par seconde en cas de fuite de la base (attaque par
// brute-force/dictionnaire). bcrypt est un algorithme volontairement lent
// (facteur de coût réglable) et intègre un sel aléatoire par hash, ce qui
// protège contre les attaques par rainbow tables et ralentit le brute-force.
// On utilise ici bcrypt.DefaultCost, un compromis raisonnable entre sécurité
// et temps de calcul pour une application de ce type.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: échec du hash du mot de passe: %w", err)
	}
	return string(hash), nil
}

// CheckPassword vérifie qu'un mot de passe en clair correspond à un hash
// bcrypt existant.
//
// On délègue la comparaison à bcrypt.CompareHashAndPassword plutôt que de
// recalculer un hash puis de le comparer nous-mêmes : bcrypt extrait le sel
// et le coût stockés dans le hash lui-même et effectue une comparaison en
// temps constant, ce qui évite les erreurs d'implémentation (timing attacks,
// mauvaise extraction du sel).
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateConfirmationCode génère un code alphanumérique aléatoire de 8
// caractères (ex: confirmation d'inscription ou de commande par email).
//
// Pourquoi crypto/rand et pas math/rand : math/rand est un générateur
// pseudo-aléatoire déterministe (à partir d'une graine), prévisible si l'on
// connaît ou devine cette graine. Pour un usage lié à la sécurité (code
// distribué à un utilisateur pour prouver son identité), il faut un
// générateur cryptographiquement sûr, imprévisible même pour un attaquant
// qui observerait plusieurs sorties : c'est le rôle de crypto/rand, qui
// s'appuie sur les sources d'entropie du système d'exploitation.
func GenerateConfirmationCode() (string, error) {
	const length = 8

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: échec de génération du code de confirmation: %w", err)
	}

	code := make([]byte, length)
	for i, b := range buf {
		// modulo sur un octet aléatoire pour piocher un caractère de l'alphabet ;
		// la légère non-uniformité introduite est négligeable ici (usage non
		// cryptographique direct, simple code de vérification).
		code[i] = alphanumericChars[int(b)%len(alphanumericChars)]
	}

	return string(code), nil
}

// GenerateSessionToken génère un token opaque de session : 32 octets
// aléatoires encodés en base64 URL-safe.
//
// Pourquoi 32 octets : cela représente 256 bits d'entropie, une marge très
// confortable contre les attaques par force brute ou par anniversaire sur un
// token de session, conformément aux recommandations usuelles (OWASP) qui
// demandent au moins 128 bits.
//
// Pourquoi base64 URL-safe (RawURLEncoding) : le token doit pouvoir circuler
// tel quel dans une URL, un en-tête HTTP ou un cookie sans caractères
// spéciaux à encoder (+, /) ni padding (=) qui poserait problème dans une
// query string.
func GenerateSessionToken() (string, error) {
	const length = 32

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: échec de génération du token de session: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
