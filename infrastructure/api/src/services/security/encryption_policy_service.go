package security

import (
		"github.com/nas-ai/api/src/domain/files"
"path/filepath"
	"strings"


)

// ==============================================================================
// ENCRYPTION POLICY SERVICE - Smart Defaults & User Freedom
// ==============================================================================
//
// This service implements the hybrid encryption policy system:
// - Users have the final say (FORCE_USER / FORCE_NONE overrides)
// - AUTO mode uses intelligent policies based on file type and hardware limits
//
// Policy Flow:
//   1. Check user override (FORCE_USER → USER, FORCE_NONE → NONE)
//   2. If AUTO: Check hardware limit (file too large → NONE)
//   3. If AUTO: Check file extension policies (sensitive types → USER)
//   4. Default: NONE
//
// ==============================================================================

const (
	// PolicyMaxEncryptSizeBytes is the maximum file size for encryption (500MB)
	// This limit ensures compatibility with resource-constrained hardware (Raspberry Pi)
	PolicyMaxEncryptSizeBytes = 500 * 1024 * 1024

	// Override constants
	OverrideAuto      = "AUTO"
	OverrideForceUser = "FORCE_USER"
	OverrideForceNone = "FORCE_NONE"
)

// sensitiveExtensions defines file types that should be encrypted by default
// These are typically document types that may contain sensitive information
var sensitiveExtensions = map[string]bool{
	// Office Documents
	".pdf":  true,
	".docx": true,
	".doc":  true,
	".xlsx": true,
	".xls":  true,
	".pptx": true,
	".ppt":  true,

	// Security-Critical Files
	".key": true, // Private keys
	".pem": true, // Certificates
	".p12": true, // PKCS#12 keystores
	".pfx": true, // PFX certificates
	".crt": true, // Certificates
	".cer": true, // Certificates

	// Database Files
	".db":     true,
	".sqlite": true,
	".sql":    true,

	// Configuration Files (may contain secrets)
	".env":    true,
	".config": true,
	".ini":    true,
}

// EncryptionPolicyService determines encryption mode based on policies and user overrides
type EncryptionPolicyService struct {
	// Future: Could inject SystemSettingsRepository for dynamic policies
}

// NewEncryptionPolicyService creates a new encryption policy service
func NewEncryptionPolicyService() *EncryptionPolicyService {
	return &EncryptionPolicyService{}
}

// DetermineMode determines the encryption mode for a file upload
//
// Parameters:
//   - filename: The name of the file being uploaded
//   - sizeBytes: The size of the file in bytes
//   - userOverride: User's override preference (AUTO, FORCE_USER, FORCE_NONE)
//
// Returns:
//   - EncryptionMode: The determined encryption mode (NONE, USER, SYSTEM)
//
// Logic:
//  1. If userOverride == FORCE_USER → USER (customer is king)
//  2. If userOverride == FORCE_NONE → NONE
//  3. If AUTO mode:
//     a. If file size > PolicyMaxEncryptSizeBytes → NONE (hardware limit)
//     b. If file extension matches policy → USER
//  4. Default → NONE
func (s *EncryptionPolicyService) DetermineMode(filename string, sizeBytes int64, userOverride string) files.EncryptionMode {
	// Normalize override to uppercase
	override := strings.ToUpper(strings.TrimSpace(userOverride))

	// STEP 1: Check user override (highest priority)
	switch override {
	case OverrideForceUser:
		return files.EncryptionUser
	case OverrideForceNone:
		return files.EncryptionNone
	}

	// STEP 2: AUTO mode - apply smart policies
	if override == OverrideAuto || override == "" {
		// Check hardware limit first (performance/resource constraint)
		if sizeBytes > PolicyMaxEncryptSizeBytes {
			return files.EncryptionNone
		}

		// Check file extension policy
		ext := strings.ToLower(filepath.Ext(filename))
		if sensitiveExtensions[ext] {
			return files.EncryptionUser
		}
	}

	// STEP 3: Default to no encryption
	return files.EncryptionNone
}

// IsSensitiveExtension checks if a file extension is considered sensitive
// This is a helper method for external services that need to query the policy
func (s *EncryptionPolicyService) IsSensitiveExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return sensitiveExtensions[ext]
}

// GetMaxEncryptionSize returns the maximum file size for encryption
func (s *EncryptionPolicyService) GetMaxEncryptionSize() int64 {
	return PolicyMaxEncryptSizeBytes
}
