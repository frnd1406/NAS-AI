package operations

import (
	"fmt"
	"sync"
	"time"

	"github.com/nas-ai/api/src/config"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/sirupsen/logrus"
)

const (
	// Thresholds
	DiskThresholdPercent = 90.0
	RAMThresholdPercent  = 95.0

	// Deduplication window
	AlertCooldown = 24 * time.Hour
)

type AlertService struct {
	emailService *EmailService
	cfg          *config.Config
	logger       *logrus.Logger

	// State for deduplication
	lastDiskAlert time.Time
	lastRAMAlert  time.Time
	mu            sync.Mutex
}

func NewAlertService(emailService *EmailService, cfg *config.Config, logger *logrus.Logger) *AlertService {
	return &AlertService{
		emailService: emailService,
		cfg:          cfg,
		logger:       logger,
	}
}

// RunSystemChecks checks Disk and RAM usage and alerts if critical
// Should be called periodically (e.g., every 5 minutes)
func (s *AlertService) RunSystemChecks() {
	s.checkDisk()
	s.checkRAM()
}

func (s *AlertService) checkDisk() {
	// Check root partition (inside container)
	// In Docker, this usually reflects the host volume mounted at / or /mnt/data
	usage, err := disk.Usage("/")
	if err != nil {
		s.logger.WithError(err).Error("AlertService: Failed to check disk usage")
		return
	}

	if usage.UsedPercent > DiskThresholdPercent {
		s.logger.WithField("usage", usage.UsedPercent).Warn("AlertService: Disk usage critical")
		s.sendAlert("Disk", fmt.Sprintf("Server Disk Space is CRITICAL: %.2f%% used", usage.UsedPercent), &s.lastDiskAlert)
	}
}

func (s *AlertService) checkRAM() {
	v, err := mem.VirtualMemory()
	if err != nil {
		s.logger.WithError(err).Error("AlertService: Failed to check RAM usage")
		return
	}

	if v.UsedPercent > RAMThresholdPercent {
		s.logger.WithField("usage", v.UsedPercent).Warn("AlertService: RAM usage critical")
		s.sendAlert("RAM", fmt.Sprintf("Server RAM is CRITICAL: %.2f%% used", v.UsedPercent), &s.lastRAMAlert)
	}
}

func (s *AlertService) sendAlert(resourceType, message string, lastAlertPtr *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cooldown
	if time.Since(*lastAlertPtr) < AlertCooldown {
		s.logger.WithField("type", resourceType).Info("AlertService: Suppressing alert (cooldown active)")
		return
	}

	// Send Email
	s.logger.WithField("type", resourceType).Info("AlertService: Sending email alert")

	// Assuming ADMIN_EMAIL is set or we use a configured recipient
	// Using hardcoded Subject prefix for filtering
	subject := fmt.Sprintf("🚨 CRITICAL ALERT: %s", resourceType)

	// We need a target email. Using EMAIL_FROM as fallback if no admin email in config
	targetEmail := s.cfg.EmailFrom // fallback
	// Better: If your config has AdminEmail, use that. Assuming EmailFrom for now or "admin@localhost"

	// A simple text body
	body := fmt.Sprintf(`
	CRITICAL SYSTEM ALERT
	---------------------
	Resource: %s
	Message:  %s
	Time:     %s

	Please investigate immediately.	This alert will be suppressed for 24 hours.
	`, resourceType, message, time.Now().Format(time.RFC3339))

	err := s.emailService.SendGenericEmail(targetEmail, subject, body)
	if err != nil {
		s.logger.WithError(err).Error("AlertService: Failed to send alert email")
		return
	}

	// Update cooldown
	*lastAlertPtr = time.Now()
}
