package files

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
)

// StorageMediaLibraryHandler returns the canonical per-user media folder and ensures it exists.
// Clients (iOS Photo Sync) must upload/list only this relative path inside the user Home.
func StorageMediaLibraryHandler(storage content.StorageService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		scoped, ok := scopedStorage(c, storage)
		if !ok {
			return
		}
		_ = scoped
		c.JSON(http.StatusOK, gin.H{
			"folder":      content.MediaLibraryFolder,
			"path":        content.MediaLibraryRel,
			"description": "Per-user photo library relative to Home (backed up under homes/{userId}/)",
		})
	}
}
