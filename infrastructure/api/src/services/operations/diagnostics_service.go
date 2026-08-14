package operations

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/security"
)

type DiagnosticResult struct {
	Component string `json:"component"`
	Status    string `json:"status"` // "HEALTHY", "DOWN"
	LatencyMs int64  `json:"latency_ms"`
	Message   string `json:"message"`
}

type SystemHealthReport struct {
	OverallStatus string             `json:"overall_status"` // "HEALTHY", "DEGRADED", "DOWN"
	Checks        []DiagnosticResult `json:"checks"`
	Timestamp     time.Time          `json:"timestamp"`
}

type DiagnosticsService struct {
	storage    content.StorageService
	encryption security.EncryptionServiceInterface
	db         *sqlx.DB // Assuming SQLx is used
}

func NewDiagnosticsService(storage content.StorageService, encryption security.EncryptionServiceInterface, db *sqlx.DB) *DiagnosticsService {
	return &DiagnosticsService{
		storage:    storage,
		encryption: encryption,
		db:         db,
	}
}

func (s *DiagnosticsService) RunFullDiagnosis(ctx context.Context) *SystemHealthReport {
	checks := []DiagnosticResult{}
	overallStatus := "HEALTHY"

	// 1. Probe Storage
	checks = append(checks, s.probeStorage(ctx))

	// 2. Probe Encryption
	checks = append(checks, s.probeEncryption(ctx))

	// 3. Probe Database
	checks = append(checks, s.probeDatabase(ctx))

	// Determine Overall Status
	for _, check := range checks {
		if check.Status != "HEALTHY" {
			overallStatus = "DEGRADED" // Or "DOWN" if critical
			// If storage or DB is down, it's pretty bad.
			if check.Component == "Database" || check.Component == "Storage" {
				overallStatus = "DOWN"
			}
		}
	}

	return &SystemHealthReport{
		OverallStatus: overallStatus,
		Checks:        checks,
		Timestamp:     time.Now(),
	}
}

// bufferFile implements multipart.File interface for testing
type bufferFile struct {
	*bytes.Reader
}

func (b *bufferFile) Close() error { return nil }
func (b *bufferFile) ReadAt(p []byte, off int64) (n int, err error) {
	return b.Reader.ReadAt(p, off)
}
func (b *bufferFile) Seek(offset int64, whence int) (int64, error) {
	return b.Reader.Seek(offset, whence)
}

func (s *DiagnosticsService) probeStorage(ctx context.Context) DiagnosticResult {
	start := time.Now()
	probeID := uuid.New().String()
	probeFilename := fmt.Sprintf(".probe_%s.tmp", probeID)
	probeContent := []byte(fmt.Sprintf("ping-%s", probeID))

	// Create dummy multipart file
	file := &bufferFile{Reader: bytes.NewReader(probeContent)}
	header := &multipart.FileHeader{
		Filename: probeFilename,
		Size:     int64(len(probeContent)),
		Header:   make(textproto.MIMEHeader),
	}
	header.Header.Set("Content-Type", "text/plain")

	// 1. Write
	// NOTE: We assume root path "" or "/"?
	// StorageService.Save usually takes a directory. Let's use ".system_probes" or just root.
	// Since we want to test ability to write, root aka "" is best if allowed.
	_, err := s.storage.Save("", file, header)
	if err != nil {
		return DiagnosticResult{
			Component: "Storage",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Write failed: %v", err),
		}
	}

	// 2. Read
	// Open returns *os.File, os.FileInfo, string, error
	f, _, _, err := s.storage.Open(probeFilename)
	if err != nil {
		// Try to cleanup even if read failed
		_ = s.storage.Delete(probeFilename)
		return DiagnosticResult{
			Component: "Storage",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Read failed: %v", err),
		}
	}
	defer f.Close()

	// Verify content
	readContent, err := io.ReadAll(f)
	if err != nil {
		_ = s.storage.Delete(probeFilename)
		return DiagnosticResult{
			Component: "Storage",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Read content failed: %v", err),
		}
	}

	if !bytes.Equal(readContent, probeContent) {
		_ = s.storage.Delete(probeFilename)
		return DiagnosticResult{
			Component: "Storage",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   "Content mismatch",
		}
	}

	// 3. Delete
	err = s.storage.Delete(probeFilename)
	if err != nil {
		return DiagnosticResult{
			Component: "Storage",
			Status:    "DEGRADED", // Not fully down, but cleanup failed
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Delete failed: %v", err),
		}
	}

	return DiagnosticResult{
		Component: "Storage",
		Status:    "HEALTHY",
		LatencyMs: time.Since(start).Milliseconds(),
		Message:   "Write/Read/Delete successful",
	}
}

func (s *DiagnosticsService) probeEncryption(ctx context.Context) DiagnosticResult {
	start := time.Now()
	testPayload := []byte("healthcheck-payload")

	// Check if configured
	if !s.encryption.IsConfigured() {
		return DiagnosticResult{
			Component: "Encryption",
			Status:    "SKIPPED",
			LatencyMs: 0,
			Message:   "Vault not configured",
		}
	}

	// Check if unlocked
	if !s.encryption.IsUnlocked() {
		return DiagnosticResult{
			Component: "Encryption",
			Status:    "LOCKED", // Healthy but locked?
			LatencyMs: 0,
			Message:   "Vault is locked",
		}
	}

	// Encrypt
	encrypted, err := s.encryption.EncryptData(testPayload)
	if err != nil {
		return DiagnosticResult{
			Component: "Encryption",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Encryption failed: %v", err),
		}
	}

	// Decrypt
	decrypted, err := s.encryption.DecryptData(encrypted)
	if err != nil {
		return DiagnosticResult{
			Component: "Encryption",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Decryption failed: %v", err),
		}
	}

	if !bytes.Equal(decrypted, testPayload) {
		return DiagnosticResult{
			Component: "Encryption",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   "Decrypted data mismatch",
		}
	}

	return DiagnosticResult{
		Component: "Encryption",
		Status:    "HEALTHY",
		LatencyMs: time.Since(start).Milliseconds(),
		Message:   "Crypto operations verified",
	}
}

func (s *DiagnosticsService) probeDatabase(ctx context.Context) DiagnosticResult {
	start := time.Now()

	if s.db == nil {
		return DiagnosticResult{
			Component: "Database",
			Status:    "SKIPPED",
			Message:   "Database not configured",
		}
	}

	// Ping or Select 1
	err := s.db.PingContext(ctx)
	if err != nil {
		return DiagnosticResult{
			Component: "Database",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Ping failed: %v", err),
		}
	}

	// Optional: Select 1 for deeper check
	var result int
	err = s.db.GetContext(ctx, &result, "SELECT 1")
	if err != nil {
		return DiagnosticResult{
			Component: "Database",
			Status:    "DOWN",
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("Query failed: %v", err),
		}
	}

	return DiagnosticResult{
		Component: "Database",
		Status:    "HEALTHY",
		LatencyMs: time.Since(start).Milliseconds(),
		Message:   "Connection verified",
	}
}
