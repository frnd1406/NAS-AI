package auth_repo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/nas-ai/api/src/domain/auth"

	"github.com/jmoiron/sqlx"
	"github.com/nas-ai/api/src/database"
)

// UserRepositoryX handles user data access using sqlx
type UserRepositoryX struct {
	db     *database.DBX
	logger *slog.Logger
}

// NewUserRepositoryX creates a new user repository with sqlx
func NewUserRepositoryX(db *database.DBX, logger *slog.Logger) *UserRepositoryX {
	return &UserRepositoryX{
		db:     db,
		logger: logger,
	}
}

// CreateUser creates a new user in the database
func (r *UserRepositoryX) CreateUser(ctx context.Context, username, email, passwordHash string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		INSERT INTO users (username, email, password_hash, role, email_verified)
		VALUES ($1, $2, $3, 'user', FALSE)
		RETURNING id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
	`

	err := r.db.QueryRowxContext(ctx, query, username, email, passwordHash).StructScan(user)
	if err != nil {
		r.logger.Error("Failed to create user",
			slog.String("error", err.Error()),
			slog.String("email", email),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info("User created successfully",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return user, nil
}

// FindByEmail finds a user by email address
func (r *UserRepositoryX) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := r.db.GetContext(ctx, user, query, email)
	if err == sql.ErrNoRows {
		return nil, nil // User not found (not an error)
	}

	if err != nil {
		r.logger.Error("Failed to find user by email",
			slog.String("error", err.Error()),
			slog.String("email", email),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByUsername finds a user by username
func (r *UserRepositoryX) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	err := r.db.GetContext(ctx, user, query, username)
	if err == sql.ErrNoRows {
		return nil, nil // User not found (not an error)
	}

	if err != nil {
		r.logger.Error("Failed to find user by username",
			slog.String("error", err.Error()),
			slog.String("username", username),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByID finds a user by their ID
func (r *UserRepositoryX) FindByID(ctx context.Context, id string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil // User not found (not an error)
	}

	if err != nil {
		r.logger.Error("Failed to find user by ID",
			slog.String("error", err.Error()),
			slog.String("user_id", id),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByIDs finds multiple users by their IDs (batch query)
func (r *UserRepositoryX) FindByIDs(ctx context.Context, ids []string) ([]*auth.User, error) {
	if len(ids) == 0 {
		return []*auth.User{}, nil
	}

	users := []*auth.User{}

	// Use IN query with sqlx.In helper
	query, args, err := sqlx.In(`
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE id IN (?)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Rebind for PostgreSQL ($1, $2, etc.)
	query = r.db.Rebind(query)

	err = r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		r.logger.Error("Failed to find users by IDs",
			slog.String("error", err.Error()),
			slog.Int("count", len(ids)),
		)
		return nil, fmt.Errorf("failed to find users: %w", err)
	}

	return users, nil
}

// UpdateUser updates user information
func (r *UserRepositoryX) UpdateUser(ctx context.Context, user *auth.User) error {
	query := `
		UPDATE users
		SET username = :username, email = :email, updated_at = NOW()
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		r.logger.Error("Failed to update user",
			slog.String("error", err.Error()),
			slog.String("user_id", user.ID),
		)
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.Info("User updated successfully",
		slog.String("user_id", user.ID),
	)
	return nil
}

// VerifyEmail marks a user's email as verified
func (r *UserRepositoryX) VerifyEmail(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET email_verified = TRUE, verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to verify user email",
			slog.String("error", err.Error()),
			slog.String("user_id", userID),
		)
		return fmt.Errorf("failed to verify email: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.Info("User email verified successfully",
		slog.String("user_id", userID),
	)
	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepositoryX) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, newPasswordHash, userID)
	if err != nil {
		r.logger.Error("Failed to update user password",
			slog.String("error", err.Error()),
			slog.String("user_id", userID),
		)
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.Info("User password updated successfully",
		slog.String("user_id", userID),
	)
	return nil
}

// List returns a paginated list of users
func (r *UserRepositoryX) List(ctx context.Context, limit, offset int) ([]*auth.User, error) {
	users := []*auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list users",
			slog.String("error", err.Error()),
			slog.Int("limit", limit),
			slog.Int("offset", offset),
		)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepositoryX) Count(ctx context.Context) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM users`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		r.logger.Error("Failed to count users",
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// DeleteUser soft-deletes a user (or hard-delete if you prefer)
func (r *UserRepositoryX) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to delete user",
			slog.String("error", err.Error()),
			slog.String("user_id", userID),
		)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.Info("User deleted successfully",
		slog.String("user_id", userID),
	)
	return nil
}
