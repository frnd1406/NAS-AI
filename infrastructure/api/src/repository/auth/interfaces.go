package auth_repo

import (
	"context"

	domain "github.com/nas-ai/api/src/domain/auth"
)

// UserRepositoryInterface defines contract for user repository operations
type UserRepositoryInterface interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	CreateUser(ctx context.Context, username, email, passwordHash string) (*domain.User, error)
}
