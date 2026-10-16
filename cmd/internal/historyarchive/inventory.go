package historyarchive

import (
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

//go:embed sources.json
var sourcesJSON []byte

// pair:m5-concept pure
type SourceInventory struct {
	Version int          `json:"version"`
	Sources []SourceRule `json:"sources"`
	Sinks   []SinkRule   `json:"sinks"`
}

// SourceRule describes selection policy; it does not implement a scanner.
// Patterns are slash-separated path.Match selectors relative to RootToken.
// A match selects that file, or the regular-file descendants of that directory.
// Collectors must still enforce their own safe traversal and quiescence rules.
// Every supported producer namespace/backend needs a declaration: scoped and
// legacy Pair paths, configured global Couch stores, and explicit slot stores.
// pair:m5-concept pure
type SourceRule struct {
	Kind             string   `json:"kind"`
	Owner            string   `json:"owner"`
	RootToken        string   `json:"root_token"`
	Patterns         []string `json:"patterns"`
	Mode             string   `json:"mode"`
	Retention        string   `json:"retention"`
	SnapshotRequired bool     `json:"snapshot_required"`
	ArtifactFamilies []string `json:"artifact_families,omitempty"`
}

// SinkRule is a half-open age interval in seconds. Zero MaxAgeSeconds means no upper bound.
// pair:m5-concept pure
type SinkRule struct {
	Kind          string `json:"kind"`
	MinAgeSeconds int64  `json:"min_age_seconds"`
	MaxAgeSeconds int64  `json:"max_age_seconds"`
	Sink          string `json:"sink"`
}

func Defaults() (SourceInventory, error) { return DecodeInventory(sourcesJSON) }
func DecodeInventory(b []byte) (SourceInventory, error) {
	var i SourceInventory
	if err := decodeStrict(b, &i); err != nil {
		return i, err
	}
	if i.Version != Version || len(i.Sources) == 0 || len(i.Sinks) == 0 {
		return i, errors.New("invalid inventory version or empty rules")
	}
	seen := map[string]bool{}
	for _, s := range i.Sources {
		if !validKindName(s.Kind) || seen[s.Kind] || s.Owner == "" || !validRootToken(s.RootToken) || len(s.Patterns) == 0 || s.Retention == "" || (s.Mode != "continuous" && s.Mode != "on-expiry") {
			return i, fmt.Errorf("invalid or duplicate source %q", s.Kind)
		}
		for _, p := range s.Patterns {
			if !validPath(p) {
				return i, errors.New("invalid source pattern")
			}
		}
		seen[s.Kind] = true
	}
	for n, r := range i.Sinks {
		if (r.Kind != "*" && !seen[r.Kind]) || r.Sink == "" || r.MinAgeSeconds < 0 || (r.MaxAgeSeconds != 0 && r.MaxAgeSeconds <= r.MinAgeSeconds) {
			return i, errors.New("invalid sink rule")
		}
		for _, prev := range i.Sinks[:n] {
			sameKind := r.Kind == prev.Kind || r.Kind == "*" || prev.Kind == "*"
			overlap := (r.MaxAgeSeconds == 0 || prev.MinAgeSeconds < r.MaxAgeSeconds) && (prev.MaxAgeSeconds == 0 || r.MinAgeSeconds < prev.MaxAgeSeconds)
			if sameKind && overlap {
				return i, errors.New("ambiguous sink rules")
			}
		}
	}
	for k := range seen {
		covered := false
		for _, r := range i.Sinks {
			covered = covered || r.Kind == "*" || r.Kind == k
		}
		if !covered {
			return i, fmt.Errorf("missing sink rule for %s", k)
		}
	}
	return i, nil
}
func validKindName(k string) bool {
	if k == "" {
		return false
	}
	for _, c := range k {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return k[0] != '-'
}
func validRootToken(k string) bool {
	switch k {
	case "pair-data", "claude-home", "codex-home", "qoder-home", "ops-worktree", "git-common", "couch-store", "slot-store":
		return true
	}
	return false
}
func (i SourceInventory) Resolve(kind string) (SourceRule, error) {
	for _, r := range i.Sources {
		if r.Kind == kind {
			return r, nil
		}
	}
	return SourceRule{}, fmt.Errorf("unknown archive kind %q", kind)
}
func (i SourceInventory) Route(kind string, age time.Duration) (string, error) {
	if _, err := i.Resolve(kind); err != nil {
		return "", err
	}
	if age < 0 {
		return "", errors.New("negative source age")
	}
	seconds := int64(age / time.Second)
	for _, r := range i.Sinks {
		if (r.Kind == kind || r.Kind == "*") && seconds >= r.MinAgeSeconds && (r.MaxAgeSeconds == 0 || seconds < r.MaxAgeSeconds) {
			return r.Sink, nil
		}
	}
	return "", errors.New("no sink rule for source age")
}

// ResolveRoot takes environment values explicitly so inventory policy remains pure.
func (i SourceInventory) ResolveRoot(kind, override, home string, env map[string]string) (string, error) {
	r, err := i.Resolve(kind)
	if err != nil {
		return "", err
	}
	root := override
	if root == "" {
		switch r.RootToken {
		case "pair-data":
			root = env["PAIR_DATA_DIR"]
			if root == "" {
				base := env["XDG_DATA_HOME"]
				if base == "" && home != "" {
					base = filepath.Join(home, ".local", "share")
				}
				if base != "" {
					root = filepath.Join(base, "pair")
				}
			}
		case "couch-store":
			root = env["COUCH_STORE_DIR"]
			if root == "" {
				base := env["XDG_DATA_HOME"]
				if base == "" && home != "" {
					base = filepath.Join(home, ".local", "share")
				}
				if base != "" {
					root = filepath.Join(base, "pair", "couch")
				}
			}
		// slot-store intentionally has no environment fallback: COUCH_STORE_DIR
		// identifies the global namespace, not the numbered slot backend.
		case "claude-home":
			root = env["CLAUDE_CONFIG_DIR"]
			if root == "" && home != "" {
				root = filepath.Join(home, ".claude")
			}
		case "codex-home":
			root = env["CODEX_HOME"]
			if root == "" && home != "" {
				root = filepath.Join(home, ".codex")
			}
		case "qoder-home":
			root = env["QODER_HOME"]
			if root == "" && home != "" {
				root = filepath.Join(home, ".qoder")
			}
		}
	}
	if !filepath.IsAbs(root) || strings.ContainsRune(root, 0) {
		return "", fmt.Errorf("absolute root override required for %s", r.RootToken)
	}
	return filepath.Clean(root), nil
}
