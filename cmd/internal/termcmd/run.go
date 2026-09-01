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
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/ptychild"
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
	ReportShortcutError(error)
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
				mux.inheritSize(host)
			}
		}()
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
					rt.ReportShortcutError(err)
				}
				timer.StopAndDrain()
				rename = nil
				return
			}
			if err := mux.refreshRename(rename.tabID, rename.editor); err != nil {
				rt.ReportShortcutError(err)
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
								rt.ReportShortcutError(err)
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
								rt.ReportShortcutError(err)
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
						case event.release:
							mux.writeActive(rawMouse)
						case event.button == 64:
							if mux.appMouseMode() {
								mux.writeActive(rawMouse)
							} else {
								_ = rt.RunZellijAction("scroll-up")
							}
						case event.button == 65:
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
			rt.ReportShortcutError(err)
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

// An SGR (1006) mouse event is "\x1b[<button;col;rowT" where T is 'M' for a
// press and 'm' for a RELEASE. Both terminators must be recognized: treating
// 'm' as "sequence not finished yet" parks the release — and then every
// keystroke behind it — in pumpStdin's `held` buffer, which reads as a dead
// keyboard, and leaves the child app holding an unmatched button-press (nvim
// stays in a mouse drag, i.e. stuck in visual selection).
const sgrMouseTerminators = "Mm"

type mousePressEvent struct {
	button  int
	x       int
	y       int
	release bool
}

func parseSGRMousePress(data []byte) (mousePressEvent, bool) {
	s := string(data)
	if !strings.HasPrefix(s, "\x1b[<") || s == "" {
		return mousePressEvent{}, false
	}
	term := s[len(s)-1:]
	if !strings.Contains(sgrMouseTerminators, term) {
		return mousePressEvent{}, false
	}
	var event mousePressEvent
	if _, err := fmt.Sscanf(s, "\x1b[<%d;%d;%d"+term, &event.button, &event.x, &event.y); err != nil {
		return mousePressEvent{}, false
	}
	event.release = term == "m"
	return event, true
}

func parseSGRMousePressPrefix(data []byte) (mousePressEvent, []byte, []byte, bool) {
	if !bytes.HasPrefix(data, []byte("\x1b[<")) {
		return mousePressEvent{}, nil, data, false
	}
	end := bytes.IndexAny(data, sgrMouseTerminators)
	if end < 0 {
		return mousePressEvent{}, nil, data, false
	}
	raw := data[:end+1]
	event, ok := parseSGRMousePress(raw)
	if !ok {
		return mousePressEvent{}, nil, data, false
	}
	return event, raw, data[end+1:], true
}

func findSGRMousePress(data []byte) ([]byte, mousePressEvent, []byte, []byte, bool) {
	start := bytes.Index(data, []byte("\x1b[<"))
	if start < 0 {
		return data, mousePressEvent{}, nil, nil, false
	}
	event, raw, rest, ok := parseSGRMousePressPrefix(data[start:])
	if !ok {
		return data, mousePressEvent{}, nil, nil, false
	}
	return data[:start], event, raw, rest, true
}

func isSGRMousePrefix(data []byte) bool {
	return bytes.HasPrefix([]byte("\x1b[<"), data) ||
		(bytes.HasPrefix(data, []byte("\x1b[<")) && bytes.IndexAny(data, sgrMouseTerminators) < 0)
}

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

type ptyChunk struct {
	id   int
	data []byte
	err  error
}

type terminalMux struct {
	mu        sync.Mutex
	shellName string
	shellArgs []string
	stdout    io.Writer
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
}

type activeRename struct {
	tabID  int
	editor RenameEditor
}

func newTerminalMux(shellName string, shellArgs []string, stdout, stderr io.Writer, rt Runtime) *terminalMux {
	return &terminalMux{
		shellName: shellName,
		shellArgs: shellArgs,
		stdout:    stdout,
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
			m.output <- ptyChunk{id: id, data: batch.Raw}
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

func (m *terminalMux) copyActiveOutput() {
	for {
		select {
		case chunk := <-m.output:
			if chunk.err != nil {
				m.removeTab(chunk.id)
				continue
			}
			// No buffering here any more: ptychild.Child appends to its own
			// ring BEFORE the sink runs, so a switch racing a chunk still
			// repaints a current screen.
			if m.isActive(chunk.id) {
				_, _ = m.stdout.Write(chunk.data)
			}
		case <-m.done:
			return
		}
	}
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
		m.redrawTab(activeSnapshot)
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

func (m *terminalMux) inheritSize(host hostty.Host) {
	m.captureSize(host)
	m.mu.Lock()
	childSize := m.childSizeLocked()
	m.mu.Unlock()
	m.resizeAll(childSize)
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
func (m *terminalMux) childSizeLocked() ptychild.Size {
	return ptychild.Size{Rows: m.rows, Cols: m.cols}
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

func (m *terminalMux) paneTitleLocked() string {
	if len(m.tabs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m.tabs))
	for i, tab := range m.tabs {
		if i == m.active {
			parts = append(parts, "["+tab.name+"]")
		} else {
			parts = append(parts, tab.name)
		}
	}
	return strings.Join(parts, " ")
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
func (m *terminalMux) redrawTab(replay []byte) {
	_, _ = io.WriteString(m.stdout, hostty.HomeAndClear)
	_, _ = m.stdout.Write(replay)
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
	_, _ = io.WriteString(m.stdout, hostty.ResetRegion)
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

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return runZellij(cmdArgs, os.Stdout)
}

func (OSRuntime) RunZellijActionQuiet(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return runZellij(cmdArgs, io.Discard)
}

func (OSRuntime) ReportShortcutError(err error) {
	fmt.Fprintf(os.Stderr, "pair term: global shortcut: %v\n", err)
}

func runZellij(args []string, stdout io.Writer) error {
	cmd := exec.Command("zellij", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (OSRuntime) ShellCommand() (string, []string) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, []string{"-i"}
}
