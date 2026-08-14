package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfigWithViper loads configuration using Viper
// Supports: .env files, environment variables, config.yaml
// CRITICAL: Validates required fields and fails fast!
func LoadConfigWithViper() (*Config, error) {
	v := viper.New()

	// === Configuration Sources (priority order) ===
	// 1. Environment variables (highest priority)
	// 2. .env file
	// 3. config.yaml file
	// 4. Default values (lowest priority)

	// Set config file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")            // Look in current directory
	v.AddConfigPath("./config")     // Look in config directory
	v.AddConfigPath("/etc/nas-api") // Look in /etc for production

	// Read config file (optional - won't fail if not found)
	_ = v.ReadInConfig()

	// Enable environment variable support
	v.AutomaticEnv()
	v.SetEnvPrefix("") // No prefix, use exact env var names
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// === Set Default Values ===
	setDefaults(v)

	// === Bind Environment Variables ===
	bindEnvVars(v)

	// === Load Secrets with File Support ===
	jwtSecret, err := loadSecretValue(v, "jwt_secret", "jwt_secret_file")
	if err != nil {
		return nil, err
	}

	monitoringToken, err := loadSecretValue(v, "monitoring_token", "monitoring_token_file")
	if err != nil {
		return nil, err
	}

	dbPassword, err := loadSecretValue(v, "db_password", "db_password_file")
	if err != nil {
		return nil, err
	}

	// === Build Config Struct ===
	cfg := &Config{
		// Server
		Port:        v.GetString("port"),
		BindAddr:    v.GetString("bind_addr"),
		Environment: v.GetString("env"),
		LogLevel:    v.GetString("log_level"),

		// TLS (Transport encryption against network sniffing)
		TLSCertFile: v.GetString("tls_cert_file"),
		TLSKeyFile:  v.GetString("tls_key_file"),

		// Rate Limiting
		RateLimitPerMin: v.GetInt("rate_limit_per_min"),

		// CORS
		CORSOrigins: parseStringSlice(v.GetString("cors_origins")),

		// JWT
		JWTSecret:     jwtSecret,
		JWTSecretFile: v.GetString("jwt_secret_file"),

		// Monitoring
		MonitoringToken: monitoringToken,

		// Database
		DatabaseURL:       v.GetString("database_url"),
		DatabaseHost:      v.GetString("db_host"),
		DatabasePort:      v.GetString("db_port"),
		DatabaseUser:      v.GetString("db_user"),
		DatabasePass:      dbPassword,
		DatabaseName:      v.GetString("db_name"),
		DBMaxOpenConns:    v.GetInt("db_max_open_conns"),
		DBMaxIdleConns:    v.GetInt("db_max_idle_conns"),
		DBConnMaxLifetime: v.GetString("db_conn_max_lifetime"),
		DBConnMaxIdleTime: v.GetString("db_conn_max_idle_time"),
		DBSSLMode:         v.GetString("db_sslmode"),

		// Redis
		RedisURL:           v.GetString("redis_url"),
		RedisHost:          v.GetString("redis_host"),
		RedisPort:          v.GetString("redis_port"),
		RedisTLS:           v.GetBool("redis_tls"),
		RedisTLSSkipVerify: v.GetBool("redis_tls_skip_verify"),

		// Email
		ResendAPIKey: v.GetString("resend_api_key"),
		EmailFrom:    v.GetString("email_from"),
		FrontendURL:  v.GetString("frontend_url"),

		// Cloudflare
		CloudflareAPIToken: v.GetString("cloudflare_api_token"),
		CloudflareR2Bucket: v.GetString("cloudflare_r2_bucket"),

		// WebAuthn (resolved below once FrontendURL/CORS are known)

		// Backup
		BackupSchedule:       v.GetString("backup_schedule"),
		BackupRetentionCount: v.GetInt("backup_retention_count"),
		BackupStoragePath:    v.GetString("backup_storage_path"),

		// AI/semantic search
		AIServiceURL: v.GetString("ai_service_url"),

		// LLM for RAG
		OllamaURL: v.GetString("ollama_url"),
		LLMModel:  v.GetString("llm_model"),

		// Internal Security
		InternalAPISecret: v.GetString("internal_api_secret"),
		InviteCode:        v.GetString("invite_code"),

		// Consistency check
		ConsistencyCheckIntervalMin: v.GetInt("consistency_check_interval_min"),
	}

	// Resolve sslmode default per environment if unset
	if cfg.DBSSLMode == "" {
		cfg.DBSSLMode = defaultSSLMode(cfg.Environment)
	}

	// Build DatabaseURL if not provided
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DatabaseUser,
			cfg.DatabasePass,
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseName,
			cfg.DBSSLMode,
		)
	}

	// Build RedisURL if not provided
	if cfg.RedisURL == "" {
		cfg.RedisURL = fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	}

	// Allowed roots for operator-configurable storage paths
	cfg.AllowedStorageRoots = parseStorageRoots(v.GetString("allowed_storage_roots"))

	// Resolve WebAuthn relying-party settings (derive from FrontendURL/CORS).
	cfg.WebAuthnRPID, cfg.WebAuthnRPDisplayName, cfg.WebAuthnRPOrigins = resolveWebAuthnConfig(
		v.GetString("webauthn_rp_id"),
		v.GetString("webauthn_rp_display_name"),
		v.GetString("webauthn_rp_origins"),
		cfg.FrontendURL,
		cfg.CORSOrigins,
	)

	// === Additional Validation ===
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// setDefaults sets default values for all configuration options
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("port", "8080")
	v.SetDefault("bind_addr", "0.0.0.0")
	v.SetDefault("env", "development")
	v.SetDefault("log_level", "info")

	// TLS defaults (empty => in dev an ephemeral self-signed cert is used)
	v.SetDefault("tls_cert_file", "")
	v.SetDefault("tls_key_file", "")

	// Allowed roots for configurable storage paths
	v.SetDefault("allowed_storage_roots", DefaultAllowedStorageRoots)

	// Rate limiting defaults
	v.SetDefault("rate_limit_per_min", 100)

	// CORS defaults
	v.SetDefault("cors_origins", "http://localhost:5173")

	// Database defaults
	v.SetDefault("db_host", "localhost")
	v.SetDefault("db_port", "5433")
	v.SetDefault("db_user", "nas_user")
	v.SetDefault("db_password", "nas_dev_password")
	v.SetDefault("db_name", "nas_db")
	v.SetDefault("db_max_open_conns", 25)
	v.SetDefault("db_max_idle_conns", 5)
	v.SetDefault("db_conn_max_lifetime", "5m")
	v.SetDefault("db_conn_max_idle_time", "10m")

	// Redis defaults
	v.SetDefault("redis_host", "localhost")
	v.SetDefault("redis_port", "6380")
	v.SetDefault("redis_tls", false)
	v.SetDefault("redis_tls_skip_verify", false)

	// Email defaults
	v.SetDefault("email_from", "NAS.AI <noreply@your-domain.com>")
	v.SetDefault("frontend_url", "https://your-domain.com")
	v.SetDefault("cloudflare_r2_bucket", "nas-ai-storage")

	// Backup defaults
	v.SetDefault("backup_schedule", "0 3 * * *")
	v.SetDefault("backup_retention_count", 7)
	v.SetDefault("backup_storage_path", "/mnt/backups")

	// AI agent default
	v.SetDefault("ai_service_url", "http://ai-knowledge-agent:5000")

	// LLM for RAG
	v.SetDefault("ollama_url", "http://localhost:11434")
	v.SetDefault("llm_model", "qwen2.5:3b")

	// Consistency check (orphan cleanup)
	v.SetDefault("consistency_check_interval_min", 5)
}

