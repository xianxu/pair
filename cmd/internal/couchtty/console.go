package couchtty

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// chunk is one child write on its way to the screen.
type chunk struct {
	id   string
	data []byte
}

type pane struct {
	label string
	child *ptychild.Child

	// bell is sticky until the operator looks at this actor. The row's job is
	// to say who wants attention, so a signal that cleared itself on the next
	// repaint would be invisible in practice.
	bell bool

	// rowDirty is the same shape for the reserved row: an INACTIVE pane's
	// erase or margin reset is real, it just cannot be acted on yet. The first
	// version consumed the child's latch for every pane and acted on it only
	// for the active one, so a background child's damage was thrown away and
	// attaching to it would land on a screen with no status row.
	rowDirty bool
}

// Console routes the operator's terminal to one child at a time.
//
// It is the THIN IO SHELL and holds no policy: every decision it makes is a
// call into a pure function in this package. It drives hostty.Host rather than
// x/term and os/signal directly, which is what makes the resize path and the
// restore-on-teardown path testable without a terminal.
type Console struct {
	host   hostty.Host
	stdin  io.Reader
	stderr io.Writer

	mu     sync.Mutex
	panes  map[string]*pane
	order  []string
	active string
	notice string
	size   ptychild.Size

	// paintPending means a repaint was wanted while the host stream was
	// mid-sequence, and is owed as soon as it is safe.
	paintPending bool

	// hostScan frames the bytes the console has WRITTEN to the host.
	//
	// It has to be this stream, not the child's. Asking the child was the first
	// shape of this fix and it was unsound (M2 BR-21): ptychild's pump feeds its
	// Screen before calling the sink, and the console drains a buffered channel
	// later, so by the time it asked about the chunk it had just written, the
	// answer described a LATER chunk the child had since read. Framing what we
	// write is race-free by construction -- there is exactly one writer.
	hostScan ptychild.Screen

	// Run is the ONLY goroutine that writes to the host. Everything that wants
	// the screen sends here instead of writing.
	//
	// The first fix for BR-21 framed the console's own output but left
	// applyLayout and the hotkey path writing from other goroutines, so a
	// SIGWINCH or a keypress could still splice into the child's stream. Making
	// the writer singular removes the class rather than the two instances:
	// there is no longer a way to reach the screen except through the loop that
	// tracks where the stream is.
	chunks  chan chunk
	resized chan struct{}
	hotkeys chan struct{}
	stop    chan struct{}
	once    sync.Once
}

// errw is where the console reports its own failures. Separate from the host
// because a host that cannot go raw may equally be unable to render.
func (c *Console) errw() io.Writer {
	if c.stderr != nil {
		return c.stderr
	}
	return os.Stderr
}

func New(host hostty.Host, stdin io.Reader) *Console {
	c := &Console{
		host:    host,
		stdin:   stdin,
		panes:   map[string]*pane{},
		chunks:  make(chan chunk, 256),
		resized: make(chan struct{}, 1),
		hotkeys: make(chan struct{}, 8),
		stop:    make(chan struct{}),
	}
	if s, err := host.Size(); err == nil {
		c.size = s
	}
	return c
}

// SetErrorWriter redirects the console's own diagnostics, so a test can read
// them instead of the process's stderr.
func (c *Console) SetErrorWriter(w io.Writer) { c.stderr = w }

// ChildSize is what a new child should be sized to: the host, minus the
// reserved row. Handed to PtyRunner so the FIRST frame is already right --
// spawning at the host height and reflowing is a whole redraw for a full-screen
// agent harness.
func (c *Console) ChildSize() ptychild.Size {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ptychild.Size{Rows: ChildRows(c.size.Rows), Cols: c.size.Cols}
}

// Deliver is the sink handed to the runner: it hands a child's output to the
// console loop.
//
// It BLOCKS when the buffer is full rather than dropping, and that reversal is
// deliberate. The first version dropped, justified by "the ring still has it,
// so the next repaint is correct" -- but nothing repaints from the ring at this
// milestone (that arrives with M3's attach path), so a drop was silent, permanent
// output loss on a slow screen (M2 BR-29). Blocking applies back-pressure to the
// pty instead, which is what a terminal does anyway.
//
// It still yields to stop, so teardown cannot deadlock behind a child that is
// mid-write.
func (c *Console) Deliver(id string, data []byte) {
	select {
	case c.chunks <- chunk{id: id, data: data}:
	case <-c.stop:
	}
}

