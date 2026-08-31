package couchtty

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// chunk is one child write on its way to the screen.
type chunk struct {
	id   string
	data []byte
}

type childExit struct {
	id   string
	code int
}

type pane struct {
	tree    couchcore.Worktree
	thread  couchcore.ThreadAddress
	process couchcore.ProcessIdentity
	actorID couchcore.ActorID
	label   string
	desc    string
	child   *ptychild.Child

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
// It is the integration controller: reusable decisions live in pure functions
// in this package, while Console owns event ordering and transient UI
// transitions as it drives hostty.Host. It never calls x/term or os/signal
// directly, which keeps resize and teardown testable without a terminal.
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
	// the switcher is a focus with no actor behind it.
	focus Focus

	actionable ActionableThreadProvider
	menu       MenuState
	menuReady  bool

	// menuHeld carries a partial escape sequence across reads.
	menuHeld []byte

	// Ops dispatches an operator action. Injected so the console never learns
	// what an operation IS -- it names one and couchcore runs it, which is
	// what keeps the panel from growing a private verb (#148's design test).
	ops    func(couchcore.OperationCall) (any, error)
	forget func(couchcore.Worktree, couchcore.ActorID) error
	feed   *Feed
	size   ptychild.Size
	// expectedExits are exact child handles whose successful Park already
	// authorized shutdown. They bridge the race between the child-exit channel
	// and the asynchronous operation-completion channel.
	expectedExits map[string]bool

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
	chunks          chan chunk
	resized         chan struct{}
	switching       chan string
	input           chan []byte
	exited          chan childExit
	operationQueue  *operationQueue
	refreshRequests chan struct{}
	refreshResults  chan menuRefreshResult
	refreshSchedule RefreshSchedule
	previewResults  chan menuPreviewResult
	previewSchedule PreviewSchedule
	previewCancel   context.CancelFunc
	previewRunning  uint64
	lifetime        context.Context
	cancelLifetime  context.CancelFunc
	stop            chan struct{}
	once            sync.Once
	workers         sync.WaitGroup
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
	lifetime, cancelLifetime := context.WithCancel(context.Background())
	c := &Console{
		host:            host,
		stdin:           stdin,
		panes:           map[string]*pane{},
		chunks:          make(chan chunk, 256),
		resized:         make(chan struct{}, 1),
		switching:       make(chan string, 8),
		input:           make(chan []byte, 64),
		exited:          make(chan childExit, 64),
		operationQueue:  newOperationQueue(16),
		refreshRequests: make(chan struct{}, 1),
		refreshResults:  make(chan menuRefreshResult, 1),
		previewResults:  make(chan menuPreviewResult, 1),
		expectedExits:   map[string]bool{},
		lifetime:        lifetime,
		cancelLifetime:  cancelLifetime,
		stop:            make(chan struct{}),
		feed:            NewFeed(8),
	}
	if s, err := host.Size(); err == nil {
		c.size = s
	}
	return c
}

// SetForget injects registry removal. Production passes Couch.Forget; keeping
// it as a narrow seam lets Console own exit ordering without owning registry
// persistence or process policy.
func (c *Console) SetForget(f func(couchcore.Worktree, couchcore.ActorID) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forget = f
}

