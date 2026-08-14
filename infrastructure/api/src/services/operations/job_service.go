package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
)

// Redis keys and streams
const (
	JobStreamName    = "ai:jobs"
	JobResultPrefix  = "ai:results:"
	JobConsumerGroup = "ai-workers"
	JobResultTTL     = 1 * time.Hour
	JobTimeout       = 120 * time.Second
)

// JobStatus represents the status of an AI job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// AIJob represents a queued AI job
type AIJob struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	Status    JobStatus `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// AIJobResult represents the result of an AI job
type AIJobResult struct {
	JobID       string                   `json:"job_id"`
	Status      JobStatus                `json:"status"`
	Mode        string                   `json:"mode,omitempty"`       // "search" or "answer"
	Intent      map[string]interface{}   `json:"intent,omitempty"`     // Intent classification
	Answer      string                   `json:"answer,omitempty"`     // For answer mode
	Files       []map[string]interface{} `json:"files,omitempty"`      // For search mode
	Sources     []map[string]interface{} `json:"sources,omitempty"`    // Cited sources
	Confidence  string                   `json:"confidence,omitempty"` // HOCH/MITTEL/NIEDRIG
	Query       string                   `json:"query,omitempty"`      // Original query
	Error       string                   `json:"error,omitempty"`      // Error message if failed
	CreatedAt   time.Time                `json:"created_at"`
	CompletedAt *time.Time               `json:"completed_at,omitempty"`
}

// JobService manages AI job queuing via Redis
type JobService struct {
	redis  *database.RedisClient
	logger *logrus.Logger
}

// NewJobService creates a new JobService
func NewJobService(redis *database.RedisClient, logger *logrus.Logger) *JobService {
	return &JobService{
		redis:  redis,
		logger: logger,
	}
}

// CreateJob creates a new AI job and pushes it to the Redis stream
func (s *JobService) CreateJob(ctx context.Context, query string) (*AIJob, error) {
	jobID := uuid.New().String()
	now := time.Now()

	job := &AIJob{
		ID:        jobID,
		Query:     query,
		Status:    JobStatusPending,
		CreatedAt: now,
	}

	// Push to Redis Stream
	err := s.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: JobStreamName,
		Values: map[string]interface{}{
			"job_id":     jobID,
			"query":      query,
			"created_at": now.Format(time.RFC3339),
		},
	}).Err()

	if err != nil {
		s.logger.WithError(err).Error("Failed to add job to stream")
		return nil, fmt.Errorf("failed to queue job: %w", err)
	}

	// Also store initial status in a key for quick lookup
	initialResult := &AIJobResult{
		JobID:     jobID,
		Status:    JobStatusPending,
		Query:     query,
		CreatedAt: now,
	}

	resultJSON, err := json.Marshal(initialResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial result: %w", err)
	}

	err = s.redis.Set(ctx, JobResultPrefix+jobID, resultJSON, JobResultTTL).Err()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to store initial job status")
		// Non-fatal - job is still queued
	}

	s.logger.WithFields(logrus.Fields{
		"job_id": jobID,
		"query":  truncateForLog(query, 50),
	}).Info("AI job created and queued")

	return job, nil
}

// GetJobResult retrieves the result of an AI job
func (s *JobService) GetJobResult(ctx context.Context, jobID string) (*AIJobResult, error) {
	resultJSON, err := s.redis.Get(ctx, JobResultPrefix+jobID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job result: %w", err)
	}

	var result AIJobResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// UpdateJobResult updates the result of an AI job (called by worker)
func (s *JobService) UpdateJobResult(ctx context.Context, result *AIJobResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	err = s.redis.Set(ctx, JobResultPrefix+result.JobID, resultJSON, JobResultTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"job_id": result.JobID,
		"status": result.Status,
	}).Info("AI job result updated")

	return nil
}

// EnsureConsumerGroup creates the consumer group if it doesn't exist
func (s *JobService) EnsureConsumerGroup(ctx context.Context) error {
	err := s.redis.XGroupCreateMkStream(ctx, JobStreamName, JobConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
