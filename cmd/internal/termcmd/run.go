// Package termcmd provides the right-side user terminal wrapper for Pair's
// workbench layout.
package termcmd

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/draftroute"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"strconv"
)

type Runtime interface {
	CachedDraftPaneID() (string, bool)
	CurrentPaneID() string
	ListPanesJSON() ([]byte, error)
	LastLeftPaneID() (string, error)
	RecordLastLeftPaneID(string) error
	LastTerminalPaneID() (string, error)
	RecordLastTerminalPaneID(string) error
	TerminalPaneIDs() ([]string, error)
	RegisterTerminalPane() error
	RunZellijAction(args ...string) error
	RunZellijActionQuiet(args ...string) error
	ShellCommand() (string, []string)
}

// rightTerminalClassifier is a terminal_command that RoleForPane classifies as
// PaneRoleRightTerminal. The fast path in focusedWorkbenchPanes synthesises this
// pane's own report (#220) — the role is known by construction there, but
// RoleForPaneWith still wants a pane to read. Pinned by a test so a change to
// RoleForPane's predicate cannot silently make the synthesised pane classify as
// PaneRoleOther and route every terminal chord wrong.
const rightTerminalClassifier = "pair term"

const rightTerminalPaneShell = `zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunWithRuntime(args, stdin, stdout, stderr, OSRuntime{})
}

func RunWithRuntime(args []string, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	fs := flag.NewFlagSet("term", flag.ContinueOnError)
	fs.SetOutput(stderr)
	testShortcut := fs.String("test-shortcut", "", "exercise a workbench shortcut without starting a shell")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: pair term [--test-shortcut CHORD]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *testShortcut != "" {
		chord, ok := namedChord(*testShortcut)
		if !ok {
			fmt.Fprintf(stderr, "term: unknown shortcut %q\n", *testShortcut)
			return 2
		}
		if err := handleChord(chord, rt, stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "term: %v\n", err)
			return 1
		}
		return 0
	}
	return runShell(stdin, stdout, stderr, rt)
}

func namedChord(name string) (workbenchshortcut.Chord, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alt+j":
		return workbenchshortcut.ChordAltJ, true
	case "alt+k":
		return workbenchshortcut.ChordAltK, true
	case "alt+t":
		return workbenchshortcut.ChordAltT, true
	case "alt+w":
		return workbenchshortcut.ChordAltW, true
	case "alt+r":
		return workbenchshortcut.ChordAltR, true
	case "alt+shift+d":
		return workbenchshortcut.ChordAltShiftD, true
	case "alt+x":
		return workbenchshortcut.ChordAltX, true
	case "alt+/":
		return workbenchshortcut.ChordAltSlash, true
	case "alt+shift+c":
		return workbenchshortcut.ChordAltShiftC, true
	case "ctrl+alt+c":
		return workbenchshortcut.ChordCtrlAltC, true
	case "alt+left", "alt+left-arrow":
		return workbenchshortcut.ChordAltLeft, true
	case "alt+right", "alt+right-arrow":
		return workbenchshortcut.ChordAltRight, true
	case "alt+shift+enter", "alt+shift+return":
		return workbenchshortcut.ChordAltShiftEnter, true
	case "alt+shift+left", "alt+shift+left-arrow":
		return workbenchshortcut.ChordAltShiftLeft, true
	case "alt+shift+right", "alt+shift+right-arrow":
		return workbenchshortcut.ChordAltShiftRight, true
	case "alt+shift+t":
		return workbenchshortcut.ChordAltShiftT, true
	default:
		return workbenchshortcut.ChordUnknown, false
	}
}

func handleChord(chord workbenchshortcut.Chord, rt Runtime, stdin io.Reader, stdout io.Writer) error {
	if decision, ok := workbenchshortcut.DecideGlobal(chord); ok {
		return runDecision(decision, workbenchPanes{}, rt, stdin, stdout)
	}
	panes, err := focusedWorkbenchPanes(rt)
	if err != nil {
		return err
	}
	lastLeft, err := rt.LastLeftPaneID()
	if err != nil {
		return err
	}
	terminalIDs, err := rt.TerminalPaneIDs()
	if err != nil {
		terminalIDs = nil
	}
	decision := workbenchshortcut.Decide(workbenchshortcut.ShortcutInput{
		Role:           workbenchshortcut.RoleForPaneWith(panes.focused, terminalIDs),
		Chord:          chord,
		FocusedPaneID:  panes.focused.ID,
		LastLeftPaneID: lastLeft,
		DraftPaneID:    panes.draft.ID,
	})
	return runDecision(decision, panes, rt, stdin, stdout)
}

type workbenchPanes struct {
	focused zellijpane.Pane
	draft   zellijpane.Pane
}

// registered reports whether paneID is a live self-registered `pair term` pane.
//
// Both fast paths gate on this rather than on ZELLIJ_PANE_ID alone (BR-5): the
// env var says which pane we are, not what KIND of pane, and synthesising a
// right-terminal role from it unchecked would misroute every chord in any pane
// that happens to run this code. The registry is the signal that says
// "right terminal" — the same one RoleForPaneWith uses to recognise split
// halves that zellij reports without a terminal_command.
func registered(rt Runtime, paneID string) bool {
	ids, err := rt.TerminalPaneIDs()
	if err != nil {
		return false
	}
	return workbenchshortcut.Registered(ids, paneID)
}

func focusedWorkbenchPanes(rt Runtime) (workbenchPanes, error) {
	// Fast path (#220). run.go's own comment below records why this is sound:
	// bytes on `pair term`'s stdin can only mean its OWN pane is the input, so
	// on the live path the focused pane IS this right terminal. The caller
	// needs exactly two things from the result — the focused pane (for
	// RoleForPaneWith and its ID) and the draft's ID — and both are available
	// without listing panes: the current id from the env, and the draft id from
	// the sidecar draftroute already caches and validates.
	//
	// Synthesised with the terminal_command that classifies it, rather than a
	// bare ID, so RoleForPaneWith reaches the same verdict it would from the
	// real report instead of relying on a registry lookup this path does not do.
	if currentID := rt.CurrentPaneID(); currentID != "" && registered(rt, currentID) {
		// rt.CachedDraftPaneID, not draftroute's env reader: the Runtime already
		// exposes this seam (run.go:29) and reaching past it makes the branch
		// untestable (BR-4).
		if draftID, ok := rt.CachedDraftPaneID(); ok {
			return workbenchPanes{
				focused: zellijpane.Pane{ID: currentID, TerminalCommand: rightTerminalClassifier},
				draft:   zellijpane.Pane{ID: draftID},
			}, nil
		}
	}
	data, err := rt.ListPanesJSON()
	if err != nil {
		return workbenchPanes{}, err
	}
	var out workbenchPanes
	var haveOwn bool
	current := rt.CurrentPaneID()
	for _, pane := range zellijpane.Parse(data) {
		// Bytes on pair term's stdin can only mean its OWN pane is the input
		// target, so the process's pane id outranks the is_focused scan —
		// zellij reports per-client focus and several panes can carry
		// is_focused at once (draft + terminal seen live in the tiled smoke).
		if !pane.IsPlugin && current != "" && pane.ID == current {
			out.focused, haveOwn = pane, true
		}
		if pane.IsFocused && !haveOwn {
			out.focused = pane
		}
		if workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleLeftDraft {
			out.draft = pane
		}
	}
	if out.focused.ID == "" {
		return workbenchPanes{}, fmt.Errorf("no focused zellij pane found")
	}
	return out, nil
}

func runDecision(decision workbenchshortcut.ShortcutDecision, panes workbenchPanes, rt Runtime, stdin io.Reader, stdout io.Writer) error {
	if decision.Disposition != workbenchshortcut.DispositionHandle {
		return nil
	}
	if decision.RecordLastLeftPaneID != "" {
		if err := rt.RecordLastLeftPaneID(decision.RecordLastLeftPaneID); err != nil {
			return err
		}
	}
	if decision.RecordLastTerminalPaneID != "" {
		if err := rt.RecordLastTerminalPaneID(decision.RecordLastTerminalPaneID); err != nil {
			return err
		}
	}
	if decision.DraftLuaFunction != "" {
		return draftroute.RouteLua(rt, decision.DraftLuaFunction, decision.FocusDraft)
	}
	switch decision.Action {
	case workbenchshortcut.ActionNewTab, workbenchshortcut.ActionCloseTab, workbenchshortcut.ActionRenameTab:
		return nil
	case workbenchshortcut.ActionFocusPane:
		if decision.TargetPaneID == "" {
			return nil
		}
		return rt.RunZellijAction("focus-pane-id", decision.TargetPaneID)
	case workbenchshortcut.ActionTerminalPrevTab, workbenchshortcut.ActionTerminalNextTab, workbenchshortcut.ActionTerminalNewTab:
		chord, ok := workbenchshortcut.TabChordFor(decision.Action)
		if !ok {
			return nil
		}
		return layoutcmd.SwitchRightTerminalTab(rt, chord)
	case workbenchshortcut.ActionFocusRightTerminal:
		// One picker for the right-terminal jump (shared with draft nvim and
		// pair wrap): id-based, preferring the recorded last-used split half.
		return layoutcmd.FocusRightTerminal(rt)
	case workbenchshortcut.ActionSplitTerminalDown:
		return splitTerminalDown(rt)
	case workbenchshortcut.ActionToggleFocusedLayout:
		if layoutcmd.RunToggleFocused(nil, rt, io.Discard) != 0 {
			return fmt.Errorf("toggle focused layout failed")
		}
		return nil
	default:
		return nil
	}
}

type terminalHost interface {
	hostty.Host
	io.Reader
	ReadContext(context.Context, []byte) (int, error)
}

func runShell(stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	if err := rt.RegisterTerminalPane(); err != nil {
		fmt.Fprintf(stderr, "term: register terminal pane: %v\n", err)
	}
	env, err := runtimebundle.TerminalEnvironment(workbenchshortcut.DataDirFromEnv())
	if err != nil {
		fmt.Fprintf(stderr, "term: terminal profile: %v\n", err)
		return 1
	}
	input, _ := stdin.(*os.File)
	output, _ := stdout.(*os.File)
	if err := runShellOnHost(hostty.NewOSHost(input, output), rt, env...); err != nil {
		fmt.Fprintf(stderr, "term: %v\n", err)
		return 1
	}
	return 0
}

// runShellOnHost joins input and resize producers while parent transport is
// still owned, then releases presentation before restoring the host terminal.
func runShellOnHost(host terminalHost, rt Runtime, env ...string) (result error) {
	defer func() { result = errors.Join(result, host.Close()) }()
	restore, err := host.MakeRaw()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, restore()) }()
	size, err := host.Size()
	if err != nil {
		return err
	}
	name, args := rt.ShellCommand()
	mux := newTerminalMux(name, args, host, rt)
	mux.rows, mux.cols = size.Rows, size.Cols
	mux.shellEnv = append([]string(nil), env...)
	defer func() { mux.closeAll(); result = errors.Join(result, mux.closeErr) }()
	if err := mux.newTab(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	inputDone, resizeDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(inputDone)
		pumpStdinContext(ctx, host, mux, rt, io.Discard, newRealEscapeTimer())
		mux.mu.Lock()
		mux.stopLocked(nil)
		mux.mu.Unlock()
	}()
	go func() {
		defer close(resizeDone)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-host.Resized():
				if !ok {
					return
				}
				mux.inheritSize(host)
			}
		}
	}()
	var terminated <-chan os.Signal
	if signals, ok := host.(hostty.TerminationHost); ok {
		terminated = signals.Terminated()
	}
	select {
	case <-mux.done:
	case <-terminated:
	case <-mux.presenter.Failed():
		result = mux.presenter.Failure()
	}
	cancel()
	<-inputDone
	<-resizeDone
	mux.mu.Lock()
	result = errors.Join(result, mux.failure)
	mux.mu.Unlock()
	return result
}

type ptyWriter interface {
	writeEvents([]terminal.InputEvent)
	newTab() error
	closeActive()
	beginRename() (int, RenameEditor, error)
	refreshRename(int, RenameEditor)
	finishRename(int, RenameOutcome) error
	previousTab()
	nextTab()
	appMouseMode() bool
	// activeChildOwnsScreen reports whether the active tab's child is a
	// full-screen app, so the pump forwards role-scoped chords to it (#227).
	activeChildOwnsScreen() bool
	// reportError puts a diagnostic on the pane THROUGH the writer loop.
	// On the interface because the input goroutine is where these arise, and
	// it must not reach the pane's fd directly (#199 M2).
	reportError(error)
}

// EscapeTimer is the stdin pump's one escape-ambiguity deadline. Whoever owns
// the pending bytes owns the timer: a rename session arms it for a lone
// pending ESC (expiry cancels the rename); the plain path arms it for any
// held chord/mouse prefix (expiry forwards the prefix to the child). The two
// owners are exclusive — held is drained before a read is processed and a
// read that begins a rename never refills it — so one timer serves both (#234).
type EscapeTimer interface {
	C() <-chan time.Time
	Reset(time.Duration)
	StopAndDrain()
}

type realEscapeTimer struct {
	timer *time.Timer
}

func newRealEscapeTimer() *realEscapeTimer {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	return &realEscapeTimer{timer: timer}
}

func (t *realEscapeTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t *realEscapeTimer) Reset(after time.Duration) {
	t.StopAndDrain()
	t.timer.Reset(after)
}

func (t *realEscapeTimer) StopAndDrain() {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
}

type stdinResult struct {
	data []byte
	err  error
}

type renameSession struct {
	tabID   int
	editor  RenameEditor
	decoder RenameDecoderState
}

func pumpStdin(stdin io.Reader, mux ptyWriter, rt Runtime, stdout io.Writer) {
	pumpStdinWithTimer(stdin, mux, rt, stdout, newRealEscapeTimer())
}

func pumpStdinWithTimer(stdin io.Reader, mux ptyWriter, rt Runtime, stdout io.Writer, timer EscapeTimer) {
	pumpStdinContext(context.Background(), stdin, mux, rt, stdout, timer)
}

func pumpStdinContext(ctx context.Context, stdin io.Reader, mux ptyWriter, rt Runtime, stdout io.Writer, timer EscapeTimer) {
	results := make(chan stdinResult, 1)
	done := make(chan struct{})
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer timer.StopAndDrain()
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			var n int
			var err error
			if r, ok := stdin.(interface {
				ReadContext(context.Context, []byte) (int, error)
			}); ok {
				n, err = r.ReadContext(readCtx, buf)
			} else {
				n, err = stdin.Read(buf)
			}
			result := stdinResult{data: append([]byte(nil), buf[:n]...), err: err}
			select {
			case results <- result:
			case <-readCtx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	// Production reads are cancellation-aware. Finite test readers reach EOF.
	defer func() { cancel(); <-done }()
	var decoder terminal.Decoder
	var rename *renameSession
	handle := func(events []terminal.InputEvent) {
		var pending []terminal.InputEvent
		flushPending := func() {
			if len(pending) > 0 {
				mux.writeEvents(pending)
				pending = nil
			}
		}
		defer flushPending()
		for _, event := range events {
			if event.Reply {
				continue
			}
			if rename != nil {
				flushPending()
				if _, ok := inputChord(event); ok {
					continue
				}
				if _, key := event.Event.(uv.KeyPressEvent); !key {
					continue
				}
				if len(event.Canonical) == 0 {
					continue
				}
				var edits []RenameEvent
				rename.decoder, edits, _ = DecodeRenameInput(rename.decoder, event.Canonical, true, false)
				for _, edit := range edits {
					if edit.Kind == RenameConsume {
						continue
					}
					var outcome RenameOutcome
					rename.editor, outcome = rename.editor.Apply(edit)
					if outcome.Kind != RenameOutcomeNone {
						if err := mux.finishRename(rename.tabID, outcome); err != nil {
							mux.reportError(err)
						}
						rename = nil
						return
					}
					mux.refreshRename(rename.tabID, rename.editor)
				}
				continue
			}
			if chord, ok := inputChord(event); ok {
				flushPending()
				if workbenchshortcut.RightTerminalChordPassesThrough(chord) && mux.activeChildOwnsScreen() {
					mux.writeEvents([]terminal.InputEvent{event})
					continue
				}
				if chord == workbenchshortcut.ChordAltR {
					id, editor, err := mux.beginRename()
					if err != nil {
						mux.reportError(err)
						return
					}
					rename = &renameSession{tabID: id, editor: editor}
					continue
				}
				if !handleTerminalChord(chord, mux, rt) {
					if err := handleChord(chord, rt, stdin, stdout); err != nil {
						mux.reportError(err)
					}
				}
				continue
			}
			if wheel, ok := event.Event.(uv.MouseWheelEvent); ok && !mux.appMouseMode() {
				flushPending()
				switch wheel.Button {
				case uv.MouseWheelUp:
					_ = rt.RunZellijAction("scroll-up")
				case uv.MouseWheelDown:
					_ = rt.RunZellijAction("scroll-down")
				}
				continue
			}
			if _, mouse := event.Event.(uv.MouseEvent); mouse {
				flushPending()
				mux.writeEvents([]terminal.InputEvent{event})
				continue
			}
			pending = append(pending, event)
		}
	}
	flush := func() {
		events, err := decoder.FlushEscape()
		if err != nil {
			mux.reportError(err)
		} else {
			handle(events)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C():
			flush()
		case result := <-results:
			events, err := decoder.Feed(result.data)
			if err != nil {
				mux.reportError(err)
				return
			}
			handle(events)
			if decoder.PendingEscape() {
				timer.Reset(workbenchshortcut.EscapeAmbiguity)
			} else {
				timer.StopAndDrain()
			}
			if result.err != nil {
				flush()
				if rename != nil {
					_ = mux.finishRename(rename.tabID, RenameOutcome{Kind: RenameOutcomeCancel, Name: rename.editor.Original()})
				}
				return
			}
		}
	}
}

// inputChord preserves supported physical aliases while canonical bytes allow
// negotiated keyboard encodings to share the existing product shortcut table.
func inputChord(event terminal.InputEvent) (workbenchshortcut.Chord, bool) {
	if _, key := event.Event.(uv.KeyPressEvent); !key {
		return workbenchshortcut.ChordUnknown, false
	}
	if chord, ok := workbenchshortcut.DecodeChord(event.Raw); ok {
		return chord, true
	}
	return workbenchshortcut.DecodeChord(event.Canonical)
}

func handleTerminalChord(chord workbenchshortcut.Chord, mux ptyWriter, rt Runtime) bool {
	switch chord {
	case workbenchshortcut.ChordAltT, workbenchshortcut.ChordAltShiftT:
		// ChordAltShiftT is the from-anywhere new-tab (#243), delivered here as
		// a global that #227 never passes through, so a full-screen child cannot
		// eat it. Same action as the local ChordAltT (ARCH-DRY). newTab's error
		// is discarded as at the local ChordAltT: a failed spawn surfaces on the
		// child's own EOF path, and reporting it here posts into the writer loop
		// a test driving handleTerminalChord directly has not started.
		_ = mux.newTab()
		return true
	case workbenchshortcut.ChordAltW:
		mux.closeActive()
		return true
	case workbenchshortcut.ChordAltLeft, workbenchshortcut.ChordAltShiftLeft:
		// Alt+Shift+Left is the from-anywhere form (#216). In THIS pane it needs
		// no delivery — it is already here, so it calls the same mux method
		// Alt+Left does. One implementation of tab switching, two callers.
		mux.previousTab()
		return true
	case workbenchshortcut.ChordAltRight, workbenchshortcut.ChordAltShiftRight:
		mux.nextTab()
		return true
	case workbenchshortcut.ChordAltShiftD:
		if err := splitTerminalDown(rt); err != nil {
			mux.reportError(err)
		}
		return true
	case workbenchshortcut.ChordAltShiftEnter:
		_ = layoutcmd.RunToggleFocused(nil, rt, io.Discard)
		return true
	default:
		return false
	}
}

func splitTerminalDown(rt Runtime) error {
	if _, ok, err := currentRightTerminalPane(rt); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("right terminal pane not found")
	}
	// Native tiled split: the invoking terminal holds the client focus (the
	// chord arrived on its stdin, which only happens when this pane is
	// focused), so `--direction down` splits this pane. Deliberately NO
	// --near-current-pane: live smoke showed it makes zellij 0.44.3 create
	// the pane invisibly (process spawned, pane absent from the layout).
	// Quiet because new-pane prints the created pane id to stdout.
	return rt.RunZellijActionQuiet(
		"new-pane",
		"--direction", "down",
		"--name", "terminal",
		"--",
		"sh",
		"-c",
		rightTerminalPaneShell,
	)
}

func currentRightTerminalPane(rt Runtime) (zellijpane.Pane, bool, error) {
	// Fast path (#220): this runs INSIDE `pair term`, so CurrentPaneID() is our
	// own pane, and `pair term` self-registers at startup. A current id present
	// in the live registry is therefore a right terminal by construction — the
	// same conclusion the walk below reaches, without the 590ms that
	// `list-panes --json` costs on zellij 0.44.3. The only caller
	// (splitTerminalDown, run.go:545) discards the pane and reads `ok`.
	if currentID := rt.CurrentPaneID(); currentID != "" && registered(rt, currentID) {
		return zellijpane.Pane{ID: currentID}, true, nil
	}
	data, err := rt.ListPanesJSON()
	if err != nil {
		return zellijpane.Pane{}, false, err
	}
	panes := zellijpane.Parse(data)
	terminalIDs, err := rt.TerminalPaneIDs()
	if err != nil {
		terminalIDs = nil
	}
	isTerminal := func(pane zellijpane.Pane) bool {
		return workbenchshortcut.RoleForPaneWith(pane, terminalIDs) == workbenchshortcut.PaneRoleRightTerminal
	}
	currentID := rt.CurrentPaneID()
	if currentID != "" {
		for _, pane := range panes {
			if pane.ID == currentID && isTerminal(pane) {
				return pane, true, nil
			}
		}
	}
	for _, pane := range panes {
		if pane.IsFocused && isTerminal(pane) {
			return pane, true, nil
		}
	}
	for _, pane := range panes {
		if isTerminal(pane) {
			return pane, true, nil
		}
	}
	return zellijpane.Pane{}, false, nil
}

// The SGR mouse parser moved to cmd/internal/mouseinput (pair#172): couch needs
// the identical decode, and a second copy would be a second wire format. Every
// move-time alias is gone -- an alias with no caller is dead surface that makes
// a move look bigger than it was.

func findSGRMousePress(data []byte) ([]byte, mouseinput.Event, []byte, []byte, bool) {
	return mouseinput.Find(data)
}

func isSGRMousePrefix(data []byte) bool { return mouseinput.IsPrefix(data) }

type OSRuntime struct{}

func (OSRuntime) ListPanesJSON() ([]byte, error) {
	return exec.Command("zellij", "action", "list-panes", "--json", "--command", "--state").Output()
}

func (OSRuntime) CachedDraftPaneID() (string, bool) {
	return draftroute.CachedDraftPaneIDFromEnv()
}

func (OSRuntime) CurrentPaneID() string {
	return os.Getenv("ZELLIJ_PANE_ID")
}

func (OSRuntime) LastLeftPaneID() (string, error) {
	store := workbenchshortcut.LastLeftPaneStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return store.Read()
}

func (OSRuntime) RecordLastLeftPaneID(id string) error {
	store := workbenchshortcut.LastLeftPaneStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return store.Write(id)
}

func (OSRuntime) LastTerminalPaneID() (string, error) {
	store := workbenchshortcut.LastTerminalPaneStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return store.Read()
}

func (OSRuntime) TerminalPaneIDs() ([]string, error) {
	return workbenchshortcut.LiveTerminalPaneIDsFromEnv(func(pid int) bool {
		return procutil.Alive(strconv.Itoa(pid))
	})
}

func (OSRuntime) RegisterTerminalPane() error {
	paneID := os.Getenv("ZELLIJ_PANE_ID")
	if paneID == "" {
		return nil
	}
	reg := workbenchshortcut.TerminalPaneRegistry{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return reg.Register(paneID, os.Getpid())
}

func (OSRuntime) RecordLastTerminalPaneID(id string) error {
	store := workbenchshortcut.LastTerminalPaneStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return store.Write(id)
}

// NEITHER method gives a subprocess the pane's descriptors, and that is the
// point: this Runtime is handed to `layoutcmd` and `draftroute` too
// (`run.go:181,194,198,514`), so a rule enforced at `termcmd`'s call sites
// would not cover the five sites in those packages -- and would not cover the
// next site anyone adds. Making the RUNTIME incapable is the property that
// holds without an enumeration to maintain.
//
// `pair term` is a full-screen process on a shared pty. `zellij action` output
// on that pty lands wherever the child's cursor happens to be, outside the
// writer loop and outside the paint gate; a FAILING action does the same on
// stderr, per wheel tick. There is no `termcmd` context in which that output
// belongs on the pane, so the failure is returned as an error and reported
// through the writer loop (`terminalMux.reportError`) instead.
//
// The two methods therefore behave identically here. Quiet is kept because the
// Runtime interface is shared with callers whose descriptors are not a live
// terminal.
func (OSRuntime) RunZellijAction(args ...string) error {
	return OSRuntime{}.RunZellijActionQuiet(args...)
}

func (OSRuntime) RunZellijActionQuiet(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return runZellijCaptured(cmdArgs)
}

// runZellij takes BOTH descriptors. It hardwired cmd.Stderr = os.Stderr for
// every caller, so routing stdout through Quiet closed the quiet path and left
// the noisy one open: a FAILING `zellij action scroll-up` wrote the pane on
// every wheel tick, outside the writer loop and outside the paint gate, and
// after #199 M4 there is no frame to absorb it.
func runZellij(args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command("zellij", args...)
	// NO STDIN EITHER. The pane's stdin is in RAW MODE and carries the
	// operator's keystrokes; handing it to a subprocess lets that process
	// consume them, and none of termcmd's verbs (focus-pane-id, scroll-up,
	// scroll-down, rename-pane) read stdin at all. Both this file and
	// atlas/architecture.md claimed "neither descriptor" while this line gave
	// away a third one.
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// diagnosticWidth bounds what any single diagnostic may put on the pane. Wide
// enough for a useful message, far short of wrapping onto the child's area.
const diagnosticWidth = 200

// zellijErrorDetailWidth bounds the subprocess half specifically, so a usage
// dump cannot crowd out the part of the message we wrote.
const zellijErrorDetailWidth = 120

// runZellijCaptured runs the action with BOTH descriptors captured, and folds
// what the subprocess said into the returned error.
//
// Keeping the pane clean must not mean destroying the diagnostic. The first
// version of this milestone sent both descriptors to io.Discard, and since the
// wheel-tick callers also drop the error (`_ = rt.RunZellijAction("scroll-up")`),
// a failing action became completely silent -- strictly worse than the noise it
// replaced. The bytes go into the error, where a caller can report or log them,
// and never onto the pane's fd.
func runZellijCaptured(args []string) error {
	var out, errb bytes.Buffer
	err := runZellij(args, &out, &errb)
	if err == nil {
		return nil
	}
	detail := strings.TrimSpace(errb.String())
	if detail == "" {
		detail = strings.TrimSpace(out.String())
	}
	if detail == "" {
		return err
	}
	// NOT sanitized here. That is deliberate: `reportError` sanitizes at the
	// EGRESS, the single point every diagnostic reaches the pane, and a second
	// strip at this producer would be two places holding one safety decision --
	// the shape that has cost this issue several rounds. A mutation deleting a
	// strip here is correctly GREEN, because the safety boundary is elsewhere.
	//
	// The bound stays, as a SIZE guard rather than a safety one: an unbounded
	// usage dump wrapped into an error is carried around by every caller, and
	// the egress would trim it to nothing useful anyway.
	detail = rowtext.Fit(detail, zellijErrorDetailWidth)
	if detail == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, detail)
}

func (OSRuntime) ShellCommand() (string, []string) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, []string{"-i"}
}
