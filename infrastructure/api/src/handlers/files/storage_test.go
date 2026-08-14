package files

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func setupStorageTest(t *testing.T) (*content.StorageManager, *logrus.Logger, string) {
	t.Helper()
	base := t.TempDir()
	logger := logrus.New()
	store, err := storage.NewLocalStore(base)
	require.NoError(t, err)
	svc := content.NewStorageManager(store, nil, nil, logger)
	return svc, logger, base
}

func TestStorageList_PathTraversalForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, _ := setupStorageTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/files?path=../../etc/passwd", nil)
	c.Request = req
	c.Set("request_id", "test")

	StorageListHandler(svc, logger)(c)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestStorageDownload_PathTraversalForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, _ := setupStorageTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/download?path=../../etc/passwd", nil)
	c.Request = req
	c.Set("request_id", "test")

	// Pass nil for honeyfileService in tests
	StorageDownloadHandler(svc, nil, logger)(c)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestStorageDownload_FileOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)

	// create file
	target := filepath.Join(base, "hello.txt")
	require.NoError(t, os.WriteFile(target, []byte("hi"), 0o644))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/download?path=/hello.txt", nil)
	c.Request = req
	c.Set("request_id", "test")

	// Pass nil for honeyfileService in tests
	StorageDownloadHandler(svc, nil, logger)(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "attachment; filename=\"hello.txt\"", w.Header().Get("Content-Disposition"))
}
