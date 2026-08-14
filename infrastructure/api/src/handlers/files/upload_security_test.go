package files

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/domain/files"
	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock objects for upload test
type MockPolicyService struct{ mock.Mock }

func (m *MockPolicyService) DetermineMode(filename string, size int64, override string) files.EncryptionMode {
	return files.EncryptionNone
}

type MockAIService struct{ mock.Mock }

func (m *MockAIService) Ask(ctx context.Context, query string, options map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (m *MockAIService) NotifyUpload(filePath, fileID, mimeType, content string) {}
func (m *MockAIService) NotifyUploadSync(ctx context.Context, filePath, fileID, mimeType, content string) error {
	return nil
}
func (m *MockAIService) NotifyDelete(ctx context.Context, filePath, fileID string) error { return nil }
func (m *MockAIService) IsFileIndexable(mimeType string) bool                            { return true }

func TestStorageUpload_Security(t *testing.T) {
	gin.SetMode(gin.TestMode)
	base := t.TempDir()
	logger := logrus.New()
	store, _ := storage.NewLocalStore(base)
	storageSvc := content.NewStorageManager(store, nil, nil, logger)

	policySvc := &MockPolicyService{}
	aiSvc := &MockAIService{}

	t.Run("Reject Path Traversal", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("path", "../etc/passwd")
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/upload", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		StorageUploadHandler(storageSvc, policySvc, nil, aiSvc, logger)(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid path")
	})

	t.Run("Reject Null Byte Injection", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("path", "image.png\x00.php")
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/upload", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		StorageUploadHandler(storageSvc, policySvc, nil, aiSvc, logger)(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid path")
	})

	t.Run("Reject File Disquised as Image (MIME Masquerading)", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("path", "/uploads")

		// Create a fake "image.jpg" that is actually a shell script
		part, _ := writer.CreateFormFile("file", "malicious.jpg")
		part.Write([]byte("#!/bin/bash\necho 'hacked'"))
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/upload", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		StorageUploadHandler(storageSvc, policySvc, nil, aiSvc, logger)(c)

		// Expect 400 Bad Request because content detection will see it's text/plain, not image/jpeg
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid file content")
	})
}
