// Package pathscope resolves physical filesystem paths without creating them.
package pathscope

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalFuture resolves existing ancestors without creating missing roots.
func CanonicalFuture(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("Couch root must be absolute: %q", path)
	}
	path = filepath.Clean(path)
	physical, err := filepath.EvalSymlinks(path)
	if err == nil {
		return physical, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("unresolved Couch root symlink %q: %w", path, err)
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", err
	}
	physical, err = CanonicalFuture(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(physical, filepath.Base(path)), nil
}

// Within reports lexical containment of already canonical physical paths.
func Within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// Directory resolves a present or future directory and checks physical confinement.
// It does not create directories or change their permissions.
func Directory(root, path, name string) (string, error) {
	physical, err := CanonicalFuture(path)
	if err != nil {
		return "", fmt.Errorf("%s isolation root: %w", name, err)
	}
	if !Within(root, physical) {
		return "", fmt.Errorf("%s must remain inside COUCH_ISOLATED_ROOT: %q", name, physical)
	}
	if info, err := os.Stat(physical); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("%s isolation root is not a directory: %q", name, physical)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return physical, nil
}
