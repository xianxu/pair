package titlepoller

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijsession"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// OSRuntime implements Runtime with real zellij/cmux/fs/process calls. The fs
// primitives (ReadFile/WriteFile/Remove/ModTime) come from the embedded
// osfs.FS (#93 M3).
type OSRuntime struct {
	osfs.FS
	ctx        context.Context
	projection *titleProjection
	err        error
}

func NewOSRuntime() OSRuntime {
	raw := os.Getenv("PAIR_LAUNCH_ORDINAL")
	ordinal, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || ordinal == 0 || strconv.FormatUint(ordinal, 10) != raw {
		return OSRuntime{err: errors.New("title projection requires exact launch ordinal")}
	}
	return OSRuntime{ctx: context.Background(), projection: newTitleProjection(contextcmd.EnvFromOS(), ordinal)}
}
func (r OSRuntime) Err() error {
	if r.err != nil {
		return r.err
	}
	if r.projection != nil {
		return r.projection.Err()
	}
	return errors.New("title projection unavailable")
}
func (r OSRuntime) WithContext(ctx context.Context) OSRuntime { r.ctx = ctx; return r }
func (r OSRuntime) context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}
func (r OSRuntime) command(build func(context.Context) *exec.Cmd) ([]byte, error) {
	ctx, cancel := context.WithTimeout(r.context(), 2*time.Second)
	defer cancel()
	cmd := build(ctx)
	cmd.WaitDelay = 100 * time.Millisecond
	return cmd.Output()
}

func (OSRuntime) Now() time.Time { return time.Now() }
func (r OSRuntime) Sleep(d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-r.context().Done():
	}
}
func (OSRuntime) Getpid() string { return strconv.Itoa(os.Getpid()) }

// WriteFile publishes title PID and workspace-owner records through a sibling
// temporary file and rename, so abrupt lease termination cannot expose a prefix.
func (r OSRuntime) WriteFile(path, data string) error { return r.FS.WriteAtomic(path, data) }
func (OSRuntime) ProcessAlive(p string) bool          { return procutil.Alive(p) }
func (OSRuntime) ProcessCommand(p string) string      { return procutil.Command(p) }

// SessionAlive reports whether a zellij server is running session. An EXITED
// (resurrectable) row is gone: `--short` lists those too, which kept pollers
// ticking for days after their session ended (pair#420).
func (r OSRuntime) SessionAlive(session string) bool {
	out, err := r.command(func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, "zellij", "list-sessions", "--no-formatting")
	})
	if err != nil {
		return false
	}
	return zellijsession.Live(string(out), session)
}

func (r OSRuntime) RenamePane(session, paneID, title string) error {
	_, err := r.command(func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, "zellij", "--session", session, "action",
			"rename-pane", "--pane-id", paneID, title)
	})
	return err
}

func (OSRuntime) CmuxAvailable() bool {
	_, err := exec.LookPath("cmux")
	return err == nil
}

func (r OSRuntime) CmuxRenameWorkspace(title string) error {
	_, err := r.command(func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, "cmux", "rename-workspace", title)
	})
	return err
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

// ContextCount shares the context command's projection and this title's compact
// owner-launch evidence. Empty denotes unresolved or cancelled evidence.
func (r OSRuntime) ContextCount(tag, agent string) string {
	p := r.projection
	if p == nil {
		return ""
	}
	return p.count(tag, agent)
}

// SessionActivity returns only the established inventory root's authorized
// activity timestamp.
func (r OSRuntime) SessionActivity(tag, agent string) (time.Time, bool) {
	p := r.projection
	if p == nil {
		return time.Time{}, false
	}
	return p.activity(tag, agent)
}
