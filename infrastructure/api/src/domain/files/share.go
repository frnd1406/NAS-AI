package files

import (
	"encoding/json"
	"time"
)

// ShareType represents how a file is shared
// Maps to PostgreSQL ENUM: share_type ('LINK', 'INTERNAL_USER')
type ShareType string

const (
	ShareTypeLink         ShareType = "LINK"          // Public/password-protected share link
	ShareTypeInternalUser ShareType = "INTERNAL_USER" // Share with registered user
)

// IsValid checks if the share type is valid
func (s ShareType) IsValid() bool {
	switch s {
	case ShareTypeLink, ShareTypeInternalUser:
		return true
	}
	return false
}

// SharePermission represents access level for a share
type SharePermission string

const (
	SharePermRead  SharePermission = "read"
	SharePermWrite SharePermission = "write"
	SharePermAdmin SharePermission = "admin"
)

// IsValid checks if the permission is valid
func (p SharePermission) IsValid() bool {
	switch p {
	case SharePermRead, SharePermWrite, SharePermAdmin:
		return true
	}
	return false
}

// EncryptedKeyMaterial holds re-wrapped encryption keys for shares
// Stored as JSONB for flexibility in key formats
type EncryptedKeyMaterial struct {
	// Algorithm used for wrapping (e.g., "XChaCha20-Poly1305")
	Algorithm string `json:"algorithm"`

	// The wrapped/re-encrypted key (base64 encoded)
	WrappedKey string `json:"wrapped_key"`

	// Salt used for key derivation if password-protected (base64 encoded)
	Salt string `json:"salt,omitempty"`

	// Nonce used for wrapping (base64 encoded)
	Nonce string `json:"nonce,omitempty"`

	// Key derivation parameters
	Argon2Params *Argon2Params `json:"argon2_params,omitempty"`

	// Version for future compatibility
	Version int `json:"version,omitempty"`
}

// Share represents a file share
// Maps to PostgreSQL table: shares
type Share struct {
	// Primary key
	ID string `json:"id" db:"id"`

	// What is being shared
	FileID string `json:"file_id" db:"file_id"`

	// Who created the share
	CreatedBy string `json:"created_by" db:"created_by"`

	// Share type
	ShareType ShareType `json:"share_type" db:"share_type"`

	// For LINK shares: unique URL token
	Token *string `json:"token,omitempty" db:"token"`

	// For INTERNAL_USER shares: target user
	SharedWithUserID *string `json:"shared_with_user_id,omitempty" db:"shared_with_user_id"`

	// Cryptographic key material for re-wrapped encryption
	EncryptedKeyMaterial *EncryptedKeyMaterialJSON `json:"encrypted_key_material,omitempty" db:"encrypted_key_material"`

	// Password protection for LINK shares (Argon2id hash)
	PasswordHash *string `json:"-" db:"password_hash"` // Never expose in JSON!

	// Access control
	Permissions SharePermission `json:"permissions" db:"permissions"`

	// Expiration
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`

	// Usage tracking
	AccessCount    int        `json:"access_count" db:"access_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty" db:"last_accessed_at"`

	// Audit timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// EncryptedKeyMaterialJSON wraps EncryptedKeyMaterial for sqlx JSONB scanning
type EncryptedKeyMaterialJSON struct {
	EncryptedKeyMaterial
}

// Scan implements the sql.Scanner interface for JSONB
func (e *EncryptedKeyMaterialJSON) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, &e.EncryptedKeyMaterial)
}

// Value implements the driver.Valuer interface for JSONB
func (e EncryptedKeyMaterialJSON) Value() (interface{}, error) {
	if e.EncryptedKeyMaterial == (EncryptedKeyMaterial{}) {
		return nil, nil
	}
	return json.Marshal(e.EncryptedKeyMaterial)
}

// ShareResponse is the safe representation for API responses
type ShareResponse struct {
	ID               string          `json:"id"`
	FileID           string          `json:"file_id"`
	ShareType        ShareType       `json:"share_type"`
	Token            *string         `json:"token,omitempty"`
	SharedWithUserID *string         `json:"shared_with_user_id,omitempty"`
	Permissions      SharePermission `json:"permissions"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
	AccessCount      int             `json:"access_count"`
	HasPassword      bool            `json:"has_password"`
	IsExpired        bool            `json:"is_expired"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ToResponse converts a Share to a safe ShareResponse
func (s *Share) ToResponse() ShareResponse {
	return ShareResponse{
		ID:               s.ID,
		FileID:           s.FileID,
		ShareType:        s.ShareType,
		Token:            s.Token,
		SharedWithUserID: s.SharedWithUserID,
		Permissions:      s.Permissions,
		ExpiresAt:        s.ExpiresAt,
		AccessCount:      s.AccessCount,
		HasPassword:      s.HasPassword(),
		IsExpired:        s.IsExpired(),
		CreatedAt:        s.CreatedAt,
	}
}

// IsExpired checks if the share has expired
func (s *Share) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// HasPassword checks if the share is password-protected
func (s *Share) HasPassword() bool {
	return s.PasswordHash != nil && *s.PasswordHash != ""
}

// IsLinkShare checks if this is a public link share
func (s *Share) IsLinkShare() bool {
	return s.ShareType == ShareTypeLink
}

// IsInternalShare checks if this is an internal user share
func (s *Share) IsInternalShare() bool {
	return s.ShareType == ShareTypeInternalUser
}

// CanAccess checks if a user can access this share
func (s *Share) CanAccess(userID string) bool {
	// Link shares are accessible to anyone (password check happens elsewhere)
	if s.IsLinkShare() {
		return true
	}

	// Internal shares require matching user ID
	if s.SharedWithUserID != nil && *s.SharedWithUserID == userID {
		return true
	}

	return false
}

// CreateShareRequest represents the API request to create a share
type CreateShareRequest struct {
	FileID           string          `json:"file_id" validate:"required,uuid"`
	ShareType        ShareType       `json:"share_type" validate:"required"`
	SharedWithUserID *string         `json:"shared_with_user_id,omitempty" validate:"omitempty,uuid"`
	Password         *string         `json:"password,omitempty"`
	Permissions      SharePermission `json:"permissions" validate:"required"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
}
