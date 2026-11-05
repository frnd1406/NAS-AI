package files

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression: handlers used to write the per-request scoped storage back into
// the factory's captured argument, so concurrent requests from different users
// could read each other's homes.
func TestScopedHandlers_ConcurrentUsersStayIsolated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)

	for _, uid := range []string{testUserA, testUserB} {
		writeHomeFile(t, base, uid, "whoami.txt", []byte(uid))
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test")
		c.Set("user_id", c.GetHeader("X-Test-User"))
		c.Next()
	})
	router.GET("/download", StorageDownloadHandler(svc, nil, logger))

	var wg sync.WaitGroup
	errs := make(chan string, 400)
	for i := 0; i < 400; i++ {
		uid := testUserA
		if i%2 == 1 {
			uid = testUserB
		}
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/download?path=whoami.txt", nil)
			req.Header.Set("X-Test-User", uid)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK || w.Body.String() != uid {
				errs <- fmt.Sprintf("user %s got %d %q", uid, w.Code, w.Body.String())
			}
		}(uid)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}

// Regression: /files/content was registered without AuthMiddleware.
func TestFilesContentRoute_RequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)
	writeHomeFile(t, base, testUserA, "secret.txt", []byte("top secret"))

	h := NewHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	router := gin.New()
	h.RegisterV1Routes(router.Group("/api/v1"))

	for _, p := range []string{"secret.txt", "homes/" + testUserA + "/secret.txt"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/files/content?path="+p, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, p)
		assert.NotContains(t, w.Body.String(), "top secret", p)
	}
}

func TestFileContent_ScopedToCallerHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)

	writeHomeFile(t, base, testUserA, "notes.txt", []byte("mine"))
	writeHomeFile(t, base, testUserB, "notes.txt", []byte("theirs"))
	require.NoError(t, os.WriteFile(filepath.Join(base, "shared-root.txt"), []byte("root"), 0o644))
	homeA, err := filepath.Abs(filepath.Join(base, content.UserHomeRel(testUserA)))
	require.NoError(t, err)

	router := gin.New()
	router.Use(asUser(testUserA))
	router.GET("/content", FileContentHandler(svc, logger))

	cases := []struct {
		path string
		code int
		body string
	}{
		{"notes.txt", http.StatusOK, "mine"},
		{"/notes.txt", http.StatusOK, "mine"},
		{filepath.ToSlash(filepath.Join(homeA, "notes.txt")), http.StatusOK, "mine"},
		{"homes/" + testUserB + "/notes.txt", http.StatusForbidden, ""},
		{"../" + testUserB + "/notes.txt", http.StatusForbidden, ""},
		{"shared-root.txt", http.StatusNotFound, ""},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/content?path="+tc.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, tc.code, w.Code, tc.path)
		if tc.body != "" {
			assert.Equal(t, tc.body, w.Body.String(), tc.path)
		} else {
			assert.NotContains(t, w.Body.String(), "theirs", tc.path)
			assert.NotContains(t, w.Body.String(), "root", tc.path)
		}
	}
}

func TestFileContent_ActiveContentServedAsText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)

	payload := []byte("<html><script>alert(1)</script></html>")
	writeHomeFile(t, base, testUserA, "page.html", payload)
	writeHomeFile(t, base, testUserA, "noext", payload)
	writeHomeFile(t, base, testUserA, "logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))

	router := gin.New()
	router.Use(asUser(testUserA))
	router.GET("/content", FileContentHandler(svc, logger))

	for _, p := range []string{"page.html", "noext", "logo.svg"} {
		req := httptest.NewRequest(http.MethodGet, "/content?path="+p, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, p)
		assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain"), "%s: %s", p, w.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "sandbox")
	}
}

// Regression: the delivery service resolved paths against the shared root even
// though the handler validated them against the user's home.
func TestSmartDownload_ScopedToCallerHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)

	writeHomeFile(t, base, testUserA, "doc.txt", []byte("home copy"))
	require.NoError(t, os.WriteFile(filepath.Join(base, "doc.txt"), []byte("root copy"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(base, "root-only.txt"), []byte("root only"), 0o644))

	deliverySvc := content.NewContentDeliveryService(svc, security.NewEncryptionService("", logger), logger)
	router := gin.New()
	router.Use(asUser(testUserA))
	router.GET("/download", SmartDownloadHandler(svc, nil, deliverySvc, logger))

	req := httptest.NewRequest(http.MethodGet, "/download?path=doc.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "home copy", w.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/download?path=root-only.txt", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, w.Body.String(), "root only")
}

