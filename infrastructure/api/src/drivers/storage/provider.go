package storage

import (
	"context"
	"io"
	"time"
)

// StorageEntry represents a file or directory item
type StorageEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"` // Relative path
	Size     int64     `json:"size"`
	IsDir    bool      `json:"isDir"`
	ModTime  time.Time `json:"modTime"`
	MimeType string    `json:"mimeType"`
}

// StorageProvider defines the interface for underlying storage backends (Local, S3, etc.)
type StorageProvider interface {
	// Writers
	WriteFile(ctx context.Context, path string, data io.Reader) (int64, error)
	Delete(ctx context.Context, path string) error
	Move(ctx context.Context, src, dst string) error
	Mkdir(ctx context.Context, path string) error

	// Readers
	ReadFile(ctx context.Context, path string) (io.ReadCloser, error)
	List(ctx context.Context, path string) ([]StorageEntry, error)
	Stat(ctx context.Context, path string) (*StorageEntry, error)

	// Utils
	GetFullPath(path string) (string, error) // For legacy direct access if needed
}
