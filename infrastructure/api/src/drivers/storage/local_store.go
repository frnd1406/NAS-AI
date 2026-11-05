package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/nas-ai/api/src/pathsafe"
)

// ErrPathTraversal aliases the shared pathsafe error so existing errors.Is
// checks in the handlers keep matching.
var ErrPathTraversal = pathsafe.ErrPathEscape

type LocalStore struct {
	basePath string
}

func NewLocalStore(basePath string) (*LocalStore, error) {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve base path: %w", err)
	}

	if err := os.MkdirAll(absBase, 0o755); err != nil {
		return nil, fmt.Errorf("ensure base path: %w", err)
	}

	return &LocalStore{
		basePath: absBase,
	}, nil
}

func (s *LocalStore) sanitizePath(rel string) (string, error) {
	return pathsafe.SafeJoin(s.basePath, rel)
}

func (s *LocalStore) WriteFile(ctx context.Context, relPath string, data io.Reader) (int64, error) {
	target, err := s.sanitizePath(relPath)
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(target)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, data)
	if err != nil {
		os.Remove(target) // Cleanup
		return 0, err
	}

	return n, nil
}

func (s *LocalStore) ReadFile(ctx context.Context, relPath string) (io.ReadCloser, error) {
	target, err := s.sanitizePath(relPath)
	if err != nil {
		return nil, err
	}

	return os.Open(target)
}

func (s *LocalStore) Delete(ctx context.Context, relPath string) error {
	target, err := s.sanitizePath(relPath)
	if err != nil {
		return err
	}
	return os.RemoveAll(target)
}

func (s *LocalStore) Move(ctx context.Context, srcRel, dstRel string) error {
	src, err := s.sanitizePath(srcRel)
	if err != nil {
		return err
	}
	dst, err := s.sanitizePath(dstRel)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	return os.Rename(src, dst)
}

func (s *LocalStore) Mkdir(ctx context.Context, relPath string) error {
	target, err := s.sanitizePath(relPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(target, 0o755)
}

func (s *LocalStore) List(ctx context.Context, relPath string) ([]StorageEntry, error) {
	target, err := s.sanitizePath(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}

	var items []StorageEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		mimeType := ""
		if !info.IsDir() {
			mimeType = mime.TypeByExtension(filepath.Ext(e.Name()))
		}

		relItem := filepath.Join(relPath, e.Name())

		items = append(items, StorageEntry{
			Name:     e.Name(),
			Path:     relItem,
			Size:     info.Size(),
			IsDir:    info.IsDir(),
			ModTime:  info.ModTime(),
			MimeType: mimeType,
		})
	}

	return items, nil
}

func (s *LocalStore) Stat(ctx context.Context, relPath string) (*StorageEntry, error) {
	target, err := s.sanitizePath(relPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	mimeType := ""
	if !info.IsDir() {
		mimeType = mime.TypeByExtension(filepath.Ext(info.Name()))
	}

	return &StorageEntry{
		Name:     info.Name(),
		Path:     relPath,
		Size:     info.Size(),
		IsDir:    info.IsDir(),
		ModTime:  info.ModTime(),
		MimeType: mimeType,
	}, nil
}

func (s *LocalStore) GetFullPath(relPath string) (string, error) {
	return s.sanitizePath(relPath)
}
