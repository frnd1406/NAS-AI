package content_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
)

const (
	pathsUserA = "11111111-1111-1111-1111-111111111111"
	pathsUserB = "22222222-2222-2222-2222-222222222222"
)

func newScopedPair(t *testing.T) (base string, a, b content.StorageService) {
	t.Helper()
	base = t.TempDir()
	store, err := storage.NewLocalStore(base)
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	if a, err = mgr.ScopeToUser(pathsUserA); err != nil {
		t.Fatal(err)
	}
	if b, err = mgr.ScopeToUser(pathsUserB); err != nil {
		t.Fatal(err)
	}
	return base, a, b
}

func TestScopeToUser_InvalidIDs(t *testing.T) {
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	for _, id := range []string{"", "  ", "user-1", "../" + pathsUserA} {
		if _, err := mgr.ScopeToUser(id); !errors.Is(err, content.ErrInvalidUserID) {
			t.Errorf("ScopeToUser(%q) = %v, want ErrInvalidUserID", id, err)
		}
	}
}

func TestTrash_RejectsIDsOutsideTrash(t *testing.T) {
	base, a, _ := newScopedPair(t)
	keep := filepath.Join(base, content.UserHomeRel(pathsUserA), "keep.txt")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"", ".", "/", "..", "../keep.txt", "../../" + pathsUserB, "a/../../keep.txt", "a\x00b"} {
		if err := a.DeleteFromTrash(id); !errors.Is(err, content.ErrPathTraversal) {
			t.Errorf("DeleteFromTrash(%q) = %v, want ErrPathTraversal", id, err)
		}
		if err := a.RestoreFromTrash(id); err == nil {
			t.Errorf("RestoreFromTrash(%q) succeeded", id)
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("home content was removed: %v", err)
	}
}

func TestTrash_DeleteAndRestoreRoundTrip(t *testing.T) {
	base, a, _ := newScopedPair(t)
	home := filepath.Join(base, content.UserHomeRel(pathsUserA))
	if err := os.MkdirAll(filepath.Join(home, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.txt", "two.txt"} {
		if err := os.WriteFile(filepath.Join(home, "docs", name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := a.Delete("docs/" + name); err != nil {
			t.Fatal(err)
		}
	}

	items, err := a.ListTrash()
	if err != nil || len(items) != 2 {
		t.Fatalf("ListTrash = %v, %v; want 2 items", items, err)
	}
	if err := a.RestoreFromTrash(items[0].ID); err != nil {
		t.Fatalf("restore %q: %v", items[0].ID, err)
	}
	if err := a.DeleteFromTrash(items[1].ID); err != nil {
		t.Fatalf("delete %q: %v", items[1].ID, err)
	}
	if _, err := os.Stat(filepath.Join(home, items[0].OriginalPath)); err != nil {
		t.Errorf("restored file missing: %v", err)
	}
	if left, _ := a.ListTrash(); len(left) != 0 {
		t.Errorf("trash not empty: %v", left)
	}
}

func TestRename_RejectsMultiSegmentNames(t *testing.T) {
	base, a, _ := newScopedPair(t)
	src := filepath.Join(base, content.UserHomeRel(pathsUserA), "f.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", ".", "..", "../x", "../../" + pathsUserB + "/x", "a/b", `a\b`, "a\x00b"} {
		if err := a.Rename("f.txt", name); !errors.Is(err, content.ErrPathTraversal) {
			t.Errorf("Rename to %q = %v, want ErrPathTraversal", name, err)
		}
	}
	if err := a.Rename("f.txt", "g..h.txt"); err != nil {
		t.Errorf("legitimate rename failed: %v", err)
	}
}

func TestSave_RejectsUnsafeFilename(t *testing.T) {
	_, a, _ := newScopedPair(t)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "ok.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r := multipart.NewReader(&body, w.Boundary())
	form, err := r.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	fh := form.File["file"][0]

	for _, name := range []string{"..", "."} {
		fh.Filename = name
		f, err := fh.Open()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.Save("", f, fh); !errors.Is(err, content.ErrPathTraversal) {
			t.Errorf("Save(%q) = %v, want ErrPathTraversal", name, err)
		}
		_ = f.Close()
	}
}
