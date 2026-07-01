package opener

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
)

// OSRuntime implements Runtime with real zellij/nvim/exec/fs calls.
type OSRuntime struct{}

func NewOSRuntime() OSRuntime { return OSRuntime{} }

func (OSRuntime) Sleep(d time.Duration)        { time.Sleep(d) }
func (OSRuntime) Getpid() string               { return strconv.Itoa(os.Getpid()) }
func (OSRuntime) ProcessAlive(pid string) bool { return procutil.Alive(pid) }

func (OSRuntime) ReadFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

func (OSRuntime) WriteFile(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0644)
}

// WriteAtomic writes via a sibling temp file + rename so a concurrent reader
// never sees a torn write (the shell's `> .tmp && mv -f` for .viewport).
func (OSRuntime) WriteAtomic(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (OSRuntime) Remove(path string) { _ = os.Remove(path) }

func (OSRuntime) FileSize(path string) (int64, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}

func (OSRuntime) Touch(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

func (OSRuntime) Executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0111 != 0
}

// RenderScrollback runs `pair scrollback-render` in-process (scrollbackcmd.Run,
// #92) rather than shelling out — the render is synchronous, so no subprocess is
// needed (ARCH-DRY; drops the shell's `$PAIR_HOME/bin/pair` dependency here).
func (OSRuntime) RenderScrollback(raw, events, ansi string) error {
	if code := scrollbackcmd.Run([]string{raw, events, ansi}, io.Discard, io.Discard); code != 0 {
		return &renderError{code: code}
	}
	return nil
}

type renderError struct{ code int }

func (e *renderError) Error() string { return "scrollback-render exit " + strconv.Itoa(e.code) }

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
// if >1 candidate exists — under pair's two-pane invariant the draft pane is
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

	cmd := exec.Command("sh", "-c", script)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = statusF
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	// Reap the detached child asynchronously so it doesn't linger as a zombie
	// under this (short-lived) launcher; the setsid child is already reparented
	// away, so this Wait only cleans our own bookkeeping.
	go func() { _ = cmd.Wait() }()
	return pid, nil
}

// RunViewer execs nvim on file with luaPath config as a HELD child (inherits the
// floating pane's tty), returning when the user quits.
func (OSRuntime) RunViewer(luaPath, file string, extraEnv []string) error {
	cmd := exec.Command("nvim", "-u", luaPath, file)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
