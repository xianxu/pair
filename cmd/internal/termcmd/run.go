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
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/draftroute"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"golang.org/x/term"
)

type Runtime interface {
	CachedDraftPaneID() (string, bool)
	ListPanesJSON() ([]byte, error)
	LastLeftPaneID() (string, error)
	RecordLastLeftPaneID(string) error
	RunZellijAction(args ...string) error
	ReportShortcutError(error)
	ShellCommand() (string, []string)
}

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
	decision := workbenchshortcut.Decide(workbenchshortcut.ShortcutInput{
		Role:           workbenchshortcut.RoleForPane(panes.focused),
		Chord:          chord,
		FocusedPaneID:  panes.focused.ID,
		LastLeftPaneID: lastLeft,
		DraftPaneID:    panes.draft.ID,
	})
	return runDecision(decision, panes, rt, stdin, stdout)
}

type workbenchPanes struct {
	focused  zellijpane.Pane
	draft    zellijpane.Pane
	terminal zellijpane.Pane
}

func focusedWorkbenchPanes(rt Runtime) (workbenchPanes, error) {
	data, err := rt.ListPanesJSON()
	if err != nil {
		return workbenchPanes{}, err
	}
	var out workbenchPanes
	for _, pane := range zellijpane.Parse(data) {
		if pane.IsFocused {
			out.focused = pane
		}
		switch workbenchshortcut.RoleForPane(pane) {
		case workbenchshortcut.PaneRoleLeftDraft:
			out.draft = pane
		case workbenchshortcut.PaneRoleRightTerminal:
			out.terminal = pane
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
		if panes.terminal.ID == "" {
			return nil
		}
		return rt.RunZellijAction("focus-pane-id", panes.terminal.ID)
	case workbenchshortcut.ActionToggleFocusedLayout:
		if layoutcmd.RunToggleFocused(nil, rt, io.Discard) != 0 {
			return fmt.Errorf("toggle focused layout failed")
		}
		return nil
	default:
		return nil
	}
}

func runShell(stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	name, args := rt.ShellCommand()
	_ = layoutcmd.AlignFloatingTerminal(rt)
	stdinFile, _ := stdin.(*os.File)
	var oldState *term.State
	if stdinFile != nil {
		s, err := term.MakeRaw(int(stdinFile.Fd()))
		if err != nil {
			fmt.Fprintf(stderr, "term: MakeRaw: %v\n", err)
			return 1
		}
		oldState = s
		defer func() { _ = term.Restore(int(stdinFile.Fd()), oldState) }()
	}

	mux := newTerminalMux(name, args, stdout, stderr, rt)
	if stdinFile != nil {
		mux.captureSize(stdinFile)
	}
	if err := mux.newTab(); err != nil {
		fmt.Fprintf(stderr, "term: %v\n", err)
		return 1
	}
	defer mux.closeAll()
	defer mux.restoreTerminal()

	if stdinFile != nil {
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		defer signal.Stop(winch)
		go func() {
			for range winch {
				mux.inheritSize(stdinFile)
			}
		}()
		winch <- syscall.SIGWINCH
	}

	go pumpStdin(stdin, mux, rt, stdout)
	mux.copyActiveOutput()

	if stdinFile != nil && oldState != nil {
		_ = term.Restore(int(stdinFile.Fd()), oldState)
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
						switch event.button {
						case 64:
							if mux.appMouseMode() {
								mux.writeActive(rawMouse)
							} else {
								_ = rt.RunZellijAction("scroll-up")
							}
						case 65:
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
	case workbenchshortcut.ChordAltShiftEnter:
		_ = layoutcmd.RunToggleFocused(nil, rt, io.Discard)
		return true
	default:
		return false
	}
}

type mousePressEvent struct {
	button int
	x      int
	y      int
}

func parseSGRMousePress(data []byte) (mousePressEvent, bool) {
	s := string(data)
	if !strings.HasPrefix(s, "\x1b[<") || !strings.HasSuffix(s, "M") {
		return mousePressEvent{}, false
	}
	var event mousePressEvent
	if _, err := fmt.Sscanf(s, "\x1b[<%d;%d;%dM", &event.button, &event.x, &event.y); err != nil {
		return mousePressEvent{}, false
	}
	return event, true
}

func parseSGRMousePressPrefix(data []byte) (mousePressEvent, []byte, []byte, bool) {
	if !bytes.HasPrefix(data, []byte("\x1b[<")) {
		return mousePressEvent{}, nil, data, false
	}
	end := bytes.IndexByte(data, 'M')
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
		(bytes.HasPrefix(data, []byte("\x1b[<")) && bytes.IndexByte(data, 'M') < 0)
}

type OSRuntime struct{}

type terminalTab struct {
	id     int
	name   string
	cmd    *exec.Cmd
	pty    *os.File
	buffer []byte
	mouse  bool
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

	cmd := exec.Command(m.shellName, m.shellArgs...)
	cmd.Env = os.Environ()
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("pty.Start: %w", err)
	}
	m.applyPTYSize(ptmx)
	tab := &terminalTab{id: id, name: name, cmd: cmd, pty: ptmx}

	m.mu.Lock()
	m.tabs = append(m.tabs, tab)
	m.active = len(m.tabs) - 1
	m.mu.Unlock()
	m.renamePane()
	m.redrawTab(tab)

	go m.readPTY(tab)
	return nil
}

func (m *terminalMux) readPTY(tab *terminalTab) {
	buf := make([]byte, 4096)
	for {
		n, err := tab.pty.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			m.output <- ptyChunk{id: tab.id, data: chunk}
		}
		if err != nil {
			m.output <- ptyChunk{id: tab.id, err: err}
			return
		}
	}
}

