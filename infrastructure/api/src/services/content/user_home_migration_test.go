package content_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
)

func writeLegacyPhoto(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(content.MediaLibraryRel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The legacy Fotos/iPhone folder is shared, so a single user may adopt it —
// that is the single-user NAS upgrade path.
func TestLegacyMigration_SingleUserAdoptsPhotos(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.NewLocalStore(tmp)
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	uid := "55555555-5555-5555-5555-555555555555"

	writeLegacyPhoto(t, tmp, "urlaub.jpg")
	if err := mgr.EnsureUserHome(uid); err != nil {
		t.Fatal(err)
	}

	moved := filepath.Join(tmp, "homes", uid, filepath.FromSlash(content.MediaLibraryRel), "urlaub.jpg")
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("photo was not migrated into the user home: %v", err)
	}
}

// With a second home present, ownership of the shared folder is ambiguous —
// migrating would silently take the photos away from the other account.
func TestLegacyMigration_SkippedWithMultipleHomes(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.NewLocalStore(tmp)
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	uidA := "66666666-6666-6666-6666-666666666666"
	uidB := "77777777-7777-7777-7777-777777777777"

	if err := mgr.EnsureUserHome(uidA); err != nil {
		t.Fatal(err)
	}
	if err := mgr.EnsureUserHome(uidB); err != nil {
		t.Fatal(err)
	}

	// Legacy photos appear only now, with two homes already present.
	writeLegacyPhoto(t, tmp, "geteilt.jpg")
	if err := mgr.EnsureUserHome(uidB); err != nil {
		t.Fatal(err)
	}

	legacy := filepath.Join(tmp, filepath.FromSlash(content.MediaLibraryRel), "geteilt.jpg")
	if _, err := os.Stat(legacy); err != nil {
		t.Errorf("shared legacy photo was moved despite multiple homes: %v", err)
	}
	claimed := filepath.Join(tmp, "homes", uidB, filepath.FromSlash(content.MediaLibraryRel), "geteilt.jpg")
	if _, err := os.Stat(claimed); err == nil {
		t.Error("user B claimed the shared legacy photo")
	}
}

// EnsureUserHome runs on every authenticated request; repeated calls must be
// a no-op rather than an "already exists" error.
func TestEnsureUserHome_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.NewLocalStore(tmp)
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	uid := "88888888-8888-8888-8888-888888888888"

	for i := 0; i < 3; i++ {
		if err := mgr.EnsureUserHome(uid); err != nil {
			t.Fatalf("EnsureUserHome call %d: %v", i+1, err)
		}
	}
}
