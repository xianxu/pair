package wrapcmd

import (
	"fmt"
	"io"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
	xansi "github.com/charmbracelet/x/ansi"
	xparser "github.com/charmbracelet/x/ansi/parser"
	"github.com/charmbracelet/x/vt"
)

const (
	terminalControlParamsMax  = 32
	terminalControlDataMax    = 64
	terminalDimensionMaxAxis  = 4096
	terminalDimensionMaxCells = 262144
)

type terminalModel struct {
	mu sync.Mutex

	emulator     *vt.Emulator
	observer     terminalControlObserver
	altScreen    bool
	synchronized bool
	replyCloser  io.Closer
	drainDone    chan struct{}
	closeDone    chan struct{}
	closed       bool
	closeErr     error
	final        terminalSnapshot
}

type terminalControlObserver struct {
	visible  bool
	parser   *xansi.Parser
	csiBytes uint8
}

type terminalSnapshot struct {
	Width, Height int
	Cursor        uv.Position
	CursorVisible bool
	AltScreen     bool
	Cells         []uv.Cell
}

func (s terminalSnapshot) CellAt(x, y int) *uv.Cell {
	if s.Width <= 0 || s.Height <= 0 || x < 0 || y < 0 || x >= s.Width || y >= s.Height || x >= len(s.Cells) {
		return nil
	}
	if y > (len(s.Cells)-1-x)/s.Width {
		return nil
	}
	return &s.Cells[y*s.Width+x]
}

func newTerminalModel(width, height int) (*terminalModel, error) {
	return newTerminalModelWithReplyCloserAssertion(width, height, func(writer io.Writer) bool {
		_, ok := writer.(io.Closer)
		return ok
	})
}

func newTerminalModelWithReplyCloserAssertion(width, height int, assertCloser func(io.Writer) bool) (*terminalModel, error) {
	if err := validateTerminalDimensions(width, height); err != nil {
		return nil, err
	}

	emulator := vt.NewEmulator(width, height)
	replyWriter := emulator.InputPipe()
	replyCloser, ok := replyWriter.(io.Closer)
	if !assertCloser(replyWriter) || !ok {
		_ = emulator.Close()
		return nil, fmt.Errorf("terminal reply writer %T does not implement io.Closer", replyWriter)
	}

	model := &terminalModel{
		emulator:     emulator,
		synchronized: true,
		replyCloser:  replyCloser,
		drainDone:    make(chan struct{}),
		closeDone:    make(chan struct{}),
	}
	go func() {
		defer close(model.drainDone)
		_, _ = io.Copy(io.Discard, emulator)
	}()
	return model, nil
}

func (m *terminalModel) Feed(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return io.ErrClosedPipe
	}
	if _, err := m.emulator.Write(data); err != nil {
		return err
	}
	m.observer.Feed(data)
	m.altScreen = m.emulator.IsAltScreen()
	return nil
}

func (m *terminalModel) Resize(cols, rows int) error {
	if err := m.PrepareResize(cols, rows); err != nil {
		return err
	}
	return m.CommitResize()
}

func (m *terminalModel) PrepareResize(cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return io.ErrClosedPipe
	}
	if err := validateTerminalDimensions(cols, rows); err != nil {
		return err
	}
	m.synchronized = false
	m.emulator.Resize(cols, rows)
	m.altScreen = m.emulator.IsAltScreen()
	return nil
}

func (m *terminalModel) CommitResize() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return io.ErrClosedPipe
	}
	m.observer.visible = false
	m.synchronized = true
	return nil
}

func validateTerminalDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("terminal dimensions must be positive: %dx%d", width, height)
	}
	if width > terminalDimensionMaxAxis || height > terminalDimensionMaxAxis {
		return fmt.Errorf("terminal dimensions exceed maximum axis %d: %dx%d", terminalDimensionMaxAxis, width, height)
	}
	if width > terminalDimensionMaxCells/height {
		return fmt.Errorf("terminal dimensions exceed maximum area %d: %dx%d", terminalDimensionMaxCells, width, height)
	}
	return nil
}

