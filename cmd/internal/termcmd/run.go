// Package termcmd provides the right-side user terminal wrapper for Pair's
// workbench layout.
package termcmd

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/draftroute"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/rowtext"
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

func focusedWorkbenchPanes(rt Runtime) (workbenchPanes, error) {
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

// teardown stops the host's resize watcher BEFORE any child pty is closed.
//
// A function rather than two defers, because as two defers the ordering was
// EMERGENT -- it depended on LIFO registration order, and the #146 migration
// silently inverted it by registering host.Close() up next to NewOSHost (BR-2).
// The hazard is concrete: a SIGWINCH arriving during teardown runs resizeAll ->
// Child.Resize -> ptmx.Fd() concurrently with ptmx.Close(), the use-after-close
// workshop/lessons.md records from the scribecmd bug.
//
// Making it explicit is also what makes it TESTABLE: defer ordering inside a
// function that needs a real tty cannot be asserted, and BR-15 is that a fix
// defended only by a comment is not defended.
func teardown(host hostty.Host, closeChildren func()) {
	_ = host.Close()
	closeChildren()
}

func runShell(stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	name, args := rt.ShellCommand()
	// Self-register this pane as a live right terminal: zellij's pane report
	// can't identify split panes (no terminal_command for --direction-created
	// panes; #118 tab-strip titles), so the registry is how every consumer —
	// including this process's own chord routing — recognizes them.
	if err := rt.RegisterTerminalPane(); err != nil {
		fmt.Fprintf(stderr, "term: register terminal pane: %v\n", err)
	}
	stdinFile, _ := stdin.(*os.File)
	stdoutFile, _ := stdout.(*os.File)
	host := hostty.NewOSHost(stdinFile, stdoutFile)

	var restore func() error
	if stdinFile != nil {
		r, err := host.MakeRaw()
		if err != nil {
			fmt.Fprintf(stderr, "term: %v\n", err)
			return 1
		}
		restore = r
		defer func() { _ = restore() }()
	}

	mux := newTerminalMux(name, args, stdout, stderr, rt)
	if stdinFile != nil {
		mux.captureSize(host)
	}
	if err := mux.newTab(); err != nil {
		fmt.Fprintf(stderr, "term: %v\n", err)
		return 1
	}
	defer teardown(host, mux.closeAll)
	defer mux.restoreTerminal()

	if stdinFile != nil {
		go func() {
			for range host.Resized() {
				mux.resizeThroughWriter(host)
			}
		}()
		// loop-exempt: the initial sizing runs BEFORE copyActiveOutput starts
		// (below), so there is no loop to serialize against and nothing else
		// writing. Every later resize goes through resizeThroughWriter.
		mux.inheritSize(host)
	}

	go pumpStdin(stdin, mux, rt, stdout)
	mux.copyActiveOutput()

	if restore != nil {
		_ = restore()
	}
	return 0
}

type ptyWriter interface {
	writeActive([]byte)
	newTab() error
	closeActive()
	beginRename() (int, RenameEditor, error)
	refreshRename(int, RenameEditor) error
	finishRename(int, RenameOutcome) error
	previousTab()
	nextTab()
	appMouseMode() bool
	// reportError puts a diagnostic on the pane THROUGH the writer loop.
	// On the interface because the input goroutine is where these arise, and
	// it must not reach the pane's fd directly (#199 M2).
	reportError(error)
}

type RenameTimer interface {
	C() <-chan time.Time
	Reset(time.Duration)
	StopAndDrain()
}

type realRenameTimer struct {
	timer *time.Timer
}

func newRealRenameTimer() *realRenameTimer {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	return &realRenameTimer{timer: timer}
}

func (t *realRenameTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t *realRenameTimer) Reset(after time.Duration) {
	t.StopAndDrain()
	t.timer.Reset(after)
}

func (t *realRenameTimer) StopAndDrain() {
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
	pumpStdinWithTimer(stdin, mux, rt, stdout, newRealRenameTimer())
}

func pumpStdinWithTimer(stdin io.Reader, mux ptyWriter, rt Runtime, stdout io.Writer, timer RenameTimer) {
	results := make(chan stdinResult, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdin.Read(buf)
			result := stdinResult{err: err}
			if n > 0 {
				result.data = append([]byte(nil), buf[:n]...)
			}
			results <- result
			if err != nil {
				return
			}
		}
	}()

	var held []byte
	var rename *renameSession

	applyRename := func(data []byte, flushEscape, eof bool) {
		if rename == nil {
			return
		}
		var events []RenameEvent
		var exited bool
		rename.decoder, events, exited = DecodeRenameInput(rename.decoder, data, flushEscape, eof)
		for _, event := range events {
			if event.Kind == RenameConsume {
				continue
			}
			var outcome RenameOutcome
			rename.editor, outcome = rename.editor.Apply(event)
			if outcome.Kind != RenameOutcomeNone {
				if err := mux.finishRename(rename.tabID, outcome); err != nil {
					mux.reportError(err)
				}
				timer.StopAndDrain()
				rename = nil
				return
			}
			if err := mux.refreshRename(rename.tabID, rename.editor); err != nil {
				mux.reportError(err)
			}
		}
		if exited {
			timer.StopAndDrain()
			rename = nil
			return
		}
		if len(rename.decoder.Pending) == 1 && rename.decoder.Pending[0] == 0x1b {
			timer.Reset(50 * time.Millisecond)
		} else {
			timer.StopAndDrain()
		}
	}

	for {
		select {
		case <-timer.C():
			applyRename(nil, true, false)
		case result := <-results:
			if len(result.data) > 0 {
				if rename != nil {
					applyRename(result.data, false, false)
					if result.err != nil {
						if rename != nil {
							applyRename(nil, false, true)
						}
						return
					}
					continue
				}
				data := append(held, result.data...)
				held = nil
				for len(data) > 0 {
					chordBefore, chord, _, chordRest, chordOK := workbenchshortcut.FindChord(data)
					mouseBefore, event, rawMouse, mouseRest, mouseOK := findSGRMousePress(data)
					if chordOK && (!mouseOK || len(chordBefore) <= len(mouseBefore)) {
						if len(chordBefore) > 0 {
							mux.writeActive(chordBefore)
						}
						if chord == workbenchshortcut.ChordAltR {
							tabID, editor, err := mux.beginRename()
							if err != nil {
								mux.reportError(err)
								data = nil
								continue
							}
							rename = &renameSession{tabID: tabID, editor: editor}
							applyRename(chordRest, false, false)
							data = nil
							continue
						}
						if !handleTerminalChord(chord, mux, rt) {
							if err := handleChord(chord, rt, stdin, stdout); err != nil {
								mux.reportError(err)
							}
						}
						data = chordRest
						continue
					}
					if mouseOK {
						if len(mouseBefore) > 0 {
							mux.writeActive(mouseBefore)
						}
						switch {
						// A release is never a wheel tick (the wheel reports
						// press-only), so it always passes straight through —
						// the child needs it to close its drag.
						case event.Release:
							mux.writeActive(rawMouse)
						case event.Button == mouseinput.WheelUp:
							if mux.appMouseMode() {
								mux.writeActive(rawMouse)
							} else {
								_ = rt.RunZellijAction("scroll-up")
							}
						case event.Button == mouseinput.WheelDown:
							if mux.appMouseMode() {
								mux.writeActive(rawMouse)
							} else {
								_ = rt.RunZellijAction("scroll-down")
							}
						default:
							mux.writeActive(rawMouse)
						}
						data = mouseRest
						continue
					}
					if workbenchshortcut.IsChordPrefix(data) || isSGRMousePrefix(data) {
						held = append(held, data...)
						break
					}
					mux.writeActive(data)
					data = nil
				}
			}
			if result.err != nil {
				if rename != nil {
					applyRename(nil, false, true)
				} else if len(held) > 0 {
					mux.writeActive(held)
				}
				return
			}
		}
	}
}

