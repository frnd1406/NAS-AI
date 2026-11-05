package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all configuration for the API server
type Config struct {
	// Server
	Port     string
	BindAddr string

	// TLS (Transport encryption against network sniffing)
	TLSCertFile string
	TLSKeyFile  string

	// CORS (Whitelist - NO WILDCARD!)
	CORSOrigins []string

	// Rate Limiting
	RateLimitPerMin int

	// Logging
	LogLevel string

	// JWT (Phase 2 - but validate now!)
	JWTSecret     string
	JWTSecretFile string

	// Database (Phase 2)
	DatabaseURL  string
	DatabaseHost string
	DatabasePort string
	DatabaseUser string
	DatabasePass string
	DatabaseName string
	DBSSLMode    string

	// DB Pool Configuration
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime string
	DBConnMaxIdleTime string

	// Redis (Phase 2)
	RedisURL           string
	RedisHost          string
	RedisPort          string
	RedisTLS           bool
	RedisTLSSkipVerify bool

	// Email (Phase 3 - Resend)
	ResendAPIKey string
	EmailFrom    string
	FrontendURL  string

	// Cloudflare (Phase 3)
	CloudflareAPIToken string
	CloudflareR2Bucket string

	// WebAuthn / FIDO2 (YubiKey second factor). Empty RPID => feature disabled.
	WebAuthnRPID          string
	WebAuthnRPDisplayName string
	WebAuthnRPOrigins     []string

	// Roots under which operator-configurable storage paths must live
	// (backup destination, storage path, path validation).
	AllowedStorageRoots []string

	// FilesStorageRoot is the plain (unencrypted) files mount, e.g. /mnt/data.
	FilesStorageRoot string
	// VaultStorageRoot is the encrypted vault mount, e.g. /media/frnd14.
	// Must not overlap FilesStorageRoot.
	VaultStorageRoot string

	// Environment
	Environment string

	// Monitoring (agent ingestion)
	MonitoringToken string

	// AI/semantic search
	AIServiceURL string

	// LLM for RAG
	OllamaURL string
	LLMModel  string

	// Internal Security (Shared Secret)
	InternalAPISecret string

	// Abuse Prevention
	InviteCode string

	// Backup configuration
	BackupSchedule       string
	BackupRetentionCount int
	BackupStoragePath    string

	// Consistency check (orphan cleanup)
	ConsistencyCheckIntervalMin int
}

// LoadConfig loads configuration using Viper (supports .env, config.yaml, and env vars)
// CRITICAL: Fails fast if required secrets are missing!
func LoadConfig() (*Config, error) {
	return LoadConfigWithViper()
}

