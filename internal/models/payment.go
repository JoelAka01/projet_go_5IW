package models

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidCardNumber = errors.New("models: numero de carte invalide")
	ErrInvalidExpiry     = errors.New("models: date d'expiration invalide (format attendu MM/AA)")
	ErrCardExpired       = errors.New("models: carte expiree")
	ErrInvalidCVC        = errors.New("models: CVC invalide")
)

type CardDetails struct {
	Number   string
	ExpMonth int
	ExpYear  int
	CVC      string
}

func (c CardDetails) Last4() string {
	digits := onlyDigits(c.Number)
	if len(digits) < 4 {
		return digits
	}
	return digits[len(digits)-4:]
}

func (c CardDetails) Validate(now time.Time) error {
	digits := onlyDigits(c.Number)
	if len(digits) < 13 || len(digits) > 19 || !luhnValid(digits) {
		return ErrInvalidCardNumber
	}

	if c.ExpMonth < 1 || c.ExpMonth > 12 || c.ExpYear < 1000 {
		return ErrInvalidExpiry
	}

	expiry := time.Date(c.ExpYear, time.Month(c.ExpMonth)+1, 1, 0, 0, 0, 0, time.UTC)
	if !now.Before(expiry) {
		return ErrCardExpired
	}

	if len(c.CVC) < 3 || len(c.CVC) > 4 || !isAllDigits(c.CVC) {
		return ErrInvalidCVC
	}

	return nil
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func luhnValid(digits string) bool {
	if digits == "" {
		return false
	}
	sum := 0
	alternate := false
	for i := len(digits) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(string(digits[i]))
		if err != nil {
			return false
		}
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}
	return sum%10 == 0
}
