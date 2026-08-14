package content

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/nas-ai/api/src/pathsafe"
)

// newPathTestEncryptedStorage builds the service without storage/encryption
// dependencies. Only path confinement is exercised here, and neither
// ListEncrypted nor DeleteEncrypted touch those collaborators.
func newPathTestEncryptedStorage(t *testing.T, base string) *EncryptedStorageService {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)

	return &EncryptedStorageService{
		encryptedBasePath: base,
		logger:            logger,
	}
}

// climbingPaths escape the base directory and must be rejected outright.
var climbingPaths = []string{
	"../../etc/passwd",
	"../victim.txt",
	"subdir/../../../etc/passwd",
}

// rootedPaths look absolute but are interpreted relative to the base directory,
// so they must resolve inside the base and never reach the real filesystem root.
var rootedPaths = []string{
	"/etc/passwd",
	"/",
}

func TestEncryptedStorage_ListEncrypted_TraversalBlocked(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "enc")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}

	// A file next to (but outside) the base directory must never be listed.
	if err := os.WriteFile(filepath.Join(root, "outside.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	service := newPathTestEncryptedStorage(t, base)

	for _, relPath := range climbingPaths {
		if _, err := service.ListEncrypted(relPath); err == nil {
			t.Errorf("expected %q to be rejected", relPath)
		} else if !errors.Is(err, pathsafe.ErrPathEscape) {
			t.Errorf("expected ErrPathEscape for %q, got: %v", relPath, err)
		}
	}

	// Rooted paths stay inside the base: they may succeed, but must never
	// expose anything from outside it.
	for _, relPath := range append(rootedPaths, "..") {
		entries, err := service.ListEncrypted(relPath)
		if err != nil {
			continue // rejected outright is fine too
		}
		for _, entry := range entries {
			if entry.Name == "outside.txt" {
				t.Errorf("listing %q leaked a file outside the base directory", relPath)
			}
		}
	}
}

func TestEncryptedStorage_DeleteEncrypted_TraversalBlocked(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "enc")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}
	service := newPathTestEncryptedStorage(t, base)

	// A victim file outside the base directory must survive every attempt.
	victim := filepath.Join(root, "victim.txt")
	if err := os.WriteFile(victim, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("write victim: %v", err)
	}

	for _, relPath := range climbingPaths {
		if err := service.DeleteEncrypted(relPath); err == nil {
			t.Errorf("expected delete of %q to be rejected", relPath)
		}
	}

	for _, relPath := range append(rootedPaths, "/etc") {
		_ = service.DeleteEncrypted(relPath)
	}

	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("victim file outside the base directory was removed: %v", err)
	}
}

// TestEncryptedStorage_DeleteEncrypted_SiblingPrefixBlocked covers the bug in
// the previous strings.HasPrefix check: "<base>-evil" shares a string prefix
// with "<base>" but is a different directory.
func TestEncryptedStorage_DeleteEncrypted_SiblingPrefixBlocked(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "enc")
	sibling := filepath.Join(root, "enc-evil")

	for _, dir := range []string{base, sibling} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	secret := filepath.Join(sibling, "secret.txt.enc")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	service := newPathTestEncryptedStorage(t, base)

	if err := service.DeleteEncrypted("../enc-evil/secret.txt.enc"); err == nil {
		t.Error("expected sibling-directory delete to be rejected")
	}
	if _, err := os.Stat(secret); err != nil {
		t.Fatalf("file in sibling directory was deleted: %v", err)
	}
}

func TestEncryptedStorage_DeleteEncrypted_InsideBaseWorks(t *testing.T) {
	base := t.TempDir()
	service := newPathTestEncryptedStorage(t, base)

	target := filepath.Join(base, "note.txt.enc")
	if err := os.WriteFile(target, []byte("data"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if err := service.DeleteEncrypted("note.txt"); err != nil {
		t.Fatalf("legitimate delete failed: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, stat err: %v", err)
	}
}

func TestEncryptedStorage_SetEncryptedBasePath_Normalizes(t *testing.T) {
	base := t.TempDir()
	service := newPathTestEncryptedStorage(t, base)

	// A path containing a traversal segment must be stored in absolute,
	// cleaned form so later confinement checks compare correctly.
	messy := filepath.Join(base, "sub", "..", "sub")
	if err := service.SetEncryptedBasePath(messy); err != nil {
		t.Fatalf("SetEncryptedBasePath: %v", err)
	}

	got := service.GetEncryptedBasePath()
	want := filepath.Join(base, "sub")
	if got != want {
		t.Errorf("base path not normalized: got %q, want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("base path is not absolute: %q", got)
	}
}
