package files

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
	"github.com/stretchr/testify/require"
)

func TestFileContentHandler_RequiresAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, _ := setupStorageTest(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/content?path=note.txt", nil)

	FileContentHandler(svc, logger)(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFileContentHandler_UsesAuthenticatedUserHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, root := setupStorageTest(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "note.txt"), []byte("shared-root content"), 0o644))
	require.NoError(t, os.WriteFile(userFilePath(t, svc, "note.txt"), []byte("private content"), 0o644))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/content?path=note.txt", nil)
	c.Set("request_id", "content-test")
	c.Set("user_id", downloadTestUserID)

	FileContentHandler(svc, logger)(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "private content", w.Body.String())
}

func TestFileContentHandler_BlocksAnotherUserHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, _ := setupStorageTest(t)
	otherID := "22222222-2222-2222-2222-222222222222"
	require.NoError(t, svc.EnsureUserHome(otherID))
	other, err := svc.ForUser(otherID)
	require.NoError(t, err)
	otherPath, err := other.GetFullPath("secret.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(otherPath, []byte("other user's secret"), 0o644))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/content?path="+content.UserHomeRel(otherID)+"/secret.txt", nil)
	c.Set("request_id", "content-test")
	c.Set("user_id", downloadTestUserID)

	FileContentHandler(svc, logger)(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.NotContains(t, w.Body.String(), "other user's secret")
}
