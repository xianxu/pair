// Package contextcmd implements the pair-context command body.
package contextcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/ctxmeter"
	"github.com/xianxu/pair/cmd/internal/transcript"
)

type Env struct {
	Home        string
	XDGDataHome string
	PairDataDir string
}

func EnvFromOS() Env {
	return Env{
		Home:        os.Getenv("HOME"),
		XDGDataHome: os.Getenv("XDG_DATA_HOME"),
		PairDataDir: os.Getenv("PAIR_DATA_DIR"),
	}
}

func Run(args []string, env Env, stdout io.Writer) int {
	if len(args) < 2 {
		return 0
	}
	tag, agent := args[0], args[1]
	path := TranscriptPath(env, tag, agent)
	if path == "" {
		return 0
	}
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
		fmt.Fprintln(stdout, ctxmeter.Humanize(n))
	}
	return 0
}

// TranscriptPath resolves the native transcript file for (tag, agent) — the
// session-id + pane-cwd + agent-specific resolution shared by `pair context`
// and the #93 title poller's activity-mtime check. Returns "" when the session
// isn't resolvable yet.
func TranscriptPath(env Env, tag, agent string) string {
	dataDir := resolveDataDir(env)
	sid := transcript.SessionID(dataDir, tag, agent)
	if sid == "" {
		return ""
	}
	cwd := paneCwd(dataDir, tag, agent)
	return transcript.Resolve(agent, sid, cwd, env.Home)
}

func resolveDataDir(env Env) string {
	if env.PairDataDir != "" {
		return env.PairDataDir
	}
	base := env.XDGDataHome
	if base == "" {
		base = filepath.Join(env.Home, ".local", "share")
	}
	return filepath.Join(base, "pair")
}

func paneCwd(dataDir, tag, agent string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
	if err != nil {
		return ""
	}
	var p struct {
		Cwd string `json:"cwd"`
	}
	if json.Unmarshal(b, &p) != nil {
		return ""
	}
	return p.Cwd
}
