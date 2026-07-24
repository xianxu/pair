package entrypoint

import (
	"fmt"
	"path/filepath"
	"strings"
)

type AssetRootInput struct {
	PairHome        string
	Executable      string
	DefaultPairHome string
	EmbeddedRoot    string
	// ValidRoot reports whether a candidate directory is a Pair asset root. The
	// caller checks every marker (ValidRootMarkers) exists there (#99 M5c — was
	// bin/pair-shell, now the always-present zellij layout, since the shell
	// launcher is retired).
	ValidRoot func(root string) bool
}

type AssetRoot struct {
	Root   string
	Source string
}

func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
	valid := input.ValidRoot
	if valid == nil {
		valid = func(string) bool { return false }
	}

	candidates := make([]assetRootCandidate, 0, 3)
	if input.PairHome != "" {
		candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
	}
	if input.Executable != "" {
		candidates = append(candidates, assetRootCandidate{
			root:   filepath.Dir(filepath.Dir(input.Executable)),
			source: "executable sibling",
		})
	}
	if input.DefaultPairHome != "" {
		candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
	}
	if input.EmbeddedRoot != "" {
		candidates = append(candidates, assetRootCandidate{root: input.EmbeddedRoot, source: "embedded runtime"})
	}

	seen := map[string]bool{}
	checked := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		root := filepath.Clean(candidate.root)
		if root == "." || seen[root] {
			continue
		}
		seen[root] = true
		checked = append(checked, root)
		if valid(root) {
			return AssetRoot{Root: root, Source: candidate.source}, nil
		}
	}

	if len(checked) == 0 {
		checked = append(checked, "<none>")
	}
	return AssetRoot{}, fmt.Errorf("pair assets not found; set PAIR_HOME to a Pair checkout/install root containing %s and %s (checked: %s)",
		filepath.Join("zellij", "layouts", "main-2.kdl"),
		filepath.Join("zellij", "layouts", "main-3.kdl"),
		strings.Join(checked, ", "))
}

// ValidRootMarkers are the tracked layouts every Pair asset root must carry.
func ValidRootMarkers(root string) []string {
	return []string{
		filepath.Join(root, "zellij", "layouts", "main-2.kdl"),
		filepath.Join(root, "zellij", "layouts", "main-3.kdl"),
	}
}

type assetRootCandidate struct {
	root   string
	source string
}
