// pair-context — print one agent pane's current context size (humanized
// token count), or nothing. Invoked as `pair-context <tag> <agent>` by the
// pair-title poller. Tolerant: any failure prints nothing and exits 0, so a
// hiccup never garbles the pane title.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/ctxmeter"
	"github.com/xianxu/pair/cmd/internal/transcript"
)

func main() {
	if len(os.Args) < 3 {
		return
	}
	tag, agent := os.Args[1], os.Args[2]
	dataDir := os.Getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(os.Getenv("HOME"), ".local", "share")
		}
		dataDir = filepath.Join(base, "pair")
	}
	sid := transcript.SessionID(dataDir, tag, agent)
	if sid == "" {
		return
	}
	cwd := paneCwd(dataDir, tag, agent) // "" for codex/agy is fine (Resolve ignores it)
	path := transcript.Resolve(agent, sid, cwd, os.Getenv("HOME"))
	if path == "" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
		fmt.Println(ctxmeter.Humanize(n))
	}
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
