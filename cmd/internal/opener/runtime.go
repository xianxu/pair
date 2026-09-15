package opener

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// OSRuntime implements Runtime with real zellij/nvim/exec/fs calls. The fs
// primitives (ReadFile/WriteFile/WriteAtomic/Remove/FileSize/Touch/Executable)
// come from the embedded osfs.FS (#93 M3).
type OSRuntime struct{ osfs.FS }

func NewOSRuntime() OSRuntime { return OSRuntime{} }

func (OSRuntime) EstablishedSessionID(dataDir, scopeKey, tag, agent string) (string, sessioninventory.BindingStatus) {
	home, _ := os.UserHomeDir()
	runtime := sessioninventory.NewOSRuntime(home, dataDir)
	query, err := sessioninventory.QuerySession(runtime, scopeKey, tag, sessioninventory.Agent(agent))
	if err != nil || query.Root == nil {
		return "", query.Status
	}
	return query.Root.NativeID, query.Status
}

func (OSRuntime) Sleep(d time.Duration)        { time.Sleep(d) }
func (OSRuntime) Getpid() string               { return strconv.Itoa(os.Getpid()) }
func (OSRuntime) ProcessAlive(pid string) bool { return procutil.Alive(pid) }

// RenderScrollback runs `pair scrollback-render` in-process (scrollbackcmd.Run,
// #92) rather than shelling out — the render is synchronous, so no subprocess is
// needed (ARCH-DRY; drops the shell's `$PAIR_HOME/bin/pair` dependency here).
func (OSRuntime) RenderScrollback(raw, events, ansi, viewport string) error {
	if code := scrollbackcmd.RunWithEnv([]string{"--viewport", viewport, raw, events, ansi}, os.Getenv, io.Discard, io.Discard); code != 0 {
		return &renderError{code: code}
	}
	return nil
}

type renderError struct{ code int }

func (e *renderError) Error() string { return fmt.Sprintf("scrollback-render exit %d", e.code) }

// AgentPaneID returns the first non-plugin, non-floating, titled (≠ "draft")
// pane id from `zellij action list-panes --json`, or "" — the Go port of the
// shell's jq recursive-descent selector.
func (OSRuntime) AgentPaneID() string {
	out, err := exec.Command("zellij", "action", "list-panes", "--json").Output()
	if err != nil {
		return ""
	}
	var root interface{}
	if json.Unmarshal(out, &root) != nil {
		return ""
	}
	return firstAgentPaneID(root)
}

// firstAgentPaneID recursively walks the decoded JSON for the first object that
// is a real (non-plugin, non-floating) titled pane and returns its id. Map
// iteration order is Go-random (vs jq's document order), but that only matters
// if >1 candidate exists — under Pair's one-draft-pane invariant the draft pane is
// excluded by title and the floating viewers by is_floating, so exactly one pane
// matches and the pick is deterministic in practice.
func firstAgentPaneID(v interface{}) string {
	switch t := v.(type) {
	case map[string]interface{}:
		if isAgentPane(t) {
			if id := paneIDString(t["id"]); id != "" {
				return id
			}
		}
		for _, child := range t {
			if id := firstAgentPaneID(child); id != "" {
				return id
			}
		}
	case []interface{}:
		for _, child := range t {
			if id := firstAgentPaneID(child); id != "" {
				return id
			}
		}
	}
	return ""
}

func isAgentPane(m map[string]interface{}) bool {
	plugin, hasPlugin := m["is_plugin"].(bool)
	floating, hasFloating := m["is_floating"].(bool)
	title, hasTitle := m["title"].(string)
	if !hasPlugin || !hasFloating || !hasTitle {
		return false
	}
	return !plugin && !floating && title != "" && title != "draft"
}

func paneIDString(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return strconv.Itoa(int(n))
	case string:
		return n
	}
	return ""
}

func (OSRuntime) DumpScreen(paneID string) (string, error) {
	out, err := exec.Command("zellij", "action", "dump-screen", "--pane-id", "terminal_"+paneID).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// StartDetached launches `sh -c script` in a new session (setsid) so a floating
// -pane teardown can't reach it. Go's SysProcAttr.Setsid replaces the shell's
// setsid / perl POSIX::setsid fork (works on macOS + Linux). stderr → statusPath.
func (OSRuntime) StartDetached(script string, extraEnv []string, statusPath string) (string, error) {
	statusF, err := os.OpenFile(statusPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer statusF.Close()
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return "", err
	}
	defer devNull.Close()

	cmd := exec.Command("sh", "-c", childGate+script)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = statusF
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	lease, err := startProtected(cmd)
	if err != nil {
		return "", err
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	// Reap the detached child asynchronously so it doesn't linger as a zombie
	// under this (short-lived) launcher; the setsid child is already reparented
	// away, so this Wait only cleans our own bookkeeping.
	go func() { _ = cmd.Wait(); _ = lease.Close() }()
	return pid, nil
}

// RunViewer execs nvim on file with luaPath config as a HELD child (inherits the
// floating pane's tty), returning when the user quits.
func (OSRuntime) RunViewer(luaPath, file string, extraEnv []string) error {
	cmd := exec.Command("sh", "-c", childGate+`exec nvim -u "$1" "$2"`, "pair-viewer", luaPath, file)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	lease, err := startProtected(cmd)
	if err != nil {
		return err
	}
	defer lease.Close()
	return cmd.Wait()
}

// The child cannot touch managed artifacts until its actual incarnation is
// registered. Parent death before the grant closes the pipe: the child exits.
const childGate = "IFS= read -r pair_gate <&3 || exit 1\nexec 3<&-\n"

func startProtected(cmd *exec.Cmd) (*storagegc.ProcessLease, error) {
	env := map[string]string{}
	for _, entry := range cmd.Env {
		k, v, ok := strings.Cut(entry, "=")
		if ok {
			env[k] = v
		}
	}
	var coordinator *storagegc.Coordinator
	var ownerErr error
	owner, ownerErr := storagegc.SelectedOwner(env["PAIR_DATA_DIR"], env["PAIR_SCOPE_KEY"], env["PAIR_TAG"])
	if ownerErr != nil {
		return nil, fmt.Errorf("retention child owner: %w", ownerErr)
	}
	coordinator, ownerErr = storagegc.NewCoordinator(owner.DataDir)
	if ownerErr != nil {
		return nil, ownerErr
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.ExtraFiles = []*os.File{reader}
	if err := cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	_ = reader.Close()
	fail := func(err error) (*storagegc.ProcessLease, error) { _ = writer.Close(); _ = cmd.Wait(); return nil, err }
	process, err := storagegc.CurrentProcessIdentity(cmd.Process.Pid)
	if err != nil {
		return fail(err)
	}
	id, err := coordinator.AcquireRoleProcessTarget(context.Background(), owner, process, env["PAIR_RETENTION_ROLE"], env["PAIR_RETENTION_START_ID"], env["PAIR_RETENTION_TARGET"])
	if err != nil {
		return fail(err)
	}
	lease := &storagegc.ProcessLease{Coordinator: coordinator, Owner: owner, ID: id}
	if _, err := io.WriteString(writer, "ready\n"); err != nil {
		_, _ = fail(err)
		_ = lease.Close()
		return nil, err
	}
	_ = writer.Close()
	return lease, nil
}
