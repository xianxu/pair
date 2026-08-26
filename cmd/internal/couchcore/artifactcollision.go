package couchcore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// ThreadArtifactCollisionChecker reports whether Pair already owns any durable
// tag-scoped state outside ThreadStore for a prospective composite address.
type ThreadArtifactCollisionChecker interface {
	Collides(ThreadAddress) (bool, error)
}

type NoThreadArtifactCollisions struct{}

func (NoThreadArtifactCollisions) Collides(ThreadAddress) (bool, error) { return false, nil }

type ScopedThreadArtifactCollisionChecker struct{ GlobalDataDir string }

func NewScopedThreadArtifactCollisionChecker(globalDataDir string) ScopedThreadArtifactCollisionChecker {
	return ScopedThreadArtifactCollisionChecker{GlobalDataDir: globalDataDir}
}

func (c ScopedThreadArtifactCollisionChecker) Collides(address ThreadAddress) (bool, error) {
	if err := validateThreadAddress(address); err != nil {
		return false, err
	}
	if c.GlobalDataDir == "" {
		return false, errors.New("artifact collision checker has no Pair data directory")
	}
	scope := launcher.RepoScope{Key: address.RepoScope}
	paths := launcher.NewScopedPaths(c.GlobalDataDir, scope, string(address.Tag))
	entries, err := os.ReadDir(paths.ScopeDir())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("scan scoped Pair artifacts: %w", err)
	}
	for _, entry := range entries {
		if isTagScopedArtifactName(entry.Name(), string(address.Tag)) {
			return true, nil
		}
	}

	raw, err := os.ReadFile(filepath.Join(c.GlobalDataDir, "session-names.jsonl"))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Pair session-name index: %w", err)
	}
	for _, entry := range launcher.ParseSessionNameIndex(string(raw)).Entries {
		if entry.ScopeKey == address.RepoScope && entry.Tag == string(address.Tag) {
			return true, nil
		}
	}
	return false, nil
}

func isTagScopedArtifactName(name, tag string) bool {
	exact := map[string]bool{
		"ledger-" + tag + ".jsonl":        true,
		"draft-" + tag + ".md":            true,
		"log-" + tag + ".md":              true,
		"queue-" + tag:                    true,
		"agent-" + tag:                    true,
		"agent-pid-" + tag:                true,
		"agent-output-" + tag:             true,
		"agent-picks-" + tag:              true,
		"adapt-" + tag + ".jsonl":         true,
		"outer-tty-" + tag:                true,
		"nvim-pid-" + tag + "-draft":      true,
		"nvim-pid-" + tag + "-scrollback": true,
	}
	if exact[name] {
		return true
	}
	for _, prefix := range []string{
		"config-" + tag + "-",
		"agent-ready-" + tag + "-",
		"pane-" + tag + "-",
		"scrollback-" + tag + "-",
		"changelog-" + tag + "-",
		"draft-" + tag + "-",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
