package couchtty

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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
	desc  string
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

	// root is the actor `ctrl-space` goes home to: the FIRST child attached,
	// which is "whatever session couch launched in" delivered by convention
	// (Decision 1). Nothing here knows what brain is.
	root string

	// focus is what the terminal is pointed at. It is not the same as `active`:
	// the panel is a focus with no actor behind it.
	focus Focus

	// query is the panel's typeahead buffer, and resolve is the match rule --
	// INJECTED rather than implemented, so the panel resolves exactly what the
	// CLI and #148's advisor resolve (Decision 12). Nil degrades to showing
	// everything rather than to a private match rule.
	query   string
	resolve func(string) []couchcore.Worktree
	notice  string
	size    ptychild.Size

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
	chunks    chan chunk
	resized   chan struct{}
	hotkeys   chan struct{}
	switching chan string
	panelKeys chan byte
	stop      chan struct{}
	once      sync.Once
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
		host:      host,
		stdin:     stdin,
		panes:     map[string]*pane{},
		chunks:    make(chan chunk, 256),
		resized:   make(chan struct{}, 1),
		switching: make(chan string, 8),
		panelKeys: make(chan byte, 64),
		hotkeys:   make(chan struct{}, 8),
		stop:      make(chan struct{}),
	}
	if s, err := host.Size(); err == nil {
		c.size = s
	}
	return c
}

// SetResolver injects the panel's match rule. Production passes
// `couch.LookupTrees`; without it the seam is one nothing uses.
func (c *Console) SetResolver(f func(string) []couchcore.Worktree) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resolve = f
}

