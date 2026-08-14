package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockAIAgentService mocks intelligence.AIAgentServiceInterface
type MockAIAgentService struct {
	mock.Mock
}

func (m *MockAIAgentService) NotifyUpload(path, fileID, mimeType, text string) {
	m.Called(path, fileID, mimeType, text)
}

func (m *MockAIAgentService) NotifyDelete(ctx context.Context, path, fileID string) error {
	args := m.Called(ctx, path, fileID)
	return args.Error(0)
}
