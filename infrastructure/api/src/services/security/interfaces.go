package security

import (
	"context"

	"github.com/nas-ai/api/src/domain/files"
)

// JWTServiceInterface defines contract for JWT operations
type JWTServiceInterface interface {
	GenerateAccessToken(userID, email string) (string, error)
	GenerateRefreshToken(userID, email string) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
	ExtractClaims(tokenString string) (*TokenClaims, error)
}

// PasswordServiceInterface defines contract for password operations
type PasswordServiceInterface interface {
	HashPassword(password string) (string, error)
	ComparePassword(hashedPassword, password string) error
	ValidatePasswordStrength(password string) error
}

// TokenServiceInterface defines contract for token operations
type TokenServiceInterface interface {
	GenerateVerificationToken(ctx context.Context, userID string) (string, error)
	IsTokenRevoked(ctx context.Context, userID string, tokenIssuedAtUnix int64) bool
}

// EncryptionPolicyServiceInterface defines contract for encryption policy
type EncryptionPolicyServiceInterface interface {
	DetermineMode(filename string, size int64, override string) files.EncryptionMode
}

// EncryptionServiceInterface defines contract for data encryption operations
type EncryptionServiceInterface interface {
	EncryptData(plaintext []byte) ([]byte, error)
	DecryptData(ciphertext []byte) ([]byte, error)
	IsUnlocked() bool
	Unlock(masterPassword string) error
	Lock() error
	Setup(masterPassword string) error
	IsConfigured() bool
	GetVaultPath() string
	SetVaultPath(path string) error
}
