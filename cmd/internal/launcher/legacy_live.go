package launcher

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

type legacyPaneMeta struct {
	CWD string `json:"cwd"`
}

func legacyLiveSessionsForScope(rt Runtime, live []Session, index SessionNameIndex, scope RepoScope, globalDataDir string) []Session {
	if globalDataDir == "" {
		return nil
	}
	var out []Session
	for _, session := range live {
		if session.State == SessionExited || !strings.HasPrefix(session.Name, "pair-") {
			continue
		}
		if _, ok := index.ownerOf(session.Name); ok {
			continue
		}
		tag := strings.TrimPrefix(session.Name, "pair-")
		agent, ok := legacyPaneAgentForScope(rt, globalDataDir, scope, tag)
		if !ok {
			continue
		}
		session.Tag = tag
		session.RepoName = scope.DisplayName
		session.Agent = agent
		out = append(out, session)
	}
	return out
}

func legacyPaneAgentForScope(rt Runtime, globalDataDir string, scope RepoScope, tag string) (string, bool) {
	names, err := rt.ReadDir(globalDataDir)
	if err != nil {
		return "", false
	}
	prefix := "pane-" + tag + "-"
	for _, name := range names {
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		raw, err := rt.ReadFile(filepath.Join(globalDataDir, name))
		if err != nil {
			continue
		}
		var meta legacyPaneMeta
		if err := json.Unmarshal([]byte(raw), &meta); err != nil || meta.CWD == "" {
			continue
		}
		if pathWithinRoot(meta.CWD, scope.Root) {
			agent := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".json")
			if agent == "" {
				agent = rt.InferAgent(tag)
			}
			return agent, true
		}
	}
	return "", false
}

func pathWithinRoot(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}
