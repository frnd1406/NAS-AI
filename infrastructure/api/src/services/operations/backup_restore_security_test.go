package operations

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func newTestBackupService(t *testing.T) (*BackupService, string, string) {
	t.Helper()

	root := t.TempDir()
	dataPath := filepath.Join(root, "data")
	backupPath := filepath.Join(root, "backups")

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// The temp root is the only allowed root for this test.
	svc, err := NewBackupService(dataPath, backupPath, []string{root}, logger)
	if err != nil {
		t.Fatalf("NewBackupService failed: %v", err)
	}
	return svc, dataPath, root
}

// writeArchive builds a .tar.gz containing the given entries.
func writeArchive(t *testing.T, path string, entries map[string]string) {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range entries {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o777, // deliberately permissive: restore must not honor this
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write body %q: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
}

// A crafted archive must not be able to write outside the data directory.
func TestRestoreBackup_ZipSlipBlocked(t *testing.T) {
	svc, dataPath, root := newTestBackupService(t)

	archive := filepath.Join(svc.backupPath, "evil.tar.gz")
	writeArchive(t, archive, map[string]string{
		"../../escaped.txt": "owned",
	})

	err := svc.RestoreBackup("evil.tar.gz")
	if err == nil {
		t.Fatal("expected restore of a traversal archive to fail")
	}
	if !strings.Contains(err.Error(), "escapes") && !strings.Contains(err.Error(), "archive entry") {
		t.Fatalf("expected a path-escape error, got: %v", err)
	}

	// Nothing may have been written outside the data directory.
	for _, p := range []string{filepath.Join(root, "escaped.txt"), filepath.Join(filepath.Dir(root), "escaped.txt")} {
		if _, statErr := os.Stat(p); statErr == nil {
			t.Fatalf("file escaped the data dir: %s", p)
		}
	}
	if _, statErr := os.Stat(filepath.Join(dataPath, "escaped.txt")); statErr == nil {
		t.Fatal("unexpected file inside data dir")
	}
}

// A benign archive restores, and archive-supplied permissions are not trusted.
func TestRestoreBackup_ValidArchiveUsesFixedModes(t *testing.T) {
	svc, dataPath, _ := newTestBackupService(t)

	archive := filepath.Join(svc.backupPath, "good.tar.gz")
	writeArchive(t, archive, map[string]string{
		"notes.txt": "hello",
	})

	if err := svc.RestoreBackup("good.tar.gz"); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	restored := filepath.Join(dataPath, "notes.txt")
	info, err := os.Stat(restored)
	if err != nil {
		t.Fatalf("restored file missing: %v", err)
	}
	if got := info.Mode().Perm(); got != restoreFileMode {
		t.Fatalf("expected fixed mode %o, got %o", restoreFileMode, got)
	}
}

// The backup destination must stay inside the configured roots.
func TestSetBackupPath_RejectsPathOutsideAllowedRoots(t *testing.T) {
	svc, _, _ := newTestBackupService(t)

	if err := svc.SetBackupPath("/etc/nas-backups"); err == nil {
		t.Fatal("expected a path outside the allowed roots to be rejected")
	}
	if _, err := os.Stat("/etc/nas-backups"); err == nil {
		t.Fatal("rejected path must not have been created")
	}
}

func TestSetBackupPath_AcceptsPathInsideAllowedRoots(t *testing.T) {
	svc, _, root := newTestBackupService(t)

	target := filepath.Join(root, "other-backups")
	if err := svc.SetBackupPath(target); err != nil {
		t.Fatalf("expected a path inside the allowed roots to be accepted: %v", err)
	}
}

// A backup id must not be usable to address files outside the backup directory.
func TestDeleteBackup_RejectsTraversalID(t *testing.T) {
	svc, _, root := newTestBackupService(t)

	victim := filepath.Join(root, "victim.txt")
	if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// filepath.Base reduces this to "victim.txt" inside backupPath, so the
	// outside file must survive regardless of the outcome.
	_ = svc.DeleteBackup("../victim.txt")

	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("file outside the backup dir was deleted: %v", err)
	}
}