func (m *terminalMux) copyActiveOutput() {
	for {
		select {
		case chunk := <-m.output:
			if chunk.err != nil {
				m.removeTab(chunk.id)
				continue
			}
			m.appendBuffer(chunk.id, chunk.data)
			if m.isActive(chunk.id) {
				_, _ = m.stdout.Write(chunk.data)
			}
		case <-m.done:
			return
		}
	}
}

func (m *terminalMux) appendBuffer(id int, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tab := range m.tabs {
		if tab.id != id {
			continue
		}
		tab.buffer = append(tab.buffer, data...)
		if len(tab.buffer) > 128*1024 {
			tab.buffer = tab.buffer[len(tab.buffer)-128*1024:]
		}
		tab.mouse = updateMouseMode(tab.mouse, data)
		return
	}
}

func updateMouseMode(current bool, data []byte) bool {
	s := string(data)
	for {
		idx := strings.Index(s, "\x1b[?")
		if idx < 0 {
			return current
		}
		s = s[idx+3:]
		end := strings.IndexAny(s, "hl")
		if end < 0 {
			return current
		}
		mode := s[end]
		params := s[:end]
		for _, param := range strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' }) {
			switch param {
			case "1000", "1002", "1003", "1006":
				current = mode == 'h'
			}
		}
		s = s[end+1:]
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
	if tab != nil {
		_, _ = tab.pty.Write(data)
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
	if tab == nil {
		return
	}
	_ = tab.pty.Close()
	_ = tab.cmd.Process.Kill()
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
	tab := m.activeTabLocked()
	m.mu.Unlock()
	m.renamePane()
	m.redrawTab(tab)
}

func (m *terminalMux) appMouseMode() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	tab := m.activeTabLocked()
	return tab != nil && tab.mouse
}

func (m *terminalMux) removeTab(id int) {
	m.mu.Lock()
	var removed *terminalTab
	var active *terminalTab
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
	_ = removed.pty.Close()
	_ = removed.cmd.Wait()
	if empty {
		close(m.done)
		return
	}
	_ = m.setPaneTitle(title)
	if !preserveRename {
		m.redrawTab(active)
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

func (m *terminalMux) inheritSize(stdinFile *os.File) {
	m.captureSize(stdinFile)
	m.mu.Lock()
	childSize := m.childSizeLocked()
	m.mu.Unlock()
	m.resizeAll(childSize)
}

func (m *terminalMux) captureSize(stdinFile *os.File) {
	size, err := pty.GetsizeFull(stdinFile)
	if err != nil || size == nil {
		return
	}
	m.mu.Lock()
	m.rows = size.Rows
	m.cols = size.Cols
	m.mu.Unlock()
}

func (m *terminalMux) childSizeLocked() *pty.Winsize {
	if m.rows == 0 || m.cols == 0 {
		return nil
	}
	return &pty.Winsize{Rows: m.rows, Cols: m.cols}
}

func (m *terminalMux) applyPTYSize(f *os.File) {
	m.mu.Lock()
	size := m.childSizeLocked()
	m.mu.Unlock()
	if size != nil {
		_ = pty.Setsize(f, size)
	}
}

func (m *terminalMux) resizeAll(size *pty.Winsize) {
	m.mu.Lock()
	tabs := append([]*terminalTab(nil), m.tabs...)
	m.mu.Unlock()
	for _, tab := range tabs {
		if size != nil {
			_ = pty.Setsize(tab.pty, size)
		}
	}
}

func (m *terminalMux) closeAll() {
	m.mu.Lock()
	tabs := append([]*terminalTab(nil), m.tabs...)
	m.mu.Unlock()
	for _, tab := range tabs {
		_ = tab.pty.Close()
		if tab.cmd.Process != nil {
			_ = tab.cmd.Process.Kill()
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

func (m *terminalMux) redrawTab(tab *terminalTab) {
	if tab == nil {
		return
	}
	_, _ = io.WriteString(m.stdout, "\x1b[1;1H\x1b[J")
	_, _ = m.stdout.Write(tab.buffer)
}

func (m *terminalMux) restoreTerminal() {
	_, _ = io.WriteString(m.stdout, "\x1b[r")
}

func (OSRuntime) ListPanesJSON() ([]byte, error) {
	return exec.Command("zellij", "action", "list-panes", "--json", "--command", "--state").Output()
}

func (OSRuntime) CachedDraftPaneID() (string, bool) {
	return draftroute.CachedDraftPaneIDFromEnv()
}

func (OSRuntime) LastLeftPaneID() (string, error) {
	store := workbenchshortcut.LastLeftPaneStore{DataDir: pairDataDir(), Tag: os.Getenv("PAIR_TAG")}
	return store.Read()
}

func (OSRuntime) RecordLastLeftPaneID(id string) error {
	store := workbenchshortcut.LastLeftPaneStore{DataDir: pairDataDir(), Tag: os.Getenv("PAIR_TAG")}
	return store.Write(id)
}

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return runZellij(cmdArgs...)
}

func (OSRuntime) ReportShortcutError(err error) {
	fmt.Fprintf(os.Stderr, "pair term: global shortcut: %v\n", err)
}

func runZellij(args ...string) error {
	cmd := exec.Command("zellij", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
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

func pairDataDir() string {
	if v := os.Getenv("PAIR_DATA_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v + "/pair"
	}
	if v := os.Getenv("HOME"); v != "" {
		return v + "/.local/share/pair"
	}
	return "."
}
