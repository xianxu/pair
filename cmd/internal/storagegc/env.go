package storagegc

import (
	"errors"
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
