package content_test

import (
	"testing"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
)

// Reserved infrastructure names must be rejected as the first path segment for
// every client-facing operation of a scoped manager — otherwise delete/move on
// ".trash" recurses the trash into itself, and case variants slip past filters.
func TestScopedManager_RejectsReservedFirstSegment(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.NewLocalStore(tmp)
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	uid := "33333333-3333-3333-3333-333333333333"
	scoped, err := mgr.ForUser(uid)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.EnsureUserHome(uid); err != nil {
		t.Fatal(err)
	}

	blocked := []string{
		".trash",
		".trash/x.txt",
		"vault",
		"Vault/x.txt",
		".system/config",
		"homes",
		"homes/" + uid + "/x.txt",
		"/.trash/x.txt",
	}
	for _, p := range blocked {
		if _, err := scoped.GetFullPath(p); err == nil {
			t.Errorf("GetFullPath(%q) must fail", p)
		}
		if err := scoped.Delete(p); err == nil {
			t.Errorf("Delete(%q) must fail", p)
		}
		if err := scoped.Mkdir(p); err == nil {
			t.Errorf("Mkdir(%q) must fail", p)
		}
	}

	// Normal names keep working.
	if err := scoped.Mkdir("Dokumente"); err != nil {
		t.Fatalf("Mkdir(Dokumente): %v", err)
	}
	if _, err := scoped.GetFullPath("Dokumente/notiz.txt"); err != nil {
		t.Fatalf("GetFullPath(Dokumente/notiz.txt): %v", err)
	}
	// Reserved names deeper in the tree are fine — only the first segment is infra.
	if err := scoped.Mkdir("Dokumente/vault"); err != nil {
		t.Fatalf("Mkdir(Dokumente/vault): %v", err)
	}
}

func TestIsReservedRootName_CaseInsensitive(t *testing.T) {
	for _, name := range []string{"vault", "Vault", "VAULT", " .trash ", ".System", "HOMES"} {
		if !content.IsReservedRootName(name) {
			t.Errorf("IsReservedRootName(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "Dokumente", "vault2", "my.trash"} {
		if content.IsReservedRootName(name) {
			t.Errorf("IsReservedRootName(%q) = true, want false", name)
		}
	}
}
