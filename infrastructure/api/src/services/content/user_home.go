package content

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const userHomesDir = "homes"

// UserHomeRel returns the storage-relative home directory for a user.
func UserHomeRel(userID string) string {
	return filepath.ToSlash(filepath.Join(userHomesDir, strings.TrimSpace(userID)))
}

// ForUser returns a storage manager scoped to that user's private home.
// Client paths stay relative to Home ("" = home root, "Fotos/iPhone" = …/Fotos/iPhone).
func (s *StorageManager) ForUser(userID string) (*StorageManager, error) {
	id := strings.TrimSpace(userID)
	if id == "" {
		return nil, fmt.Errorf("user id required")
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	home := UserHomeRel(id)
	clone := *s
	clone.homePrefix = home
	clone.trashPath = filepath.ToSlash(filepath.Join(home, ".trash"))
	return &clone, nil
}

// EnsureUserHome creates the private home tree (idempotent).
func (s *StorageManager) EnsureUserHome(userID string) error {
	scoped, err := s.ForUser(userID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, rel := range []string{
		scoped.homePrefix,
		filepath.ToSlash(filepath.Join(scoped.homePrefix, "Fotos")),
		filepath.ToSlash(filepath.Join(scoped.homePrefix, "Fotos", "iPhone")),
		scoped.trashPath,
	} {
		if err := s.store.Mkdir(ctx, rel); err != nil {
			return fmt.Errorf("ensure home %s: %w", rel, err)
		}
	}
	return nil
}

// mapIn translates a client-relative path into the on-disk path under the user home.
func (s *StorageManager) mapIn(rel string) (string, error) {
	if s.homePrefix == "" {
		return rel, nil
	}
	clean := strings.Trim(filepath.ToSlash(rel), "/")
	if clean == "." {
		clean = ""
	}
	if hasDotDotSegment(clean) {
		return "", ErrPathTraversal
	}
	// Never allow addressing another user's home (or the homes/ root) from the client.
	if clean == userHomesDir || strings.HasPrefix(clean, userHomesDir+"/") {
		return "", ErrPathTraversal
	}
	if clean == "" {
		return s.homePrefix, nil
	}
	return filepath.ToSlash(filepath.Join(s.homePrefix, clean)), nil
}

// mapOut strips the home prefix so API clients never see homes/{uuid}/…
func (s *StorageManager) mapOut(absRel string) string {
	if s.homePrefix == "" {
		return absRel
	}
	p := strings.Trim(filepath.ToSlash(absRel), "/")
	prefix := strings.Trim(s.homePrefix, "/")
	if p == prefix {
		return ""
	}
	if strings.HasPrefix(p, prefix+"/") {
		return strings.TrimPrefix(p, prefix+"/")
	}
	return p
}

func hasDotDotSegment(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// ensureHomeDir makes sure the scoped home exists before the first list/upload.
func (s *StorageManager) ensureHomeDir() error {
	if s.homePrefix == "" {
		return nil
	}
	if err := s.store.Mkdir(context.Background(), s.homePrefix); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}
