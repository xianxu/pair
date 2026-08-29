package titlepoller

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// OSRuntime implements Runtime with real zellij/cmux/fs/process calls. The fs
// primitives (ReadFile/WriteFile/Remove/ModTime) come from the embedded
// osfs.FS (#93 M3).
type OSRuntime struct{ osfs.FS }

func NewOSRuntime() OSRuntime { return OSRuntime{} }

func (OSRuntime) Now() time.Time                 { return time.Now() }
func (OSRuntime) Sleep(d time.Duration)          { time.Sleep(d) }
func (OSRuntime) Getpid() string                 { return strconv.Itoa(os.Getpid()) }
func (OSRuntime) ProcessAlive(p string) bool     { return procutil.Alive(p) }
func (OSRuntime) ProcessCommand(p string) string { return procutil.Command(p) }

// SessionAlive reports whether `zellij list-sessions --short` lists an exact
// match for session (the shell's `grep -qx "$SESSION"`).
func (OSRuntime) SessionAlive(session string) bool {
	out, err := exec.Command("zellij", "list-sessions", "--short").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == session {
			return true
		}
	}
	return false
}

func (OSRuntime) RenamePane(session, paneID, title string) error {
	return exec.Command("zellij", "--session", session, "action",
		"rename-pane", "--pane-id", paneID, title).Run()
}

func (OSRuntime) CmuxAvailable() bool {
	_, err := exec.LookPath("cmux")
	return err == nil
}

func (OSRuntime) CmuxRenameWorkspace(title string) error {
	return exec.Command("cmux", "rename-workspace", title).Run()
}

// PaneFiles globs pane-<tag>-*.json and decodes each into a PaneInfo (agent from
// the filename; pane_id from the JSON). The file's "cwd" is intentionally not
// decoded — no title carries a cwd since #133.
func (OSRuntime) PaneFiles(dataDir, tag string) []PaneInfo {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return nil
	}
	matches, _ := filepath.Glob(paths.PaneGlob())
	var out []PaneInfo
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var p struct {
			PaneID string `json:"pane_id"`
		}
		if json.Unmarshal(b, &p) != nil {
			continue
		}
		agent, ok := paths.AgentFromPane(m)
		if !ok {
			continue
		}
		out = append(out, PaneInfo{
			Agent:  agent,
			PaneID: p.PaneID,
		})
	}
	return out
}

// ContextCount returns the humanized context-window size for (tag, agent) by
// invoking the exact `pair context` code path in-process (no subprocess) — the
// #92-landed contextcmd.Run, captured to a buffer. Empty when unresolved.
func (OSRuntime) ContextCount(tag, agent string) string {
	var buf bytes.Buffer
	contextcmd.Run([]string{tag, agent}, contextcmd.EnvFromOS(), &buf)
	return strings.TrimSpace(buf.String())
}

// SessionActivity returns only the established inventory root's authorized
// activity timestamp.
func (OSRuntime) SessionActivity(tag, agent string) (time.Time, bool) {
	env := contextcmd.EnvFromOS()
	runtime := sessioninventory.NewOSRuntime(env.Home, env.PairDataDir)
	query, err := sessioninventory.QuerySession(runtime, env.PairScopeKey, tag, sessioninventory.Agent(agent))
	if err != nil {
		return time.Time{}, false
	}
	activity, ok, err := sessioninventory.ActivityForSession(runtime, query)
	if err != nil || !ok {
		return time.Time{}, false
	}
	return activity.LastActivityAt, true
}
