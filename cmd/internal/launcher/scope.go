package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
)

var scopedComponentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// RepoScope is the hidden repo identity Pair uses to keep display tags local to
// a repo while avoiding sidecar/session collisions between same-name repos.
type RepoScope struct {
	Root        string
	DisplayName string
	Key         string
}

// ResolveRepoScope derives a stable hidden scope key from the cleaned repo root.
// The display name intentionally remains human-readable and never includes Key.
func ResolveRepoScope(root string) (RepoScope, error) {
	clean := filepath.Clean(root)
	if clean == "." || clean == string(filepath.Separator) {
		return RepoScope{}, fmt.Errorf("empty repo root")
	}
	sum := sha256.Sum256([]byte(clean))
	return RepoScope{
		Root:        clean,
		DisplayName: repoDisplayName(clean),
		Key:         hex.EncodeToString(sum[:])[:16],
	}, nil
}

// ValidateRepoScopeKey validates an already-resolved hidden scope before it is
// used as a path component. Resolution and validation are separate because
// persisted records and child environments carry only the key.
func ValidateRepoScopeKey(key string) error {
	if !scopedComponentPattern.MatchString(key) {
		return fmt.Errorf("invalid repo scope key %q", key)
	}
	return nil
}

// ValidatePairTag validates the durable tag component shared by Pair sidecars,
// native sessions, and Couch thread records.
func ValidatePairTag(tag string) error {
	if !scopedComponentPattern.MatchString(tag) {
		return fmt.Errorf("invalid pair tag %q", tag)
	}
	return nil
}

func repoDisplayName(root string) string {
	base := filepath.Base(root)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "pair"
	}
	return base
}

// NormalizeDisplayComponent converts a user-facing repo/tag component into the
// filesystem/session-safe spelling used for public names. It does not include
// hidden scope identity; callers store that separately.
func NormalizeDisplayComponent(raw string) string {
	if raw == "" {
		return "pair"
	}
	out := make([]rune, 0, len(raw))
	for _, r := range raw {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "pair"
	}
	return string(out)
}
