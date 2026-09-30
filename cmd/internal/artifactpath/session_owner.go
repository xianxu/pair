package artifactpath

import (
	"path/filepath"
	"strings"
)

// SessionArtifactOwner decodes only exact draft and raw scrollback paths used
// by live Pair panes. Reconstructing the path prevents prefix/tag ambiguity.
func SessionArtifactOwner(root, path string, agents []string) (StorageOwner, bool) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return StorageOwner{}, false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return StorageOwner{}, false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	scope := ""
	switch {
	case len(parts) == 1:
	case len(parts) == 3 && parts[0] == "repos":
		scope = parts[1]
	default:
		return StorageOwner{}, false
	}
	name := filepath.Base(path)
	if tag, ok := TagFromHistorySidecar(name); ok {
		owner, err := NewStorageOwner(root, scope, tag)
		if err == nil {
			p, err := ResolveScoped(owner.Directory(), tag)
			if err == nil && p.Draft() == path {
				return owner, true
			}
		}
	}
	for _, agent := range agents {
		prefix, suffix := "scrollback-", "-"+agent+".raw"
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		tag := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
		owner, err := NewStorageOwner(root, scope, tag)
		if err != nil {
			continue
		}
		p, err := ResolveScoped(owner.Directory(), tag)
		if err == nil && p.ScrollbackRaw(agent) == path {
			return owner, true
		}
	}
	return StorageOwner{}, false
}