// bindEnvVars explicitly binds environment variables to config keys
func bindEnvVars(v *viper.Viper) {
	// Server
	_ = v.BindEnv("port", "PORT")
	_ = v.BindEnv("bind_addr", "BIND_ADDR")
	_ = v.BindEnv("env", "ENV")
	_ = v.BindEnv("log_level", "LOG_LEVEL")

	// TLS
	_ = v.BindEnv("tls_cert_file", "TLS_CERT_FILE")
	_ = v.BindEnv("tls_key_file", "TLS_KEY_FILE")

	// Storage roots
	_ = v.BindEnv("allowed_storage_roots", "ALLOWED_STORAGE_ROOTS")

	// WebAuthn
	_ = v.BindEnv("webauthn_rp_id", "WEBAUTHN_RP_ID")
	_ = v.BindEnv("webauthn_rp_display_name", "WEBAUTHN_RP_DISPLAY_NAME")
	_ = v.BindEnv("webauthn_rp_origins", "WEBAUTHN_RP_ORIGINS")

	// Rate limiting
	_ = v.BindEnv("rate_limit_per_min", "RATE_LIMIT_PER_MIN")

	// CORS
	_ = v.BindEnv("cors_origins", "CORS_ORIGINS")

	// JWT
	_ = v.BindEnv("jwt_secret", "JWT_SECRET")
	_ = v.BindEnv("jwt_secret_file", "JWT_SECRET_FILE")
	_ = v.BindEnv("monitoring_token", "MONITORING_TOKEN")
	_ = v.BindEnv("monitoring_token_file", "MONITORING_TOKEN_FILE")

	// Database
	_ = v.BindEnv("database_url", "DATABASE_URL")
	_ = v.BindEnv("db_host", "DB_HOST")
	_ = v.BindEnv("db_port", "DB_PORT")
	_ = v.BindEnv("db_user", "DB_USER")
	_ = v.BindEnv("db_password", "DB_PASSWORD")
	_ = v.BindEnv("db_password_file", "POSTGRES_PASSWORD_FILE")
	_ = v.BindEnv("db_name", "DB_NAME")
	_ = v.BindEnv("db_max_open_conns", "DB_MAX_OPEN_CONNS")
	_ = v.BindEnv("db_max_idle_conns", "DB_MAX_IDLE_CONNS")
	_ = v.BindEnv("db_conn_max_lifetime", "DB_CONN_MAX_LIFETIME")
	_ = v.BindEnv("db_conn_max_idle_time", "DB_CONN_MAX_IDLE_TIME")
	_ = v.BindEnv("db_sslmode", "DB_SSLMODE")

	// Redis
	_ = v.BindEnv("redis_url", "REDIS_URL")
	_ = v.BindEnv("redis_host", "REDIS_HOST")
	_ = v.BindEnv("redis_port", "REDIS_PORT")
	_ = v.BindEnv("redis_tls", "REDIS_TLS")
	_ = v.BindEnv("redis_tls_skip_verify", "REDIS_TLS_SKIP_VERIFY")

	// Email
	_ = v.BindEnv("resend_api_key", "RESEND_API_KEY")
	_ = v.BindEnv("email_from", "EMAIL_FROM")
	_ = v.BindEnv("frontend_url", "FRONTEND_URL")

	// Cloudflare
	_ = v.BindEnv("cloudflare_api_token", "CLOUDFLARE_API_TOKEN")
	_ = v.BindEnv("cloudflare_r2_bucket", "CLOUDFLARE_R2_BUCKET")

	// Backup
	_ = v.BindEnv("backup_schedule", "BACKUP_SCHEDULE")
	_ = v.BindEnv("backup_retention_count", "BACKUP_RETENTION_COUNT")
	_ = v.BindEnv("backup_storage_path", "BACKUP_STORAGE_PATH")

	// AI agent
	_ = v.BindEnv("ai_service_url", "AI_SERVICE_URL")

	// LLM for RAG
	_ = v.BindEnv("ollama_url", "OLLAMA_URL")
	_ = v.BindEnv("llm_model", "LLM_MODEL")

	// Internal Security
	_ = v.BindEnv("internal_api_secret", "INTERNAL_API_SECRET")
	_ = v.BindEnv("invite_code", "INVITE_CODE")

	// Consistency check
	_ = v.BindEnv("consistency_check_interval_min", "CONSISTENCY_CHECK_INTERVAL_MIN")
}

