package clipcmd

import (
	"io"
	"strings"
	"testing"
	"time"
)

type paneColor struct{ id, bg string }
type resetSched struct {
	id string
	d  time.Duration
}
type execCall struct {
	path string
	args []string
}

// fakeRuntime is a scriptable Runtime for the clip-pipeline orchestration tests.
type fakeRuntime struct {
	// clipboard
	copied    string
	pasteText string
	pasteOK   bool
	// panes: returned for any ListPanes call; lastListCommand records the flag
	panes           string
	panesErr        error
	lastListCommand bool
	// fs
	wrote      map[string]string
	executable map[string]bool
	// zellij actions
	setColors      []paneColor
	scheduledReset []resetSched
	focused        []string
	moved          []string
	keys           []byte
	// spawn/exec
	subprocess []execCall
	execd      []execCall
}

func newFake() *fakeRuntime {
	return &fakeRuntime{wrote: map[string]string{}, executable: map[string]bool{}}
}

func (f *fakeRuntime) WriteFile(p, d string) error { f.wrote[p] = d; return nil }
func (f *fakeRuntime) Executable(p string) bool    { return f.executable[p] }
func (f *fakeRuntime) ClipboardCopy(t string) error {
	f.copied = t
	return nil
}
func (f *fakeRuntime) ClipboardPaste() (string, bool) { return f.pasteText, f.pasteOK }
func (f *fakeRuntime) ListPanes(command bool) (string, error) {
	f.lastListCommand = command
	return f.panes, f.panesErr
}
func (f *fakeRuntime) SetPaneColor(id, bg string) {
	f.setColors = append(f.setColors, paneColor{id, bg})
}
func (f *fakeRuntime) ResetPaneColorAfter(id string, d time.Duration) {
	f.scheduledReset = append(f.scheduledReset, resetSched{id, d})
}
func (f *fakeRuntime) FocusPane(id string) error { f.focused = append(f.focused, id); return nil }
func (f *fakeRuntime) MoveFocus(dir string)      { f.moved = append(f.moved, dir) }
func (f *fakeRuntime) WriteKey(b byte)           { f.keys = append(f.keys, b) }
func (f *fakeRuntime) RunSubprocess(path string, args ...string) error {
	f.subprocess = append(f.subprocess, execCall{path, args})
	return nil
}
func (f *fakeRuntime) ExecReplace(path string, args ...string) error {
	f.execd = append(f.execd, execCall{path, args})
	return nil
}
func (f *fakeRuntime) Log(string)      {}
func (f *fakeRuntime) LogFresh(string) {}

const (
	agentFocusedPanes = `[
	  {"id":0,"is_plugin":false,"is_focused":true,"is_floating":false,
	   "title":"claude [~/workspace/parley.nvim]",
	   "terminal_command":"sh -c exec pair-wrap --scrollback-log /d/s.raw claude"},
	  {"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,
	   "title":"draft","terminal_command":"sh -c exec nvim -u /h/init.lua /d/draft-t.md"}
	]`
	draftFocusedPanes = `[
	  {"id":0,"is_plugin":false,"is_focused":false,"is_floating":false,
	   "title":"claude","terminal_command":"sh -c exec pair-wrap claude"},
	  {"id":1,"is_plugin":false,"is_focused":true,"is_floating":false,
	   "title":"draft","terminal_command":"sh -c exec nvim -u /h/init.lua /d/draft-t.md"}
	]`
)

func copyOpts() CopyOnSelectOptions { return CopyOnSelectOptions{PairHome: "/h"} }

