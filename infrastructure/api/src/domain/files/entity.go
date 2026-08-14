package files

import (
	"encoding/json"
	"time"
)

// EncryptionMode represents the encryption status of a file
// Maps to PostgreSQL ENUM: encryption_mode ('NONE', 'SYSTEM', 'USER')
type EncryptionMode string

const (
	EncryptionNone   EncryptionMode = "NONE"   // Raw storage, no encryption (max performance)
	EncryptionSystem EncryptionMode = "SYSTEM" // Server-side encryption with system key
	EncryptionUser   EncryptionMode = "USER"   // User-side encryption (zero-knowledge)
)

// IsValid checks if the encryption mode is valid
func (e EncryptionMode) IsValid() bool {
	switch e {
	case EncryptionNone, EncryptionSystem, EncryptionUser:
		return true
	}
	return false
}

// EncryptionMetadata holds cryptographic parameters for encrypted files
// Stored as JSONB in the database for flexibility
type EncryptionMetadata struct {
	// Algorithm used (e.g., "XChaCha20-Poly1305")
	Algorithm string `json:"algorithm,omitempty"`

	// Nonce/IV used for encryption (base64 encoded)
	Nonce string `json:"nonce,omitempty"`

	// Salt for key derivation (base64 encoded)
	Salt string `json:"salt,omitempty"`

	// Argon2id parameters for key derivation
	Argon2Params *Argon2Params `json:"argon2_params,omitempty"`

	// Wrapped key for SYSTEM encryption (base64 encoded)
	WrappedKey string `json:"wrapped_key,omitempty"`

	// Key version for rotation support
	KeyVersion int `json:"key_version,omitempty"`
}

// Argon2Params holds the parameters for Argon2id key derivation
// Hardcoded for portability as per Master-Plan
type Argon2Params struct {
	Time    uint32 `json:"time"`    // Iterations
	Memory  uint32 `json:"memory"`  // Memory in KiB
	Threads uint8  `json:"threads"` // Parallelism
}

// File represents a file stored in the NAS
// Maps to PostgreSQL table: files
type File struct {
	// Primary key
	ID string `json:"id" db:"id"`

	// Ownership
	OwnerID string `json:"owner_id" db:"owner_id"`

	// File identity
	Filename string `json:"filename" db:"filename"`
	MimeType string `json:"mime_type" db:"mime_type"`

	// Storage location (relative path from storage root)
	StoragePath string `json:"storage_path" db:"storage_path"`

	// Size and integrity
	SizeBytes int64   `json:"size_bytes" db:"size_bytes"`
	Checksum  *string `json:"checksum,omitempty" db:"checksum"`

	// Encryption metadata
	EncryptionStatus   EncryptionMode          `json:"encryption_status" db:"encryption_status"`
	EncryptionMetadata *EncryptionMetadataJSON `json:"encryption_metadata,omitempty" db:"encryption_metadata"`

	// Audit timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// EncryptionMetadataJSON wraps EncryptionMetadata for sqlx JSONB scanning
type EncryptionMetadataJSON struct {
	EncryptionMetadata
}

// Scan implements the sql.Scanner interface for JSONB
func (e *EncryptionMetadataJSON) Scan(value interface{}) error {
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

	return json.Unmarshal(bytes, &e.EncryptionMetadata)
}

// Value implements the driver.Valuer interface for JSONB
func (e EncryptionMetadataJSON) Value() (interface{}, error) {
	if e.EncryptionMetadata == (EncryptionMetadata{}) {
		return nil, nil
	}
	return json.Marshal(e.EncryptionMetadata)
}

// FileResponse is the safe representation for API responses
type FileResponse struct {
	ID               string         `json:"id"`
	OwnerID          string         `json:"owner_id"`
	Filename         string         `json:"filename"`
	MimeType         string         `json:"mime_type"`
	SizeBytes        int64          `json:"size_bytes"`
	Checksum         *string        `json:"checksum,omitempty"`
	EncryptionStatus EncryptionMode `json:"encryption_status"`
	IsEncrypted      bool           `json:"is_encrypted"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// ToResponse converts a File to a safe FileResponse
func (f *File) ToResponse() FileResponse {
	return FileResponse{
		ID:               f.ID,
		OwnerID:          f.OwnerID,
		Filename:         f.Filename,
		MimeType:         f.MimeType,
		SizeBytes:        f.SizeBytes,
		Checksum:         f.Checksum,
		EncryptionStatus: f.EncryptionStatus,
		IsEncrypted:      f.IsEncrypted(),
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
	}
}

// IsEncrypted returns true if the file has any encryption
func (f *File) IsEncrypted() bool {
	return f.EncryptionStatus != EncryptionNone
}

// IsUserEncrypted returns true if the file uses user-side encryption
func (f *File) IsUserEncrypted() bool {
	return f.EncryptionStatus == EncryptionUser
}

// IsDeleted returns true if the file is soft-deleted
func (f *File) IsDeleted() bool {
	return f.DeletedAt != nil
}

// HumanSize returns a human-readable file size
func (f *File) HumanSize() string {
	const unit = 1024
	if f.SizeBytes < unit {
		return formatInt(f.SizeBytes) + " B"
	}
	div, exp := int64(unit), 0
	for n := f.SizeBytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return formatFloat(float64(f.SizeBytes)/float64(div)) + " " + string("KMGTPE"[exp]) + "iB"
}

// Helper functions for HumanSize
func formatInt(n int64) string {
	return string(rune('0' + n))
}

func formatFloat(f float64) string {
	// Simple 1 decimal place formatting
	i := int64(f * 10)
	if i%10 == 0 {
		return string(rune('0'+i/10)) + ""
	}
	return string(rune('0'+i/10)) + "." + string(rune('0'+i%10))
}