// LoadConfigFromEnv is the legacy configuration loader (kept for backward compatibility)
// Use LoadConfig() instead, which now uses Viper
func LoadConfigFromEnv() (*Config, error) {
	cfg := &Config{
		// Defaults
		Port:                        getEnv("PORT", "8080"),
		BindAddr:                    getEnv("BIND_ADDR", "0.0.0.0"),
		TLSCertFile:                 getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:                  getEnv("TLS_KEY_FILE", ""),
		LogLevel:                    getEnv("LOG_LEVEL", "info"),
		Environment:                 getEnv("ENV", "development"),
		RateLimitPerMin:             getEnvInt("RATE_LIMIT_PER_MIN", 100),
		BackupSchedule:              getEnv("BACKUP_SCHEDULE", "0 3 * * *"),
		BackupRetentionCount:        getEnvInt("BACKUP_RETENTION_COUNT", 7),
		BackupStoragePath:           getEnv("BACKUP_STORAGE_PATH", "/mnt/backups"),
		ConsistencyCheckIntervalMin: getEnvInt("CONSISTENCY_CHECK_INTERVAL_MIN", 5),
		AIServiceURL:                getEnv("AI_SERVICE_URL", "http://ai-knowledge-agent:5000"),
		OllamaURL:                   getEnv("OLLAMA_URL", "http://localhost:11434"),
		LLMModel:                    getEnv("LLM_MODEL", "qwen2.5:3b"),
	}

	// CORS Origins (Whitelist)
	corsOrigins := getEnv("CORS_ORIGINS", "http://localhost:5173")
	cfg.CORSOrigins = strings.Split(corsOrigins, ",")
	for i := range cfg.CORSOrigins {
		cfg.CORSOrigins[i] = strings.TrimSpace(cfg.CORSOrigins[i])
	}

	// JWT Secret - REQUIRED (even if not used in Phase 1)
	// Fail-fast principle: Better fail now than at runtime!
	if secretFile := os.Getenv("JWT_SECRET_FILE"); secretFile != "" {
		secret, err := readSecretFromFile(secretFile)
		if err != nil {
			return nil, err
		}
		cfg.JWTSecret = secret
		cfg.JWTSecretFile = secretFile
	} else {
		cfg.JWTSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if cfg.JWTSecret == "" {
			return nil, fmt.Errorf("CRITICAL: JWT_SECRET environment variable is required (no defaults allowed)")
		}
	}

	// Validate JWT secret strength (min 32 chars)
	if err := ValidateJWTSecret(cfg.JWTSecret); err != nil {
		return nil, err
	}

	// Database Configuration (Phase 2)
	// Support both DATABASE_URL (single string) or individual components
	// DB_SSLMODE enables TLS to Postgres against in-network sniffing.
	cfg.DBSSLMode = getEnv("DB_SSLMODE", defaultSSLMode(cfg.Environment))
	cfg.DatabaseURL = getEnv("DATABASE_URL", "")
	if cfg.DatabaseURL == "" {
		// Build from components (for docker-compose dev)
		cfg.DatabaseHost = getEnv("DB_HOST", "localhost")
		cfg.DatabasePort = getEnv("DB_PORT", "5433")
		cfg.DatabaseUser = getEnv("DB_USER", "nas_user")
		cfg.DatabasePass = getEnv("DB_PASSWORD", "nas_dev_password")
		cfg.DatabaseName = getEnv("DB_NAME", "nas_db")
		cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DatabaseUser, cfg.DatabasePass, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName, cfg.DBSSLMode)
	}

	// Redis Configuration (Phase 2)
	// REDIS_TLS enables TLS to Redis against in-network sniffing.
	cfg.RedisTLS = getEnvBool("REDIS_TLS", false)
	cfg.RedisTLSSkipVerify = getEnvBool("REDIS_TLS_SKIP_VERIFY", false)
	cfg.RedisURL = getEnv("REDIS_URL", "")
	if cfg.RedisURL == "" {
		cfg.RedisHost = getEnv("REDIS_HOST", "localhost")
		cfg.RedisPort = getEnv("REDIS_PORT", "6380")
		cfg.RedisURL = fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	}

	// Email Configuration (Phase 3)
	cfg.ResendAPIKey = getEnv("RESEND_API_KEY", "")
	cfg.EmailFrom = getEnv("EMAIL_FROM", "NAS.AI <noreply@your-domain.com>")
	cfg.FrontendURL = getEnv("FRONTEND_URL", "https://your-domain.com")

	// Cloudflare Configuration (Phase 3)
	cfg.CloudflareAPIToken = getEnv("CLOUDFLARE_API_TOKEN", "")
	cfg.CloudflareR2Bucket = getEnv("CLOUDFLARE_R2_BUCKET", "nas-ai-storage")

	// Allowed roots for operator-configurable storage paths
	cfg.AllowedStorageRoots = parseStorageRoots(getEnv("ALLOWED_STORAGE_ROOTS", DefaultAllowedStorageRoots))

	cfg.FilesStorageRoot = strings.TrimSpace(getEnv("FILES_STORAGE_ROOT", DefaultFilesStorageRoot))
	cfg.VaultStorageRoot = strings.TrimSpace(getEnv("VAULT_STORAGE_ROOT", DefaultVaultStorageRoot))
	if err := validateStorageRoots(cfg.FilesStorageRoot, cfg.VaultStorageRoot, cfg.AllowedStorageRoots); err != nil {
		return nil, err
	}

	// WebAuthn / FIDO2 (YubiKey second factor)
	cfg.WebAuthnRPID, cfg.WebAuthnRPDisplayName, cfg.WebAuthnRPOrigins = resolveWebAuthnConfig(
		getEnv("WEBAUTHN_RP_ID", ""),
		getEnv("WEBAUTHN_RP_DISPLAY_NAME", ""),
		getEnv("WEBAUTHN_RP_ORIGINS", ""),
		cfg.FrontendURL,
		cfg.CORSOrigins,
	)

	// Monitoring
	cfg.MonitoringToken = strings.TrimSpace(getEnv("MONITORING_TOKEN", ""))
	if len(cfg.MonitoringToken) < 16 {
		return nil, fmt.Errorf("CRITICAL: MONITORING_TOKEN must be at least 16 characters")
	}

	if strings.TrimSpace(cfg.BackupSchedule) == "" {
		return nil, fmt.Errorf("CRITICAL: BACKUP_SCHEDULE is required")
	}
	if cfg.BackupRetentionCount < 1 {
		return nil, fmt.Errorf("CRITICAL: BACKUP_RETENTION_COUNT must be >= 1")
	}
	if strings.TrimSpace(cfg.BackupStoragePath) == "" {
		return nil, fmt.Errorf("CRITICAL: BACKUP_STORAGE_PATH is required")
	}

	return cfg, nil
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt gets environment variable as int with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		_, err := fmt.Sscanf(value, "%d", &result)
		if err == nil {
			return result
		}
	}
	return fallback
}

