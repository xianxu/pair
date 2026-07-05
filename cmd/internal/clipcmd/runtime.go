package clipcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/osfs"
)

// OSRuntime implements Runtime with real clipboard / zellij / exec / fs calls.
// The fs primitives (WriteFile / Executable) come from the embedded osfs.FS.
type OSRuntime struct{ osfs.FS }

func NewOSRuntime() OSRuntime { return OSRuntime{} }

// clipboardTool is one candidate OS-clipboard command tried in preference order.
type clipboardTool struct {
	name string
	args []string
}

var copyTools = []clipboardTool{
	{"pbcopy", nil},
	{"wl-copy", nil},
	{"xclip", []string{"-selection", "clipboard", "-i"}},
}

var pasteTools = []clipboardTool{
	{"pbpaste", nil},
	{"wl-paste", []string{"--no-newline"}},
	{"xclip", []string{"-selection", "clipboard", "-o"}},
}

// ClipboardCopy mirrors text onto the OS clipboard via the first available tool.
// If none is installed it silently does nothing, matching the shell's `command
// -v` cascade (copy-on-select still hands off the selection either way).
func (OSRuntime) ClipboardCopy(text string) error {
	for _, t := range copyTools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, t.args...)
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return nil
}

// ClipboardPaste reads the OS clipboard. ok=false only when no clipboard tool is
// installed (clipboard-to-pane's hard error); a found-but-failed or empty read
// returns ("", true), which the caller treats as "nothing to paste".
func (OSRuntime) ClipboardPaste() (string, bool) {
	for _, t := range pasteTools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			continue
		}
		out, err := exec.Command(path, t.args...).Output()
		if err != nil {
			return "", true
		}
		return string(out), true
	}
	return "", false
}

func (OSRuntime) ListPanes(command bool) (string, error) {
	args := []string{"action", "list-panes", "--json"}
	if command {
		args = append(args, "--command")
	}
	out, err := exec.Command("zellij", args...).Output()
	return string(out), err
}

func (OSRuntime) SetPaneColor(id, bg string) {
	_ = exec.Command("zellij", "action", "set-pane-color", "--pane-id", id, "--bg", bg).Run()
}

// startDetached launches cmd in its own session (setsid) with /dev/null stdio and
// returns immediately (does NOT wait) — the shared detach idiom for work that must
// outlive this process: the flash's bg reset, and copy-on-select's post-hook paste
// orchestrator (#100). Closing our devNull handle after Start() is safe — the child
// already holds its own dup'd fds. Extracted so both detaching callers share one
// copy (ARCH-DRY).
func startDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devNull, err := os.Open(os.DevNull); err == nil {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devNull, devNull, devNull
		defer devNull.Close()
	}
	if cmd.Start() == nil {
		go func() { _ = cmd.Wait() }()
	}
}

// ResetPaneColorAfter schedules the flash's bg reset in a detached session
// (setsid), so it survives the caller exiting — copy-on-select execs into
// clipboard-to-pane immediately after the flash, replacing itself. Mirrors
// flash-pane.sh's `( sleep; reset ) & disown`, in the same idiom as opener's
// detached changelog distiller (#93 M2).
func (OSRuntime) ResetPaneColorAfter(id string, d time.Duration) {
	secs := fmt.Sprintf("%.3f", d.Seconds())
	// Pass secs + id as positional args to sh so nothing is interpolated into
	// the script text (injection-safe, though ids come from zellij's own output).
	startDetached(exec.Command("sh", "-c",
		`sleep "$1"; zellij action set-pane-color --pane-id "$2" --reset`,
		"sh", secs, id))
}

// SpawnDetached starts path in its own session (setsid) with /dev/null stdio,
// inheriting the environment, and returns immediately — the copy-on-select hook's
// escape from zellij's copy_command reap (#100): the hook returns fast while the
// orchestrator runs on, unreaped. Shares startDetached with the flash's bg reset.
func (OSRuntime) SpawnDetached(path string, args ...string) {
	startDetached(exec.Command(path, args...))
}

// FocusPane targets a pane by id, trying the bare form then the terminal_<id>
// form (list-panes ids can be either, and zellij's parser accepts both).
func (OSRuntime) FocusPane(id string) error {
	if err := exec.Command("zellij", "action", "focus-pane-id", id).Run(); err == nil {
		return nil
	}
	return exec.Command("zellij", "action", "focus-pane-id", "terminal_"+id).Run()
}

func (OSRuntime) MoveFocus(dir string) {
	_ = exec.Command("zellij", "action", "move-focus", dir).Run()
}

func (OSRuntime) WriteKey(b byte) {
	_ = exec.Command("zellij", "action", "write", strconv.Itoa(int(b))).Run()
}

// RunSubprocess runs path as a child and returns (call-and-return), inheriting
// the environment so flash-pane.sh sees PAIR_FLASH_* / XDG_CACHE_HOME.
func (OSRuntime) RunSubprocess(path string, args ...string) error {
	return exec.Command(path, args...).Run()
}

// ExecReplace replaces this process with path (the shell's `exec`), so the
// clipboard-to-pane hand-off is the terminal step. Returns only on failure.
func (OSRuntime) ExecReplace(path string, args ...string) error {
	return syscall.Exec(path, append([]string{path}, args...), os.Environ())
}

// Log appends a line to the clipboard-debug.log diagnostic
// (${XDG_CACHE_HOME:-$HOME/.cache}/pair/clipboard-debug.log). Best-effort — the
// finicky copy pipeline relies on this to confirm zellij is even invoking us.
func (OSRuntime) Log(line string) { writeDebugLog(line, os.O_APPEND) }

// LogFresh truncates the diagnostic then writes line — copy-on-select (the
// pipeline head) calls it once so the log holds one selection's chain and can't
// grow unbounded. (The source truncated inside clipboard-to-pane, mid-chain,
// which clobbered copy-on-select's own lines; truncating at the head keeps them.)
func (OSRuntime) LogFresh(line string) { writeDebugLog(line, os.O_TRUNC) }

func writeDebugLog(line string, mode int) {
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		cache = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	path := filepath.Join(cache, "pair", "clipboard-debug.log")
	if os.MkdirAll(filepath.Dir(path), 0755) != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|mode, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}
