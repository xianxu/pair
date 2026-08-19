// Package codexsid resolves a live codex session id by walking the agent's
// process tree for its open rollout file. It's the canonical home for the
// ps-descendants + lsof + rollout-regex walk that slug and sessionwatch each
// grew their own copy of (#93 M3 extracts it for review-target; those two hot
// -path packages can adopt it later).
package codexsid

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/transcript"
)

// ResolveSessionID reads the codex agent's root pid from
// $dataDir/agent-pid-<tag>, BFS-walks its process descendants, and greps each
// process's open files for the live rollout jsonl — returning the session UUID,
// or "" when the pidfile is absent/empty or no rollout is open.
func ResolveSessionID(dataDir, tag string) string {
	raw, err := os.ReadFile(filepath.Join(dataDir, "agent-pid-"+tag))
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(raw))
	if root == "" {
		return ""
	}
	for _, pid := range procutil.DescendantPIDs(root, procutil.ProcessChildren()) {
		for _, name := range procutil.LsofNames(pid) {
			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
				return sid
			}
		}
	}
	return ""
}
