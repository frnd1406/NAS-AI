package mocks

import (
	"context"

	"github.com/google/uuid"
	files_repo "github.com/nas-ai/api/src/repository/files"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// MockFileEmbeddingRepository
// ============================================================

type MockFileEmbeddingRepository struct {
	mock.Mock
}

func (m *MockFileEmbeddingRepository) DeleteByFileID(ctx context.Context, fileID string) (int64, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFileEmbeddingRepository) GetOrphanCandidates(ctx context.Context, limit, offset int) ([]files_repo.FileEmbeddingEntry, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]files_repo.FileEmbeddingEntry), args.Error(1)
}

func (m *MockFileEmbeddingRepository) CountAll(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockFileEmbeddingRepository) IndexFile(ctx context.Context, fileID string, content string) error {
	args := m.Called(ctx, fileID, content)
	return args.Error(0)
}

// ============================================================
// MockHoneyfileRepository
// ============================================================

type MockHoneyfileRepository struct {
	mock.Mock
}

func (m *MockHoneyfileRepository) IsHoneyfile(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockHoneyfileRepository) Created(ctx context.Context, filePath, fileType string, createdBy *uuid.UUID) (*files_repo.Honeyfile, error) {
	return m.Create(ctx, filePath, fileType, createdBy)
}

func (m *MockHoneyfileRepository) Create(ctx context.Context, filePath, fileType string, createdBy *uuid.UUID) (*files_repo.Honeyfile, error) {
	args := m.Called(ctx, filePath, fileType, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*files_repo.Honeyfile), args.Error(1)
}

func (m *MockHoneyfileRepository) CreateHoneyfile(ctx context.Context, filePath, fileType string) error {
	args := m.Called(ctx, filePath, fileType)
	return args.Error(0)
}

func (m *MockHoneyfileRepository) RecordEvent(ctx context.Context, honeyfileID uuid.UUID, event *files_repo.HoneyfileEvent) error {
	args := m.Called(ctx, honeyfileID, event)
	return args.Error(0)
}

func (m *MockHoneyfileRepository) GetIDByPath(ctx context.Context, rawPath string) (uuid.UUID, error) {
	args := m.Called(ctx, rawPath)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockHoneyfileRepository) IncrementTrigger(ctx context.Context, rawPath string) (uuid.UUID, error) {
	args := m.Called(ctx, rawPath)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockHoneyfileRepository) GetAllPaths(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockHoneyfileRepository) EnsureTable(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHoneyfileRepository) ListAll(ctx context.Context) ([]files_repo.Honeyfile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]files_repo.Honeyfile), args.Error(1)
}

func (m *MockHoneyfileRepository) Delete(ctx context.Context, filePath string) error {
	args := m.Called(ctx, filePath)
	return args.Error(0)
}