// validateConfig validates the configuration after it's been loaded
func validateConfig(cfg *Config) error {
	// Validate JWT secret strength
	if err := ValidateJWTSecret(cfg.JWTSecret); err != nil {
		return err
	}

	if len(cfg.MonitoringToken) < 16 {
		return fmt.Errorf("CRITICAL: MONITORING_TOKEN must be at least 16 characters")
	}

	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return fmt.Errorf("CRITICAL: Database URL is required")
	}

	if strings.TrimSpace(cfg.RedisURL) == "" {
		return fmt.Errorf("CRITICAL: Redis URL is required")
	}

	// Validate CORS origins (no wildcards allowed)
	for _, origin := range cfg.CORSOrigins {
		if origin == "*" {
			return fmt.Errorf("CRITICAL: CORS wildcard (*) is not allowed - use explicit origins")
		}
	}

	if strings.TrimSpace(cfg.BackupSchedule) == "" {
		return fmt.Errorf("CRITICAL: Backup schedule is required")
	}
	if cfg.BackupRetentionCount < 1 {
		return fmt.Errorf("CRITICAL: Backup retention count must be >= 1")
	}
	if strings.TrimSpace(cfg.BackupStoragePath) == "" {
		return fmt.Errorf("CRITICAL: Backup storage path is required")
	}

	if strings.TrimSpace(cfg.AIServiceURL) == "" {
		return fmt.Errorf("CRITICAL: AI_SERVICE_URL is required for semantic search")
	}

	// Validate environment
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
	}
	if !validEnvs[cfg.Environment] {
		return fmt.Errorf("CRITICAL: Invalid environment '%s' (must be: development, staging, production, or test)", cfg.Environment)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.LogLevel] {
		return fmt.Errorf("CRITICAL: Invalid log level '%s' (must be: debug, info, warn, or error)", cfg.LogLevel)
	}

	return nil
}

