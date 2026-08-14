package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/services/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type validatePathResponse struct {
	Valid    bool   `json:"valid"`
	Exists   bool   `json:"exists"`
	Writable bool   `json:"writable"`
	Message  string `json:"message"`
}

func TestValidatePathHandler_ValidWritablePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	// ValidatePath now confines probes to the configured storage roots; the
	// temp dir lives under /tmp, so it has to be allowed explicitly.
	t.Setenv("ALLOWED_STORAGE_ROOTS", dir)
	logger := logrus.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"path":"` + dir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/validate-path", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// ValidatePath only uses logger and os/filepath, so we can pass nil for other deps
	svc := config.NewSettingsService(nil, nil, nil, nil, logger)
	ValidatePathHandler(svc)(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp validatePathResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Valid)
	require.True(t, resp.Exists)
	require.True(t, resp.Writable)
	require.Equal(t, "path is valid", resp.Message)
}

func TestValidatePathHandler_RelativePathInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"path":"relative/path"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/validate-path", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	svc := config.NewSettingsService(nil, nil, nil, nil, logger)
	ValidatePathHandler(svc)(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp validatePathResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Valid)
	require.False(t, resp.Exists)
	require.False(t, resp.Writable)
	require.Contains(t, resp.Message, "absolute")
}

func TestValidatePathHandler_NonWritablePath(t *testing.T) {
	// root bypasses file permissions, so a 0o555 directory is still writable
	// and this test cannot observe what it is meant to check.
	if os.Geteuid() == 0 {
		t.Skip("cannot test non-writable directories as root")
	}

	gin.SetMode(gin.TestMode)
	logger := logrus.New()

	base := t.TempDir()
	t.Setenv("ALLOWED_STORAGE_ROOTS", base)
	dir := filepath.Join(base, "readonly")
	require.NoError(t, os.MkdirAll(dir, 0o555))
	defer os.Chmod(dir, 0o755) // ensure cleanup works
	defer os.RemoveAll(dir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"path":"` + dir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/validate-path", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	svc := config.NewSettingsService(nil, nil, nil, nil, logger)
	ValidatePathHandler(svc)(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp validatePathResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Valid)
	require.True(t, resp.Exists)
	require.False(t, resp.Writable)
	require.Contains(t, resp.Message, "writable")
}

func TestValidatePathHandler_OutsideAllowedRootsRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()

	// Only the temp dir is allowed, so /etc — which exists, is a directory and
	// would otherwise be probed and written to — must be refused outright.
	t.Setenv("ALLOWED_STORAGE_ROOTS", t.TempDir())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"path":"/etc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/validate-path", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	svc := config.NewSettingsService(nil, nil, nil, nil, logger)
	ValidatePathHandler(svc)(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp validatePathResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Valid)
	require.False(t, resp.Exists)
	require.False(t, resp.Writable)
	require.Contains(t, resp.Message, "außerhalb der erlaubten Verzeichnisse")
}

func TestValidatePathHandler_DefaultRootsRejectArbitraryPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()

	// No ALLOWED_STORAGE_ROOTS set: the default /mnt,/media must still apply.
	t.Setenv("ALLOWED_STORAGE_ROOTS", "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"path":"/media/../etc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/validate-path", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	svc := config.NewSettingsService(nil, nil, nil, nil, logger)
	ValidatePathHandler(svc)(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp validatePathResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Valid)
	require.Contains(t, resp.Message, "außerhalb der erlaubten Verzeichnisse")
}