func handleTerminalChord(chord workbenchshortcut.Chord, mux ptyWriter, rt Runtime) bool {
	switch chord {
	case workbenchshortcut.ChordAltT:
		_ = mux.newTab()
		return true
	case workbenchshortcut.ChordAltW:
		mux.closeActive()
		return true
	case workbenchshortcut.ChordAltLeft:
		mux.previousTab()
		return true
	case workbenchshortcut.ChordAltRight:
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

type terminalTab struct {
	id   int
	name string

	// child is nil in tests that exercise only naming and tab bookkeeping.
	// The pty, the replay ring and the mouse/alt-screen state all live in it
	// now (#146): `pair term` and `couch` are the same switcher one layer
	// apart, so the child half is shared and only the POLICY differs.
	child *ptychild.Child
}

// ptyChunk is anything the writer loop may act on. `data` from a child is
// SCANNED and forwarded; `own` is a console-originated write (a paint, a
// redraw, a diagnostic) that is gated instead.
//
// One type rather than two channels: the ordering between a child's bytes and
// our own is the whole subject of this milestone, and two channels would let a
// select choose between them in an order nothing pins.
type ptyChunk struct {
	id   int
	data []byte
	err  error

	// own is a console-originated write. Never fed to the gate's scanner --
	// see gate below.
	own []byte
	// diag is a console-originated DIAGNOSTIC: same gate, different deferral
	// policy (queued rather than coalesced -- see owedDiag).
	diag []byte
	// takeover means this write replaces the whole screen (redrawTab), so the
	// scan resets and any owed paint is dropped rather than flushed against a
	// screen that no longer exists. couch's third gate rule.
	takeover bool
	// rowDirty means the child may have destroyed the reserved row -- a margin
	// reset, RIS, an alt-screen transition, or an ERASE. DECSTBM restricts
	// scrolling, not erasing, so a full-screen app's startup clear takes the row
	// while the region is still perfectly intact.
	rowDirty bool
	// replay is the CHILD-originated half of a takeover's bytes, tracked apart
	// from the console-originated prefix so the gate can be fed the former and
	// not the latter.
	replay []byte
	// drained is closed after this event is fully handled; tests wait on it
	// instead of sleeping.
	drained chan struct{}
	// onWriter runs ON THE WRITER GOROUTINE. Two uses: a resize, which must
	// serialize against paints (ARCH-ORDER), and a test reading gate state
	// without racing the loop that owns it.
	onWriter func()
}

// paneWriter is the ONLY way to reach the pane's file descriptor, and it is
// deliberately NOT an io.Writer.
//
// That is the whole design: with no Write method, `fmt.Fprintf(m.pane, …)`,
// `io.WriteString(m.pane, …)` and `m.pane.Write(…)` are all COMPILE ERRORS, so
// a new door cannot be opened by reflex. The predecessor was a test that
// scanned run.go for `m.stdout` -- which missed Fprintf (the most idiomatic
// spelling, and the one this file used pre-M2), missed a second file in the
// same package, and could be fooled by an unrelated comment. Scanning for
// violations is weaker than making them unrepresentable.
//
// Every write states a reason at the call site. The gated writers pass their
// own; the three exemptions pass theirs.
type paneWriter struct{ w io.Writer }

func (p paneWriter) raw(reason string, b []byte) {
	_ = reason // documentation at the call site, not a runtime value
	if len(b) == 0 {
		return
	}
	_, _ = p.w.Write(b)
}

func (p paneWriter) rawString(reason, s string) { p.raw(reason, []byte(s)) }

type terminalMux struct {
	mu        sync.Mutex
	shellName string
	shellArgs []string
	pane      paneWriter
	stderr    io.Writer
	rt        Runtime
	paneID    string
	tabs      []*terminalTab
	active    int
	nextID    int
	output    chan ptyChunk
	done      chan struct{}
	rows      uint16
	cols      uint16
	rename    *activeRename

	// THE GATE. Owned by the writer loop alone, so it needs no lock: every
	// mutation happens in copyActiveOutput.
	//
	// hostScan is fed CHILD bytes ONLY. Feeding our own escapes in would let it
	// frame our bytes with the child's partial and report safe precisely when
	// it is not -- the half couch got wrong first (atlas/couch.md).
	hostScan ptychild.Screen
	// owed is a PAINT deferred because the child's stream was mid-sequence.
	// Deferred and OWED: dropping it leaves a stale row nothing repaints.
	owed []byte
	// captureIDForTest, when set, runs on the writer goroutine inside a resize.
	captureIDForTest func()
	// owedDiag are DIAGNOSTICS deferred the same way, kept separately because
	// they coalesce differently. A paint is a rendering of current state, so a
	// later one supersedes an earlier one; an error is an EVENT, and letting a
	// routine repaint swallow it loses the only report the operator gets.
	owedDiag [][]byte
}

type activeRename struct {
	tabID  int
	editor RenameEditor
}

func newTerminalMux(shellName string, shellArgs []string, stdout, stderr io.Writer, rt Runtime) *terminalMux {
	return &terminalMux{
		shellName: shellName,
		shellArgs: shellArgs,
		pane:      paneWriter{w: stdout},
		stderr:    stderr,
		rt:        rt,
		paneID:    os.Getenv("ZELLIJ_PANE_ID"),
		active:    -1,
		output:    make(chan ptyChunk, 64),
		done:      make(chan struct{}),
	}
}

func (m *terminalMux) newTab() error {
	m.mu.Lock()
	m.nextID++
	id := m.nextID
	name := fmt.Sprintf("terminal %d", id)
	m.mu.Unlock()

	m.mu.Lock()
	size := m.childSizeLocked()
	m.mu.Unlock()

	ready := make(chan struct{})
	child, err := ptychild.Start(ptychild.Options{
		Argv: append([]string{m.shellName}, m.shellArgs...),
		Size: size,
		// The sink hands each chunk to the existing pump. Routing it to the
		// screen stays this mux's decision -- ptychild never learns which tab
		// is active.
		Sink: func(batch ptychild.OutputBatch) {
			<-ready
			// batch.RowDirty, NOT Child.TakeRowDirty(): readLoop already
			// DRAINED the flag into this batch whenever a Sink is set, so a
			// separate call would read false forever and the strip would never
			// come back after a full-screen child's startup clear (finding 7).
			m.output <- ptyChunk{id: id, data: batch.Raw, rowDirty: batch.RowDirty}
		},
	})
	if err != nil {
		return err
	}
	tab := &terminalTab{id: id, name: name, child: child}

	m.mu.Lock()
	m.tabs = append(m.tabs, tab)
	m.active = len(m.tabs) - 1
	m.mu.Unlock()
	m.renamePane()
	// Clear before releasing startup output. The child's pump may already have
	// read bytes, but its sink is gated until the tab is registered and the
	// screen is ready; replaying that same buffer here would duplicate it when
	// the queued live copy arrives (BR-9).
	m.redrawTab(nil)
	close(ready)

	// The child's own pump feeds Sink; when it ends, the tab is gone.
	go func() {
		child.Wait()
		m.output <- ptyChunk{id: id, err: io.EOF}
	}()
	return nil
}

// copyActiveOutput is THE writer. Every byte that reaches the pane passes
// through this one goroutine -- child output, redraws, paints, diagnostics --
// because a second writer is how a paint lands inside a child's escape
// sequence. atlas/couch.md states the mechanism: "a pty read boundary falls
// wherever the kernel puts it, so a paint written between two chunks can land
// inside one of the child's escape sequences."
func (m *terminalMux) copyActiveOutput() {
	for {
		select {
		case chunk := <-m.output:
			m.handleChunk(chunk)
		case <-m.done:
			return
		}
	}
}

// handleChunk runs on the writer goroutine ONLY, which is what lets hostScan and
// owed be plain fields with no lock.
func (m *terminalMux) handleChunk(chunk ptyChunk) {
	defer func() {
		if chunk.drained != nil {
			close(chunk.drained)
		}
	}()

	if chunk.onWriter != nil {
		chunk.onWriter()
	}

	switch {
	case chunk.err != nil:
		m.removeTab(chunk.id)

	case chunk.takeover:
		m.applyTakeover(chunk.replay)

	case chunk.diag != nil:
		m.writeDiag(chunk.diag)

	case chunk.own != nil:
		m.writeOwn(chunk.own)

	case chunk.data == nil:
		// A bare event (a drain or an inspect). It carries no bytes in either
		// direction, so it must touch neither the gate nor the owed slot.

	default:
		// THE GATE MODELS THE TERMINAL, so it is fed exactly what the terminal
		// is shown -- no more, no less.
		//
		// An earlier version fed EVERY chunk and wrote only the active tab's.
		// A background tab emitting a partial escape then pinned the gate
		// mid-sequence against a terminal that had seen none of it, deferring
		// paints indefinitely; and any byte written without being fed left the
		// gate blind to a sequence the terminal really was inside.
		//
		// FeedFraming, not Feed: the gate needs sequence boundaries and nothing
		// else, and Feed additionally RETAINS output for notification
		// observers -- which this consumer has none of, so it would be an
		// unbounded buffer growing behind a terminal that never reads it.
		//
		// No buffering here any more: ptychild.Child appends to its own ring
		// BEFORE the sink runs, so a switch racing a chunk still repaints a
		// current screen.
		if m.isActive(chunk.id) {
			m.hostScan.FeedFraming(chunk.data)
			// Not a USER of the gate but the thing it MODELS: gating a child
			// against its own stream state deadlocks it against itself. Fed
			// immediately above, so model and terminal move together.
			m.pane.raw("child output", chunk.data)
		}
		m.flushOwed()
		if chunk.rowDirty {
			m.paintStripInline()
		}
	}
}

// applyTakeover replaces the whole screen: clear, replay, then settle the gate.
// Runs on the writer goroutine, whether reached through the channel or called
// inline by another handler already on it (see removeTab).
func (m *terminalMux) applyTakeover(replay []byte) {
	// couch's THIRD gate rule (console.go:992-995): the screen is being replaced
	// wholesale, so whatever partial sequence the old content left is no longer
	// on screen to be corrupted. Reset the scan, and DROP the owed paint rather
	// than flushing it against a screen that is gone.
	pendingDiag := m.owedDiag
	m.hostScan = ptychild.Screen{}
	m.owed = nil
	m.owedDiag = nil

	// THE ONE EXEMPTION from the gate, and it is deliberate rather than
	// overlooked. Every other console-originated write consults hostScan; this
	// one cannot, because it is what makes the gate's state meaningful again:
	// HomeAndClear discards the screen the old scan described, so consulting
	// that scan first would defer a write against a screen about to cease
	// existing. The reset above is what earns the exemption -- it happens
	// BEFORE these writes, so nothing downstream reads a stale mid-sequence.
	//
	// Enumerated and enforced by TestEveryConsoleWriteIsGatedOrExplicitlyExempt.
	m.pane.rawString("takeover: clears the screen the old scan described", hostty.HomeAndClear)
	m.pane.raw("takeover: replay, fed to the gate immediately below", replay)
	// The replay is CHILD bytes and the terminal has now seen them, so the gate
	// must too -- it is replay-safe (ptychild strips queries and cuts at
	// ReplaySafeEnd) but "usually ends at a boundary" is an assumption, and the
	// gate exists so nothing has to assume.
	m.hostScan.FeedFraming(replay)

	// Diagnostics survive a takeover -- unlike a paint, an error is not made
	// obsolete by the screen being replaced. But they go through writeDiag, NOT
	// straight to the pane: the replay we just fed may have ended mid-sequence,
	// and writing into it is the exact corruption this milestone exists to
	// prevent. Re-queued if so, and flushed at the next boundary.
	for _, d := range pendingDiag {
		m.writeDiag(d)
	}

	// The takeover just cleared the screen, so the row it cleared is ours to
	// put back. Inline, not posted: this runs on the writer goroutine.
	m.paintStripInline()
}

// writeDiag writes a diagnostic, or QUEUES it if the stream is mid-sequence.
// Queued, not coalesced -- see owedDiag.
func (m *terminalMux) writeDiag(b []byte) {
	if m.hostScan.MidSequence() {
		m.owedDiag = append(m.owedDiag, b)
		return
	}
	m.pane.raw("diagnostic, gate consulted", b)
}

// writeOwn writes console-originated bytes, or defers them if the child's stream
// is mid-sequence.
func (m *terminalMux) writeOwn(b []byte) {
	if m.hostScan.MidSequence() {
		// DEFERRED AND OWED. Dropping it would leave a stale row that nothing
		// repaints -- the failure couch names explicitly.
		//
		// One slot, and a later paint REPLACES an earlier one. That is correct
		// for this payload and would be wrong for a stream: the row is a
		// rendering of current state, so the freshest paint is the only one
		// worth landing, and queueing them would draw a burst of stale rows on
		// the next boundary. It is a coalescing slot, not a buffer.
		m.owed = b
		return
	}
	m.pane.raw("paint, gate consulted", b)
}

// flushOwed writes deferred work once the child's stream reaches a boundary.
//
// Called from the CHILD-DATA branch only, and that is correct rather than an
// oversight: nothing else can clear the gate. A console write cannot -- if we
// owed, we were mid-sequence, and our own bytes are never fed to the scanner --
// and a takeover resets rather than flushes. Adding a call to the console
// branches looks like defence and is provably dead.
//
// KNOWN GAP, deferred to M3 with the strip: a child that goes silent while
// mid-sequence strands the owed write indefinitely. Nothing paints in M2 so it
// is unreachable here; M3 owes it a flush deadline, since "the child stopped
// mid-escape" is indistinguishable from "the child is slow".
func (m *terminalMux) flushOwed() {
	if m.hostScan.MidSequence() {
		return
	}
	// Diagnostics first: they were queued before the paint that may describe a
	// state they explain, and every one of them lands.
	for _, d := range m.owedDiag {
		m.pane.raw("owed diagnostic, boundary reached", d)
	}
	m.owedDiag = nil
	if m.owed == nil {
		return
	}
	b := m.owed
	m.owed = nil
	m.pane.raw("owed paint, boundary reached", b)
}

// reportError puts a diagnostic on the pane through the writer loop.
//
// stderr is the SAME terminal as stdout here, so a `term:` line printed from the
// input goroutine while a child is mid-sequence corrupts the pane exactly as a
// stray paint would -- and after #199 M4 there is no frame to absorb it. Startup
// diagnostics (before copyActiveOutput runs) still write stderr directly: there
// is no loop yet and no child to corrupt.
func (m *terminalMux) reportError(err error) {
	if err == nil {
		return
	}
	// SANITIZED HERE, at the single point diagnostics reach the pane -- not at
	// each producer. An error's text can come from anywhere: a subprocess's
	// stderr, a filesystem path, an operator-typed tab name in a wrapped error.
	// Filtering one producer moves the hazard to the next one, which is the
	// mistake #208 made with `ps` output and fixed by filtering at emission.
	text := rowtext.SanitizeAndFit("pair term: "+err.Error(), diagnosticWidth)
	m.enqueue(ptyChunk{diag: []byte(text + "\r\n")})
}

// paintOwn queues a console-originated write onto the writer loop.
func (m *terminalMux) paintOwn(b []byte) {
	m.enqueue(ptyChunk{own: b})
}

// enqueue posts an event and returns. It does NOT wait for the loop to handle
// it, and that is load-bearing rather than a shortcut.
//
// Waiting deadlocks: `runShell` calls newTab() -- which redraws -- at run.go:253
// and only starts copyActiveOutput at :270, so a synchronous enqueue would hang
// `pair term` on startup, before the loop that would release it exists. The
// first version did exactly that and hung the suite.
//
// Nothing is lost by not waiting. Ordering, which is what this milestone is
// about, comes from the channel: every write reaches the pane through one loop
// in the order it was posted. A caller that needed to observe its own write
// would be a caller reading the terminal, and there is none.
//
// NO CHANNEL, NO QUEUE. A mux built by struct literal -- which some tests do to
// exercise one method -- has a nil output channel, so the event is handled
// inline. Safe rather than a hole in the envelope, and the condition says why:
// `output == nil` means copyActiveOutput could never have been started, so
// there is no other writer to interleave with.
func (m *terminalMux) enqueue(chunk ptyChunk) {
	if m.output == nil {
		m.handleChunk(chunk)
		return
	}
	select {
	case m.output <- chunk:
	case <-m.done:
	}
}

// enqueueAndWait posts an event and blocks until the loop has handled it. Only
// for tests, which is why it is not what enqueue does -- see the deadlock above.
func (m *terminalMux) enqueueAndWait(chunk ptyChunk) {
	if m.output == nil {
		m.handleChunk(chunk)
		return
	}
	done := make(chan struct{})
	chunk.drained = done
	select {
	case m.output <- chunk:
	case <-m.done:
		return
	}
	select {
	case <-done:
	case <-m.done:
	}
}

// drainForTest waits until every event queued so far has been handled.
//
// A BARE event, deliberately: an earlier version enqueued `own: []byte{}` and
// the empty write took the owed slot, so the drain silently destroyed the paint
// the test was about to assert. A probe must not perturb what it observes.
func (m *terminalMux) drainForTest() { m.enqueueAndWait(ptyChunk{}) }

// midSequenceForTest reports the gate's state from the writer goroutine.
func (m *terminalMux) midSequenceForTest() bool {
	var mid bool
	if m.output == nil {
		return m.hostScan.MidSequence()
	}
	done := make(chan struct{})
	select {
	case m.output <- ptyChunk{drained: done, onWriter: func() { mid = m.hostScan.MidSequence() }}:
	case <-m.done:
		return false
	}
	<-done
	return mid
}

func (m *terminalMux) isActive(id int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active < 0 || m.active >= len(m.tabs) {
		return false
	}
	return m.tabs[m.active].id == id
}

func (m *terminalMux) writeActive(data []byte) {
	m.mu.Lock()
	tab := m.activeTabLocked()
	m.mu.Unlock()
	if tab != nil && tab.child != nil {
		_, _ = tab.child.Write(data)
	}
}

func (m *terminalMux) closeActive() {
	m.mu.Lock()
	if len(m.tabs) <= 1 {
		m.mu.Unlock()
		return
	}
	tab := m.activeTabLocked()
	m.mu.Unlock()
	if tab == nil || tab.child == nil {
		return
	}
	// Close kills and lets the child's own pump reap; calling Wait here too
	// would be a second Wait on the same process.
	_ = tab.child.Close()
}

func (m *terminalMux) beginRename() (int, RenameEditor, error) {
	m.mu.Lock()
	tab := m.activeTabLocked()
	if tab == nil {
		m.mu.Unlock()
		return 0, RenameEditor{}, fmt.Errorf("rename terminal tab: no active tab")
	}
	editor := NewRenameEditor(tab.name)
	tabID := tab.id
	m.rename = &activeRename{tabID: tabID, editor: editor}
	title := m.renamePaneTitleLocked(tabID, editor)
	m.mu.Unlock()
	if err := m.setPaneTitle(title); err != nil {
		m.mu.Lock()
		if m.rename != nil && m.rename.tabID == tabID {
			m.rename = nil
		}
		m.mu.Unlock()
		return 0, RenameEditor{}, fmt.Errorf("start terminal tab rename: %w", err)
	}
	return tabID, editor, nil
}

func (m *terminalMux) refreshRename(tabID int, editor RenameEditor) error {
	m.mu.Lock()
	m.rename = &activeRename{tabID: tabID, editor: editor}
	title := m.renamePaneTitleLocked(tabID, editor)
	m.mu.Unlock()
	if err := m.setPaneTitle(title); err != nil {
		return fmt.Errorf("refresh terminal tab rename: %w", err)
	}
	return nil
}

func (m *terminalMux) finishRename(tabID int, outcome RenameOutcome) error {
	m.mu.Lock()
	if outcome.Kind == RenameOutcomeCommit {
		if tab := m.tabByIDLocked(tabID); tab != nil {
			tab.name = outcome.Name
		}
	}
	m.rename = nil
	title := m.paneTitleLocked()
	m.mu.Unlock()
	if err := m.setPaneTitle(title); err != nil {
		return fmt.Errorf("finish terminal tab rename: %w", err)
	}
	return nil
}

func (m *terminalMux) previousTab() {
	m.switchRelative(-1)
}

func (m *terminalMux) nextTab() {
	m.switchRelative(1)
}

func (m *terminalMux) switchRelative(delta int) {
	m.mu.Lock()
	if len(m.tabs) == 0 {
		m.mu.Unlock()
		return
	}
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	snapshot := replaySnapshotLocked(m.activeTabLocked())
	m.mu.Unlock()
	m.renamePane()
	m.redrawTab(snapshot)
}

func (m *terminalMux) appMouseMode() bool {
	m.mu.Lock()
	tab := m.activeTabLocked()
	m.mu.Unlock()
	return tab != nil && tab.child != nil && tab.child.Mouse()
}

// removeTab runs ON THE WRITER GOROUTINE -- its only caller is handleChunk, on
// a child's EOF -- so it must never post to the channel that goroutine drains.
//
// It did, via redrawTab, and that is a deadlock rather than a slow path: with
// the buffer full (a child exiting while its output is backed up, which is
// exactly when a child exits under load) the send blocks forever, because the
// only goroutine that could drain it is the one blocked in the send. The pane
// wedges permanently.
//
// THE RULE: a handler running on the writer goroutine applies its own writes
// INLINE. Anything that posts belongs to a caller that is not the loop.
func (m *terminalMux) removeTab(id int) {
	m.mu.Lock()
	var removed *terminalTab
	var active *terminalTab
	var activeSnapshot []byte
	empty := false
	activeID := 0
	title := ""
	preserveRename := false
	if tab := m.activeTabLocked(); tab != nil {
		activeID = tab.id
	}
	for i, tab := range m.tabs {
		if tab.id != id {
			continue
		}
		removed = tab
		m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
		if len(m.tabs) == 0 {
			empty = true
		} else {
			activeFound := false
			for j, remaining := range m.tabs {
				if remaining.id == activeID {
					m.active = j
					activeFound = true
					break
				}
			}
			if !activeFound && m.active >= len(m.tabs) {
				m.active = len(m.tabs) - 1
			}
		}
		active = m.activeTabLocked()
		activeSnapshot = replaySnapshotLocked(active)
		if m.rename != nil {
			title = m.renamePaneTitleLocked(m.rename.tabID, m.rename.editor)
			preserveRename = true
		} else {
			title = m.paneTitleLocked()
		}
		break
	}
	m.mu.Unlock()
	if removed == nil {
		return
	}
	if removed.child != nil {
		_ = removed.child.Close()
	}
	if empty {
		close(m.done)
		return
	}
	_ = m.setPaneTitle(title)
	if !preserveRename {
		m.applyTakeover(activeSnapshot)
	}
}

func (m *terminalMux) activeTabLocked() *terminalTab {
	if m.active < 0 || m.active >= len(m.tabs) {
		return nil
	}
	return m.tabs[m.active]
}

func (m *terminalMux) tabByIDLocked(id int) *terminalTab {
	for _, tab := range m.tabs {
		if tab.id == id {
			return tab
		}
	}
	return nil
}

// resizeThroughWriter runs the resize ON the writer goroutine.
//
// inheritSize does not write the pane today, but it becomes a writer in M3 (the
// strip is repainted on resize), and ARCH-ORDER already asserts that resize and
// paint "serialize by construction" on the writer loop. Routing it here makes
// that claim true now rather than at the milestone that depends on it -- and a
// resize racing a paint is precisely the interleaving that produces a strip
// drawn at the old width.
func (m *terminalMux) resizeThroughWriter(host hostty.Host) {
	m.enqueue(ptyChunk{onWriter: func() {
		// captureIDForTest lets a test observe WHICH goroutine this runs on,
		// through the PRODUCTION entry point. Reverting this method to a direct
		// inheritSize call was measured green while the test drove an
		// injectable helper instead -- testing the seam is not testing the path.
		if m.captureIDForTest != nil {
			m.captureIDForTest()
		}
		m.inheritSize(host)
	}})
}

func (m *terminalMux) inheritSize(host hostty.Host) {
	m.captureSize(host)
	m.mu.Lock()
	childSize := m.childSizeLocked()
	m.mu.Unlock()
	m.resizeAll(childSize)
	// The region and the row's width both changed. Inline: every caller of this
	// is either pre-loop (the initial sizing) or already on it (the resize
	// posts through resizeThroughWriter).
	m.paintStripInline()
}

func (m *terminalMux) captureSize(host hostty.Host) {
	size, err := host.Size()
	if err != nil {
		return
	}
	m.mu.Lock()
	m.rows = size.Rows
	m.cols = size.Cols
	m.mu.Unlock()
}

// childSizeLocked is the size a tab gets. `pair term` gives its children the
// whole terminal; couch subtracts a row here. That difference is the policy
// each caller keeps.
// childSizeLocked is the pane MINUS the row the strip occupies.
//
// NewReservation is the validating door: it refuses a pane too short to reserve
// from, and the child then gets the whole thing with no strip drawn -- a
// zero-row pty is not a thing, and a pane that short has no room for chrome
// anyway.
func (m *terminalMux) childSizeLocked() ptychild.Size {
	res, err := hostty.NewReservation(m.rows, hostty.EdgeBottom)
	if err != nil {
		return ptychild.Size{Rows: m.rows, Cols: m.cols}
	}
	return ptychild.Size{Rows: res.ChildRows(), Cols: m.cols}
}

// reservationLocked is the strip's row. Same validating door; the zero value
// draws nothing, which is what a too-short pane should do.
func (m *terminalMux) reservationLocked() hostty.Reservation {
	res, err := hostty.NewReservation(m.rows, hostty.EdgeBottom)
	if err != nil {
		return hostty.Reservation{}
	}
	return res
}

// stripModelLocked snapshots what the row should say.
func (m *terminalMux) stripModelLocked() StripModel {
	tabs := make([]TabChip, 0, len(m.tabs))
	for _, t := range m.tabs {
		tabs = append(tabs, TabChip{Name: t.name})
	}
	return StripModel{Tabs: tabs, Active: m.active}
}

// paintStrip renders the row and posts it through the writer loop.
//
// Re-Reserve BEFORE painting, every time. Not belt-and-braces: a child that
// reset margins (nvim does, on startup and on quit) dropped the region a moment
// ago, and painting into an unreserved screen puts the row where the child's
// content belongs. couch learned this the same way.
func (m *terminalMux) paintStrip() {
	if b := m.stripBytes(); b != nil {
		m.paintOwn(b)
	}
}

// paintStripInline is the same paint for a caller ALREADY on the writer
// goroutine. A handler on the loop applies its writes inline rather than
// posting to the channel it drains -- the rule BR-25 cost this milestone a
// wedged pane to learn.
func (m *terminalMux) paintStripInline() {
	if b := m.stripBytes(); b != nil {
		m.writeOwn(b)
	}
}

// stripBytes renders the row, region first. Nil when there is no room.
func (m *terminalMux) stripBytes() []byte {
	m.mu.Lock()
	res := m.reservationLocked()
	model := m.stripModelLocked()
	cols := int(m.cols)
	m.mu.Unlock()

	if res.Rows == 0 || cols <= 0 {
		return nil
	}
	return []byte(res.Reserve() + res.Paint(RenderStrip(cols, model).Body))
}

func (m *terminalMux) resizeAll(size ptychild.Size) {
	if size.Rows == 0 || size.Cols == 0 {
		return
	}
	m.mu.Lock()
	tabs := append([]*terminalTab(nil), m.tabs...)
	m.mu.Unlock()
	for _, tab := range tabs {
		if tab.child != nil {
			_ = tab.child.Resize(size)
		}
	}
}

func (m *terminalMux) closeAll() {
	m.mu.Lock()
	tabs := append([]*terminalTab(nil), m.tabs...)
	m.mu.Unlock()
	for _, tab := range tabs {
		if tab.child != nil {
			_ = tab.child.Close()
		}
	}
}

func (m *terminalMux) renamePane() {
	m.mu.Lock()
	title := m.paneTitleLocked()
	m.mu.Unlock()
	if title == "" {
		return
	}
	_ = m.setPaneTitle(title)
}

func (m *terminalMux) setPaneTitle(title string) error {
	if m.paneID != "" {
		return m.rt.RunZellijAction("rename-pane", "--pane-id", m.paneID, title)
	}
	return m.rt.RunZellijAction("rename-pane", title)
}

// paneTitleLocked is the DEGRADED title: the active tab's name, nothing else.
//
// It used to pack every tab into one rename argument -- `one [work] three` --
// because a rename was the only channel tab state had. The strip is that channel
// now (#199 M3), so the title goes back to being a label.
//
// It still exists, and is still kept current, because the zellij pane title is
// the only label visible when the pane is NOT focused, and two consumers read
// it. Derived rather than remembered (finding 9):
//
//	grep -rn "\.Title" cmd --include=*.go | grep -v _test.go
//
//	launcher/layoutflow.go:62         Title == "terminal" | HasPrefix "[terminal"
//	workbenchshortcut/shortcut.go:189 title == "terminal" | HasPrefix "terminal "
//
// Both arms also match on the pane's COMMAND (`pair term`), so neither depends
// on the title alone -- which is what makes degrading it safe, and is a
// measurement rather than a hope. Note the two disagree about the bracket form,
// so the packed title matched shortcut.go's arm NEVER: that consumer has always
// classified this pane by command.
func (m *terminalMux) paneTitleLocked() string {
	if len(m.tabs) == 0 {
		return ""
	}
	name := m.tabs[0].name
	if m.active >= 0 && m.active < len(m.tabs) {
		name = m.tabs[m.active].name
	}
	// The `terminal ` prefix is LOAD-BEARING, not decoration.
	//
	// `RoleForPane` classifies by `HasPrefix(title, "terminal ")` when the
	// pane's command is unavailable, and that classification is what routes the
	// operator's global shortcuts. A bare active-tab name -- `work` -- matches
	// nothing, so renaming a tab would silently cost that pane its keybindings.
	// The packed form happened to keep classifying only because it began with
	// the FIRST tab's default name; degrading without the prefix trades one
	// accident for a worse one.
	if strings.HasPrefix(name, "terminal") {
		return name
	}
	return "terminal " + name
}

func (m *terminalMux) renamePaneTitleLocked(tabID int, editor RenameEditor) string {
	if len(m.tabs) == 0 {
		return ""
	}
	text := []rune(editor.Text())
	cursor := editor.Cursor()
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(text) {
		cursor = len(text)
	}
	field := string(text[:cursor]) + "│" + string(text[cursor:])
	parts := make([]string, 0, len(m.tabs))
	found := false
	for _, tab := range m.tabs {
		if tab.id == tabID {
			found = true
			parts = append(parts, "[rename: "+field+"]")
		} else {
			parts = append(parts, tab.name)
		}
	}
	if !found {
		parts = append(parts, "[rename: "+field+"]")
	}
	return strings.Join(parts, " ")
}

// redrawTab repaints from a SNAPSHOT taken by the caller under m.mu — it never
// touches the mutex itself. Callers already hold the lock immediately before
// calling, so snapshotting there costs nothing and avoids inventing a "no caller
// may hold m.mu" contract whose violation mode would be a deadlock.
//
// redrawTab repaints from a REPLAY taken by the caller under m.mu.
//
// The query stripping lives in ptychild.Child.Replay, NOT here. It used to be
// composed at this site while Replay went uncalled -- two places holding one
// decision about what a repaint may contain, and couch's attach path in M3
// would have made it three (BR-20).
// redrawTab is the WHOLESALE TAKEOVER: clear the screen, replay the tab. It
// goes through the writer loop like everything else -- before M2 it wrote from
// the Run goroutine while the pump wrote from its own, which is exactly the two
// writers this milestone removes.
func (m *terminalMux) redrawTab(replay []byte) {
	m.enqueue(ptyChunk{replay: replay, takeover: true})
}

// replaySnapshotLocked is what a repaint of this tab should write. Caller must
// hold m.mu — the child's pump appends concurrently.
func replaySnapshotLocked(tab *terminalTab) []byte {
	if tab == nil || tab.child == nil {
		return nil
	}
	return tab.child.Replay()
}

func (m *terminalMux) restoreTerminal() {
	// Teardown writes DIRECTLY: the loop may already be gone, and a
	// half-restored terminal is worse than an un-gated write when the child is
	// finished with the screen anyway. Same reasoning as couch's release().
	m.pane.rawString("teardown: the loop may already be gone", hostty.ResetRegion)
}

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

// runZellijCaptured runs the action with BOTH descriptors captured, and folds
// what the subprocess said into the returned error.
//
// Keeping the pane clean must not mean destroying the diagnostic. The first
// version of this milestone sent both descriptors to io.Discard, and since the
// wheel-tick callers also drop the error (`_ = rt.RunZellijAction("scroll-up")`),
// a failing action became completely silent -- strictly worse than the noise it
// replaced. The bytes go into the error, where a caller can report or log them,
// and never onto the pane's fd.
// diagnosticWidth bounds what any single diagnostic may put on the pane. Wide
// enough for a useful message, far short of wrapping onto the child's area.
const diagnosticWidth = 200

// zellijErrorDetailWidth bounds the subprocess half specifically, so a usage
// dump cannot crowd out the part of the message we wrote.
const zellijErrorDetailWidth = 120

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
