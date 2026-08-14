package content

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/nas-ai/api/src/pathsafe"
)

// newPathTestHoneyfileService builds the service without a repository. The
// path checks in Create/Delete run before any repository call, so the DB
// dependency is never reached for the rejection cases tested here.
func newPathTestHoneyfileService(t *testing.T, root string) *HoneyfileService {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)

	return &HoneyfileService{
		logger:      logger,
		storageRoot: root,
		cache:       make(map[string]bool),
	}
}

func TestHoneyfileService_Create_RejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	service := newPathTestHoneyfileService(t, root)

	unsafePaths := []string{
		"/etc/passwd",
		"/etc/cron.d/evil",
		"../escape",
		"../../etc/passwd",
		"sub/../../escape.txt",
	}

	for _, rawPath := range unsafePaths {
		t.Run(rawPath, func(t *testing.T) {
			// A nil repository would panic if the path check did not reject
			// the input first — reaching the DB layer is itself a failure.
			honeyfile, err := service.Create(context.Background(), rawPath, "it", nil)
			if err == nil {
				t.Fatalf("expected %q to be rejected, got honeyfile %v", rawPath, honeyfile)
			}
			if !errors.Is(err, pathsafe.ErrPathEscape) {
				t.Errorf("expected ErrPathEscape, got: %v", err)
			}
		})
	}

	// Nothing outside the root may have been created.
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape")); !os.IsNotExist(err) {
		t.Errorf("a file was created outside the storage root: %v", err)
	}
}

func TestHoneyfileService_Delete_RejectsUnsafePaths(t *testing.T) {
	service := newPathTestHoneyfileService(t, t.TempDir())

	for _, rawPath := range []string{"/etc/passwd", "../escape"} {
		if err := service.Delete(context.Background(), rawPath); err == nil {
			t.Errorf("expected delete of %q to be rejected", rawPath)
		} else if !errors.Is(err, pathsafe.ErrPathEscape) {
			t.Errorf("expected ErrPathEscape for %q, got: %v", rawPath, err)
		}
	}
}

func TestHoneyfileService_ResolvePath(t *testing.T) {
	root := t.TempDir()
	service := newPathTestHoneyfileService(t, root)

	// Relative paths and absolute paths already inside the root both resolve
	// to the same confined location.
	for _, rawPath := range []string{"decoy/passwords.txt", filepath.Join(root, "decoy/passwords.txt")} {
		got, err := service.resolvePath(rawPath)
		if err != nil {
			t.Fatalf("resolvePath(%q): %v", rawPath, err)
		}
		want := filepath.Join(root, "decoy", "passwords.txt")
		if got != want {
			t.Errorf("resolvePath(%q) = %q, want %q", rawPath, got, want)
		}
	}
}

// TestHoneyfileService_WriteHoneyfile_NoOverwrite verifies the O_EXCL behaviour
// that replaced the stat-then-write TOCTOU window: an existing file is never
// overwritten.
func TestHoneyfileService_WriteHoneyfile_NoOverwrite(t *testing.T) {
	root := t.TempDir()
	service := newPathTestHoneyfileService(t, root)

	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if err := service.writeHoneyfile(target, []byte("honeyfile content")); err == nil {
		t.Error("expected write to an existing file to fail")
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(content) != "original" {
		t.Errorf("existing file was overwritten: %q", content)
	}

	// A fresh path in a not-yet-existing subdirectory must succeed.
	fresh := filepath.Join(root, "nested", "decoy.txt")
	if err := service.writeHoneyfile(fresh, []byte("bait")); err != nil {
		t.Fatalf("writeHoneyfile on fresh path: %v", err)
	}
	if content, err := os.ReadFile(fresh); err != nil || string(content) != "bait" {
		t.Errorf("unexpected content %q (err %v)", content, err)
	}
}
