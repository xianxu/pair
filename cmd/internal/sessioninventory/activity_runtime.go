package sessioninventory

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
)

// ActivityCatalogFingerprint uses the same platform metadata as native files.
// A missing catalog is an explicit dependency state, never an empty catalog.
func (r OSRuntime) ActivityCatalogFingerprint() (ArtifactFingerprint, error) {
	paths, err := artifactpath.ResolveSelectedScope(r.pairRoot.Path)
	if err != nil {
		return ArtifactFingerprint{}, err
	}
	path := paths.SessionInventoryCatalog()
	info, err := os.Lstat(path)
	if err != nil {
		return ArtifactFingerprint{}, err
	}
	return FingerprintFileInfo(path, info)
}
