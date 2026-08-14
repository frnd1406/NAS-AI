package security

import (
	"fmt"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService handles password hashing and validation
type PasswordService struct {
	cost int
}

// NewPasswordService creates a new password service
// Using bcrypt cost 12 as per SECURITY_HANDBOOK.pdf requirements
func NewPasswordService() *PasswordService {
	return &PasswordService{
		cost: 12, // bcrypt cost factor (SECURITY requirement)
	}
}

// HashPassword hashes a plaintext password using bcrypt
func (s *PasswordService) HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// ComparePassword compares a plaintext password with a hash
func (s *PasswordService) ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePasswordStrength validates password meets minimum requirements
func (s *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	var hasUpper, hasLower, hasDigit bool

	for _, r := range password { // iterate runes to support unicode characters
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsNumber(r):
			hasDigit = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one number")
	}

	return nil
}
