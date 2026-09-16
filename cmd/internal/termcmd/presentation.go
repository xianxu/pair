package termcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// muxMutex serializes product changes; canceled child publication can leave
// admission without spawning a detached lock waiter.
type muxMutex struct {
	once  sync.Once
	token chan struct{}
}

func (m *muxMutex) LockContext(ctx context.Context) error {
	m.once.Do(func() { m.token = make(chan struct{}, 1) })
	select {
	case m.token <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (m *muxMutex) Lock()   { _ = m.LockContext(context.Background()) }
func (m *muxMutex) Unlock() { <-m.token }

type terminalTab struct {
	id    int
	name  string
	child *ptychild.Child
}
type activeRename struct {
	tabID  int
	editor RenameEditor
}
type terminalMux struct {
	mu             muxMutex
	shellName      string
	shellArgs      []string
	shellEnv       []string
	closeErr       error
	presenter      *terminal.Presenter
	rt             Runtime
	paneID         string
	tabs           []*terminalTab
	active, nextID int
	rows, cols     uint16
	rename         *activeRename
	notice         string
	failure        error
	closing        bool
	done           chan struct{}
	doneOnce       sync.Once
	closeOnce      sync.Once
	workers        sync.WaitGroup
}

func newTerminalMux(shell string, args []string, parent ttyio.Writer, rt Runtime) *terminalMux {
	return &terminalMux{shellName: shell, shellArgs: args, presenter: terminal.NewPresenter(parent, terminal.ChildRequested), rt: rt, paneID: os.Getenv("ZELLIJ_PANE_ID"), active: -1, rows: 24, cols: 80, done: make(chan struct{})}
}
func (m *terminalMux) stopLocked(err error) {
	m.failure = errors.Join(m.failure, err)
	m.doneOnce.Do(func() { close(m.done) })
}
func (m *terminalMux) childSizeLocked() ptychild.Size {
	rows := m.rows
	if rows > 1 {
		rows--
	}
	return ptychild.Size{Rows: rows, Cols: m.cols}
}
func (m *terminalMux) chromeLocked(active int) ([]terminal.Cell, error) {
	return m.chromeForGeometryLocked(active, m.rows, m.cols)
}
func (m *terminalMux) chromeForGeometryLocked(active int, rows, cols uint16) ([]terminal.Cell, error) {
	if rows < 2 {
		return nil, nil
	}
	model := m.stripModelLocked()
	model.Active = active
	text := RenderStrip(int(cols), model).Body
	if m.notice != "" {
		text = m.notice
	}
	return terminal.StyledRows(text, int(cols), 1)
}
func (m *terminalMux) selectLocked(index int) error {
	if index < 0 || index >= len(m.tabs) || m.tabs[index].child == nil {
		return errors.New("term: missing tab endpoint")
	}
	chrome, err := m.chromeLocked(index)
	if err != nil {
		return err
	}
	if err = m.presenter.Select(context.Background(), m.tabs[index].child.Endpoint(), terminal.Geometry{Cols: int(m.cols), Rows: int(m.rows)}, chrome); err != nil {
		return err
	}
	m.active = index
	return nil
}
func (m *terminalMux) admitTab(tab *terminalTab) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing {
		return errors.New("term: closing")
	}
	m.tabs = append(m.tabs, tab)
	if err := m.selectLocked(len(m.tabs) - 1); err != nil {
		m.tabs = m.tabs[:len(m.tabs)-1]
		m.stopLocked(err)
		return err
	}
	return nil
}
func (m *terminalMux) newTab() error {
	m.mu.Lock()
	if m.closing {
		m.mu.Unlock()
		return errors.New("term: closing")
	}
	m.workers.Add(1)
	handedOff := false
	defer func() {
		if !handedOff {
			m.workers.Done()
		}
	}()
	m.nextID++
	id := m.nextID
	size := m.childSizeLocked()
	m.mu.Unlock()
	ready := make(chan struct{})
	child, err := ptychild.Start(ptychild.Options{Argv: append([]string{m.shellName}, m.shellArgs...), Size: size, Env: append([]string(nil), m.shellEnv...), Sink: func(ctx context.Context, b ptychild.OutputBatch) error {
		select {
		case <-ready:
		case <-ctx.Done():
			return ctx.Err()
		}
		return m.handleOutput(ctx, id, b)
	}})
	if err != nil {
		return err
	}
	tab := &terminalTab{id: id, name: fmt.Sprintf("terminal %d", id), child: child}
	if err = m.admitTab(tab); err != nil {
		close(ready)
		child.Close()
		return err
	}
	close(ready)
	m.renamePane()
	handedOff = true
	go func() { defer m.workers.Done(); <-child.Exited(); m.removeTab(id) }()
	return nil
}
func (m *terminalMux) handleOutput(ctx context.Context, id int, b ptychild.OutputBatch) error {
	if err := m.mu.LockContext(ctx); err != nil {
		return err
	}
	defer m.mu.Unlock()
	if m.closing {
		return context.Canceled
	}
	tab := m.tabByIDLocked(id)
	if tab == nil {
		return nil
	}
	if b.Err != nil {
		m.stopLocked(b.Err)
		return b.Err
	}
	active := m.activeTabLocked() == tab
	if _, err := m.presenter.EmitEffects(ctx, b.Terminal.Effects, terminal.EffectPolicy{Bell: active, Title: active, Clipboard: active, Notifications: true}); err != nil {
		m.stopLocked(err)
		return err
	}
	if err := m.presenter.Present(ctx, tab.child.Endpoint()); err != nil {
		m.stopLocked(err)
		return err
	}
	return nil
}
func (m *terminalMux) writeEvent(event terminal.InputEvent) {
	m.writeEvents([]terminal.InputEvent{event})
}
func (m *terminalMux) writeEvents(events []terminal.InputEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing || m.activeTabLocked() == nil {
		return
	}
	for _, event := range events {
		if event.Reply {
			continue
		}
		if err := m.presenter.Input(context.Background(), event.Event); err != nil {
			m.stopLocked(err)
			return
		}
	}
}
func (m *terminalMux) switchRelative(delta int) {
	m.mu.Lock()
	if len(m.tabs) == 0 || m.closing {
		m.mu.Unlock()
		return
	}
	index := (m.active + delta + len(m.tabs)) % len(m.tabs)
	err := m.selectLocked(index)
	if err != nil {
		m.stopLocked(err)
	}
	m.mu.Unlock()
	if err == nil {
		m.renamePane()
	}
}
func (m *terminalMux) previousTab() { m.switchRelative(-1) }
func (m *terminalMux) nextTab()     { m.switchRelative(1) }
func (m *terminalMux) closeActive() {
	m.mu.Lock()
	tab := m.activeTabLocked()
	canClose := len(m.tabs) > 1
	m.mu.Unlock()
	if canClose && tab != nil && tab.child != nil {
		// The product owner must release gestures and retire presentation
		// before Close disposes the endpoint; the exit watcher is idempotent.
		m.removeTab(tab.id)
	}
}
func (m *terminalMux) removeTab(id int) {
	m.mu.Lock()
	if m.closing {
		m.mu.Unlock()
		return
	}
	index := -1
	for i, t := range m.tabs {
		if t.id == id {
			index = i
			break
		}
	}
	if index < 0 {
		m.mu.Unlock()
		return
	}
	removed := m.tabs[index]
	if err := m.presenter.Flush(context.Background()); err != nil {
		m.stopLocked(err)
		m.mu.Unlock()
		return
	}
	if len(m.tabs) == 1 {
		m.active = -1
		m.stopLocked(nil)
		m.mu.Unlock()
		return
	}
	selected := m.activeTabLocked()
	candidate := m.active
	if index == m.active {
		candidate = (index + 1) % len(m.tabs)
		if err := m.selectLocked(candidate); err != nil {
			m.stopLocked(err)
			m.mu.Unlock()
			return
		}
		selected = m.tabs[candidate]
	}
	m.tabs = append(m.tabs[:index], m.tabs[index+1:]...)
	for i, t := range m.tabs {
		if t == selected {
			m.active = i
			break
		}
	}
	if removed.child != nil {
		if err := m.presenter.Retire(context.Background(), removed.child.Endpoint()); err != nil {
			m.stopLocked(err)
		}
	}
	m.paintStripLocked()
	m.mu.Unlock()
	if removed.child != nil {
		removed.child.Close()
	}
	m.renamePane()
}
func (m *terminalMux) appMouseMode() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.activeTabLocked()
	return t != nil && t.child != nil && t.child.Endpoint().Modes().Tracking != 0
}
func (m *terminalMux) activeChildOwnsScreen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.activeTabLocked()
	return t != nil && t.child != nil && t.child.Endpoint().Modes().AltScreen
}
func (m *terminalMux) beginRename() (int, RenameEditor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.activeTabLocked()
	if t == nil {
		return 0, RenameEditor{}, errors.New("rename terminal tab: no active tab")
	}
	editor := NewRenameEditor(t.name)
	m.rename = &activeRename{tabID: t.id, editor: editor}
	m.paintStripLocked()
	return t.id, editor, nil
}
func (m *terminalMux) refreshRename(id int, editor RenameEditor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rename = &activeRename{tabID: id, editor: editor}
	m.paintStripLocked()
}
func (m *terminalMux) finishRename(id int, outcome RenameOutcome) error {
	m.mu.Lock()
	if outcome.Kind == RenameOutcomeCommit {
		if t := m.tabByIDLocked(id); t != nil {
			t.name = outcome.Name
		}
	}
	m.rename = nil
	m.paintStripLocked()
	title := m.paneTitleLocked()
	m.mu.Unlock()
	if err := m.setPaneTitle(title); err != nil {
		return fmt.Errorf("finish terminal tab rename: %w", err)
	}
	return nil
}
func (m *terminalMux) paintStripLocked() {
	if m.presenter == nil || m.closing || m.activeTabLocked() == nil {
		return
	}
	chrome, err := m.chromeLocked(m.active)
	if err == nil {
		err = m.presenter.UpdateChrome(context.Background(), chrome)
	}
	if err != nil {
		m.stopLocked(err)
	}
}
func (m *terminalMux) paintStrip() { m.mu.Lock(); defer m.mu.Unlock(); m.paintStripLocked() }
func (m *terminalMux) reportError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notice = rowtext.SanitizeAndFit("pair term: "+err.Error(), diagnosticWidth)
	m.paintStripLocked()
}
func (m *terminalMux) inheritSize(host hostty.Host) {
	size, err := host.Size()
	if err != nil || size.Rows == 0 || size.Cols == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing {
		return
	}
	active := m.activeTabLocked()
	if active == nil {
		return
	}
	chrome, err := m.chromeForGeometryLocked(m.active, size.Rows, size.Cols)
	if err != nil {
		m.stopLocked(err)
		return
	}
	err = m.presenter.ResizeLayout(context.Background(), terminal.Geometry{Cols: int(size.Cols), Rows: int(size.Rows)}, chrome, func(g terminal.Geometry) error {
		return active.child.ResizePTY(ptychild.Size{Rows: uint16(g.Rows), Cols: uint16(g.Cols)})
	})
	if err != nil {
		m.stopLocked(err)
		return
	}
	m.rows, m.cols = size.Rows, size.Cols
	for _, t := range m.tabs {
		if t != active && t.child != nil {
			if err := t.child.Resize(m.childSizeLocked()); err != nil {
				m.stopLocked(err)
				return
			}
		}
	}
	m.paintStripLocked()
}
func (m *terminalMux) closeAll() {
	m.closeOnce.Do(func() {
		m.mu.Lock()
		m.closing = true
		tabs := append([]*terminalTab(nil), m.tabs...)
		m.stopLocked(nil)
		m.mu.Unlock()
		if m.presenter != nil {
			m.closeErr = errors.Join(m.closeErr, m.presenter.Release(context.Background()))
		}
		for _, t := range tabs {
			if t.child != nil {
				m.closeErr = errors.Join(m.closeErr, t.child.Close())
			}
		}
		m.workers.Wait()
	})
}
func (m *terminalMux) renamePane() {
	m.mu.Lock()
	title := m.paneTitleLocked()
	m.mu.Unlock()
	if title != "" {
		_ = m.setPaneTitle(title)
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

func (m *terminalMux) stripModelLocked() StripModel {
	tabs := make([]TabChip, 0, len(m.tabs))
	for _, t := range m.tabs {
		tabs = append(tabs, TabChip{Name: t.name})
	}
	model := StripModel{Tabs: tabs, Active: m.active}
	if m.rename != nil {
		// Not found leaves Tab at -1, which the renderer treats as marking
		// nothing -- the same contract as an out-of-range Active.
		field := RenameField{Tab: -1, Text: m.rename.editor.Field()}
		for i, t := range m.tabs {
			if t.id == m.rename.tabID {
				field.Tab = i
				break
			}
		}
		model.Rename = &field
	}
	return model
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
	name := m.tabs[0].name
	if m.active >= 0 && m.active < len(m.tabs) {
		name = m.tabs[m.active].name
	}
	// The `terminal ` prefix is LOAD-BEARING, not decoration. `RoleForPane`
	// classifies by the title alone when the pane's command is unavailable, and
	// that classification routes the operator's global shortcuts -- so a bare
	// active-tab name (`work`) would silently cost the pane its keybindings.
	//
	// ASK the consumer's predicate; do not restate it. An earlier version tested
	// `HasPrefix(name, "terminal")`, which is an approximation that disagrees on
	// every name starting with `terminal` and continuing: a tab renamed
	// `terminals` produced the title `terminals`, which classifies as
	// PaneRoleOther -- the exact failure the prefix exists to prevent, delivered
	// by the code preventing it (BR-56).
	if workbenchshortcut.TitleIdentifiesRightTerminal(name) {
		return name
	}
	return "terminal " + name
}
