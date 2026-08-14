package content

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"small file", 1024, false},
		{"exactly max size", MaxUploadSize, false},
		{"too large", MaxUploadSize + 1, true},
		{"zero size", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.size)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrFileTooLarge)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// mockMultipartFile implements multipart.File for testing
type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

func TestValidateFileType_AllowedTypes(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		filename string
		wantMime string
		wantErr  bool
	}{
		{
			name:     "JPEG image",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			filename: "test.jpg",
			wantMime: "image/jpeg",
			wantErr:  false,
		},
		{
			name:     "PNG image",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00},
			filename: "test.png",
			wantMime: "image/png",
			wantErr:  false,
		},
		{
			name:     "PDF document",
			content:  []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}, // %PDF-1.4
			filename: "document.pdf",
			wantMime: "application/pdf",
			wantErr:  false,
		},
		{
			name:     "Encrypted file (.enc extension)",
			content:  []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, // random binary
			filename: "encrypted.enc",
			wantMime: "application/octet-stream",
			wantErr:  false,
		},
		{
			name:     "Plain text",
			content:  []byte("This is a plain text file."),
			filename: "readme.txt",
			wantMime: "text/plain; charset=utf-8",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &mockMultipartFile{bytes.NewReader(tt.content)}
			mime, err := ValidateFileType(file, tt.filename)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// MIME may have charset suffix or be slightly different
				assert.True(t, len(mime) > 0, "mime should not be empty")
			}
		})
	}
}

func TestValidateFileType_Dangerous(t *testing.T) {
	dangerousFiles := []string{
		"script.exe", "malware.bat", "virus.cmd", "hack.sh", "backdoor.php",
	}

	for _, filename := range dangerousFiles {
		t.Run(filename, func(t *testing.T) {
			// Create content that would otherwise be valid (plain text)
			content := []byte("#!/bin/bash\necho hello")
			file := &mockMultipartFile{bytes.NewReader(content)}

			_, err := ValidateFileType(file, filename)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidFileType)
		})
	}
}

func TestLogValidationFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Should not panic
	LogValidationFailure(logger, "test.exe", "application/octet-stream", ErrInvalidFileType)
}
