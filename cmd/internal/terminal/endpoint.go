package terminal

import (
	"context"
	"errors"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type EffectKind uint8

const (
	BellEffect EffectKind = iota
	TitleEffect
	DirectoryEffect
	ClipboardEffect
	NotificationEffect
)

// Effect is one output-side action. Position identifies the committed output
// batch, Sequence orders effects within the endpoint's lifetime.
type Effect struct {
	Kind               EffectKind
	EndpointID         string
	Position, Sequence uint64
	Text, Selection    string
	Data               []byte
}
type Output struct {
	Generation, Position uint64
	Effects              []Effect
}
type Modes struct {
	MouseEpoch                                                         uint64
	Tracking                                                           int
	SGR, Focus, Paste, ApplicationCursor, ApplicationKeypad, AltScreen bool
	Keyboard                                                           uint32
}
type replyBuffer struct {
	data    []byte
	failure error
}

func (b *replyBuffer) Write(p []byte) (int, error) {
	if b.failure != nil {
		return 0, b.failure
	}
	if len(p) > MaxInputBytes-len(b.data) {
		b.failure = ErrBackpressure
		return 0, b.failure
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

// Endpoint is the only interpreter of one child's terminal output. Its mutex
// orders backend mutations and admissions to the child's sole FIFO writer.
type Endpoint struct {
	mu                                    sync.Mutex
	id                                    string
	backend                               *vt.Emulator
	input                                 *InputWriter
	replies                               replyBuffer
	geometry                              Geometry
	generation, epoch, position, sequence uint64
	now, syncStarted                      time.Time
	syncState                             uint8 // 0 idle, 1 withheld, 2 timed-out/recovered
	published                             Frame
	effects                               []Effect
	failure                               error
	closed                                bool
	closeDone                             chan struct{}
}

func NewEndpoint(id string, g Geometry, out ttyio.Writer) (*Endpoint, error) {
	if id == "" || out == nil {
		return nil, errors.New("terminal: endpoint needs identity and writer")
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	backend, err := vt.NewEmulatorWithLimits(g.Cols, g.Rows, vt.DefaultLimits())
	if err != nil {
		return nil, err
	}
	e := &Endpoint{closeDone: make(chan struct{}), id: id, backend: backend, geometry: g, epoch: 1}
	backend.SetReplyWriter(&e.replies)
	backend.SetCallbacks(vt.Callbacks{
		Bell:             func() { e.effect(Effect{Kind: BellEffect}) },
		Title:            func(s string) { e.effect(Effect{Kind: TitleEffect, Text: s}) },
		WorkingDirectory: func(s string) { e.effect(Effect{Kind: DirectoryEffect, Text: s}) },
		ClipboardWrite: func(s string, b []byte) {
			e.effect(Effect{Kind: ClipboardEffect, Selection: s, Data: append([]byte(nil), b...)})
		},
		ClipboardQuery: func(s string) {
			if validSelection(s) {
				_, err := e.replies.Write([]byte("\x1b]52;" + s + ";\x1b\\"))
				if err != nil {
					e.failure = err
				}
			}
		},
		Notification: func(title, body string) { e.effect(Effect{Kind: NotificationEffect, Selection: title, Text: body}) },
		EnableMode: func(m ansi.Mode) {
			if m == ansi.DECMode(2026) && e.syncState == 0 {
				e.published = e.capture()
				e.syncState = 1
				e.syncStarted = e.now
			}
		},
		DisableMode: func(m ansi.Mode) {
			if m == ansi.DECMode(2026) {
				e.syncState = 0
			}
		},
	})
	if err := DefaultProfile().InstallQueries(backend, &e.replies); err != nil {
		backend.Close()
		return nil, err
	}
	e.input = NewInputWriter(out, MaxInputPackets, MaxInputBytes)
	e.published = e.capture()
	return e, nil
}
func validSelection(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("cpqs01234567", r) {
			return false
		}
	}
	return true
}
func (e *Endpoint) effect(v Effect) {
	if len(e.effects) >= MaxPendingEvents {
		e.failure = ErrBackpressure
		return
	}
	e.sequence++
	v.EndpointID = e.id
	v.Sequence = e.sequence
	e.effects = append(e.effects, v)
}
func (e *Endpoint) check() error {
	if e.closed {
		return os.ErrClosed
	}
	if e.failure != nil {
		return e.failure
	}
	return e.input.Failure()
}
func (e *Endpoint) commitReplies() error {
	err := errors.Join(e.backend.TakeReplyError(), e.replies.failure)
	if err == nil && e.failure != nil {
		err = e.failure
	}
	if err == nil {
		err = e.input.Enqueue(e.replies.data)
	}
	e.replies.data = e.replies.data[:0]
	e.replies.failure = nil
	if err != nil {
		e.failure = err
	}
	return err
}
func (e *Endpoint) Feed(p []byte, now time.Time) (Output, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.check(); err != nil {
		return Output{}, err
	}
	e.now = now
	n, err := e.backend.Write(p)
	if !e.backend.Mode(ansi.DECMode(2026)).IsSet() {
		e.syncState = 0
	}
	e.position += uint64(n)
	if n > 0 {
		e.generation++
	}
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		e.failure = err
	}
	for i := range e.effects {
		e.effects[i].Position = e.position
	}
	out := Output{Generation: e.generation, Position: e.position, Effects: e.effects}
	e.effects = nil
	return out, errors.Join(err, e.commitReplies())
}
func (e *Endpoint) capture() Frame {
	f := Frame{EndpointID: e.id, Generation: e.generation, GeometryEpoch: e.epoch, Geometry: e.geometry, AltScreen: e.backend.IsAltScreen()}
	cur := e.backend.Cursor()
	f.Cursor = Cursor{X: cur.X, Y: cur.Y, Visible: !cur.Hidden, Blink: !cur.Steady, Shape: int(cur.Style) + 1}
	f.Cells = make([]Cell, e.geometry.Cols*e.geometry.Rows)
	for y := 0; y < e.geometry.Rows; y++ {
		for x := 0; x < e.geometry.Cols; x++ {
			if c := e.backend.CellAt(x, y); c != nil {
				f.Cells[y*e.geometry.Cols+x] = *c
			}
		}
	}
	return f
}
func (e *Endpoint) Snapshot(now time.Time) (Frame, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.check(); err != nil {
		return Frame{}, err
	}
	if e.syncState == 1 {
		if now.Sub(e.syncStarted) < SyncTimeout {
			return e.published.Clone(), nil
		}
		e.syncState = 2
	}
	if e.published.Generation != e.generation || e.published.GeometryEpoch != e.epoch {
		e.published = e.capture()
	}
	return e.published.Clone(), nil
}
func (e *Endpoint) Send(event uv.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.check(); err != nil {
		return err
	}
	switch v := event.(type) {
	case uv.KeyEvent:
		e.backend.SendKey(v)
	case uv.MouseEvent:
		e.backend.SendMouse(v)
	case uv.PasteEvent:
		if len(v.Content) > MaxPasteBytes {
			return ErrBackpressure
		}
		e.backend.Paste(v.Content)
	case uv.FocusEvent:
		e.backend.Focus()
	case uv.BlurEvent:
		e.backend.Blur()
	default:
		return fmt.Errorf("terminal: unsupported input %T", event)
	}
	return e.commitReplies()
}
func (e *Endpoint) Modes() Modes {
	e.mu.Lock()
	defer e.mu.Unlock()
	set := func(n int) bool { return e.backend.Mode(ansi.DECMode(n)).IsSet() }
	m := Modes{MouseEpoch: e.backend.MouseEpoch(), SGR: set(1006), Focus: set(1004), Paste: set(2004), ApplicationCursor: set(1), ApplicationKeypad: set(66), AltScreen: e.backend.IsAltScreen(), Keyboard: e.backend.KeyboardFlags()}
	for _, n := range []int{9, 1000, 1002, 1003} {
		if set(n) {
			m.Tracking = n
		}
	}
	return m
}
func (e *Endpoint) Resize(g Geometry, apply func(Geometry) error) error {
	if err := g.Validate(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.check(); err != nil {
		return err
	}
	if g == e.geometry {
		return nil
	}
	if apply == nil {
		return errors.New("terminal: resize requires PTY acknowledgement")
	}
	if err := apply(g); err != nil {
		return err
	}
	if err := e.backend.ResizeChecked(g.Cols, g.Rows); err != nil {
		e.failure = err
		return err
	}
	e.geometry = g
	e.epoch++
	e.generation++
	e.syncState = 0
	return e.commitReplies()
}
func (e *Endpoint) Flush(ctx context.Context) error { return e.input.Flush(ctx) }
func (e *Endpoint) Close() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		<-e.closeDone
		return
	}
	e.closed = true
	e.backend.Close()
	e.mu.Unlock()
	e.input.Close()
	close(e.closeDone)
}

// NextPublication is the only timer the presenter needs while output is held.
// Zero means no timed publication remains; idle endpoints require no repaint.
func (e *Endpoint) NextPublication() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed || e.failure != nil || e.syncState != 1 {
		return time.Time{}
	}
	return e.syncStarted.Add(SyncTimeout)
}

// SendMouse admits an event only under the exact negotiation that established
// its gesture. A mode change between inspection and encoding rejects it without
// sending an orphaned event in the child's new protocol.
func (e *Endpoint) SendMouse(event uv.MouseEvent, expectedEpoch uint64) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.check(); err != nil {
		return false, err
	}
	if e.backend.MouseEpoch() != expectedEpoch {
		return false, nil
	}
	e.backend.SendMouse(event)
	if err := e.commitReplies(); err != nil {
		return false, err
	}
	return true, nil
}