// SetOperationDispatcher installs the typed generic dispatcher. Without it,
// every effectful panel action refuses loudly rather than taking a private path.
func (c *Console) SetOperationDispatcher(f func(couchcore.OperationCall) (any, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ops = f
}

// Ops returns the injected dispatcher, so a wiring test can assert one was
// passed -- the panel renders identically without it.
func (c *Console) Ops() func(couchcore.OperationCall) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ops
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

// Attach registers a child with a synthetic legacy thread address. It remains
// as a test/helper convenience; production supplies the durable address.
func (c *Console) Attach(id, label string, child *ptychild.Child) {
	c.AttachActor(id, couchcore.ActorID(id), couchcore.Worktree(id), label, child)
}

// AttachTree registers a child with a synthetic legacy thread address and its
// working path. It remains for callers predating composite thread identity.
func (c *Console) AttachTree(id string, tree couchcore.Worktree, label string, child *ptychild.Child) {
	c.AttachActor(id, couchcore.ActorID(id), tree, label, child)
}

// AttachActor is the legacy-address test/helper form of attachThreadActor.
func (c *Console) AttachActor(handleID string, actorID couchcore.ActorID, tree couchcore.Worktree, label string, child *ptychild.Child) {
	c.attachThreadActor(handleID, actorID, couchcore.ThreadAddress{RepoScope: "legacy", Tag: couchcore.ThreadTag(actorID)}, tree, label, child)
}

// attachThreadActor registers every identity a hosted pane carries. handleID
// routes PTY bytes inside this console; actorID identifies the live registry
// incarnation; thread identifies durable work; tree is only its working path.
// It stays package-private so production callers cannot bypass the declared
// attach operation with an exact composite address.
func (c *Console) attachThreadActor(handleID string, actorID couchcore.ActorID, thread couchcore.ThreadAddress, tree couchcore.Worktree, label string, child *ptychild.Child) {
	c.attachObservedThreadActor(handleID, actorID, thread, tree, label, child, couchcore.ProcessIdentity{})
}

func (c *Console) attachObservedThreadActor(handleID string, actorID couchcore.ActorID, thread couchcore.ThreadAddress, tree couchcore.Worktree, label string, child *ptychild.Child, process couchcore.ProcessIdentity) {
	_ = c.installObservedThreadActor(c.lifetime, handleID, actorID, thread, tree, label, child, process)
}

// installObservedThreadActor commits a complete pane or no pane. The worker
// count is reserved under the same mutex as routing state, so teardown's mutex
// barrier cannot begin its final Wait between a partial map insertion and the
// exit watcher becoming owned.
func (c *Console) installObservedThreadActor(ctx context.Context, handleID string, actorID couchcore.ActorID, thread couchcore.ThreadAddress, tree couchcore.Worktree, label string, child *ptychild.Child, process couchcore.ProcessIdentity) error {
	if ctx == nil {
		ctx = c.lifetime
	}
	if handleID == "" || actorID == "" || child == nil {
		return errors.New("attach requires complete handle, actor, and terminal identities")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return errors.New("console is stopped")
	case <-child.Exited():
		return errors.New("attach terminal has already exited")
	default:
	}

	c.mu.Lock()
	select {
	case <-ctx.Done():
		c.mu.Unlock()
		return ctx.Err()
	case <-c.stop:
		c.mu.Unlock()
		return errors.New("console is stopped")
	case <-child.Exited():
		c.mu.Unlock()
		return errors.New("attach terminal has already exited")
	default:
	}
	if _, exists := c.panes[handleID]; exists {
		c.mu.Unlock()
		return fmt.Errorf("terminal handle %q is already attached", handleID)
	}
	for _, installed := range c.panes {
		if installed.thread == thread && !installed.child.Done() {
			c.mu.Unlock()
			return fmt.Errorf("thread %s/%s is already attached", thread.RepoScope, thread.Tag)
		}
	}
	c.workers.Add(1)
	c.panes[handleID] = &pane{tree: tree, thread: thread, process: process, actorID: actorID, label: label, child: child}
	c.order = append(c.order, handleID)
	if c.active == "" {
		c.active = handleID
		c.root = handleID
		c.focus = FocusActor(handleID)
	}
	if !c.menuReady {
		c.menu = NewMenuState(nil, thread)
		c.menu.Notice = infoMenuNotice("thread inventory unavailable")
		c.menuReady = true
	}
	c.mu.Unlock()
	c.requestMenuRefresh()

	go func() {
		defer c.workers.Done()
		select {
		case <-child.Exited():
			select {
			case c.exited <- childExit{id: handleID, code: child.Wait()}:
			case <-c.stop:
			}
		case <-c.stop:
		}
	}()
	return nil
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
func (c *Console) onSwitch(id string) { c.switchTo(id, false) }

// forceSwitch repaints even when the actor is already active -- which is the
// case when returning from the panel, where the SCREEN changed but the active
// actor did not.
func (c *Console) forceSwitch(id string) { c.switchTo(id, true) }

func (c *Console) switchTo(id string, force bool) {
	c.mu.Lock()
	p, known := c.panes[id]
	already := c.active == id && !force
	if known {
		c.active = id
		c.focus = FocusActor(id)
		c.menu.ActiveAddress = p.thread
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
func (c *Console) Stop() {
	c.once.Do(func() {
		c.cancelLifetime()
		close(c.stop)
	})
}

// Run owns the operator's terminal until the actor-focused last child exits or
// Stop is called. If the panel already owns focus, a last-child exit leaves it
// available for durable Park/Resume; Escape with no actor calls Stop.
func (c *Console) Run() int {
	restore, err := c.host.MakeRaw()
	if err != nil {
		// Say why. Returning a bare 1 was the other half of BR-23: the
		// operator saw an exit code and nothing else.
		fmt.Fprintf(c.errw(), "couch: cannot take the terminal: %v\n", err)
		return 1
	}
	defer c.teardown(restore)

	c.applyLayout()
	c.paintNow()

	c.workers.Add(3)
	go func() { defer c.workers.Done(); c.pumpStdin() }()
	go func() { defer c.workers.Done(); c.watchResize() }()
	go func() { defer c.workers.Done(); c.operationQueue.Run(c.stop) }()
	var terminated <-chan os.Signal
	if h, ok := c.host.(hostty.TerminationHost); ok {
		terminated = h.Terminated()
	}

	var it Interceptor
	var inputEscapeTimer, panelEscapeTimer *time.Timer
	var inputEscapeC, panelEscapeC <-chan time.Time
	var spinnerTimer *time.Timer
	var spinnerC <-chan time.Time
	var spinnerOwner MenuProgressOwner
	stopTimer := func(timer *time.Timer) {
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
	armInputEscape := func() {
		if !bytes.Equal(it.held, []byte{0x1b}) {
			inputEscapeC = nil
			return
		}
		if inputEscapeTimer == nil {
			inputEscapeTimer = time.NewTimer(escapeAmbiguity)
		} else {
			inputEscapeTimer.Reset(escapeAmbiguity)
		}
		inputEscapeC = inputEscapeTimer.C
	}
	armPanelEscape := func() {
		if !bytes.Equal(c.menuHeld, []byte{0x1b}) {
			panelEscapeC = nil
			return
		}
		if panelEscapeTimer == nil {
			panelEscapeTimer = time.NewTimer(escapeAmbiguity)
		} else {
			panelEscapeTimer.Reset(escapeAmbiguity)
		}
		panelEscapeC = panelEscapeTimer.C
	}
	stopSpinner := func() {
		if spinnerTimer != nil && !spinnerTimer.Stop() {
			select {
			case <-spinnerTimer.C:
			default:
			}
		}
		spinnerC = nil
		spinnerOwner = MenuProgressOwner{}
	}
	defer stopSpinner()
	syncSpinner := func() {
		c.mu.Lock()
		notice := c.menu.Notice
		focused := c.focus.IsPanel()
		c.mu.Unlock()
		active := focused && notice.Level == MenuNoticeProgress && notice.Owner != (MenuProgressOwner{})
		if !active {
			stopSpinner()
			return
		}
		if spinnerC != nil && spinnerOwner == notice.Owner {
			return
		}
		stopSpinner()
		spinnerOwner = notice.Owner
		if spinnerTimer == nil {
			spinnerTimer = time.NewTimer(100 * time.Millisecond)
		} else {
			spinnerTimer.Reset(100 * time.Millisecond)
		}
		spinnerC = spinnerTimer.C
	}
	route := func(raw []byte) {
		if len(raw) == 0 {
			return
		}
		c.mu.Lock()
		toPanel := c.focus.IsPanel()
		c.mu.Unlock()
		if toPanel {
			stopTimer(panelEscapeTimer)
			panelEscapeC = nil
			c.onMenuInput(raw)
			armPanelEscape()
			return
		}
		if child := c.activeChild(); child != nil {
			_, _ = child.Write(raw)
		}
	}
	processInput := func(raw []byte) {
		for {
			before, hit, rest := it.FeedHit(raw)
			route(before)
			if hit == HitNone {
				return
			}
			if hit == HitPark {
				c.onParkHotkey()
			} else {
				c.onHotkey()
			}
			raw = rest
		}
	}

	for {
		select {
		case ch := <-c.chunks:
			c.onChunk(ch)
		case <-c.resized:
			c.onResize()
		case id := <-c.switching:
			c.onSwitch(id)
		case raw := <-c.input:
			stopTimer(inputEscapeTimer)
			inputEscapeC = nil
			processInput(raw)
			armInputEscape()
		case <-inputEscapeC:
			inputEscapeC = nil
			literal := it.Flush()
			c.mu.Lock()
			toPanel := c.focus.IsPanel()
			c.mu.Unlock()
			if toPanel && bytes.Equal(literal, []byte{0x1b}) {
				c.onMenuKey(PanelKey{Kind: KeyEscape})
			} else {
				route(literal)
			}
		case <-panelEscapeC:
			panelEscapeC = nil
			c.menuHeld = nil
			c.onMenuKey(PanelKey{Kind: KeyEscape})
		case <-spinnerC:
			owner := spinnerOwner
			spinnerC = nil
			c.reduceMenu(MenuEvent{
				Kind: MenuEventTick, Attempt: owner.OperationAttempt,
				Generation: owner.PreviewGeneration,
			})
		case event := <-c.exited:
			// A child's pump delivers every chunk before it closes Exited, but
			// select may choose the exit channel while those chunks are already
			// queued. Drain them before removing the pane so its final output is
			// not discarded as belonging to an unknown child (BR-35).
			c.drainChunks()
			if c.onExit(event) {
				return event.code
			}
		case <-c.refreshRequests:
			c.advanceMenuRefresh(RefreshScheduleEvent{Kind: RefreshRequested})
		case result := <-c.refreshResults:
			c.finishMenuRefresh(result)
		case result := <-c.previewResults:
			c.finishMenuPreview(result)
		case completed := <-c.operationQueue.results:
			if c.finishOperation(completed) {
				continue
			}
		case <-terminated:
			return 0
		case <-c.stop:
			return 0
		}
		syncSpinner()
	}
}

func (c *Console) drainChunks() {
	for {
		select {
		case ch := <-c.chunks:
			c.onChunk(ch)
		default:
			return
		}
	}
}

// teardown is the one lifecycle owner: restore the visible terminal while it
// is still writable/raw, stop every event source, close the blocking input seam,
// and join every worker before Run returns.
func (c *Console) teardown(restore func() error) {
	c.release()
	if err := restore(); err != nil {
		fmt.Fprintf(c.errw(), "couch: restore terminal: %v\n", err)
	}
	c.Stop()
	if err := c.host.Close(); err != nil {
		fmt.Fprintf(c.errw(), "couch: close terminal host: %v\n", err)
	}
	if closer, ok := c.stdin.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			fmt.Fprintf(c.errw(), "couch: close terminal input: %v\n", err)
		}
	}
	// Pair with installObservedThreadActor's under-mutex Add. Once this barrier
	// passes, Stop is closed and no later attach can increment the WaitGroup.
	c.mu.Lock()
	c.mu.Unlock()
	c.workers.Wait()
}

// onExit removes a dead child from the console and registry. An active exit
// lands on the panel; an inactive exit only repaints the notice so it cannot
// steal the operator from the child they are typing in. The final child ends
// an actor-focused console, but an already panel-focused console stays up so a
// completed Park can expose the durable row that Enter resumes.
func (c *Console) onExit(event childExit) bool {
	c.mu.Lock()
	p, known := c.panes[event.id]
	if !known {
		last := len(c.panes) == 0
		c.mu.Unlock()
		return last
	}
	wasFocused := c.focus == FocusActor(event.id)
	panelFocused := c.focus.IsPanel()
	wasActive := c.active == event.id
	delete(c.panes, event.id)
	for i, id := range c.order {
		if id == event.id {
			c.order = append(c.order[:i:i], c.order[i+1:]...)
			break
		}
	}
	if event.id == c.root {
		c.root = ""
		if len(c.order) > 0 {
			c.root = c.order[0]
		}
	}
	if wasActive {
		// Panel actions address the active actor, not merely the highlighted
		// durable row. Preserve that invariant after either the root or a
		// non-root active actor exits by falling back to the current root.
		c.active = c.root
	}
	if wasFocused {
		c.focus = FocusPanel()
	}
	expected := c.consumeExpectedParkExitLocked(event.id, p.thread)
	if !expected {
		exitNotice := ExitNotice(p.actorID, p.label, event.code)
		c.feed.Push(exitNotice)
	}
	forget := c.forget
	last := len(c.panes) == 0
	c.mu.Unlock()

	if forget != nil {
		if err := forget(p.tree, p.actorID); err != nil {
			c.setNotice(fmt.Sprintf("forget %s: %v", p.label, err))
		}
	}
	c.requestMenuRefresh()
	if last && !panelFocused {
		return true
	}
	if wasFocused || panelFocused {
		c.showMenu()
	} else {
		c.paintNow()
	}
	return false
}

// release puts the terminal back: region reset, then the reserved row cleared,
// so the operator's shell does not inherit a pinned region or a stale row.
func (c *Console) release() {
	c.mu.Lock()
	rows := c.size.Rows
	c.mu.Unlock()
	// Teardown writes UNCONDITIONALLY: a half-restored terminal is worse than a
	// spliced sequence, and the child is finished with the screen by now.
	_, _ = io.WriteString(c.host,
		Release()+PaintRow(rows, "")+hostty.ResetInteractiveModes+hostty.LeaveAltScreen+hostty.ResetRegion+hostty.ShowCursor)
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
	model := StatusModel{Notice: c.feed.Latest()}
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
			c.feed.Push(BellNotice(p.actorID, p.label))
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
	c.mu.Lock()
	menuFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if menuFocused {
		c.showMenu()
		return
	}
	c.applyLayout()
	c.repaint()
}

// pumpStdin is the one blocking reader. It hands raw chunks to Run, which owns
// framing, ambiguity timers, focus transitions, and routing in one event loop.
func (c *Console) pumpStdin() {
	buf := make([]byte, 4096)
	for {
		n, err := c.stdin.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			select {
			case c.input <- chunk:
			case <-c.stop:
				return
			}
		}
		if err != nil {
			return
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
	if cur.IsPanel() {
		c.onMenuKey(PanelKey{Kind: KeyCtrlSpace})
		return
	}

	next := Up(cur, root, c.actorAlive)
	if next == cur {
		return // already at the top
	}

	c.mu.Lock()
	c.focus = next
	c.mu.Unlock()

	if next.IsPanel() {
		c.requestMenuRefresh()
		c.showMenu()
		return
	}
	c.onSwitch(next.Actor())
}

// onParkHotkey handles Pair's Alt+x chord at the Couch ownership boundary.
// It renders confirmation immediately; durable park work starts only after
// confirmation and runs off the terminal event loop.
func (c *Console) onParkHotkey() {
	c.mu.Lock()
	p := c.panes[c.active]
	isRoot := c.active != "" && c.active == c.root
	if p != nil {
		c.focus = FocusPanel()
		c.menu.ActiveAddress = p.thread
	}
	c.mu.Unlock()
	if p == nil {
		c.setNotice("park: no active thread")
		return
	}
	operation := "park"
	if isRoot {
		operation = "leave"
	}
	c.reduceMenu(MenuEvent{Kind: MenuEventParkHotkey, Operation: operation, Address: p.thread})
}

// actorAlive is the liveness predicate Up consults.
func (c *Console) actorAlive(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.panes[id]
	return ok && !p.child.Done()
}

func (c *Console) runMenuOperation(effect MenuEffect) {
	c.mu.Lock()
	fn := c.ops
	origin := c.menu.InFlight
	c.mu.Unlock()
	if fn == nil {
		c.finishOperation(operationCompletion{
			name: effect.Operation, origin: origin, err: errors.New("no action dispatcher wired"),
		})
		return
	}
	if origin.Operation != effect.Operation || origin.Attempt == 0 || origin.Attempt != effect.Attempt {
		return
	}
	requestArgs := cloneOperationArgs(effect.Args)
	key := fmt.Sprintf("menu\x00%d\x00%s", effect.Attempt, effect.Operation)
	_, err := c.operationQueue.Enqueue(operationRequest{key: key, name: effect.Operation, origin: origin, run: func() (any, error) {
		operationContext, cancelOperation := context.WithCancel(c.lifetime)
		defer cancelOperation()
		return fn(couchcore.OperationCall{Name: effect.Operation, Args: requestArgs, Implicit: true, Context: operationContext})
	}})
	if err != nil {
		c.finishOperation(operationCompletion{key: key, name: effect.Operation, origin: origin, err: err})
	}
}

// finishOperation returns true when the completion requested Console exit.
func (c *Console) finishOperation(completed operationCompletion) bool {
	err := completed.err
	address := completed.origin.Address
	if parked, ok := completed.value.(couchcore.ParkResult); ok && parked.Thread.Address != (couchcore.ThreadAddress{}) {
		address = parked.Thread.Address
	}
	startedHandleID := ""
	if started, ok := completed.value.(couchcore.StartResult); ok {
		address = started.Record.Thread
		if started.Handle != nil {
			startedHandleID = started.Handle.ID()
		}
		if err == nil {
			c.mu.Lock()
			fn := c.ops
			c.mu.Unlock()
			if fn == nil {
				err = errors.New("no action dispatcher wired")
			} else {
				_, err = fn(couchcore.OperationCall{
					Name: "attach", Context: c.lifetime, Implicit: true, TypedPayload: started,
					Args: map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag)},
				})
			}
		}
	}
	event := MenuEvent{
		Kind: MenuEventOperationResult, Operation: completed.origin.Operation,
		Attempt: completed.origin.Attempt, Address: address, Success: err == nil,
	}
	if err != nil {
		event.Error = err.Error()
	}
	c.mu.Lock()
	if event.Success && operationNeedsProjectionRefresh(event.Operation) {
		event.ProjectionAfterGeneration = c.refreshSchedule.Sequence
	}
	if completed.origin.Operation == "park" && err == nil {
		for id, p := range c.panes {
			if p.thread == address {
				c.expectedExits[id] = true
			}
		}
	}
	if c.menuReady {
		c.menu, _ = ReduceMenu(c.menu, event)
	}
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if completed.origin.Operation == "leave" && err == nil {
		c.Stop()
		return true
	}
	if completed.origin.Operation == "resume" && err == nil && startedHandleID != "" {
		c.requestMenuRefresh()
		c.forceSwitch(startedHandleID)
		return false
	}
	c.requestMenuRefresh()
	if panelFocused {
		c.showMenu()
	}
	return false
}

// consumeExpectedParkExitLocked classifies only the exact child selected by a
// Park attempt as expected. It handles either event order: while the operation
// is in flight its immutable origin is authority; after successful completion
// the exact handle marker bridges until the child-exit event arrives.
func (c *Console) consumeExpectedParkExitLocked(id string, address couchcore.ThreadAddress) bool {
	if c.expectedExits[id] {
		delete(c.expectedExits, id)
		return true
	}
	origin := c.menu.InFlight
	return origin.Operation == "park" && origin.Attempt != 0 && origin.Address == address
}

func cloneOperationArgs(args map[string]string) map[string]string {
	copy := make(map[string]string, len(args))
	for key, value := range args {
		copy[key] = value
	}
	return copy
}

// ExecuteConsoleOperation is the owner-local executor for effects that cannot
// exist in couchcore: routing the human terminal and attaching its PTY.
func (c *Console) ExecuteConsoleOperation(call couchcore.OperationCall) (any, error) {
	address := couchcore.ThreadAddress{RepoScope: call.Args["repo-scope"], Tag: couchcore.ThreadTag(call.Args["tag"])}
	switch call.Name {
	case "switch":
		c.mu.Lock()
		target := c.switchTargetForAddressLocked(address)
		c.mu.Unlock()
		if target == "" {
			return nil, fmt.Errorf("thread %s/%s is not attached to this console", address.RepoScope, address.Tag)
		}
		c.forceSwitch(target)
		return address, nil
	case "attach":
		start, ok := call.TypedPayload.(couchcore.StartResult)
		if !ok {
			return nil, fmt.Errorf("attach requires a typed start result")
		}
		if start.Record.Thread != address {
			return nil, fmt.Errorf("attach address does not match started thread")
		}
		if start.Handle == nil || start.Record.PID != start.Handle.PID() || start.Record.Identity != start.Handle.Identity() {
			return nil, fmt.Errorf("attach record/handle process identity mismatch")
		}
		th, ok := start.Handle.(couchcore.TerminalHandle)
		if !ok {
			return nil, fmt.Errorf("child has no terminal to attach")
		}
		ctx := call.Context
		if ctx == nil {
			ctx = c.lifetime
		}
		if err := c.installObservedThreadActor(ctx, start.Handle.ID(), start.Record.ID, start.Record.Thread,
			start.Record.Args.Worktree, start.Record.Args.Worktree.Repo(), th.Terminal(),
			couchcore.ProcessIdentity{PID: start.Handle.PID(), Identity: start.Handle.Identity()}); err != nil {
			return nil, err
		}
		return address, nil
	default:
		return nil, fmt.Errorf("%s is not a console-local operation", call.Name)
	}
}

func (c *Console) switchTargetForAddressLocked(address couchcore.ThreadAddress) string {
	for _, id := range c.order {
		pane := c.panes[id]
		if pane != nil && pane.thread == address && !pane.child.Done() {
			return id
		}
	}
	return ""
}

func (c *Console) setNotice(text string) {
	c.mu.Lock()
	c.feed.Push(Notice{Kind: "status", Body: text})
	c.mu.Unlock()
}
