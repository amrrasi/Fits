// Package auth provides password hashing, JWT handling, and auth middleware.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword returns nil if plain matches the stored hash.
func CheckPassword(plain, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return fmt.Errorf("auth: invalid credentials")
	}
	return nil
}
