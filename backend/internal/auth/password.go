package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLen is the lowest allowed password length for setup/login change.
	MinPasswordLen = 8
	bcryptCost     = 12
)

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword compares plaintext with a bcrypt hash.
func CheckPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ValidatePassword enforces minimum strength rules.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLen {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLen)
	}
	return nil
}
