package files

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentTestUserA = "11111111-1111-4111-8111-111111111111"
	contentTestUserB = "22222222-2222-4222-8222-222222222222"
)

func newContentTestRouter(t *testing.T, userID string) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := logrus.New()

	tmpDir, err := os.MkdirTemp("", "content_scope_test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := storage.NewLocalStore(tmpDir)
	require.NoError(t, err)
	mgr := content.NewStorageManager(store, nil, nil, logger)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
	})
	router.GET("/content", FileContentHandler(mgr, logger))
	return router, tmpDir
}

func TestFileContentHandler_ScopedToUserHome(t *testing.T) {
	router, tmpDir := newContentTestRouter(t, contentTestUserA)

	// Datei im Home von User B anlegen — darf für User A unerreichbar sein.
	otherHome := filepath.Join(tmpDir, "homes", contentTestUserB)
	require.NoError(t, os.MkdirAll(otherHome, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(otherHome, "secret.txt"), []byte("geheim"), 0o644))

	// Eigene Datei anlegen (Home wird durch den ersten Request erzeugt —
	// hier direkt vorbereiten).
	ownHome := filepath.Join(tmpDir, "homes", contentTestUserA)
	require.NoError(t, os.MkdirAll(ownHome, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ownHome, "mine.txt"), []byte("meins"), 0o644))

	// Eigener Zugriff funktioniert.
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/content?path=mine.txt", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "meins", w.Body.String())

	// Zugriff auf fremdes Home über homes/<uid> muss scheitern.
	for _, p := range []string{
		"homes/" + contentTestUserB + "/secret.txt",
		"../homes/" + contentTestUserB + "/secret.txt",
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/content?path="+url.QueryEscape(p), nil))
		assert.NotEqual(t, http.StatusOK, w.Code, "path %q must not be readable", p)
		assert.NotContains(t, w.Body.String(), "geheim", "path %q leaked content", p)
	}
}

func TestFileContentHandler_RequiresAuth(t *testing.T) {
	router, _ := newContentTestRouter(t, "")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/content?path=anything.txt", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
