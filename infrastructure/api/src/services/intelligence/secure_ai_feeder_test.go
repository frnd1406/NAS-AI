package intelligence

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nas-ai/api/src/domain/files"

	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileProvider implements FileMetadataProvider for testing
type mockFileProvider struct {
	files map[string]*files.File
}

func (m *mockFileProvider) GetFileByID(fileID string) (*files.File, error) {
	if file, ok := m.files[fileID]; ok {
		return file, nil
	}
	return nil, ErrFileNotFound
}

func (m *mockFileProvider) GetFileByPath(storagePath string) (*files.File, error) {
	for _, file := range m.files {
		if file.StoragePath == storagePath {
			return file, nil
		}
	}
	return nil, ErrFileNotFound
}

func TestBlindAgentProtocol_GetContentForIndexing_BlocksEncrypted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "blind_agent_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	unencryptedPath := filepath.Join(tmpDir, "plain.txt")
	err = os.WriteFile(unencryptedPath, []byte("This is unencrypted content"), 0644)
	require.NoError(t, err)

	encryptedPath := filepath.Join(tmpDir, "secret.txt.enc")
	encryptedContent := &bytes.Buffer{}
	err = security.EncryptStream("testpassword", bytes.NewReader([]byte("This is encrypted content")), encryptedContent)
	require.NoError(t, err)
	err = os.WriteFile(encryptedPath, encryptedContent.Bytes(), 0644)
	require.NoError(t, err)

	// Create mock file provider
	mockProvider := &mockFileProvider{
		files: map[string]*files.File{
			"file-1": {
				ID:               "file-1",
				Filename:         "plain.txt",
				StoragePath:      "plain.txt",
				EncryptionStatus: files.EncryptionNone,
			},
			"file-2": {
				ID:               "file-2",
				Filename:         "secret.txt.enc",
				StoragePath:      "secret.txt.enc",
				EncryptionStatus: files.EncryptionUser,
			},
		},
	}

	// Create feeder
	feeder := NewSecureAIFeederV2(nil, mockProvider, tmpDir, "http://localhost:5000", "", logger)

	// Test 1: Unencrypted file SHOULD be accessible
	t.Run("UnencryptedFileAllowed", func(t *testing.T) {
		reader, err := feeder.GetContentForIndexing("file-1")
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "This is unencrypted content", string(content))
	})

	// Test 2: Encrypted file MUST be blocked
	t.Run("EncryptedFileBlocked", func(t *testing.T) {
		reader, err := feeder.GetContentForIndexing("file-2")
		assert.ErrorIs(t, err, ErrEncryptedContentProtected)
		assert.Nil(t, reader)
	})

	// Test 3: Non-existent file returns ErrFileNotFound
	t.Run("FileNotFound", func(t *testing.T) {
		reader, err := feeder.GetContentForIndexing("file-nonexistent")
		assert.ErrorIs(t, err, ErrFileNotFound)
		assert.Nil(t, reader)
	})
}

func TestBlindAgentProtocol_GetEphemeralContent_DecryptsWithPassword(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ephemeral_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	password := "ephemeral-password-123"
	plaintext := "This is secret content that only exists in RAM!"

	// Create encrypted file
	encryptedPath := filepath.Join(tmpDir, "secret.txt.enc")
	encryptedContent := &bytes.Buffer{}
	err = security.EncryptStream(password, bytes.NewReader([]byte(plaintext)), encryptedContent)
	require.NoError(t, err)
	err = os.WriteFile(encryptedPath, encryptedContent.Bytes(), 0644)
	require.NoError(t, err)

	// Create unencrypted file
	unencryptedPath := filepath.Join(tmpDir, "plain.txt")
	err = os.WriteFile(unencryptedPath, []byte("Plain content"), 0644)
	require.NoError(t, err)

	// Create mock file provider
	mockProvider := &mockFileProvider{
		files: map[string]*files.File{
			"enc-file": {
				ID:               "enc-file",
				Filename:         "secret.txt.enc",
				StoragePath:      "secret.txt.enc",
				EncryptionStatus: files.EncryptionUser,
			},
			"plain-file": {
				ID:               "plain-file",
				Filename:         "plain.txt",
				StoragePath:      "plain.txt",
				EncryptionStatus: files.EncryptionNone,
			},
		},
	}

	feeder := NewSecureAIFeederV2(nil, mockProvider, tmpDir, "http://localhost:5000", "", logger)

	// Test 1: Encrypted file WITH password returns decrypted content
	t.Run("EncryptedWithPassword", func(t *testing.T) {
		reader, err := feeder.GetEphemeralContent("enc-file", password)
		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, string(content))
	})

	// Test 2: Encrypted file WITHOUT password returns ErrPasswordRequired
	t.Run("EncryptedWithoutPassword", func(t *testing.T) {
		reader, err := feeder.GetEphemeralContent("enc-file", "")
		assert.ErrorIs(t, err, ErrPasswordRequired)
		assert.Nil(t, reader)
	})

	// Test 3: Unencrypted file doesn't need password
	t.Run("UnencryptedNoPasswordNeeded", func(t *testing.T) {
		reader, err := feeder.GetEphemeralContent("plain-file", "")
		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "Plain content", string(content))
	})
}

func TestBlindAgentProtocol_LegacyCompatibility(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Legacy constructor (no file provider)
	feeder := NewSecureAIFeeder(nil, "http://localhost:5000", "", logger)

	// Legacy mode should return error when trying to use new methods
	_, err := feeder.GetContentForIndexing("any-file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file provider not configured")
}
