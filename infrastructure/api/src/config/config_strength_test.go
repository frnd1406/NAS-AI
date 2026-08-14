package config

import (
	"testing"
)

// TestValidateSecrets proves that weak secrets/configs are rejected (CWE-798/521 mitigation).
func TestValidateSecrets(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectError bool
	}{
		{
			name: "strong secrets",
			cfg: &Config{
				JWTSecret:            "dies_ist_ein_sehr_langes_und_sicheres_geheimnis_123!",
				DatabaseURL:          "postgres://user:strongpassword@localhost:5432/db",
				RedisURL:             "localhost:6379",
				Environment:          "development",
				LogLevel:             "info",
				MonitoringToken:      "monitoring-token-123456",
				RateLimitPerMin:      100,
				BackupSchedule:       "0 3 * * *",
				BackupRetentionCount: 7,
				BackupStoragePath:    "/mnt/backups",
				AIServiceURL:         "http://ai-knowledge-agent:5000",
			},
			expectError: false,
		},
		{
			name: "short JWT secret is rejected",
			cfg: &Config{
				JWTSecret:            "short",
				DatabaseURL:          "postgres://user:pass@localhost:5432/db",
				RedisURL:             "localhost:6379",
				Environment:          "development",
				LogLevel:             "info",
				MonitoringToken:      "monitoring-token-123456",
				RateLimitPerMin:      100,
				BackupSchedule:       "0 3 * * *",
				BackupRetentionCount: 7,
				BackupStoragePath:    "/mnt/backups",
			},
			expectError: true,
		},
		{
			name: "empty database URL is rejected",
			cfg: &Config{
				JWTSecret:            "dies_ist_ein_sehr_langes_und_sicheres_geheimnis_123!",
				DatabaseURL:          "",
				RedisURL:             "localhost:6379",
				Environment:          "development",
				LogLevel:             "info",
				MonitoringToken:      "monitoring-token-123456",
				RateLimitPerMin:      100,
				BackupSchedule:       "0 3 * * *",
				BackupRetentionCount: 7,
				BackupStoragePath:    "/mnt/backups",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)

			if tt.expectError && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}
