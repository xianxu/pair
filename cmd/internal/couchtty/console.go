package couchtty

import (
	"io"
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
}

// Console routes the operator's terminal to one child at a time.
//
// It is the THIN IO SHELL and holds no policy: every decision it makes is a
// call into a pure function in this package. It drives hostty.Host rather than
// x/term and os/signal directly, which is what makes the resize path and the
// restore-on-teardown path testable without a terminal.
type Console struct {
	host  hostty.Host
	stdin io.Reader

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

	chunks chan chunk
	stop   chan struct{}
	once   sync.Once
}

func New(host hostty.Host, stdin io.Reader) *Console {
	c := &Console{
		host:   host,
		stdin:  stdin,
		panes:  map[string]*pane{},
		chunks: make(chan chunk, 256),
		stop:   make(chan struct{}),
	}
	if s, err := host.Size(); err == nil {
		c.size = s
	}
	return c
}

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

// Stop tears the console down. Safe to call more than once, and from any
// goroutine.
func (c *Console) Stop() { c.once.Do(func() { close(c.stop) }) }

// Run owns the operator's terminal until the active child exits or Stop is
// called. It returns the child's exit code.
func (c *Console) Run() int {
	restore, err := c.host.MakeRaw()
	if err != nil {
		return 1
	}
	// Restoration is deferred FIRST so it runs LAST, after the region reset
	// below -- the escapes have to reach a terminal that is still ours.
	defer func() { _ = restore() }()
	defer c.release()

	c.applyLayout()
	c.repaint()

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
	_, _ = io.WriteString(c.host, Release())
	_, _ = io.WriteString(c.host, PaintRow(rows, ""))
}

func (c *Console) activeChild() *ptychild.Child {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.panes[c.active]; ok {
		return p.child
	}
	return nil
}

// applyLayout reserves the row and sizes every child to fit above it.
func (c *Console) applyLayout() {
	c.mu.Lock()
	rows := c.size.Rows
	size := ptychild.Size{Rows: ChildRows(c.size.Rows), Cols: c.size.Cols}
	children := make([]*ptychild.Child, 0, len(c.panes))
	for _, p := range c.panes {
		children = append(children, p.child)
	}
	c.mu.Unlock()

	_, _ = io.WriteString(c.host, Reserve(rows))
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
func (c *Console) repaint() {
	c.mu.Lock()
	mid := c.hostScan.MidSequence()
	if mid {
		c.paintPending = true
	}
	c.mu.Unlock()
	if mid {
		return
	}
	c.paintNow()
}

// writeHost is the ONE path to the operator's screen for child output, so the
// framing state cannot miss a byte.
func (c *Console) writeHost(p []byte) {
	c.mu.Lock()
	c.hostScan.Feed(p)
	c.mu.Unlock()
	_, _ = c.host.Write(p)
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

	_, _ = io.WriteString(c.host, Reserve(rows))
	_, _ = io.WriteString(c.host, PaintRow(rows, RenderStatusRow(cols, model)))
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
		c.writeHost(ch.data)
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
	rowDirty := p.child.TakeRowDirty()
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
	if rowDirty && isActive {
		c.repaint()
	}
}

func (c *Console) watchResize() {
	for {
		select {
		case _, ok := <-c.host.Resized():
			if !ok {
				return
			}
			if s, err := c.host.Size(); err == nil {
				c.mu.Lock()
				c.size = s
				c.mu.Unlock()
			}
			c.applyLayout()
			c.repaint()
		case <-c.stop:
			return
		}
	}
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
				c.onHotkey()
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
