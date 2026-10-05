package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const alphanumericChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: échec du hash du mot de passe: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateConfirmationCode() (string, error) {
	const length = 8

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: échec de génération du code de confirmation: %w", err)
	}

	code := make([]byte, length)
	for i, b := range buf {

		code[i] = alphanumericChars[int(b)%len(alphanumericChars)]
	}

	return string(code), nil
}

func GenerateSessionToken() (string, error) {
	const length = 32

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: échec de génération du token de session: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
