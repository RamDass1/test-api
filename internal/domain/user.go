package domain

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Credentials struct {
	User         User
	PasswordHash string
}

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", Invalid("email must not be empty")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", Invalid("email is not a valid address")
	}
	return email, nil
}

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", Invalid("name must not be empty")
	}
	if utf8.RuneCountInString(name) > 255 {
		return "", Invalid("name is too long")
	}
	return name, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return Invalid("password must be at least 8 characters")
	}
	if len(password) > 72 {
		return Invalid("password is too long")
	}
	return nil
}
