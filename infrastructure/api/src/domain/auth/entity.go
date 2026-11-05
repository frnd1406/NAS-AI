package auth

import "time"

// UserRole represents the role of a user
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// User represents a user account
type User struct {
	ID            string     `json:"id" db:"id"`
	Username      string     `json:"username" db:"username"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"` // Never expose password hash in JSON!
	Role          UserRole   `json:"role" db:"role"`       // User role (user or admin)
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// UserResponse is the safe representation for API responses (no password hash)
type UserResponse struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	Role          UserRole   `json:"role"`
	EmailVerified bool       `json:"email_verified"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ToResponse converts a User to a safe UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		VerifiedAt:    u.VerifiedAt,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// RefreshToken represents a refresh token
type RefreshToken struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"` // Never expose token hash!
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Revoked   bool      `json:"revoked" db:"revoked"`
}
