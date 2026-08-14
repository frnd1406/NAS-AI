package security

import (
		"github.com/nas-ai/api/src/domain/files"
"testing"


	"github.com/stretchr/testify/assert"
)

func TestEncryptionPolicyService_DetermineMode_ForceUser(t *testing.T) {
	service := NewEncryptionPolicyService()

	// FORCE_USER should always return USER mode, regardless of file type
	mode := service.DetermineMode("test.txt", 1024, "FORCE_USER")
	assert.Equal(t, files.EncryptionUser, mode, "FORCE_USER override should return USER mode")

	mode = service.DetermineMode("test.pdf", 1024, "FORCE_USER")
	assert.Equal(t, files.EncryptionUser, mode, "FORCE_USER override should work for any file type")
}

func TestEncryptionPolicyService_DetermineMode_ForceNone(t *testing.T) {
	service := NewEncryptionPolicyService()

	// FORCE_NONE should always return NONE mode, even for sensitive files
	mode := service.DetermineMode("secrets.pdf", 1024, "FORCE_NONE")
	assert.Equal(t, files.EncryptionNone, mode, "FORCE_NONE override should return NONE mode")

	mode = service.DetermineMode("passwords.xlsx", 1024, "FORCE_NONE")
	assert.Equal(t, files.EncryptionNone, mode, "FORCE_NONE should override policy for sensitive types")
}

func TestEncryptionPolicyService_DetermineMode_AutoWithSensitiveExtension(t *testing.T) {
	service := NewEncryptionPolicyService()

	testCases := []struct {
		filename     string
		expectedMode files.EncryptionMode
	}{
		// Sensitive documents should be encrypted
		{"contract.pdf", files.EncryptionUser},
		{"report.docx", files.EncryptionUser},
		{"spreadsheet.xlsx", files.EncryptionUser},
		{"presentation.pptx", files.EncryptionUser},

		// Security-critical files
		{"private.key", files.EncryptionUser},
		{"certificate.pem", files.EncryptionUser},
		{"keystore.p12", files.EncryptionUser},

		// Database files
		{"database.db", files.EncryptionUser},
		{"backup.sqlite", files.EncryptionUser},

		// Configuration files
		{".env", files.EncryptionUser},
		{"config.ini", files.EncryptionUser},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			mode := service.DetermineMode(tc.filename, 1024, "AUTO")
			assert.Equal(t, tc.expectedMode, mode, "AUTO mode should encrypt sensitive file: "+tc.filename)
		})
	}
}

func TestEncryptionPolicyService_DetermineMode_AutoWithNonSensitiveExtension(t *testing.T) {
	service := NewEncryptionPolicyService()

	testCases := []string{
		"photo.jpg",
		"video.mp4",
		"document.txt",
		"webpage.html",
		"script.js",
		"style.css",
		"archive.zip",
		"audio.mp3",
	}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			mode := service.DetermineMode(filename, 1024, "AUTO")
			assert.Equal(t, files.EncryptionNone, mode, "AUTO mode should NOT encrypt non-sensitive file: "+filename)
		})
	}
}

func TestEncryptionPolicyService_DetermineMode_HardwareLimit(t *testing.T) {
	service := NewEncryptionPolicyService()

	// File larger than PolicyMaxEncryptSizeBytes (500MB) should not be encrypted
	largeFileSize := int64(600 * 1024 * 1024) // 600MB

	mode := service.DetermineMode("large_document.pdf", largeFileSize, "AUTO")
	assert.Equal(t, files.EncryptionNone, mode, "Files exceeding hardware limit should not be encrypted")

	// File just under the limit should be encrypted if sensitive
	justUnderLimit := int64(400 * 1024 * 1024) // 400MB
	mode = service.DetermineMode("medium_document.pdf", justUnderLimit, "AUTO")
	assert.Equal(t, files.EncryptionUser, mode, "Files under hardware limit should follow policy")
}

func TestEncryptionPolicyService_DetermineMode_EmptyOverride(t *testing.T) {
	service := NewEncryptionPolicyService()

	// Empty override should be treated as AUTO
	mode := service.DetermineMode("test.pdf", 1024, "")
	assert.Equal(t, files.EncryptionUser, mode, "Empty override should default to AUTO mode")

	mode = service.DetermineMode("test.txt", 1024, "")
	assert.Equal(t, files.EncryptionNone, mode, "Empty override should default to AUTO mode")
}

func TestEncryptionPolicyService_DetermineMode_CaseInsensitive(t *testing.T) {
	service := NewEncryptionPolicyService()

	// Test case insensitivity for override parameter
	mode := service.DetermineMode("test.txt", 1024, "force_user")
	assert.Equal(t, files.EncryptionUser, mode, "Override should be case-insensitive")

	mode = service.DetermineMode("test.pdf", 1024, "Force_None")
	assert.Equal(t, files.EncryptionNone, mode, "Override should be case-insensitive")

	mode = service.DetermineMode("test.pdf", 1024, "auto")
	assert.Equal(t, files.EncryptionUser, mode, "AUTO should work in lowercase")
}

func TestEncryptionPolicyService_IsSensitiveExtension(t *testing.T) {
	service := NewEncryptionPolicyService()

	assert.True(t, service.IsSensitiveExtension("test.pdf"), ".pdf should be sensitive")
	assert.True(t, service.IsSensitiveExtension("test.key"), ".key should be sensitive")
	assert.False(t, service.IsSensitiveExtension("test.jpg"), ".jpg should not be sensitive")
	assert.False(t, service.IsSensitiveExtension("test.txt"), ".txt should not be sensitive")
}

func TestEncryptionPolicyService_GetMaxEncryptionSize(t *testing.T) {
	service := NewEncryptionPolicyService()

	maxSize := service.GetMaxEncryptionSize()
	assert.Equal(t, int64(500*1024*1024), maxSize, "Max encryption size should be 500MB")
}

func TestEncryptionPolicyService_DetermineMode_ExtensionCaseInsensitive(t *testing.T) {
	service := NewEncryptionPolicyService()

	// File extensions should be case-insensitive
	mode1 := service.DetermineMode("test.PDF", 1024, "AUTO")
	mode2 := service.DetermineMode("test.pdf", 1024, "AUTO")
	mode3 := service.DetermineMode("test.Pdf", 1024, "AUTO")

	assert.Equal(t, files.EncryptionUser, mode1, ".PDF should be recognized")
	assert.Equal(t, files.EncryptionUser, mode2, ".pdf should be recognized")
	assert.Equal(t, files.EncryptionUser, mode3, ".Pdf should be recognized")
}
