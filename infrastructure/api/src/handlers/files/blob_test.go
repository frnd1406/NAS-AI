package files

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupBlobStorageTest(t *testing.T) (*gin.Engine, *BlobStorageHandler, string) {
	// Create temp directory for storage
	tempDir, err := os.MkdirTemp("", "nas-ai-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	logger := logrus.New()
	logger.SetOutput(os.Stdout) // Enable logs for debugging

	store, err := storage.NewLocalStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	storageService := content.NewStorageManager(store, nil, nil, logger)

	handler := NewBlobStorageHandler(storageService, logger)

	r := gin.Default()
	r.POST("/api/v1/vault/upload/init", handler.InitUpload)
	r.POST("/api/v1/vault/upload/chunk/:id", handler.UploadChunk)
	r.POST("/api/v1/vault/upload/finalize/:id", handler.FinalizeUpload)

	return r, handler, tempDir
}

func TestChunkedUploadFlow(t *testing.T) {
	r, _, tempDir := setupBlobStorageTest(t)
	defer os.RemoveAll(tempDir)

	// 1. Init Upload
	initReqBody := map[string]interface{}{
		"filename":   "testfile.txt.enc",
		"total_size": 1024,
	}
	body, _ := json.Marshal(initReqBody)

	req, _ := http.NewRequest("POST", "/api/v1/vault/upload/init", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response Body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)
	var initResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &initResp)
	uploadID := initResp["upload_id"]
	assert.NotEmpty(t, uploadID)

	// 2. Upload Chunks (Mock encrypted data)
	chunk1 := []byte("chunk1-data")
	chunk2 := []byte("chunk2-data")

	// Upload Chunk 1
	req, _ = http.NewRequest("POST", "/api/v1/vault/upload/chunk/"+uploadID, bytes.NewBuffer(chunk1))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Upload Chunk 2
	req, _ = http.NewRequest("POST", "/api/v1/vault/upload/chunk/"+uploadID, bytes.NewBuffer(chunk2))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. Finalize
	finalizeReqBody := map[string]string{
		"path": "vault",
	}
	body, _ = json.Marshal(finalizeReqBody)
	req, _ = http.NewRequest("POST", "/api/v1/vault/upload/finalize/"+uploadID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file on disk
	expectedPath := filepath.Join(tempDir, "vault", "testfile.txt.enc")
	assert.FileExists(t, expectedPath)

	content, _ := os.ReadFile(expectedPath)
	expectedContent := append(chunk1, chunk2...)
	assert.Equal(t, expectedContent, content)
}

func TestInvalidExtension(t *testing.T) {
	r, _, tempDir := setupBlobStorageTest(t)
	defer os.RemoveAll(tempDir)

	// Attempt init with .txt (should fail)
	initReqBody := map[string]interface{}{
		"filename":   "malware.exe",
		"total_size": 1024,
	}
	body, _ := json.Marshal(initReqBody)

	req, _ := http.NewRequest("POST", "/api/v1/vault/upload/init", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
