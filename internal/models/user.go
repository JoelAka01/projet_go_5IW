package models

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	IsAdmin      bool
	Confirmed    bool
}
