package pathsafe

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSafeJoin_Allowed(t *testing.T) {
	base := t.TempDir()

	cases := map[string]string{
		"simple relative":   "a/b.txt",
		"leading slash":     "/a/b.txt",
		"current dir marks": "a/./b.txt",
		"dots in filename":  "my..notes.txt",
		"empty":             "",
		"single dot":        ".",
	}

	for name, rel := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := SafeJoin(base, rel)
			if err != nil {
				t.Fatalf("SafeJoin(%q) returned error: %v", rel, err)
			}
			ok, err := IsWithin(base, got)
			if err != nil {
				t.Fatalf("IsWithin failed: %v", err)
			}
			if !ok {
				t.Fatalf("SafeJoin(%q) = %q, which is outside base %q", rel, got, base)
			}
		})
	}
}

func TestSafeJoin_RejectsEscape(t *testing.T) {
	base := filepath.Join(t.TempDir(), "data")

	// Any ".." segment is refused, including ones that would resolve back
	// inside the base — the caller must not receive a different file silently.
	escapes := []string{
		"../escape.txt",
		"../../etc/passwd",
		"a/../../escape.txt",
		"a/b/../../../escape.txt",
		"a/../b.txt",
		"..",
	}

	for _, rel := range escapes {
		t.Run(rel, func(t *testing.T) {
			got, err := SafeJoin(base, rel)
			if err == nil {
				t.Fatalf("SafeJoin(%q) unexpectedly succeeded with %q", rel, got)
			}
			if !errors.Is(err, ErrPathEscape) {
				t.Fatalf("expected ErrPathEscape for %q, got %v", rel, err)
			}
		})
	}
}

// An absolute path is confined to the base rather than honored as-is.
func TestSafeJoin_AbsoluteInputIsConfined(t *testing.T) {
	base := t.TempDir()

	got, err := SafeJoin(base, "/etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, "etc/passwd")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A sibling directory sharing a name prefix must not count as inside.
func TestIsWithin_SiblingPrefixNotInside(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "data")
	sibling := filepath.Join(root, "data-evil")

	inside, err := IsWithin(base, sibling)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inside {
		t.Fatalf("%q must not be considered inside %q", sibling, base)
	}
}

func TestIsWithin_BaseItself(t *testing.T) {
	base := t.TempDir()

	inside, err := IsWithin(base, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inside {
		t.Fatal("base must be considered within itself")
	}
}

func TestWithinAnyRoot(t *testing.T) {
	root := t.TempDir()
	allowed := []string{filepath.Join(root, "mnt"), filepath.Join(root, "media")}

	if err := WithinAnyRoot(allowed, filepath.Join(root, "media", "usb", "backups")); err != nil {
		t.Fatalf("expected path under an allowed root to pass: %v", err)
	}
	if err := WithinAnyRoot(allowed, filepath.Join(root, "mnt")); err != nil {
		t.Fatalf("expected the root itself to pass: %v", err)
	}

	denied := WithinAnyRoot(allowed, filepath.Join(root, "etc", "cron.d"))
	if denied == nil {
		t.Fatal("expected a path outside every root to be rejected")
	}
	if !errors.Is(denied, ErrPathEscape) {
		t.Fatalf("expected ErrPathEscape, got %v", denied)
	}

	// Traversal out of an allowed root must not be accepted.
	if err := WithinAnyRoot(allowed, filepath.Join(root, "mnt", "..", "etc")); err == nil {
		t.Fatal("expected traversal out of an allowed root to be rejected")
	}
}

// No configured roots means nothing is allowed (fail closed).
func TestWithinAnyRoot_EmptyDeniesAll(t *testing.T) {
	if err := WithinAnyRoot(nil, "/mnt/data"); err == nil {
		t.Fatal("expected empty root list to deny")
	}
	if err := WithinAnyRoot([]string{"", "  "}, "/mnt/data"); err == nil {
		t.Fatal("expected blank-only root list to deny")
	}
}