// Attach registers a child. The first one attached is the active one -- and in
// M2 the only one.
func (c *Console) Attach(id, label string, child *ptychild.Child) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.panes[id] = &pane{label: label, child: child}
	c.order = append(c.order, id)
	if c.active == "" {
		c.active = id
	}
}

// PaneRowDirty reports whether a pane still owes a row repaint. Exported for
// the test that pins an inactive pane's damage surviving -- a latch thrown away
// is invisible from every other accessor.
func (c *Console) PaneRowDirty(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.panes[id]; ok {
		return p.rowDirty
	}
	return false
}

// Stop tears the console down. Safe to call more than once, and from any
// goroutine.
func (c *Console) Stop() { c.once.Do(func() { close(c.stop) }) }

// Run owns the operator's terminal until the active child exits or Stop is
// called. It returns the child's exit code.
func (c *Console) Run() int {
	restore, err := c.host.MakeRaw()
	if err != nil {
		// Say why. Returning a bare 1 was the other half of BR-23: the
		// operator saw an exit code and nothing else.
		fmt.Fprintf(c.errw(), "couch: cannot take the terminal: %v\n", err)
		return 1
	}
	// Restoration is deferred FIRST so it runs LAST, after the region reset
	// below -- the escapes have to reach a terminal that is still ours.
	defer func() { _ = restore() }()
	defer c.release()

	c.applyLayout()
	c.paintNow()

	go c.pumpStdin()
	go c.watchResize()

	exited := make(chan int, 1)
	if child := c.activeChild(); child != nil {
		go func() { exited <- child.Wait() }()
	}

	for {
		select {
		case ch := <-c.chunks:
			c.onChunk(ch)
		case <-c.resized:
			c.onResize()
		case <-c.hotkeys:
			c.onHotkey()
		case code := <-exited:
			return code
		case <-c.stop:
			return 0
		}
	}
}

// release puts the terminal back: region reset, then the reserved row cleared,
// so the operator's shell does not inherit a pinned region or a stale row.
func (c *Console) release() {
	c.mu.Lock()
	rows := c.size.Rows
	c.mu.Unlock()
	// Teardown writes UNCONDITIONALLY: a half-restored terminal is worse than a
	// spliced sequence, and the child is finished with the screen by now.
	_, _ = io.WriteString(c.host, Release()+PaintRow(rows, ""))
}

func (c *Console) activeChild() *ptychild.Child {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.panes[c.active]; ok {
		return p.child
	}
	return nil
}

// applyLayout sizes every child to fit above the reserved row. The row itself
// is drawn by the paint, so there is one gated path to the screen rather than
// two.
func (c *Console) applyLayout() {
	c.mu.Lock()
	size := ptychild.Size{Rows: ChildRows(c.size.Rows), Cols: c.size.Cols}
	children := make([]*ptychild.Child, 0, len(c.panes))
	for _, p := range c.panes {
		children = append(children, p.child)
	}
	c.mu.Unlock()

	// The resize always happens; only the SCREEN write is gated, and it goes
	// through the paint below so there is one gated path rather than two.
	for _, child := range children {
		_ = child.Resize(size)
	}
}

// repaint draws the status row when it is SAFE to do so, and defers when it is
// not.
//
// Safety here is about the child's stream, not about locking: a pty read
// boundary falls wherever the kernel puts it, so a paint written between two
// chunks can land inside one of the child's escape sequences. A real nvim under
// the console produced exactly that -- `\x1b7\x1b[12;1H\x1b[2K[brain]\x1b8`
// spliced into the middle of `\x1b[38;2;76;82;88m`, corrupting the child's
// colours and losing the row. The debt is remembered and paid by the next chunk
// that leaves the stream at a sequence boundary.
func (c *Console) repaint() { c.paintNow() }

// writeChild passes the active child's output through, tracking where the
// CHILD's stream sits. Called only from the Run goroutine.
//
// Only child bytes are fed to the scanner. Feeding our own escapes into it was
// the second shape of this bug: appending `\x1b[1;23r` to a pending
// `\x1b[38;2;76` let the scanner frame the two together as one complete
// sequence, so it reported "safe" precisely when it was not. The question the
// scanner answers is "where is the child's stream", and our writes are not part
// of it.
func (c *Console) writeChild(p []byte) {
	c.mu.Lock()
	c.hostScan.Feed(p)
	c.mu.Unlock()
	_, _ = c.host.Write(p)
}

