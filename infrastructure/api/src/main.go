package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/server"
	"github.com/sirupsen/logrus"

	_ "github.com/nas-ai/api/docs" // swagger docs
)

// @title NAS.AI API
// @version 1.0
// @description Secure file storage and management API with authentication, email verification, and password reset.
// @termsOfService https://your-domain.com/terms

// @contact.name API Support
// @contact.url https://your-domain.com/support
// @contact.email support@your-domain.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host api.your-domain.com
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey CSRFToken
// @in header
// @name X-CSRF-Token
// @description CSRF token for state-changing operations

func main() {
	// 1. Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// 2. Load configuration (FAIL-FAST if secrets missing!)
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Log startup
	logger.WithFields(logrus.Fields{
		"port":         cfg.Port,
		"environment":  cfg.Environment,
		"log_level":    cfg.LogLevel,
		"cors_origins": cfg.CORSOrigins,
		"rate_limit":   cfg.RateLimitPerMin,
	}).Info("Starting NAS.AI API server")

	// 3. Create server (all initialization happens inside)
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize server")
	}
	defer srv.Close()

	// 4. Run server (blocks until shutdown signal)
	if err := srv.Run(); err != nil {
		logger.WithError(err).Fatal("Server error")
	}
}