// parseStringSlice parses a comma-separated string into a slice
func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// PrintConfig prints the configuration (with secrets masked)
func PrintConfig(cfg *Config) {
	fmt.Println("=== Configuration ===")
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("Port: %s\n", cfg.Port)
	fmt.Printf("Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("Rate Limit: %d req/min\n", cfg.RateLimitPerMin)
	fmt.Printf("CORS Origins: %v\n", cfg.CORSOrigins)
	fmt.Printf("Backup Schedule: %s\n", cfg.BackupSchedule)
	fmt.Printf("Backup Retention: %d\n", cfg.BackupRetentionCount)
	fmt.Printf("Backup Storage Path: %s\n", cfg.BackupStoragePath)
	fmt.Printf("JWT Secret: %s\n", maskSecret(cfg.JWTSecret))
	fmt.Printf("Database: %s\n", maskConnectionString(cfg.DatabaseURL))
	fmt.Printf("Redis: %s\n", cfg.RedisURL)
	fmt.Printf("Frontend URL: %s\n", cfg.FrontendURL)
	fmt.Printf("AI Service URL: %s\n", cfg.AIServiceURL)
	fmt.Println("=====================")
}

// maskSecret masks a secret, showing only first and last 4 chars
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// maskConnectionString masks password in connection string
func maskConnectionString(s string) string {
	if strings.Contains(s, "@") {
		parts := strings.Split(s, "@")
		if len(parts) == 2 {
			userPart := strings.Split(parts[0], ":")
			if len(userPart) == 2 {
				return userPart[0] + ":***@" + parts[1]
			}
		}
	}
	return s
}

// loadSecretValue loads a secret from a file (if specified) or falls back to the direct value
func loadSecretValue(v *viper.Viper, valKey, fileKey string) (string, error) {
	if file := strings.TrimSpace(v.GetString(fileKey)); file != "" {
		secret, err := readSecretFromFile(file)
		if err != nil {
			return "", err
		}
		return secret, nil
	}
	return v.GetString(valKey), nil
}