// writeOwn emits the console's OWN bytes, and is the only way they reach the
// screen. It refuses while the child's stream is mid-sequence and records the
// debt; the next chunk that lands on a boundary pays it.
func (c *Console) writeOwn(p string) {
	c.mu.Lock()
	if c.hostScan.MidSequence() {
		c.paintPending = true
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	_, _ = io.WriteString(c.host, p)
}

// paintNow draws the row unconditionally, re-asserting the region first.
//
// The re-assertion is not belt-and-braces: a child that reset margins may have
// dropped it a moment ago, and painting into an unreserved screen is what puts
// the row where the child's content should be.
func (c *Console) paintNow() {
	c.mu.Lock()
	c.paintPending = false
	rows := c.size.Rows
	cols := int(c.size.Cols)
	model := StatusModel{Notice: c.notice}
	for _, id := range c.order {
		p := c.panes[id]
		model.Actors = append(model.Actors, StatusActor{
			Label:  p.label,
			Active: id == c.active,
			Bell:   p.bell,
		})
	}
	c.mu.Unlock()

	c.writeOwn(Reserve(rows) + PaintRow(rows, RenderStatusRow(cols, model)))
}

// onChunk routes one child write.
func (c *Console) onChunk(ch chunk) {
	c.mu.Lock()
	p, known := c.panes[ch.id]
	isActive := ch.id == c.active
	c.mu.Unlock()
	if !known {
		return
	}

	if isActive {
		c.writeChild(ch.data)
	}
	// A paint deferred while the stream was mid-sequence is owed as soon as
	// the stream is whole again.
	c.mu.Lock()
	owed := c.paintPending && !c.hostScan.MidSequence()
	c.mu.Unlock()
	if owed {
		c.paintNow()
	}
	// Derived state is consumed whether or not the child is on screen: an
	// inactive child that rings still has something to say.
	// The child's latch is per-chunk, so it is consumed for every pane -- but
	// KEPT on the pane, so an inactive child's damage survives until the
	// operator lands on it.
	if p.child.TakeRowDirty() {
		c.mu.Lock()
		p.rowDirty = true
		c.mu.Unlock()
	}
	if p.child.TakeBell() {
		c.mu.Lock()
		// An actor the operator is already looking at is not "wanting" them.
		if !isActive {
			p.bell = true
			c.notice = p.label + " wants you"
		}
		c.mu.Unlock()
		c.repaint()
		return
	}
	c.mu.Lock()
	dirty := p.rowDirty && isActive
	if dirty {
		p.rowDirty = false
	}
	c.mu.Unlock()
	if dirty {
		c.repaint()
	}
}

// watchResize turns host resizes into events for the Run loop. It deliberately
// does NOT touch the screen: see the note on the channel fields.
func (c *Console) watchResize() {
	for {
		select {
		case _, ok := <-c.host.Resized():
			if !ok {
				return
			}
			select {
			case c.resized <- struct{}{}: // coalesced; one pending is enough
			default:
			}
		case <-c.stop:
			return
		}
	}
}

// onResize runs on the Run goroutine.
func (c *Console) onResize() {
	if s, err := c.host.Size(); err == nil {
		c.mu.Lock()
		c.size = s
		c.mu.Unlock()
	}
	c.applyLayout()
	c.repaint()
}

// pumpStdin routes the operator's keystrokes, splitting around the hotkey.
func (c *Console) pumpStdin() {
	var it Interceptor
	buf := make([]byte, 4096)
	for {
		n, err := c.stdin.Read(buf)
		if n > 0 {
			in := append([]byte(nil), buf[:n]...)
			for {
				before, hit, rest := it.Feed(in)
				if len(before) > 0 {
					if child := c.activeChild(); child != nil {
						_, _ = child.Write(before)
					}
				}
				if !hit {
					break
				}
				select {
				case c.hotkeys <- struct{}{}:
				case <-c.stop:
					return
				}
				in = rest
			}
		}
		if err != nil {
			return
		}
		select {
		case <-c.stop:
			return
		default:
		}
	}
}

// onHotkey handles ctrl-space.
//
// M2 has one child and no panel, so "up one level" has nowhere to go and the
// row says so. M3 replaces this with the focus model -- the point of doing it
// here is that the INTERCEPTION is proven end to end before there is anywhere
// to land.
func (c *Console) onHotkey() {
	c.mu.Lock()
	c.notice = "ctrl-space: no other actors yet"
	c.mu.Unlock()
	c.repaint()
}
