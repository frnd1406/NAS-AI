package settings

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// SetupHandler must refuse operator-supplied storage paths outside the allowed
// roots before os.MkdirAll runs — otherwise any authenticated user can have an
// arbitrary host directory created and persisted.
func TestSetupHandler_RejectsPathOutsideAllowedRoots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	allowed := t.TempDir()
	t.Setenv("ALLOWED_STORAGE_ROOTS", allowed)

	// A sibling of the allowed root: rejected, and never created.
	victim := filepath.Join(t.TempDir(), "should-not-exist")

	for _, path := range []string{victim, "/etc/nas-evil", allowed + "/../escaped"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := strings.NewReader(`{"storagePath":"` + path + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/system/setup", body)
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		SetupHandler(logrus.New())(c)

		require.Equal(t, http.StatusBadRequest, w.Code, "path %q should be rejected", path)
		require.Contains(t, w.Body.String(), "invalid storage path")
	}

	_, err := os.Stat(victim)
	require.True(t, os.IsNotExist(err), "handler must not create the rejected directory")
}

func TestSetupHandler_RejectsRelativePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ALLOWED_STORAGE_ROOTS", t.TempDir())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"storagePath":"relative/dir"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/setup", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	SetupHandler(logrus.New())(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid storage path")
}