func (o *terminalControlObserver) Feed(data []byte) {
	if o.parser == nil {
		o.parser = new(xansi.Parser)
		o.parser.SetParamsSize(terminalControlParamsMax)
		o.parser.SetDataSize(terminalControlDataMax)
		o.parser.SetHandler(xansi.Handler{
			HandleCsi: o.handleCSI,
			HandleEsc: o.handleESC,
		})
	}
	for _, b := range data {
		before := o.parser.State()
		beforeCSI := terminalCSIState(before)
		if beforeCSI && o.csiBytes < 6 {
			o.csiBytes++
		}
		o.parser.Advance(b)
		afterCSI := terminalCSIState(o.parser.State())
		if !beforeCSI && afterCSI {
			o.csiBytes = 1
		} else if beforeCSI && !afterCSI {
			o.csiBytes = 0
		}
	}
}

func (o *terminalControlObserver) handleCSI(command xansi.Cmd, params xansi.Params) {
	if command.Prefix() != '?' || command.Intermediate() != 0 || (command.Final() != 'h' && command.Final() != 'l') {
		return
	}

	touchesVisibility := false
	standaloneShow := command.Final() == 'h' && len(params) == 1 && o.csiBytes == 5
	for i := range params {
		param, hasMore, ok := params.Param(i, 0)
		if !ok {
			standaloneShow = false
			continue
		}
		if param == 25 || param == 1047 || param == 1049 {
			touchesVisibility = true
		}
		if i != 0 || param != 25 || hasMore {
			standaloneShow = false
		}
	}
	if standaloneShow {
		o.visible = true
	} else if touchesVisibility {
		o.visible = false
	}
}

func terminalCSIState(state xparser.State) bool {
	return state == xparser.CsiEntryState || state == xparser.CsiParamState || state == xparser.CsiIntermediateState
}

func (o *terminalControlObserver) handleESC(command xansi.Cmd) {
	if command.Prefix() == 0 && command.Intermediate() == 0 && command.Final() == 'c' {
		o.visible = false
	}
}

func (m *terminalModel) Snapshot() terminalSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return cloneTerminalSnapshot(m.final)
	}
	return m.snapshotLocked()
}

func (m *terminalModel) snapshotLocked() terminalSnapshot {
	width, height := m.emulator.Width(), m.emulator.Height()
	snapshot := terminalSnapshot{
		Width:         width,
		Height:        height,
		Cursor:        m.emulator.CursorPosition(),
		CursorVisible: m.synchronized && m.observer.visible,
		AltScreen:     m.altScreen,
		Cells:         make([]uv.Cell, width*height),
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if cell := m.emulator.CellAt(x, y); cell != nil {
				snapshot.Cells[y*width+x] = *cell.Clone()
			}
		}
	}
	return snapshot
}

func cloneTerminalSnapshot(snapshot terminalSnapshot) terminalSnapshot {
	clone := snapshot
	clone.Cells = make([]uv.Cell, len(snapshot.Cells))
	for i := range snapshot.Cells {
		clone.Cells[i] = *snapshot.Cells[i].Clone()
	}
	return clone
}

func (m *terminalModel) Close() error {
	m.mu.Lock()
	if m.closed {
		closeDone := m.closeDone
		m.mu.Unlock()
		<-closeDone
		m.mu.Lock()
		closeErr := m.closeErr
		m.mu.Unlock()
		return closeErr
	}
	m.final = m.snapshotLocked()
	m.closed = true
	replyCloser := m.replyCloser
	drainDone := m.drainDone
	emulator := m.emulator
	m.mu.Unlock()

	closeErr := replyCloser.Close()
	<-drainDone
	emulatorErr := emulator.Close()
	result := emulatorErr
	if closeErr != nil {
		result = closeErr
	}

	m.mu.Lock()
	m.closeErr = result
	close(m.closeDone)
	m.mu.Unlock()
	return result
}
