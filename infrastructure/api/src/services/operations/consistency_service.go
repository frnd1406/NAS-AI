package operations

import (
		"github.com/nas-ai/api/src/repository/files"
"context"
	"os"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultReconciliationInterval is the default time between consistency checks
	DefaultReconciliationInterval = 5 * time.Minute
	// DefaultBatchSize is the number of entries to check per batch
	DefaultBatchSize = 100
)

// ConsistencyService is the "Consistency Wächter" that detects and removes
// orphaned vectors from file_embeddings when physical files no longer exist.
// It runs as a background worker and ensures eventual consistency between
// the filesystem and the AI vector index.
type ConsistencyService struct {
	db       *sqlx.DB
	repo     *files_repo.FileEmbeddingsRepository
	basePath string
	interval time.Duration
	logger   *logrus.Logger

	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	cycle    int64
}

// NewConsistencyService creates a new ConsistencyService instance
func NewConsistencyService(
	db *sqlx.DB,
	repo *files_repo.FileEmbeddingsRepository,
	basePath string,
	interval time.Duration,
	logger *logrus.Logger,
) *ConsistencyService {
	if interval <= 0 {
		interval = DefaultReconciliationInterval
	}

	return &ConsistencyService{
		db:       db,
		repo:     repo,
		basePath: basePath,
		interval: interval,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// RunReconciliation performs a single reconciliation pass.
// This is called at startup (blocking) and periodically by the background worker.
// Thread-safe: can be called concurrently with Start().
func (s *ConsistencyService) RunReconciliation(ctx context.Context) error {
	s.cycle++
	startTime := time.Now()

	s.logger.WithFields(logrus.Fields{
		"cycle":    s.cycle,
		"basePath": s.basePath,
	}).Info("Consistency reconciliation started")

	totalOrphans := 0
	totalChunksDeleted := int64(0)
	offset := 0

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Phase A: Fetch batch of entries
		entries, err := s.repo.GetOrphanCandidates(ctx, DefaultBatchSize, offset)
		if err != nil {
			s.logger.WithError(err).Error("Reconciliation failed: could not fetch candidates")
			return err
		}

		if len(entries) == 0 {
			break // No more entries
		}

		// Phase B & C: Verify and Purge
		for _, entry := range entries {
			// Skip if no file_path in metadata
			if entry.FilePath == nil || *entry.FilePath == "" {
				s.logger.WithField("file_id", entry.FileID).Debug("Skipping entry without file_path in metadata")
				continue
			}

			filePath := *entry.FilePath

			// Check if file exists on disk
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				// File is missing - this is an orphan!
				chunksDeleted, delErr := s.repo.DeleteByFileID(ctx, entry.FileID)
				if delErr != nil {
					s.logger.WithFields(logrus.Fields{
						"file_id":   entry.FileID,
						"file_path": filePath,
						"error":     delErr.Error(),
					}).Error("Failed to delete orphaned vector")
					continue
				}

				s.logger.WithFields(logrus.Fields{
					"file_id":        entry.FileID,
					"file_path":      filePath,
					"chunks_deleted": chunksDeleted,
				}).Warn("Orphaned vector detected and removed")

				totalOrphans++
				totalChunksDeleted += chunksDeleted
			}
		}

		offset += len(entries)

		// Safety: if we got fewer than batch size, we're done
		if len(entries) < DefaultBatchSize {
			break
		}
	}

	duration := time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"cycle":           s.cycle,
		"duration_ms":     duration.Milliseconds(),
		"entries_checked": offset,
		"orphans_removed": totalOrphans,
		"chunks_deleted":  totalChunksDeleted,
	}).Info("Consistency reconciliation complete")

	return nil
}

// Start begins the background reconciliation loop.
// This method blocks until Stop() is called or context is cancelled.
func (s *ConsistencyService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Warn("ConsistencyService already running")
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.WithField("interval", s.interval.String()).Info("ConsistencyService background worker started")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("ConsistencyService stopped: context cancelled")
			return
		case <-s.stopChan:
			s.logger.Info("ConsistencyService stopped: stop signal received")
			return
		case <-ticker.C:
			if err := s.RunReconciliation(ctx); err != nil {
				s.logger.WithError(err).Error("Scheduled reconciliation failed")
			}
		}
	}
}

// Stop signals the background worker to stop.
// Safe to call multiple times.
func (s *ConsistencyService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
	s.logger.Info("ConsistencyService stop requested")
}
