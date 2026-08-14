package testutils

import (
	"context"

	"github.com/nas-ai/api/src/domain/auth"
)

// ============================================================
// INTERFACES for Dependency Injection
// These match the methods used by real handlers
// ============================================================

// UserRepositoryInterface defines the contract for user repository
// Real: auth_repo.UserRepository, Mock: MockUserRepository
type UserRepositoryInterface interface {
	FindByEmail(ctx context.Context, email string) (*auth.User, error)
	FindByUsername(ctx context.Context, username string) (*auth.User, error)
	CreateUser(ctx context.Context, username, email, passwordHash string) (*auth.User, error)
}

// JWTServiceInterface defines the contract for JWT operations
type JWTServiceInterface interface {
	GenerateAccessToken(userID, email string) (string, error)
	GenerateRefreshToken(userID, email string) (string, error)
}

// PasswordServiceInterface defines the contract for password operations
type PasswordServiceInterface interface {
	HashPassword(password string) (string, error)
	ComparePassword(hash, password string) error
	ValidatePasswordStrength(password string) error
}

// TokenServiceInterface defines the contract for token revocation
type TokenServiceInterface interface {
	InvalidateUserTokens(ctx context.Context, userID string) error
	IsTokenRevoked(ctx context.Context, userID string, iat int64) bool
}
