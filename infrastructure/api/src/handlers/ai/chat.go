package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
)

type ragResult struct {
	Answer  string   `json:"answer"`
	Sources []source `json:"sources"`
}

type source struct {
	FilePath string  `json:"file_path"`
	Score    float64 `json:"score"`
	Snippet  string  `json:"snippet"`
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// AskHandler implements RAG (Retrieval Augmented Generation):
// 1. Generate embedding for the question
// 2. Find top-k relevant documents
// 3. Send question + context to LLM
// 4. Return intelligent answer with sources
// AskHandler proxies the request to the AI Knowledge Agent's /rag endpoint
func AskHandler(db *database.DB, aiServiceURL, ollamaURL, llmModel string, httpClient *http.Client, logger *logrus.Logger) gin.HandlerFunc {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}

	// AI Agent URL (internal docker network). Derived from the configured
	// AI_SERVICE_URL so it can be switched to https:// for encrypted transport.
	ragURL := strings.TrimRight(aiServiceURL, "/") + "/rag"

	return func(c *gin.Context) {
		question := strings.TrimSpace(c.Query("q"))
		if question == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing query parameter 'q'"})
			return
		}

		logger.Infof("Proxying RAG query to AI Agent: %s", question)

		// Prepare payload for AI Agent
		payload := map[string]interface{}{
			"query": question,
			"top_k": 5,
		}

		jsonBody, err := json.Marshal(payload)
		if err != nil {
			logger.WithError(err).Error("Failed to marshal RAG payload")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Call AI Agent
		req, err := http.NewRequestWithContext(c.Request.Context(), "POST", ragURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			logger.WithError(err).Error("Failed to create RAG request")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			logger.WithError(err).Error("Failed to call AI Agent")
			c.JSON(http.StatusBadGateway, gin.H{"error": "AI service unavailable"})
			return
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.WithError(err).Error("Failed to read AI Agent response")
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid response from AI service"})
			return
		}

		if resp.StatusCode != http.StatusOK {
			logger.Errorf("AI Agent returned status %d: %s", resp.StatusCode, string(body))
			c.JSON(resp.StatusCode, gin.H{"error": "AI service error", "details": string(body)})
			return
		}

		// Forward JSON response directly
		c.Data(http.StatusOK, "application/json", body)
	}
}

// generateWithOllama is deprecated/unused in this version as we proxy to Python agent
func generateWithOllama(ctx context.Context, client *http.Client, url, model, prompt string) (string, error) {
	return "", nil
}
