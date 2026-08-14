package settings

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// defaultOllamaURL returns the configured Ollama URL (OLLAMA_URL) or a sane
// fallback. Exposed here so the displayed default can point at an https://
// endpoint for encrypted transport.
func defaultOllamaURL() string {
	if v := strings.TrimSpace(os.Getenv("OLLAMA_URL")); v != "" {
		return v
	}
	return "http://host.docker.internal:11434"
}

// AIStatusHandler proxies requests to the AI Knowledge Agent's /status endpoint.
// Returns Ollama connection status, available models, and index stats.
func AIStatusHandler(aiServiceURL string, httpClient *http.Client, logger *logrus.Logger) gin.HandlerFunc {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	baseURL := strings.TrimRight(aiServiceURL, "/")

	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// Build request to AI agent
		aiReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", baseURL+"/status", nil)
		if err != nil {
			logger.WithError(err).Error("Failed to build AI status request")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		resp, err := client.Do(aiReq)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to call AI agent /status")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service unavailable"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			logger.WithError(err).Error("Failed to read AI agent status response")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
			return
		}

		c.Data(resp.StatusCode, "application/json", body)
	}
}

// AIReindexHandler proxies requests to the AI Knowledge Agent's /reindex endpoint.
// Triggers re-indexing of all files.
func AIReindexHandler(aiServiceURL string, httpClient *http.Client, logger *logrus.Logger) gin.HandlerFunc {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := strings.TrimRight(aiServiceURL, "/")

	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// Build request to AI agent
		aiReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", baseURL+"/reindex", nil)
		if err != nil {
			logger.WithError(err).Error("Failed to build AI reindex request")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		resp, err := client.Do(aiReq)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to call AI agent /reindex")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service unavailable"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			logger.WithError(err).Error("Failed to read AI agent reindex response")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
			return
		}

		logger.WithField("request_id", requestID).Info("AI reindex triggered")
		c.Data(resp.StatusCode, "application/json", body)
	}
}

// AISettingsResponse represents the AI settings
type AISettingsResponse struct {
	LLMModel         string   `json:"llm_model"`
	ClassifierModel  string   `json:"classifier_model"`
	EmbeddingModel   string   `json:"embedding_model"`
	Temperature      float64  `json:"temperature"`
	MaxTokens        int      `json:"max_tokens"`
	ContextDocuments int      `json:"context_documents"`
	AutoIndex        bool     `json:"auto_index"`
	IndexPaths       []string `json:"index_paths"`
	OllamaURL        string   `json:"ollama_url"`
}

// AISettingsGetHandler returns current AI settings
// Reads from setup.json if available, otherwise returns defaults
func AISettingsGetHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to load from setup.json first
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Warn("Failed to load setup config for AI settings")
		}

		// Build response with defaults, override from config if available
		settings := AISettingsResponse{
			LLMModel:         "llama3.2",
			ClassifierModel:  "llama3.2:1b",
			EmbeddingModel:   "mxbai-embed-large",
			Temperature:      0.7,
			MaxTokens:        500,
			ContextDocuments: 10,
			AutoIndex:        true,
			IndexPaths:       []string{"/mnt/data"},
			OllamaURL:        defaultOllamaURL(),
		}

		// Override with values from setup.json if available
		if config != nil {
			if config.AIModels.LLM != "" {
				settings.LLMModel = config.AIModels.LLM
			}
			if config.AIModels.Embedding != "" {
				settings.EmbeddingModel = config.AIModels.Embedding
			}
			if len(config.AIModels.IndexPaths) > 0 {
				settings.IndexPaths = config.AIModels.IndexPaths
			}
			settings.AutoIndex = config.AIModels.AutoIndex
		}

		c.JSON(http.StatusOK, settings)
	}
}

// AISettingsSaveHandler saves AI settings
// Persists AI model settings to setup.json
func AISettingsSaveHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var settings AISettingsResponse
		if err := json.NewDecoder(c.Request.Body).Decode(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings format"})
			return
		}

		// Load existing config
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Warn("Failed to load setup config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load config"})
			return
		}

		// If no config exists yet, create a new one
		if config == nil {
			config = &SetupConfig{
				Version:       "2.1.1",
				SetupComplete: true,
				StoragePath:   "/mnt/data",
			}
		}

		// Update AI model settings
		config.AIModels.LLM = settings.LLMModel
		config.AIModels.Embedding = settings.EmbeddingModel
		config.AIModels.AutoIndex = settings.AutoIndex
		if len(settings.IndexPaths) > 0 {
			config.AIModels.IndexPaths = settings.IndexPaths
		}

		// Persist to disk
		if err := saveSetupConfig(config); err != nil {
			logger.WithError(err).Error("Failed to save AI settings to config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id":      requestID,
			"llm_model":       settings.LLMModel,
			"embedding_model": settings.EmbeddingModel,
		}).Info("AI settings persisted to setup.json")

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Settings saved to setup.json",
		})
	}
}
