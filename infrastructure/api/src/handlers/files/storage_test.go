package files

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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
	c.Set("user_id", downloadTestUserID)

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
	c.Set("user_id", downloadTestUserID)

	// Pass nil for honeyfileService in tests
	StorageDownloadHandler(svc, nil, logger)(c)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestStorageDownload_FileOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, _ := setupStorageTest(t)

	// create file
	target := userFilePath(t, svc, "hello.txt")
	require.NoError(t, os.WriteFile(target, []byte("hi"), 0o644))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/download?path=/hello.txt", nil)
	c.Request = req
	c.Set("request_id", "test")
	c.Set("user_id", downloadTestUserID)

	// Pass nil for honeyfileService in tests
	StorageDownloadHandler(svc, nil, logger)(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "attachment; filename=\"hello.txt\"", w.Header().Get("Content-Disposition"))
}

func TestStorageDownload_ConcurrentUserIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, _ := setupStorageTest(t)
	handler := StorageDownloadHandler(svc, nil, logger)

	users := []struct {
		id      string
		content string
	}{
		{id: "11111111-1111-1111-1111-111111111111", content: "user-a"},
		{id: "22222222-2222-2222-2222-222222222222", content: "user-b"},
	}
	for _, user := range users {
		require.NoError(t, svc.EnsureUserHome(user.id))
		scoped, err := svc.ForUser(user.id)
		require.NoError(t, err)
		path, err := scoped.GetFullPath("same-name.txt")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, []byte(user.content), 0o644))
	}

	var wg sync.WaitGroup
	errs := make(chan error, 200)
	for range 100 {
		for _, user := range users {
			user := user
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest(http.MethodGet, "/download?path=same-name.txt", nil)
				c.Set("request_id", "concurrent-test")
				c.Set("user_id", user.id)
				handler(c)
				if w.Code != http.StatusOK || w.Body.String() != user.content {
					errs <- fmt.Errorf("user %s received status %d and body %q", user.id, w.Code, w.Body.String())
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
