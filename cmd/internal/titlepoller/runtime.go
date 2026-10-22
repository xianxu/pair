package titlepoller

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijsession"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// OSRuntime implements Runtime with real zellij/cmux/fs/process calls. The fs
// primitives (ReadFile/WriteFile/Remove/ModTime) come from the embedded
// osfs.FS (#93 M3).
type OSRuntime struct {
	osfs.FS
	projection           *titleProjection
	err                  error
	publication          *storagegc.Coordinator
	owner                artifactpath.StorageOwner
	workspacePublication string
}

func NewOSRuntime() OSRuntime {
	raw := os.Getenv("PAIR_LAUNCH_ORDINAL")
	ordinal, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || ordinal == 0 || strconv.FormatUint(ordinal, 10) != raw {
		return OSRuntime{err: errors.New("title projection requires exact launch ordinal")}
	}
	env := contextcmd.EnvFromOS()
	owner, err := storagegc.SelectedOwner(env.PairDataDir, env.PairScopeKey, os.Getenv("PAIR_TAG"))
	if err != nil {
		return OSRuntime{err: err}
	}
	coordinator, err := storagegc.NewCoordinator(owner.DataDir)
	if err != nil {
		return OSRuntime{err: err}
	}
	workspace := ""
	if id := os.Getenv("CMUX_WORKSPACE_ID"); id != "" {
		workspace = "cmux-owner-" + id
		if filepath.Base(workspace) != workspace {
			return OSRuntime{err: errors.New("invalid workspace publication name")}
		}
	}
	return OSRuntime{projection: newTitleProjection(env, ordinal), publication: coordinator, owner: owner, workspacePublication: workspace}
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
func (OSRuntime) Now() time.Time        { return time.Now() }
func (OSRuntime) Sleep(d time.Duration) { time.Sleep(d) }
func (OSRuntime) Getpid() string        { return strconv.Itoa(os.Getpid()) }

// WriteFile publishes only this title's PID and configured workspace ownership.
// Retention's root lock protects private staging while a publisher is alive;
// abrupt group death releases the lock for existing bounded pending recovery.
func (r OSRuntime) WriteFile(path, data string) error {
	if r.publication == nil {
		return errors.New("title metadata publisher unavailable")
	}
	physical, err := storagegc.OwnerFramePath(path)
	if err != nil {
		return err
	}
	paths, err := artifactpath.ResolveScoped(r.owner.Directory(), r.owner.Tag)
	if err != nil {
		return err
	}
	name := filepath.Base(physical)
	if physical != paths.TitlePID() && (r.workspacePublication == "" || physical != filepath.Join(r.owner.Directory(), r.workspacePublication)) {
		return errors.New("publication is outside selected title metadata")
	}
	return r.publication.PublishMetadata(context.Background(), r.owner, name, []byte(data))
}

// SessionAlive reports whether a zellij server is running session. An EXITED
// (resurrectable) row is gone: `--short` lists those too, which kept pollers
// ticking for days after their session ended (pair#420).
func (r OSRuntime) SessionAlive(session string) bool {
	// The title CLI owns and kills the entire inherited process group on lease EOF.
	out, err := (zellijquery.Source{Run: zellijquery.InheritedGroup}).Output(context.Background(), "list-sessions", "--no-formatting")
	if err != nil {
		return false
	}
	return zellijsession.Live(string(out), session)
}

func (r OSRuntime) RenamePane(session, paneID, title string) error {
	_, err := exec.Command("zellij", "--session", session, "action",
		"rename-pane", "--pane-id", paneID, title).Output()
	return err
}

func (OSRuntime) CmuxAvailable() bool {
	_, err := exec.LookPath("cmux")
	return err == nil
}

func (r OSRuntime) CmuxRenameWorkspace(title string) error {
	_, err := exec.Command("cmux", "rename-workspace", title).Output()
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
