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
	// caller checks the marker (ValidRootMarker) exists there (#99 M5c — was
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
	return AssetRoot{}, fmt.Errorf("pair assets not found; set PAIR_HOME to a Pair checkout/install root containing %s (checked: %s)",
		filepath.Join("zellij", "layouts", "main.kdl"), strings.Join(checked, ", "))
}

// ValidRootMarker is the file whose presence marks a directory as a Pair asset
// root: the zellij layout the launch reads (createflow.go). It is a tracked source
// file AND bundled into the embedded runtime, so it exists in both a checkout and
// an extracted pair-home — unlike bin/pair-wrap (a built, gitignored binary).
func ValidRootMarker(root string) string {
	return filepath.Join(root, "zellij", "layouts", "main.kdl")
}

type assetRootCandidate struct {
	root   string
	source string
}
