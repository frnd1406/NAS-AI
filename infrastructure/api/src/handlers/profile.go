package handlers

import (
		"github.com/nas-ai/api/src/repository/auth"
"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"
)

// ProfileHandler returns the current user's profile (protected route)
func ProfileHandler(
	userRepo *auth_repo.UserRepository,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		userID := c.GetString("user_id")

		// User ID comes from auth middleware
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "User not authenticated",
					"request_id": requestID,
				},
			})
			return
		}

		// Get user from database
		ctx := c.Request.Context()
		user, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			logger.WithError(err).Error("Failed to find user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to retrieve profile",
					"request_id": requestID,
				},
			})
			return
		}

		if user == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":       "user_not_found",
					"message":    "User not found",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": user.ToResponse(),
		})
	}
}
