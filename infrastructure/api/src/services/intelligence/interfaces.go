package intelligence

import (
	"context"
)

// AIAgentServiceInterface defines the interface for AI agent operations
type AIAgentServiceInterface interface {
	NotifyUpload(path, fileID, mimeType, text string)
	NotifyDelete(ctx context.Context, path, fileID string) error
}
