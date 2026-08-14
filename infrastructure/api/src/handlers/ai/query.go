package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/operations"
)

// UnifiedQueryRequest is the request body for the /query endpoint
type UnifiedQueryRequest struct {
	Query string `json:"query" binding:"required"`
}

// UnifiedQueryHandler proxies requests to the AI Knowledge Agent's /query endpoint.
// This endpoint uses AI to classify the query intent and dynamically determines:
// - Whether to return search results or an AI-generated answer
// - How many results to return based on query type
//
// Modes:
// - Async (default): Returns 202 Accepted with job_id, client polls /jobs/:id
// - Sync (?sync=true): Blocks until result is ready (backward compatibility)
func UnifiedQueryHandler(aiServiceURL string, httpClient *http.Client, jobService *operations.JobService, logger *logrus.Logger) gin.HandlerFunc {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second} // Extended timeout for LLM operations
	}
	baseURL := strings.TrimRight(aiServiceURL, "/")

	return func(c *gin.Context) {
		var req UnifiedQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid 'query' field"})
			return
		}

		if strings.TrimSpace(req.Query) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query cannot be empty"})
			return
		}

		// Check for sync mode (backward compatibility)
		syncMode := c.Query("sync") == "true"

		if syncMode || jobService == nil {
			// SYNC MODE: Direct HTTP call to AI agent (original behavior)
			handleSyncQuery(c, client, baseURL, req.Query, logger)
			return
		}

		// ASYNC MODE: Queue job and return immediately
		handleAsyncQuery(c, jobService, req.Query, logger)
	}
}

// handleSyncQuery performs a synchronous HTTP call to the AI agent
func handleSyncQuery(c *gin.Context, client *http.Client, baseURL, query string, logger *logrus.Logger) {
	logger.WithField("query", query).Info("Proxying unified query to AI agent (sync mode)")

	// Build request to AI agent
	payload := map[string]string{"query": query}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		logger.WithError(err).Error("Failed to encode query payload")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	aiReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, baseURL+"/query", &buf)
	if err != nil {
		logger.WithError(err).Error("Failed to create AI agent request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	aiReq.Header.Set("Content-Type", "application/json")

	// Call AI agent
	resp, err := client.Do(aiReq)
	if err != nil {
		logger.WithError(err).WithField("query_preview", truncateString(query, 50)).Error("AI agent call failed")
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "AI service is busy - please try again in a moment",
				"code":  "AI_TIMEOUT",
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "AI service unavailable",
				"code":  "AI_UNAVAILABLE",
			})
		}
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		logger.WithError(err).Error("Failed to read AI agent response")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read AI response"})
		return
	}

	// Forward status code and response
	c.Data(resp.StatusCode, "application/json", body)
}

// handleAsyncQuery creates a job and returns immediately
func handleAsyncQuery(c *gin.Context, jobService *operations.JobService, query string, logger *logrus.Logger) {
	logger.WithField("query", truncateString(query, 50)).Info("Creating async AI job")

	job, err := jobService.CreateJob(c.Request.Context(), query)
	if err != nil {
		logger.WithError(err).Error("Failed to create AI job")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to queue AI request",
			"code":  "JOB_CREATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"status":  "pending",
		"message": "Query submitted for processing. Poll GET /api/v1/jobs/" + job.ID + " for result.",
	})
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
