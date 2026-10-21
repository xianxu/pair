package clipcmd

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
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
	terminalPaneIDs []string
	// clipboard
	copied    string
	pasteText string
	pasteOK   bool
	// panes: returned for any ListPanes call; lastListCommand records the flag,
	// listCalls counts calls (so a test can assert the hook makes none inline).
	panes           string
	panesErr        error
	lastListCommand bool
	listCalls       int
	notices         []string
	keyPanes        []string
	focusErr        error
	clients         string
	// fs
	wrote map[string]string
	// zellij actions
	setColors      []paneColor
	scheduledReset []resetSched
	focused        []string
	keys           []byte
	// spawn/exec
	spawned []execCall
}

func newFake() *fakeRuntime {
	return &fakeRuntime{wrote: map[string]string{}}
}

func (f *fakeRuntime) WriteFile(p, d string) error { f.wrote[p] = d; return nil }
func (f *fakeRuntime) ClipboardCopy(t string) error {
	f.copied = t
	return nil
}
func (f *fakeRuntime) ClipboardPaste() (string, bool)     { return f.pasteText, f.pasteOK }
func (f *fakeRuntime) TerminalPaneIDs() ([]string, error) { return f.terminalPaneIDs, nil }

func (f *fakeRuntime) ListPanesContext(ctx context.Context) ([]byte, error) {
	f.listCalls++
	f.lastListCommand = true
	return []byte(f.panes), f.panesErr
}
func (f *fakeRuntime) CurrentPaneID() string { return "0" }
func (f *fakeRuntime) ListClientsContext(ctx context.Context) ([]byte, error) {
	if f.clients != "" {
		return []byte(f.clients), nil
	}
	for _, p := range zellijpane.Parse([]byte(f.panes)) {
		if p.IsFocused && !p.IsPlugin && !p.IsFloating {
			return []byte(fmt.Sprintf("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_%s shell\n", p.ID)), nil
		}
	}
	return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"), nil
}
func (f *fakeRuntime) NotifyRefusal(s string) error { f.notices = append(f.notices, s); return nil }
func (f *fakeRuntime) SetPaneColor(id, bg string) error {
	f.setColors = append(f.setColors, paneColor{id, bg})
	return nil
}
func (f *fakeRuntime) ResetPaneColorAfter(id string, d time.Duration) error {
	f.scheduledReset = append(f.scheduledReset, resetSched{id, d})
	return nil
}
func (f *fakeRuntime) FocusPaneContext(ctx context.Context, id string) error {
	f.focused = append(f.focused, id)
	return f.focusErr
}
func (f *fakeRuntime) WritePaneKeyContext(ctx context.Context, id string, b byte) error {
	f.keys = append(f.keys, b)
	f.keyPanes = append(f.keyPanes, id)
	return nil
}
func (f *fakeRuntime) SpawnDetached(path string, args ...string) {
	f.spawned = append(f.spawned, execCall{path, args})
}
func (f *fakeRuntime) Log(string)      {}
func (f *fakeRuntime) LogFresh(string) {}

const (
	// Legacy pre-#133 agent title (frames no longer carry a cwd). Kept on purpose:
	// role detection must not depend on the title's shape.
	agentFocusedPanes = `[
	  {"id":0,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":true,"is_floating":false,
	   "title":"claude [~/workspace/parley.nvim]",
	   "terminal_command":"sh -c exec pair wrap --scrollback-log /d/s.raw claude"},
	  {"id":1,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":false,"is_floating":false,
	   "title":"draft","terminal_command":"sh -c exec nvim -u /h/nvim/init.lua /d/draft-t.md"}
	]`
	// A selection made in the RIGHT TERMINAL. Its command identifies it.
	terminalFocusedPanes = `[
	  {"id":0,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":false,"is_floating":false,
	   "title":"claude","terminal_command":"sh -c exec pair wrap claude"},
	  {"id":4,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":true,"is_floating":false,
	   "title":"terminal","terminal_command":"sh -c exec pair term"}
	]`

	// An Alt+Shift+d split half: zellij reports NO terminal_command and a
	// tab-strip title, so only #123's registry identifies it.
	splitHalfFocusedPanes = `[
	  {"id":0,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":false,"is_floating":false,
	   "title":"claude","terminal_command":"sh -c exec pair wrap claude"},
	  {"id":5,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":true,"is_floating":false,
	   "title":"[terminal 1]","terminal_command":null}
	]`

	draftFocusedPanes = `[
	  {"id":0,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":false,"is_floating":false,
	   "title":"claude","terminal_command":"sh -c exec pair wrap claude"},
	  {"id":1,"is_plugin":false,"tab_id":0,"tab_position":0,"is_focused":true,"is_floating":false,
	   "title":"draft","terminal_command":"sh -c exec nvim -u /h/nvim/init.lua /d/draft-t.md"}
	]`
)

func copyOpts() CopyOnSelectOptions {
	return CopyOnSelectOptions{PairHome: "/h", SelfExe: "/h/bin/pair", Clipboard: clipOpts()}
}

