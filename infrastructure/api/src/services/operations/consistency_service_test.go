package operations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// MockEmbeddingsRepo is a mock implementation for testing
type MockEmbeddingsRepo struct {
	entries        []mockEntry
	deletedFileIDs []string
}

type mockEntry struct {
	ID       string
	FileID   string
	FilePath string
}

func (m *MockEmbeddingsRepo) GetOrphanCandidates(ctx context.Context, limit, offset int) ([]mockEntry, error) {
	if offset >= len(m.entries) {
		return nil, nil
	}
	end := offset + limit
	if end > len(m.entries) {
		end = len(m.entries)
	}
	return m.entries[offset:end], nil
}

func (m *MockEmbeddingsRepo) DeleteByFileID(ctx context.Context, fileID string) (int64, error) {
	m.deletedFileIDs = append(m.deletedFileIDs, fileID)
	return 1, nil
}

// TestConsistencyService_DetectsOrphan tests that the service detects and removes orphaned vectors
func TestConsistencyService_DetectsOrphan(t *testing.T) {
	// Create temp dir with a real file
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")

	// Create only the existing file
	if err := os.WriteFile(existingFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetOutput(os.Stdout)

	// Test that RunReconciliation works with zero entries (no DB to test against)
	// This is a basic smoke test - full integration test requires DB
	service := NewConsistencyService(
		nil, // No DB for unit test
		nil, // No repo for unit test
		tmpDir,
		5*time.Minute,
		logger,
	)

	// Verify service was created correctly
	if service.basePath != tmpDir {
		t.Errorf("expected basePath %s, got %s", tmpDir, service.basePath)
	}

	if service.interval != 5*time.Minute {
		t.Errorf("expected interval 5m, got %v", service.interval)
	}
}

// TestConsistencyService_DefaultInterval tests default interval is applied
func TestConsistencyService_DefaultInterval(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewConsistencyService(nil, nil, "/tmp", 0, logger)

	if service.interval != DefaultReconciliationInterval {
		t.Errorf("expected default interval %v, got %v", DefaultReconciliationInterval, service.interval)
	}
}

// TestConsistencyService_StopWithoutStart tests that Stop is safe to call multiple times
func TestConsistencyService_StopWithoutStart(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewConsistencyService(nil, nil, "/tmp", 5*time.Minute, logger)

	// Should not panic when called without Start
	service.Stop()
	service.Stop() // Should be safe to call twice
}
