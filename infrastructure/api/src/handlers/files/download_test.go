package files

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmartDownloadHandler_UnencryptedFile(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "smart_download_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create storage service
	store, err := storage.NewLocalStore(tmpDir)
	require.NoError(t, err)
	storage := content.NewStorageManager(store, nil, nil, logger)

	// Create test file
	testContent := []byte("Hello, this is test content for download!")
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Setup router
	router := gin.New()
	// Create delivery service
	encryptionSvc := security.NewEncryptionService("", logger) // Mock/Empty encryption service
	deliverySvc := content.NewContentDeliveryService(storage, encryptionSvc, logger)

	// Setup router

	router.GET("/download", SmartDownloadHandler(storage, nil, deliverySvc, logger))

	// Test download
	req := httptest.NewRequest("GET", "/download?path=test.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "test.txt")
	assert.Equal(t, testContent, w.Body.Bytes())
}

func TestSmartDownloadHandler_EncryptedFile(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "smart_download_enc_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create storage service
	store, err := storage.NewLocalStore(tmpDir)
	require.NoError(t, err)
	storage := content.NewStorageManager(store, nil, nil, logger)

	// Create and encrypt test file
	password := "testpassword123"
	testContent := []byte("Secret encrypted content that must be decrypted!")

	// Create encrypted file
	encryptedPath := filepath.Join(tmpDir, "secret.txt.enc")
	encFile, err := os.Create(encryptedPath)
	require.NoError(t, err)

	err = security.EncryptStream(password, bytes.NewReader(testContent), encFile)
	encFile.Close()
	require.NoError(t, err)

	// Setup router
	router := gin.New()
	// Create delivery service
	encryptionSvc := security.NewEncryptionService("", logger)
	deliverySvc := content.NewContentDeliveryService(storage, encryptionSvc, logger)

	// Setup router

	router.GET("/download", SmartDownloadHandler(storage, nil, deliverySvc, logger))

	// Test download without password (should fail)
	req := httptest.NewRequest("GET", "/download?path=secret.txt.enc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusLocked, w.Code)
	assert.Contains(t, w.Body.String(), "Vault is locked")

	// Test download with wrong password
	req = httptest.NewRequest("GET", "/download?path=secret.txt.enc&password=wrongpassword", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// Will start streaming but has incomplete/corrupted data

	// Test download with correct password
	req = httptest.NewRequest("GET", "/download?path=secret.txt.enc&password="+password, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "secret.txt")
	assert.Equal(t, testContent, w.Body.Bytes())
}

func TestSmartDownloadHandler_RangeRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "smart_download_range_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create storage service
	store, err := storage.NewLocalStore(tmpDir)
	require.NoError(t, err)
	storage := content.NewStorageManager(store, nil, nil, logger)

	// Create test file with known content
	testContent := make([]byte, 1000)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}
	testFile := filepath.Join(tmpDir, "range_test.bin")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Setup router
	router := gin.New()
	// Create delivery service
	encryptionSvc := security.NewEncryptionService("", logger)
	deliverySvc := content.NewContentDeliveryService(storage, encryptionSvc, logger)

	// Setup router

	router.GET("/download", SmartDownloadHandler(storage, nil, deliverySvc, logger))

	// Test Range request
	req := httptest.NewRequest("GET", "/download?path=range_test.bin", nil)
	req.Header.Set("Range", "bytes=0-99")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusPartialContent, w.Code)
	assert.Equal(t, 100, len(w.Body.Bytes()))
	assert.Equal(t, testContent[:100], w.Body.Bytes())
}
