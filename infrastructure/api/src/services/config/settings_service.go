package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	settings_repo "github.com/nas-ai/api/src/repository/settings"

	"github.com/nas-ai/api/src/config"

	"github.com/nas-ai/api/src/pathsafe"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// SettingsService handles system configuration and validation management.
type SettingsService struct {
	cfg                  *config.Config
	settingsRepo         *settings_repo.SystemSettingsRepository
	backupService        *operations.BackupService
	restartSchedulerFunc func() error
	logger               *logrus.Logger
}

// BackupSettingsDTO represents the parameters for updating backup configuration
type BackupSettingsDTO struct {
	Schedule  string
	Retention int
	Path      string
}

// NewSettingsService creates a new instance of SettingsService.
// onRestartScheduler is a callback to avoid import cycles with the scheduler package.
func NewSettingsService(
	cfg *config.Config,
	settingsRepo *settings_repo.SystemSettingsRepository,
	backupService *operations.BackupService,
	onRestartScheduler func() error,
	logger *logrus.Logger,
) *SettingsService {
	return &SettingsService{
		cfg:                  cfg,
		settingsRepo:         settingsRepo,
		backupService:        backupService,
		restartSchedulerFunc: onRestartScheduler,
		logger:               logger,
	}
}

// AllowedStorageRootsFromEnv resolves the configurable storage roots without a
// loaded *config.Config. It mirrors the loaders' behaviour (env
// ALLOWED_STORAGE_ROOTS, default config.DefaultAllowedStorageRoots) and exists
// for call sites — such as the setup handler — that are wired without a config
// reference. It never returns an empty slice, so callers always fail closed
// against a concrete root list rather than against "deny everything".
func AllowedStorageRootsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_STORAGE_ROOTS"))
	if raw == "" {
		raw = config.DefaultAllowedStorageRoots
	}

	var roots []string
	for _, r := range strings.Split(raw, ",") {
		if r = strings.TrimSpace(r); r != "" {
			roots = append(roots, r)
		}
	}
	if len(roots) == 0 {
		roots = []string{config.DefaultAllowedStorageRoots}
	}
	return roots
}

// allowedStorageRoots returns the roots operator-supplied paths are confined
// to. It prefers the loaded config and falls back to the environment when the
// service was constructed without one (tests, partial wiring).
func (s *SettingsService) allowedStorageRoots() []string {
	if s.cfg != nil && len(s.cfg.AllowedStorageRoots) > 0 {
		return s.cfg.AllowedStorageRoots
	}
	return AllowedStorageRootsFromEnv()
}

// GetBackupSettings returns the current backup configuration.
func (s *SettingsService) GetBackupSettings() BackupSettingsDTO {
	return BackupSettingsDTO{
		Schedule:  s.cfg.BackupSchedule,
		Retention: s.cfg.BackupRetentionCount,
		Path:      s.cfg.BackupStoragePath,
	}
}

// UpdateBackupSettings validates and persists new backup settings.
func (s *SettingsService) UpdateBackupSettings(ctx context.Context, settings BackupSettingsDTO) error {
	schedule := strings.TrimSpace(settings.Schedule)
	path := filepath.Clean(strings.TrimSpace(settings.Path))

	// 1. Validation
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(schedule); err != nil {
		return fmt.Errorf("invalid schedule format: %w", err)
	}
	if settings.Retention < 1 {
		return fmt.Errorf("retention must be >= 1")
	}
	if path == "" || path == "." || path == string(os.PathSeparator) {
		return fmt.Errorf("invalid backup path")
	}

	// 2. Logic Application (set path in backup service)
	if err := s.backupService.SetBackupPath(path); err != nil {
		return fmt.Errorf("failed to apply backup path: %w", err)
	}

	// 3. Update Memory State (Global Config)
	s.cfg.BackupSchedule = schedule
	s.cfg.BackupRetentionCount = settings.Retention
	s.cfg.BackupStoragePath = path

	// 4. Persistence
	err := s.settingsRepo.UpsertMany(ctx, map[string]string{
		settings_repo.SystemSettingBackupSchedule:  schedule,
		settings_repo.SystemSettingBackupRetention: fmt.Sprintf("%d", settings.Retention),
		settings_repo.SystemSettingBackupPath:      path,
	})
	if err != nil {
		return fmt.Errorf("failed to persist settings: %w", err)
	}

	// 5. Side Effects (Restart Scheduler)
	if s.restartSchedulerFunc != nil {
		if err := s.restartSchedulerFunc(); err != nil {
			return fmt.Errorf("failed to restart scheduler: %w", err)
		}
	}

	s.logger.WithField("path", path).Info("Backup settings updated successfully")
	return nil
}

// PathValidationResult details the check results for a filesystem path
type PathValidationResult struct {
	Valid    bool   `json:"valid"`
	Exists   bool   `json:"exists"`
	Writable bool   `json:"writable"`
	Message  string `json:"message"`
}

// ValidatePath checks if a path is absolute, exists, is a directory, and is writable.
func (s *SettingsService) ValidatePath(pathInput string) PathValidationResult {
	path := filepath.Clean(strings.TrimSpace(pathInput))
	res := PathValidationResult{
		Valid:    false,
		Exists:   false,
		Writable: false,
		Message:  "",
	}

	if path == "" {
		res.Message = "path is required"
		return res
	}
	if !filepath.IsAbs(path) {
		res.Message = "path must be absolute"
		return res
	}

	// Confine the probe to the configured storage roots. Without this the
	// endpoint doubles as a filesystem oracle for the whole host (existence,
	// directory-ness, writability) and creates a temp file wherever it points.
	if err := pathsafe.WithinAnyRoot(s.allowedStorageRoots(), path); err != nil {
		s.logger.WithField("path", path).Warn("validate path: rejected path outside allowed roots")
		res.Message = "Pfad liegt außerhalb der erlaubten Verzeichnisse"
		return res
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			res.Message = "path does not exist"
		} else {
			s.logger.WithError(err).Warn("validate path: stat failed")
			res.Message = "unable to read path metadata"
		}
		return res
	}

	res.Exists = true

	if !info.IsDir() {
		res.Message = "path must be a directory"
		return res
	}

	// Write Check
	tmp, err := os.CreateTemp(path, ".nas-path-check-*")
	if err != nil {
		s.logger.WithError(err).Warn("validate path: write check failed")
		res.Message = "path is not writable"
		return res
	}
	tmp.Close()
	os.Remove(tmp.Name())

	res.Writable = true
	res.Valid = true
	res.Message = "path is valid"

	return res
}
