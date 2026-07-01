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
	PairShellExists func(root string) bool
}

type AssetRoot struct {
	Root      string
	ShellPath string
	Source    string
}

func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
	exists := input.PairShellExists
	if exists == nil {
		exists = func(string) bool { return false }
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
		if exists(root) {
			return AssetRoot{
				Root:      root,
				ShellPath: PairShellPath(root),
				Source:    candidate.source,
			}, nil
		}
	}

	if len(checked) == 0 {
		checked = append(checked, "<none>")
	}
	return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
}

func PairShellPath(root string) string {
	return filepath.Join(root, "bin", "pair-shell")
}

type assetRootCandidate struct {
	root   string
	source string
}
