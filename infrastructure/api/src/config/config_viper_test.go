package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigWithViper_MissingSecretFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("MONITORING_TOKEN", "monitoring-token-123456")

	_, err := LoadConfigWithViper()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadConfigWithViper_WeakSecretFails(t *testing.T) {
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("JWT_SECRET", "short-secret")
	t.Setenv("MONITORING_TOKEN", "monitoring-token-123456")

	_, err := LoadConfigWithViper()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least")
}

func TestLoadConfigWithViper_LoadsSecretFromFile(t *testing.T) {
	secret := "file-secret-value-with-min-length-32-characters!"
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "jwt_secret")

	require.NoError(t, os.WriteFile(secretPath, []byte(secret), 0o600))

	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", secretPath)
	t.Setenv("MONITORING_TOKEN", "monitoring-token-123456")

	cfg, err := LoadConfigWithViper()
	require.NoError(t, err)

	assert.Equal(t, secret, cfg.JWTSecret)
	assert.Equal(t, secretPath, cfg.JWTSecretFile)
}

func TestLoadConfigWithViper_IgnoresConfigFileSecret(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	require.NoError(t, os.WriteFile(configPath, []byte("jwt_secret: default-from-file\n"), 0o600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(cwd)
	}()

	require.NoError(t, os.Chdir(dir))

	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("MONITORING_TOKEN", "monitoring-token-123456")

	_, err = LoadConfigWithViper()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
