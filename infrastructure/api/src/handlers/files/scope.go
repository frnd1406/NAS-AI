package files

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
)

// scopedStorage returns a per-user home view of the shared storage root.
// Without this, clients would list the API WORKDIR or the shared RAID root.
func scopedStorage(c *gin.Context, storage content.StorageService) (content.StorageService, bool) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil, false
	}
	mgr, ok := storage.(*content.StorageManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not available"})
		return nil, false
	}
	scoped, err := mgr.ForUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return nil, false
	}
	if err := mgr.EnsureUserHome(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare home"})
		return nil, false
	}
	return scoped, true
}
