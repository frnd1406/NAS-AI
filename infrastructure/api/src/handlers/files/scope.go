package files

import (
	"errors"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
)

// scopedStorage returns a per-user home view of the shared storage root.
// Without this, clients would list the API WORKDIR or the shared RAID root.
//
// Callers must keep the result in a request-local variable. Writing it back to
// the handler factory's captured argument shares it across concurrent requests,
// so one user's request could end up operating on another user's home.
func scopedStorage(c *gin.Context, storage content.StorageService) (content.StorageService, bool) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil, false
	}
	scoper, ok := storage.(content.UserScoper)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not available"})
		return nil, false
	}
	scoped, err := scoper.ScopeToUser(userID)
	if errors.Is(err, content.ErrInvalidUserID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare home"})
		return nil, false
	}
	return scoped, true
}

// contentDisposition builds a Content-Disposition header value. File names are
// user-controlled, so quotes, control characters and non-ASCII must be encoded
// (RFC 2231) instead of being interpolated into the header verbatim.
func contentDisposition(disposition, filename string) string {
	if v := mime.FormatMediaType(disposition, map[string]string{"filename": filename}); v != "" {
		return v
	}
	return disposition
}
