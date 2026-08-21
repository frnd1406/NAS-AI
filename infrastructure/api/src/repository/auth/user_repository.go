package auth_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/domain/auth"

	"github.com/sirupsen/logrus"
)

// UserRepository handles user data access
type UserRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB, logger *logrus.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(ctx context.Context, username, email, passwordHash string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		INSERT INTO users (username, email, password_hash, email_verified)
		VALUES ($1, $2, $3, FALSE)
		RETURNING id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query, username, email, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.EmailVerified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		r.logger.WithError(err).Error("Failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User created successfully")

	return user, nil
}

// FindByEmail finds a user by email address
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.EmailVerified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found (not an error)
	}

	if err != nil {
		r.logger.WithError(err).Error("Failed to find user by email")
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.EmailVerified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		r.logger.WithError(err).Error("Failed to find user by username")
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByID finds a user by their ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*auth.User, error) {
	user := &auth.User{}

	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.EmailVerified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found (not an error)
	}

	if err != nil {
		r.logger.WithError(err).Error("Failed to find user by ID")
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (r *UserRepository) UpdateUser(ctx context.Context, user *auth.User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.ID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update user")
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.WithField("user_id", user.ID).Info("User updated successfully")
	return nil
}

// VerifyEmail marks a user's email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET email_verified = TRUE, verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to verify user email")
		return fmt.Errorf("failed to verify email: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.WithField("user_id", userID).Info("User email verified successfully")
	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, newPasswordHash, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update user password")
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.WithField("user_id", userID).Info("User password updated successfully")
	return nil
}

// GetAllUsers returns all users for admin management
func (r *UserRepository) GetAllUsers(ctx context.Context) ([]auth.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.WithError(err).Error("Failed to query users")
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := make([]auth.User, 0)
	for rows.Next() {
		var user auth.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.EmailVerified,
			&user.VerifiedAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			r.logger.WithError(err).Warn("Failed to scan user row")
			continue
		}
		// Clear password hash for security
		user.PasswordHash = ""
		users = append(users, user)
	}

	return users, nil
}

// UpdateRole updates a user's role (admin only)
func (r *UserRepository) UpdateRole(ctx context.Context, userID, role string) error {
	query := `
		UPDATE users
		SET role = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, role, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update user role")
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"role":    role,
	}).Info("User role updated successfully")
	return nil
}

// DeleteUser permanently removes a user row.
func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	r.logger.WithField("user_id", userID).Warn("User deleted successfully")
	return nil
}

// CountAdmins returns how many admin accounts exist.
func (r *UserRepository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to count admins: %w", err)
	}
	return n, nil
}

// ErrLastAdmin is returned when an operation would leave the system without
// any admin account.
var ErrLastAdmin = errors.New("last admin account")

// ErrUserNotFound is returned when the target user row does not exist.
var ErrUserNotFound = errors.New("user not found")

// withAdminGuard runs fn inside a transaction that holds a row lock on every
// admin account. Counting and mutating in one locked transaction is what makes
// the last-admin check safe: two concurrent demotions of two different admins
// would otherwise both read count=2 and both succeed, leaving zero admins.
func (r *UserRepository) withAdminGuard(ctx context.Context, userID string, fn func(tx *sql.Tx, targetIsAdmin bool, adminCount int) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock all admin rows for the duration of the transaction.
	rows, err := tx.QueryContext(ctx, `SELECT id FROM users WHERE role = 'admin' FOR UPDATE`)
	if err != nil {
		return fmt.Errorf("failed to lock admin accounts: %w", err)
	}
	adminIDs := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("failed to read admin accounts: %w", err)
		}
		adminIDs[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("failed to read admin accounts: %w", err)
	}
	rows.Close()

	_, targetIsAdmin := adminIDs[userID]
	if err := fn(tx, targetIsAdmin, len(adminIDs)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// UpdateRoleGuarded changes a user's role, refusing to demote the last admin.
// Returns ErrLastAdmin or ErrUserNotFound for those cases.
func (r *UserRepository) UpdateRoleGuarded(ctx context.Context, userID, role string) error {
	return r.withAdminGuard(ctx, userID, func(tx *sql.Tx, targetIsAdmin bool, adminCount int) error {
		if targetIsAdmin && role != "admin" && adminCount <= 1 {
			return ErrLastAdmin
		}
		result, err := tx.ExecContext(ctx,
			`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, userID)
		if err != nil {
			return fmt.Errorf("failed to update user role: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return ErrUserNotFound
		}
		r.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"role":    role,
		}).Info("User role updated successfully")
		return nil
	})
}

// DeleteUserGuarded removes a user, refusing to delete the last admin.
// Returns ErrLastAdmin or ErrUserNotFound for those cases.
func (r *UserRepository) DeleteUserGuarded(ctx context.Context, userID string) error {
	return r.withAdminGuard(ctx, userID, func(tx *sql.Tx, targetIsAdmin bool, adminCount int) error {
		if targetIsAdmin && adminCount <= 1 {
			return ErrLastAdmin
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return ErrUserNotFound
		}
		r.logger.WithField("user_id", userID).Warn("User deleted successfully")
		return nil
	})
}
