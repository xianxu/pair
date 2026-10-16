package artifactpath

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
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
	if len(id) == 0 || len(id) > 512 || !utf8.ValidString(id) || id == "." || id == ".." || strings.ContainsAny(id, "/\\") {
		return "", fmt.Errorf("invalid native conversation ID")
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			return "", fmt.Errorf("invalid native conversation ID")
		}
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
