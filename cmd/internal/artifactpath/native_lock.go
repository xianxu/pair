package artifactpath

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/nativeid"
)

// NativeLockFilename validates an exact conversation identity and encodes it
// without case folding, Unicode normalization, truncation or path interpretation.
// pair:m5-concept pure
func NativeLockFilename(agent, id string) (string, error) {
	switch agent {
	case "claude", "codex", "agy", "muse", "qoder", "grok":
	default:
		return "", fmt.Errorf("invalid native lock agent %q", agent)
	}
	if err := nativeid.Validate(id); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%x.lock", agent, sha256.Sum256([]byte(id))), nil
}

// NativeLock names a permanent lock inode. Online users must never replace or
// unlink it: doing so lets two cooperating wrappers lock different inodes.
// pair:m5-concept pure
func (p PairCachePaths) NativeLock(agent, id string) (string, error) {
	if !filepath.IsAbs(p.dir) {
		return "", fmt.Errorf("invalid Pair cache directory")
	}
	name, err := NativeLockFilename(agent, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(p.dir, "native-locks", name), nil
}
