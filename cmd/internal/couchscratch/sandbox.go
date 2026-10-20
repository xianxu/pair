package couchscratch

import (
	"path/filepath"
	"strings"
)

// SandboxPolicy projects whether scratch may be selected as the child's TMPDIR.
// It is not general file-access authority. Codex's implicit TMPDIR grant needs
// no directory restriction here, even though its native sandbox remains enabled.
type SandboxPolicy struct {
	Known   bool
	Enabled bool
	Allowed []string
	Denied  []string
}

// ScratchAllowed reports whether policy permits using all of root for scratch.
// Paths are concrete absolute directories, not glob patterns or symlink queries.
// An overlapping deny includes a denied descendant, since scratch may use it.
func ScratchAllowed(root string, p SandboxPolicy) bool {
	if !p.Known || !sandboxLiteralPath(root) {
		return false
	}
	root = filepath.Clean(root)
	allowed := false
	for _, path := range p.Allowed {
		if !sandboxLiteralPath(path) {
			return false
		}
		if sandboxAncestor(filepath.Clean(path), root) {
			allowed = true
		}
	}
	denied := false
	for _, path := range p.Denied {
		if !sandboxLiteralPath(path) {
			return false
		}
		path = filepath.Clean(path)
		if sandboxAncestor(path, root) || sandboxAncestor(root, path) {
			denied = true
		}
	}
	return !p.Enabled || allowed && !denied
}

func sandboxLiteralPath(path string) bool {
	if !filepath.IsAbs(path) || strings.ContainsAny(path, "\x00*?[]\\") {
		return false
	}
	clean := filepath.Clean(path)
	return path == clean || path == clean+string(filepath.Separator)
}

func sandboxAncestor(parent, path string) bool {
	return parent == path || parent == string(filepath.Separator) || strings.HasPrefix(path, parent+string(filepath.Separator))
}