// Resolver returns the injected match rule, so a wiring test can assert one was
// actually passed -- a nil resolver still renders a panel, so nothing else
// would notice.
func (c *Console) Resolver() func(string) []couchcore.Worktree {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resolve
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
		c.root = id
		c.focus = FocusActor(id)
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

// Switch points the operator's terminal at another hosted actor.
//
// A request, not an action: it lands on the Run goroutine, which is the only
// one allowed to write to the host. Callers may be the panel, the hotkey path,
// or (in #148) the advisor's tool layer -- none of them get to touch the screen
// directly.
func (c *Console) Switch(id string) {
	select {
	case c.switching <- id:
	case <-c.stop:
	}
}

// onSwitch lands the operator on another child, running on the Run goroutine.
//
// Order is the whole contract: clear, replay the child's own screen, THEN the
// status row. Painting the row first means the landing paints over it.
func (c *Console) onSwitch(id string) {
	c.mu.Lock()
	p, known := c.panes[id]
	already := c.active == id
	if known {
		c.active = id
		c.focus = FocusActor(id)
		// Landing on an actor is looking at it: whatever it wanted is now the
		// operator's problem rather than a pending flag.
		p.bell = false
		p.rowDirty = false
	}
	c.mu.Unlock()
	if !known || already {
		// An unknown actor is not a reason to blank the operator's screen.
		return
	}

	// The replay is Replay(), not Snapshot(): a raw one still carries whatever
	// capability queries the child emitted at startup, and re-asking the host
	// terminal lands the ANSWER in the newly active child's stdin -- #127's bug
	// arriving at a new site.
	c.takeOverScreen(p.child.Replay())
	c.paintNow()
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
		case id := <-c.switching:
			c.onSwitch(id)
		case b := <-c.panelKeys:
			c.panelKey(b)
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

// takeOverScreen replaces what is on the screen wholesale -- a switch landing,
// or the panel opening.
//
// Distinct from writeOwn on purpose. An interleaved paint must WAIT for a
// sequence boundary because it is inserted into a stream that continues; a
// takeover ENDS that stream's relevance, so waiting would strand the operator
// on the previous child's screen. It resets the framing state for the same
// reason: whatever partial sequence the old child left is no longer on screen
// to be corrupted.
//
// It is still Run-goroutine-only, like every other writer.
func (c *Console) takeOverScreen(body []byte) {
	c.mu.Lock()
	c.hostScan = ptychild.Screen{}
	c.paintPending = false
	c.mu.Unlock()

	_, _ = io.WriteString(c.host, hostty.HomeAndClear)
	_, _ = c.host.Write(body)
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
	// "Active" means the operator is looking at this child. With the panel up
	// nobody is, so a child that keeps streaming must not paint over couch's
	// own screen.
	isActive := ch.id == c.active && !c.focus.IsPanel()
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
					c.mu.Lock()
					toPanel := c.focus.IsPanel()
					c.mu.Unlock()
					if toPanel {
						// The panel owns the keyboard while it is up, or a
						// child would act on keys aimed at couch.
						for _, b := range before {
							select {
							case c.panelKeys <- b:
							case <-c.stop:
								return
							}
						}
					} else if child := c.activeChild(); child != nil {
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

// onHotkey handles ctrl-space: up one level.
//
// Runs on the Run goroutine. Liveness is passed to Up rather than assumed --
// landing on a dead root actor gives the operator a frozen screen with no way
// to tell it is frozen.
func (c *Console) onHotkey() {
	c.mu.Lock()
	cur, root := c.focus, c.root
	c.mu.Unlock()

	next := Up(cur, root, c.actorAlive)
	if next == cur {
		return // already at the top
	}

	c.mu.Lock()
	c.focus = next
	c.mu.Unlock()

	if next.IsPanel() {
		c.showPanel()
		return
	}
	c.onSwitch(next.Actor())
}

// actorAlive is the liveness predicate Up consults.
func (c *Console) actorAlive(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.panes[id]
	return ok && !p.child.Done()
}

// panelModel builds the panel's data from what the console is hosting.
func (c *Console) panelModel() (*PanelModel, string, func(string) []couchcore.Worktree) {
	c.mu.Lock()
	defer c.mu.Unlock()
	trees := make([]couchcore.TreeSummary, 0, len(c.order))
	for _, id := range c.order {
		p := c.panes[id]
		var actors []couchcore.ActorView
		if !p.child.Done() {
			actors = []couchcore.ActorView{{Live: true}}
		}
		trees = append(trees, couchcore.TreeSummary{
			Tree: couchcore.Worktree(id), Name: p.label, Desc: p.desc, Actors: actors,
		})
	}
	return NewPanelModel(trees), c.query, c.resolve
}

// showPanel draws couch's own screen, filtered by whatever has been typed.
func (c *Console) showPanel() {
	m, query, resolve := c.panelModel()
	rows := m.Filter(query, resolve)
	c.takeOverScreen([]byte(RenderPanelWithQuery(query, rows)))
	c.paintNow()
}

// panelKey handles one keystroke while the panel is up.
//
// A digit is a DIRECT switch with no resolution in the path -- the Spec
// requires a route that never waits on a model turn, and this is it.
func (c *Console) panelKey(b byte) {
	switch {
	case b >= '1' && b <= '9':
		m, query, resolve := c.panelModel()
		m.Filter(query, resolve)
		if row, ok := m.Pick(int(b - '0')); ok {
			c.clearQuery()
			c.onSwitch(string(row.Tree))
			return
		}
	case b == 0x7f || b == 0x08: // backspace
		c.mu.Lock()
		if n := len(c.query); n > 0 {
			c.query = c.query[:n-1]
		}
		c.mu.Unlock()
	case b == '\r' || b == '\n':
		m, query, resolve := c.panelModel()
		m.Filter(query, resolve)
		if row, ok := m.Pick(1); ok {
			c.clearQuery()
			c.onSwitch(string(row.Tree))
			return
		}
	case b >= 0x20 && b < 0x7f:
		c.mu.Lock()
		c.query += string(b)
		c.mu.Unlock()
	default:
		return // ignore control bytes rather than filtering on them
	}
	c.showPanel()
}

func (c *Console) clearQuery() {
	c.mu.Lock()
	c.query = ""
	c.mu.Unlock()
}
