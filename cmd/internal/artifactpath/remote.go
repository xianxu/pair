package artifactpath

import (
	"errors"
	"path/filepath"
)

// RemoteStatePaths binds the security records to the already leased Couch
// namespace. Kill authority is never scoped to an individual agent worktree.
// pair:m5-concept pure
type RemoteStatePaths struct{ dir string }

func ResolveCouchRemote(couchNamespace string) (RemoteStatePaths, error) {
	if !filepath.IsAbs(couchNamespace) || filepath.Clean(couchNamespace) != couchNamespace {
		return RemoteStatePaths{}, errors.New("remote state requires an absolute clean Couch namespace")
	}
	return RemoteStatePaths{dir: filepath.Join(couchNamespace, "remote-security")}, nil
}
func (p RemoteStatePaths) Latch() string         { return filepath.Join(p.dir, "latch.json") }
func (p RemoteStatePaths) Origins() string       { return filepath.Join(p.dir, "origins.json") }
func (p RemoteStatePaths) Audit() string         { return filepath.Join(p.dir, "audit.jsonl") }
func (p RemoteStatePaths) PreviousAudit() string { return filepath.Join(p.dir, "audit.previous.jsonl") }
