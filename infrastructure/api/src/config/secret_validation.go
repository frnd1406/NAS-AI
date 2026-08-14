package config

import (
	"fmt"
	"os"
	"strings"
)

const minJWTSecretLength = 32

// readSecretFromFile reads a secret from the given path.
// It trims whitespace and returns an error if the file cannot be read or is empty.
func readSecretFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file '%s': %w", path, err)
	}

	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", fmt.Errorf("secret file '%s' is empty", path)
	}

	return secret, nil
}

// ValidateJWTSecret enforces basic strength rules for JWT secrets.
func ValidateJWTSecret(secret string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return fmt.Errorf("CRITICAL: JWT_SECRET is required for token signing")
	}

	if len(trimmed) < minJWTSecretLength {
		return fmt.Errorf("CRITICAL: JWT_SECRET must be at least %d characters (got %d)", minJWTSecretLength, len(trimmed))
	}

	return nil
}
