package storagegc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// SelectedOwner resolves the already-selected PAIR_DATA_DIR without consulting
// HOME or guessing a repository. Scoped directories encode their root and key;
// a flat directory explicitly selects the legacy namespace, even when the
// launcher also exports a logical repository scope.
func SelectedOwner(dataDir, scopeKey, tag string) (artifactpath.StorageOwner, error) {
	if !filepath.IsAbs(dataDir) {
		return artifactpath.StorageOwner{}, errors.New("PAIR_DATA_DIR must be an explicit absolute directory")
	}
	physical, err := filepath.EvalSymlinks(dataDir)
	if err != nil {
		return artifactpath.StorageOwner{}, err
	}
	if err := checkDirectory(physical, false); err != nil {
		return artifactpath.StorageOwner{}, err
	}
	root, scope := physical, ""
	if filepath.Base(filepath.Dir(physical)) == "repos" {
		scope = filepath.Base(physical)
		root = filepath.Dir(filepath.Dir(physical))
		if scopeKey != "" && scopeKey != scope {
			return artifactpath.StorageOwner{}, errors.New("PAIR_SCOPE_KEY differs from selected directory")
		}
	}
	return artifactpath.NewStorageOwner(root, scope, tag)
}

// OwnerFramePath names an absolute path in the frame SelectedOwner resolves the
// owner in: its deepest existing ancestor with symlinks evaluated, the missing
// rest and the base kept. A dangling symlink on the way is refused, so the kept
// rest is genuinely absent and cannot redirect. A caller's spelling of
// PAIR_DATA_DIR may be a symlink (macOS TMPDIR=/var/... is /private/var/...),
// so every path compared against a selected owner, or recorded for the
// collector to match capture paths against, goes through this.
func OwnerFramePath(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", errors.New("owner path must be an exact absolute path")
	}
	dir, rest := filepath.Dir(path), filepath.Base(path)
	for {
		physical, err := filepath.EvalSymlinks(dir)
		if err == nil {
			return filepath.Join(physical, rest), nil
		}
		parent := filepath.Dir(dir)
		if !errors.Is(err, fs.ErrNotExist) || parent == dir {
			return "", err
		}
		// dir exists yet does not resolve: a dangling symlink, whose target the
		// spelling cannot name. Only a truly missing component is kept as written.
		if _, lerr := os.Lstat(dir); lerr == nil {
			return "", fmt.Errorf("owner path crosses a dangling symlink: %s", dir)
		}
		dir, rest = parent, filepath.Join(filepath.Base(dir), rest)
	}
}