// (a) The HOOK mirrors the selection and detaches the orchestrator. It must run
// NONE of the slow zellij chain inline — that in-hook chain is exactly what zellij
// reaped mid-paste (#100), so the reap fix is "hook does nothing slow". WHETHER
// the hand-off happens is the orchestrator's call (#125 gates it on pane role).
func TestCopyOnSelectHookMirrorsThenDetaches(t *testing.T) {
	f := newFake()
	code := RunCopyOnSelect(copyOpts(), strings.NewReader("selected text"), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if f.copied != "selected text" {
		t.Errorf("clipboard copied %q, want the selection mirrored", f.copied)
	}
	if len(f.spawned) != 1 || f.spawned[0].path != "/h/bin/pair" ||
		len(f.spawned[0].args) != 3 || f.spawned[0].args[0] != "clip" ||
		f.spawned[0].args[1] != "copy-on-select" || f.spawned[0].args[2] != "--orchestrate" {
		t.Errorf("want one detached `/h/bin/pair clip copy-on-select --orchestrate`, got %+v", f.spawned)
	}
	if f.listCalls != 0 || len(f.setColors) != 0 || len(f.focused) != 0 || len(f.keys) != 0 || len(f.wrote) != 0 {
		t.Fatal("hook ran native work inline")
	}
}

func TestCopyOnSelectEmptySelection(t *testing.T) {
	f := newFake()
	code := RunCopyOnSelect(copyOpts(), strings.NewReader(""), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if f.copied != "" || len(f.spawned) != 0 {
		t.Errorf("empty selection should do nothing: copied=%q spawned=%+v", f.copied, f.spawned)
	}
}

// (b) The detached orchestrator flashes the source pane and hands off. Regression
// fixture: a selection in the AGENT pane while cwd is parley.nvim (title contains
// "nvim") must still hand off — in_nvim keys on terminal_command, not the title.
func TestCopyOnSelectOrchestrateHandsOff(t *testing.T) {
	f := newFake()
	f.panes = agentFocusedPanes
	f.pasteText, f.pasteOK = "hello world", true
	code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !f.lastListCommand {
		t.Error("orchestrator must call list-panes with --command")
	}
	if len(f.setColors) != 1 || f.setColors[0].id != "0" {
		t.Errorf("source flash=%v", f.setColors)
	}
	if f.wrote["/dd/quote-t"] != "hello world" || len(f.keys) != 1 || f.keyPanes[0] != "1" {
		t.Errorf("paste effects=%v keys=%v targets=%v", f.wrote, f.keys, f.keyPanes)
	}
}

// (c) Selection in the DRAFT (nvim) pane must NOT hand off (else it self-inserts).
func TestCopyOnSelectOrchestrateInNvimSkips(t *testing.T) {
	f := newFake()
	f.panes = draftFocusedPanes
	code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(f.setColors) != 0 || len(f.wrote) != 0 || len(f.focused) != 0 || len(f.keys) != 0 {
		t.Fatal("nvim selection flashed or pasted")
	}
}

// The admitted source and destination execute without a second admission process.
func TestCopyOnSelectOrchestrateUsesAdmittedFlashInProcess(t *testing.T) {
	f := newFake()
	f.panes = agentFocusedPanes
	f.pasteText, f.pasteOK = "x", true
	if code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if len(f.setColors) != 1 || len(f.keys) != 1 || f.keyPanes[0] != "1" {
		t.Fatalf("effects=%v keys=%v", f.setColors, f.keys)
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
	if !f.lastListCommand {
		t.Error("clipboard admission needs command evidence")
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

// Missing draft refuses rather than sending an unaddressed paste key.
func TestClipboardToPaneNoNvimPaneRefuses(t *testing.T) {
	f := newFake()
	f.pasteText, f.pasteOK = "x", true
	f.panes = `[{"id":0,"is_plugin":false,"tab_id":0,"tab_position":0,"terminal_command":"pair wrap claude"}]`
	if code := RunClipboardToPane(clipOpts(), f, io.Discard); code == 0 {
		t.Fatal("missing draft admitted")
	}
	if len(f.wrote) != 0 || len(f.focused) != 0 || len(f.keys) != 0 || len(f.notices) != 1 {
		t.Fatalf("unsafe fallback: %+v", f)
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

func TestFlashPaneNoPaneRefuses(t *testing.T) {
	f := newFake()
	f.panes = `[]`
	code := RunFlashPane(FlashPaneOptions{}, f, io.Discard)
	if code == 0 {
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

// (d) #125: a selection in the RIGHT TERMINAL must NOT hand off. Selecting a
// path or a command there is usually just a copy, and having it hijack the
// draft (flash + focus steal + `> ` insert) was distracting. The AGENT pane
// keeps the hand-off — case (b) above — because quoting what the agent said
// back into the draft is the point of the feature.
func TestCopyOnSelectOrchestrateSkipsRightTerminal(t *testing.T) {
	f := newFake()
	f.panes = terminalFocusedPanes
	f.terminalPaneIDs = []string{"4"}
	if code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(f.setColors) != 0 || len(f.wrote) != 0 || len(f.focused) != 0 || len(f.keys) != 0 {
		t.Fatal("terminal selection flashed or pasted")
	}
}

// (e) The same gate must hold for an Alt+Shift+d SPLIT HALF, which zellij
// reports with no terminal_command and a "[terminal N]" title — neither
// identifies it, so the gate depends on #123's registry.
func TestCopyOnSelectOrchestrateSkipsSplitTerminalHalfViaRegistry(t *testing.T) {
	f := newFake()
	f.panes = splitHalfFocusedPanes
	f.terminalPaneIDs = []string{"4", "5"}
	if code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(f.setColors) != 0 || len(f.wrote) != 0 || len(f.focused) != 0 || len(f.keys) != 0 {
		t.Fatal("split terminal selection flashed or pasted")
	}
}
