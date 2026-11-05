// Package pathsafe provides the single place where untrusted path input is
// confined to a base directory. It depends only on the standard library so it
// can be imported from drivers, services and handlers alike.
//
// Note: paths are resolved lexically, not through the filesystem. Symlinks
// inside the base directory are therefore not followed — callers that write to
// attacker-influenced directories should additionally use O_EXCL/O_NOFOLLOW.
package pathsafe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathEscape is returned when a resolved path would leave its base directory.
var ErrPathEscape = errors.New("path escapes base directory")

// SafeJoin joins rel onto base and guarantees the result stays inside base.
//
// rel is treated as rooted at base: both "a/b" and "/a/b" resolve to base/a/b.
// A ".." segment is rejected outright rather than resolved, so a caller never
// silently receives a different file than it asked for.
func SafeJoin(base, rel string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base path: %w", err)
	}

	if hasTraversalSegment(rel) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}

	// Cleaning against a leading separator drops any rooting; the separator is
	// then removed so Join cannot be handed an absolute path.
	cleaned := strings.TrimPrefix(filepath.Clean("/"+rel), "/")
	joined := filepath.Join(absBase, cleaned)

	// Defence in depth: the lexical result must still resolve inside the base.
	if !within(absBase, joined) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}
	return joined, nil
}

// hasTraversalSegment reports whether any path segment is exactly "..".
// Checking segments rather than substrings keeps legitimate names such as
// "my..notes.txt" usable.
func hasTraversalSegment(p string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(p), "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// IsWithin reports whether path lies inside base (or is base itself).
func IsWithin(base, path string) (bool, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false, fmt.Errorf("resolve base path: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("resolve path: %w", err)
	}
	return within(absBase, absPath), nil
}

// WithinAnyRoot verifies that an operator-supplied absolute path lies under one
// of the allowed roots. An empty roots list denies everything (fail closed).
func WithinAnyRoot(roots []string, path string) error {
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if within(absRoot, absPath) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s is not under an allowed root", ErrPathEscape, absPath)
}

// within reports whether target is absBase or sits below it. It uses
// filepath.Rel so that a climbing result is detected explicitly rather than
// relying on string prefixes, which mismatch on sibling names like
// "/data-evil" against base "/data".
func within(absBase, target string) bool {
	rel, err := filepath.Rel(absBase, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}