// (a) The regression: selection in the AGENT pane while cwd is parley.nvim (title
// contains "nvim") must still hand off — in_nvim keys on terminal_command.
func TestCopyOnSelectAgentPaneHandsOff(t *testing.T) {
	f := newFake()
	f.panes = agentFocusedPanes
	f.executable["/h/bin/flash-pane.sh"] = true
	code := RunCopyOnSelect(copyOpts(), strings.NewReader("selected text"), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if f.copied != "selected text" {
		t.Errorf("clipboard copied %q", f.copied)
	}
	if !f.lastListCommand {
		t.Error("copy-on-select must call list-panes with --command")
	}
	if len(f.subprocess) != 1 || f.subprocess[0].path != "/h/bin/flash-pane.sh" ||
		len(f.subprocess[0].args) != 1 || f.subprocess[0].args[0] != "0" {
		t.Errorf("flash-pane not called with focused id 0: %+v", f.subprocess)
	}
	if len(f.execd) != 1 || f.execd[0].path != "/h/bin/clipboard-to-pane.sh" {
		t.Errorf("did not exec clipboard-to-pane.sh: %+v", f.execd)
	}
}

// (b) Selection in the DRAFT (nvim) pane must NOT hand off (else it self-inserts).
func TestCopyOnSelectInNvimSkips(t *testing.T) {
	f := newFake()
	f.panes = draftFocusedPanes
	f.executable["/h/bin/flash-pane.sh"] = true
	code := RunCopyOnSelect(copyOpts(), strings.NewReader("selected text"), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if f.copied != "selected text" {
		t.Errorf("clipboard still mirrored even in nvim: %q", f.copied)
	}
	if len(f.subprocess) != 0 {
		t.Errorf("must not flash when in nvim: %+v", f.subprocess)
	}
	if len(f.execd) != 0 {
		t.Errorf("must not hand off when in nvim: %+v", f.execd)
	}
}

func TestCopyOnSelectEmptySelection(t *testing.T) {
	f := newFake()
	code := RunCopyOnSelect(copyOpts(), strings.NewReader(""), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if f.copied != "" || len(f.execd) != 0 {
		t.Errorf("empty selection should do nothing: copied=%q execd=%+v", f.copied, f.execd)
	}
}

// When flash-pane.sh isn't executable, the flash is skipped but the hand-off
// still happens (the shell's `[ -x ... ]` guard).
func TestCopyOnSelectSkipsFlashWhenNotExecutable(t *testing.T) {
	f := newFake()
	f.panes = agentFocusedPanes // executable map left empty
	code := RunCopyOnSelect(copyOpts(), strings.NewReader("x"), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(f.subprocess) != 0 {
		t.Errorf("flash should be skipped when not executable: %+v", f.subprocess)
	}
	if len(f.execd) != 1 {
		t.Errorf("hand-off should still happen: %+v", f.execd)
	}
}

func clipOpts() ClipboardToPaneOptions {
	return ClipboardToPaneOptions{DataDir: "/dd", Tag: "t", Agent: "claude"}
}

func TestClipboardToPaneStagesFocusesAndTriggers(t *testing.T) {
	f := newFake()
	f.pasteText, f.pasteOK = "hello world", true
	f.panes = agentFocusedPanes
	code := RunClipboardToPane(clipOpts(), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if got := f.wrote["/dd/quote-t"]; got != "hello world" {
		t.Errorf("quote file = %q, want the clipboard body", got)
	}
	if f.lastListCommand {
		t.Error("clipboard-to-pane must call list-panes WITHOUT --command")
	}
	if len(f.focused) != 1 || f.focused[0] != "1" {
		t.Errorf("did not focus the nvim (draft) pane 1: %+v", f.focused)
	}
	if len(f.keys) != 1 || f.keys[0] != 31 {
		t.Errorf("did not send Ctrl-_ (31): %+v", f.keys)
	}
}

func TestClipboardToPaneNoTool(t *testing.T) {
	f := newFake()
	f.pasteOK = false
	if code := RunClipboardToPane(clipOpts(), f, io.Discard); code != 1 {
		t.Fatalf("no clipboard tool → exit 1, got %d", code)
	}
}

func TestClipboardToPaneEmptyClipboard(t *testing.T) {
	f := newFake()
	f.pasteText, f.pasteOK = "", true
	code := RunClipboardToPane(clipOpts(), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(f.wrote) != 0 || len(f.keys) != 0 {
		t.Errorf("empty clipboard should stage/trigger nothing: wrote=%v keys=%v", f.wrote, f.keys)
	}
}

// No resolvable nvim pane → fall back to move-focus down, still trigger paste.
func TestClipboardToPaneNoNvimPaneFallsBack(t *testing.T) {
	f := newFake()
	f.pasteText, f.pasteOK = "x", true
	f.panes = `[{"id":0,"is_focused":true,"is_floating":false,"is_plugin":false,
	             "title":"claude","terminal_command":"sh -c exec pair-wrap claude"}]`
	code := RunClipboardToPane(clipOpts(), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(f.focused) != 0 {
		t.Errorf("should not focus a specific pane: %+v", f.focused)
	}
	if len(f.moved) != 1 || f.moved[0] != "down" {
		t.Errorf("should fall back to move-focus down: %+v", f.moved)
	}
	if len(f.keys) != 1 || f.keys[0] != 31 {
		t.Errorf("should still trigger Ctrl-_: %+v", f.keys)
	}
}

func TestFlashPaneSetsFgAndSchedulesDetachedReset(t *testing.T) {
	f := newFake()
	code := RunFlashPane(FlashPaneOptions{PaneID: "3"}, f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	// The fg phase is exactly one synchronous set-pane-color (NOT two — the reset
	// must not run synchronously, or it would block the caller's focus change).
	if len(f.setColors) != 1 || f.setColors[0] != (paneColor{"3", "#50fa7b"}) {
		t.Errorf("fg set-pane-color wrong: %+v", f.setColors)
	}
	if len(f.scheduledReset) != 1 || f.scheduledReset[0] != (resetSched{"3", 100 * time.Millisecond}) {
		t.Errorf("reset not scheduled detached: %+v", f.scheduledReset)
	}
}

func TestFlashPaneResolvesFocusedWhenNoArg(t *testing.T) {
	f := newFake()
	f.panes = agentFocusedPanes // agent pane 0 is focused
	code := RunFlashPane(FlashPaneOptions{}, f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !f.lastListCommand {
		t.Error("flash-pane resolving the focused pane must pass --command")
	}
	if len(f.setColors) != 1 || f.setColors[0].id != "0" {
		t.Errorf("did not flash resolved focused pane 0: %+v", f.setColors)
	}
}

func TestFlashPaneNoPaneNoOp(t *testing.T) {
	f := newFake()
	f.panes = `[]`
	code := RunFlashPane(FlashPaneOptions{}, f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(f.setColors) != 0 || len(f.scheduledReset) != 0 {
		t.Errorf("no pane → no flash: set=%+v reset=%+v", f.setColors, f.scheduledReset)
	}
}

func TestFlashPaneOverrides(t *testing.T) {
	f := newFake()
	code := RunFlashPane(FlashPaneOptions{PaneID: "2", FlashBG: "#ff0000", FlashMS: "250"}, f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if f.setColors[0].bg != "#ff0000" {
		t.Errorf("bg override ignored: %+v", f.setColors)
	}
	if f.scheduledReset[0].d != 250*time.Millisecond {
		t.Errorf("ms override ignored: %+v", f.scheduledReset)
	}
}
