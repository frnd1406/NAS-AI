package files

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
	"github.com/stretchr/testify/require"
)

const (
	testUserA = "11111111-1111-1111-1111-111111111111"
	testUserB = "22222222-2222-2222-2222-222222222222"
)

// asUser stands in for AuthMiddleware in router-based tests.
func asUser(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("request_id", "test")
		c.Set("user_id", userID)
		c.Next()
	}
}

// writeHomeFile creates a file inside a user's home below the storage base.
func writeHomeFile(t *testing.T, base, userID, rel string, data []byte) string {
	t.Helper()
	full := filepath.Join(base, content.UserHomeRel(userID), filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, data, 0o644))
	return full
}

// newMultipart writes an upload form with a path field and one file part and
// returns its Content-Type.
func newMultipart(t *testing.T, body *bytes.Buffer, dir, filename string, data []byte) string {
	t.Helper()
	w := multipart.NewWriter(body)
	require.NoError(t, w.WriteField("path", dir))
	part, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return w.FormDataContentType()
}