// getEnvBool gets environment variable as bool with fallback.
// Truthy values: 1, t, true, yes, on (case-insensitive).
func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "t", "true", "yes", "on":
		return true
	case "0", "f", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// DefaultAllowedStorageRoots are the roots under which configurable storage
// paths may live. /media is included so external drives stay usable.
const DefaultAllowedStorageRoots = "/mnt,/media"

// DefaultFilesStorageRoot is the plain files volume inside the API container.
const DefaultFilesStorageRoot = "/mnt/data"

// DefaultVaultStorageRoot is the encrypted vault volume inside the API container.
const DefaultVaultStorageRoot = "/media/frnd14"

// parseStorageRoots splits and normalizes a comma-separated root list.
func parseStorageRoots(raw string) []string {
	var roots []string
	for _, r := range strings.Split(raw, ",") {
		if r = strings.TrimSpace(r); r != "" {
			roots = append(roots, r)
		}
	}
	return roots
}

// validateStorageRoots ensures files/vault roots are allowed and do not overlap.
func validateStorageRoots(filesRoot, vaultRoot string, allowed []string) error {
	if filesRoot == "" || vaultRoot == "" {
		return fmt.Errorf("CRITICAL: FILES_STORAGE_ROOT and VAULT_STORAGE_ROOT must be set")
	}
	if err := pathsafeWithinAny(allowed, filesRoot); err != nil {
		return fmt.Errorf("CRITICAL: FILES_STORAGE_ROOT invalid: %w", err)
	}
	if err := pathsafeWithinAny(allowed, vaultRoot); err != nil {
		return fmt.Errorf("CRITICAL: VAULT_STORAGE_ROOT invalid: %w", err)
	}
	if storageRootsOverlap(filesRoot, vaultRoot) {
		return fmt.Errorf("CRITICAL: FILES_STORAGE_ROOT and VAULT_STORAGE_ROOT must not overlap (%s vs %s)", filesRoot, vaultRoot)
	}
	return nil
}

func pathsafeWithinAny(roots []string, path string) error {
	// Local duplicate of pathsafe.WithinAnyRoot to avoid config→pathsafe import cycles
	// if pathsafe ever imports config. Kept intentionally small.
	absPath := filepathCleanAbs(path)
	if absPath == "" {
		return fmt.Errorf("empty path")
	}
	if len(roots) == 0 {
		return fmt.Errorf("no allowed storage roots configured")
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot := filepathCleanAbs(root)
		if absRoot == "" {
			continue
		}
		if absPath == absRoot || strings.HasPrefix(absPath, absRoot+string(os.PathSeparator)) {
			return nil
		}
	}
	return fmt.Errorf("%s is outside allowed roots %v", path, roots)
}

func storageRootsOverlap(a, b string) bool {
	aa := filepathCleanAbs(a)
	bb := filepathCleanAbs(b)
	if aa == "" || bb == "" {
		return true
	}
	if aa == bb {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(aa+sep, bb+sep) || strings.HasPrefix(bb+sep, aa+sep)
}

func filepathCleanAbs(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// Lexical clean is enough at config load (mount may not exist yet in CI).
	return filepath.Clean(p)
}

// resolveWebAuthnConfig derives the effective WebAuthn relying-party settings.
// When rpID is empty it is derived from the frontend URL host (falling back to
// "localhost"). Origins default to the frontend URL plus the CORS whitelist.
func resolveWebAuthnConfig(rpID, displayName, originsRaw, frontendURL string, corsOrigins []string) (string, string, []string) {
	if strings.TrimSpace(rpID) == "" {
		if host := hostFromURL(frontendURL); host != "" {
			rpID = host
		} else {
			rpID = "localhost"
		}
	}

	if strings.TrimSpace(displayName) == "" {
		displayName = "NAS.AI"
	}

	var origins []string
	if strings.TrimSpace(originsRaw) != "" {
		for _, o := range strings.Split(originsRaw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
	} else {
		seen := map[string]bool{}
		add := func(o string) {
			o = strings.TrimSpace(o)
			if o != "" && !seen[o] {
				seen[o] = true
				origins = append(origins, o)
			}
		}
		add(frontendURL)
		for _, o := range corsOrigins {
			add(o)
		}
	}

	return strings.TrimSpace(rpID), displayName, origins
}

// hostFromURL extracts the host (without scheme/port) from a URL-ish string.
func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.Index(raw, "://"); i != -1 {
		raw = raw[i+3:]
	}
	if i := strings.IndexAny(raw, "/"); i != -1 {
		raw = raw[:i]
	}
	if i := strings.LastIndex(raw, ":"); i != -1 {
		raw = raw[:i]
	}
	return raw
}

// defaultSSLMode returns a safe default Postgres sslmode per environment.
// Production requires TLS; other environments default to disable so local
// dev against a plaintext Postgres keeps working.
func defaultSSLMode(environment string) string {
	if environment == "production" {
		return "require"
	}
	return "disable"
}