// Regression: a trash ID of ".." resolved to the home itself, and
// DeleteFromTrash then removed the whole home with RemoveAll.
func TestTrashHandlers_RejectTraversalIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)
	keep := writeHomeFile(t, base, testUserA, "Documents/keep.txt", []byte("keep"))
	other := writeHomeFile(t, base, testUserB, "keep.txt", []byte("keep"))

	for _, id := range []string{"..", ".", "../Documents", "../../" + testUserB, "/"} {
		for _, handler := range []gin.HandlerFunc{
			StorageTrashDeleteHandler(svc, logger),
			StorageTrashRestoreHandler(svc, logger),
		} {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodDelete, "/trash/x", nil)
			c.Set("request_id", "test")
			c.Set("user_id", testUserA)
			c.Params = gin.Params{{Key: "id", Value: id}}

			handler(c)

			assert.NotEqual(t, http.StatusOK, w.Code, "id %q", id)
			assert.FileExists(t, keep, "id %q", id)
			assert.FileExists(t, other, "id %q", id)
		}
	}
}

// Regression: Rename joined newName onto the parent directory, so "../" in the
// new name moved files into another user's home.
func TestRename_RejectsNamesWithPathSegments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)
	src := writeHomeFile(t, base, testUserA, "a/file.txt", []byte("data"))
	writeHomeFile(t, base, testUserB, "placeholder", nil)

	for _, name := range []string{"../../" + testUserB + "/stolen.txt", "../../../escaped.txt", "sub/x.txt", "..", ".", `a\b`} {
		body, _ := json.Marshal(renameRequest{OldPath: "a/file.txt", NewName: name})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/rename", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("request_id", "test")
		c.Set("user_id", testUserA)

		StorageRenameHandler(svc, logger)(c)

		assert.Equal(t, http.StatusForbidden, w.Code, name)
		assert.FileExists(t, src, name)
	}
	assert.NoFileExists(t, filepath.Join(base, content.UserHomeRel(testUserB), "stolen.txt"))
	assert.NoFileExists(t, filepath.Join(base, "escaped.txt"))

	body, _ := json.Marshal(renameRequest{OldPath: "a/file.txt", NewName: "renamed..v2.txt"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/rename", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserA)
	StorageRenameHandler(svc, logger)(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.FileExists(t, filepath.Join(filepath.Dir(src), "renamed..v2.txt"))
}

func TestZipDownloads_SkipSymlinks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)

	outside := filepath.Join(t.TempDir(), "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("outside secret"), 0o644))
	writeHomeFile(t, base, testUserA, "dir/ok.txt", []byte("ok"))
	home := filepath.Join(base, content.UserHomeRel(testUserA))
	require.NoError(t, os.Symlink(outside, filepath.Join(home, "dir", "link.txt")))
	require.NoError(t, os.Symlink(outside, filepath.Join(home, "toplink.txt")))

	router := gin.New()
	router.Use(asUser(testUserA))
	router.GET("/download-zip", StorageDownloadZipHandler(svc, logger))
	router.POST("/batch-download", StorageBatchDownloadHandler(svc, logger))

	readZip := func(w *httptest.ResponseRecorder) map[string]string {
		t.Helper()
		require.Equal(t, http.StatusOK, w.Code)
		zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
		require.NoError(t, err)
		out := map[string]string{}
		for _, f := range zr.File {
			rc, err := f.Open()
			require.NoError(t, err)
			data, err := io.ReadAll(rc)
			require.NoError(t, err)
			require.NoError(t, rc.Close())
			out[f.Name] = string(data)
		}
		return out
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/download-zip?path=dir", nil))
	entries := readZip(w)
	assert.Equal(t, "ok", entries["ok.txt"])
	assert.NotContains(t, entries, "link.txt")

	body := `{"paths":["dir","toplink.txt"]}`
	req := httptest.NewRequest(http.MethodPost, "/batch-download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	entries = readZip(w)
	assert.Equal(t, "ok", entries["dir/ok.txt"])
	for name, data := range entries {
		assert.NotContains(t, data, "outside secret", name)
	}
}

func TestContentDisposition_EscapesFilename(t *testing.T) {
	for _, name := range []string{`evil".html`, "line\r\nX-Injected: 1", "Straße.pdf", "plain.txt"} {
		v := contentDisposition("attachment", name)
		assert.NotContains(t, v, "\r", name)
		assert.NotContains(t, v, "\n", name)
		disp, params, err := mime.ParseMediaType(v)
		require.NoError(t, err, "%s -> %s", name, v)
		assert.Equal(t, "attachment", disp)
		if !strings.ContainsAny(name, "\r\n") {
			assert.Equal(t, name, params["filename"], v)
		}
	}
}

func TestUpload_NilAIServiceDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, logger, base := setupStorageTest(t)
	logger.SetOutput(io.Discard)

	var body bytes.Buffer
	mw := newMultipart(t, &body, "docs", "hello.txt", []byte("hello world"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/upload", &body)
	c.Request.Header.Set("Content-Type", mw)
	c.Set("user_id", testUserA)

	StorageUploadHandler(svc, &MockPolicyService{}, nil, nil, logrus.New())(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.FileExists(t, filepath.Join(base, content.UserHomeRel(testUserA), "docs", "hello.txt"))
}
