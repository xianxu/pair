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
	id                string
	batch             ptychild.OutputBatch
	focusedAtDelivery bool
	ack               chan struct{}
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

	// rowDirty is the same shape for the reserved row: an INACTIVE pane's
	// erase or margin reset is real, it just cannot be acted on yet. The first
	// version consumed the child's latch for every pane and acted on it only
	// for the active one, so a background child's damage was thrown away and
	// attaching to it would land on a screen with no status row.
	rowDirty bool

	// replayCutoff advances only after Run has processed a delivered batch.
	// A takeover therefore cannot replay bytes still queued behind the switch.
	replayCutoff uint64
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

	// focus is what the terminal is pointed at. It is not the same as `active`:
	// the switcher is a focus with no actor behind it.
	focus Focus

	actionable ActionableThreadProvider
	menu       MenuState
	menuReady  bool

	// tracker is where ctrl+backspace goes. Ephemeral by design: `previous` is
	// a property of this sitting, not of the durable thread store.
	tracker SwitchTracker

	// menuHeld carries a partial escape sequence across reads.
	menuHeld []byte

	// Ops dispatches an operator action. Injected so the console never learns
	// what an operation IS -- it names one and couchcore runs it, which is
	// what keeps the panel from growing a private verb (#148's design test).
	ops       func(couchcore.OperationCall) (any, error)
	forget    func(couchcore.Worktree, couchcore.ActorID) error
	feed      *Feed
	attention AttentionLedger
	size      ptychild.Size
	// expectedExits are exact child handles whose successful Park already
	// authorized shutdown. They bridge the race between the child-exit channel
	// and the asynchronous operation-completion channel.
	expectedExits map[string]bool

	// paintPending means a repaint was wanted while the host stream was
	// mid-sequence, and is owed as soon as it is safe.
	paintPending bool
	// deferredNotifications holds batch suffixes whose first part is a
	// notification that cannot yet be inserted into the outer host stream.
	// Keeping the original acknowledgement open backpressures that source actor
	// while other actors and the focused UI continue independently.
	deferredNotifications []chunk
	flushingNotifications bool

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
	switching chan string
	input     chan []byte
	// trace is nil unless COUCH_INPUT_TRACE names a file; see inputtrace.go.
	trace              *inputTracer
	exited             chan childExit
	operationQueue     *operationQueue
	refreshRequests    chan struct{}
	refreshResults     chan menuRefreshResult
	refreshSchedule    RefreshSchedule
	previewResults     chan menuPreviewResult
	previewSchedule    PreviewSchedule
	previewCancel      context.CancelFunc
	previewRunning     uint64
	directoryReader    DirectoryBatchReader
	completionResults  chan menuCompletionResult
	completionSchedule latestSchedule[CompletionRequest]
	completionCancel   context.CancelFunc
	completionRunning  CompletionIdentity
	lifetime           context.Context
	cancelLifetime     context.CancelFunc
	stop               chan struct{}
	once               sync.Once
	workers            sync.WaitGroup
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
		host:              host,
		stdin:             stdin,
		panes:             map[string]*pane{},
		chunks:            make(chan chunk, 256),
		resized:           make(chan struct{}, 1),
		switching:         make(chan string, 8),
		input:             make(chan []byte, 64),
		trace:             newInputTracer(),
		exited:            make(chan childExit, 64),
		operationQueue:    newOperationQueue(16),
		refreshRequests:   make(chan struct{}, 1),
		refreshResults:    make(chan menuRefreshResult, 1),
		previewResults:    make(chan menuPreviewResult, 1),
		directoryReader:   OSDirectoryBatchReader{},
		completionResults: make(chan menuCompletionResult, 1),
		expectedExits:     map[string]bool{},
		lifetime:          lifetime,
		cancelLifetime:    cancelLifetime,
		stop:              make(chan struct{}),
		feed:              NewFeed(8),
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
func (c *Console) Deliver(id string, batch ptychild.OutputBatch) {
	c.mu.Lock()
	focused := c.focus == FocusActor(id)
	c.mu.Unlock()
	ack := make(chan struct{})
	select {
	case c.chunks <- chunk{id: id, batch: batch, focusedAtDelivery: focused, ack: ack}:
		select {
		case <-ack:
		case <-c.stop:
		}
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
	c.panes[handleID] = &pane{
		tree: tree, thread: thread, process: process, actorID: actorID,
		label: label, child: child, replayCutoff: child.ReplaySafeEnd(),
	}
	c.order = append(c.order, handleID)
	if c.active == "" {
		c.active = handleID
		c.focus = FocusActor(handleID)
		// The first attach lands the operator on an actor WITHOUT going through
		// switchTo, so the tracker has to be seeded here or the actor they
		// started in is never recorded -- and the first notification hop would
		// then pin nothing instead of pinning it.
		c.tracker.Switch(thread, false)
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
func (c *Console) onSwitch(id string) { c.switchTo(id, false, arrivalOrdinary) }

// forceSwitch repaints even when the actor is already active -- which is the
// case when returning from the panel, where the SCREEN changed but the active
// actor did not.
func (c *Console) forceSwitch(id string) { c.switchTo(id, true, arrivalOrdinary) }

// arrival says HOW the operator landed on an actor. It is not decoration: the
// switch rule keys off it (only a notification hop is non-pinning), while the
// notification-acknowledgement rule deliberately does not (every landing clears
// the bell). Separating them is what keeps ctrl+backspace home from leaving the
// actor the operator is sitting in marked as still paging.
type arrival uint8

const (
	// arrivalOrdinary is a switch that is not a notification hop: the switcher's
	// Enter on an unpaged row, a post-start attach, a programmatic Switch.
	arrivalOrdinary arrival = iota
	// arrivalNotification is ctrl-space + Return on an actor that HAD a pending
	// notification. Only this one is non-pinning.
	arrivalNotification
	// arrivalPrevious is ctrl+backspace. Never a notification hop even when the
	// actor happens to be paging, because the operator is going home.
	arrivalPrevious
)

func (a arrival) viaNotification() bool { return a == arrivalNotification }

// switchTo is the funnel every landing on an actor goes through, and it owes
// two rules on each one:
//
//  1. record the landing in the switch tracker, so ctrl+backspace knows where
//     home is; and
//  2. acknowledge the landed actor's pending notifications, because the Spec's
//     rule is that an actor does not notify while the operator is attached to
//     it.
//
// Rule 2 used to live in the ctrl-space home-landing path, which #170 deleted.
// Leaving it there would have meant ctrl+backspace home to A lands on an A that
// is still lit, NewestActor() then names the actor the operator is SITTING IN,
// and the next ctrl-space opens the switcher on it instead of on whoever paged
// -- the headline behaviour, inverted.
func (c *Console) switchTo(id string, force bool, how arrival) {
	c.mu.Lock()
	p, known := c.panes[id]
	already := c.active == id && !force
	if known {
		c.active = id
		c.focus = FocusActor(id)
		c.menu.ActiveAddress = p.thread
		p.rowDirty = false
		// Unconditional: SwitchTracker itself ignores a landing on the actor
		// already current, so the rule stays in one place rather than being
		// half-enforced by whichever caller remembered.
		c.tracker.Switch(p.thread, how.viaNotification())
		// Whatever brought the operator here, they are here now.
		c.attention.Acknowledge(c.attention.Capture(p.thread))
		c.syncAttentionLocked()
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
	c.takeOverScreen(p.child.ReplayThrough(p.replayCutoff))
	c.flushDeferredNotifications()
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
			// Exhaustive on purpose. A `default: c.onHotkey()` would turn any
			// hit the console does not yet handle into "open the switcher" --
			// so the moment M2 registers alt+d, pressing it would silently open
			// the switcher until someone remembered to touch this switch too.
			switch hit {
			case HitSwitch:
				c.onHotkey()
			case HitPark:
				c.onParkHotkey()
			case HitPrevious:
				c.onPreviousHotkey()
			case HitDetach:
				c.onDetachHotkey()
			case HitRelaunch:
				c.onRelaunchHotkey()
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
		case result := <-c.completionResults:
			c.finishMenuCompletion(result)
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
	// Not a landing: on exit the operator goes to the panel, so recording one
	// would make the dead thread the return target -- the single place
	// ctrl+backspace can never usefully go.
	c.tracker.Drop(p.thread)
	c.attention.DropActor(p.thread)
	c.syncAttentionLocked()
	for i, id := range c.order {
		if id == event.id {
			c.order = append(c.order[:i:i], c.order[i+1:]...)
			break
		}
	}
	if wasActive {
		// Panel actions address the active actor, not merely the highlighted
		// durable row, so the active slot has to keep naming a live actor after
		// one exits. c.order is attach order and has already had the dead id
		// removed, so its head is the surviving actor to fall back to; empty
		// order correctly leaves no active actor at all.
		c.active = ""
		if len(c.order) > 0 {
			c.active = c.order[0]
		}
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
	c.hostScan.FeedFraming(p)
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
			Bell:   len(c.attention.Projection(p.thread)) > 0,
		})
	}
	c.mu.Unlock()

	c.writeOwn(Reserve(rows) + PaintRow(rows, RenderStatusRow(cols, model)))
}

func (c *Console) syncAttentionLocked() {
	projection := make(map[couchcore.ThreadAddress][]AttentionMessage)
	for _, p := range c.panes {
		if messages := c.attention.Projection(p.thread); len(messages) > 0 {
			projection[p.thread] = messages
		}
	}
	c.menu.Attention = projection
}

// onChunk routes one child write.
func (c *Console) onChunk(ch chunk) {
	ackHere := ch.ack != nil
	if ch.ack != nil {
		defer func() {
			if ackHere {
				close(ch.ack)
			}
		}()
	}
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

	parts := ch.batch.Parts
	if len(parts) == 0 && len(ch.batch.Raw) > 0 && ch.batch.RingEnd == 0 && ch.batch.ReplaySafeEnd == 0 {
		parts = []ptychild.OutputPart{{Bytes: ch.batch.Raw}}
	}
	attentionChanged := false
	for i, part := range parts {
		if len(part.Bytes) > 0 && isActive {
			c.writeChild(part.Bytes)
		}
		if part.Notification != nil {
			c.mu.Lock()
			unsafe := c.hostScan.MidSequence()
			if unsafe {
				ch.batch.Parts = append([]ptychild.OutputPart(nil), parts[i:]...)
				c.deferredNotifications = append(c.deferredNotifications, ch)
			}
			c.mu.Unlock()
			if unsafe {
				ackHere = false
				return
			}
			// Pair's envelope is still valid outer-terminal OSC. Couch observes
			// it but does not swallow it.
			c.writeChild(part.Notification.Raw)
			if !ch.focusedAtDelivery {
				c.mu.Lock()
				c.attention.Mark(p.thread, part.Notification.Message)
				c.syncAttentionLocked()
				c.mu.Unlock()
				attentionChanged = true
			}
		}
	}
	c.mu.Lock()
	if ch.batch.ReplaySafeEnd > p.replayCutoff {
		p.replayCutoff = ch.batch.ReplaySafeEnd
	}
	c.mu.Unlock()
	// A paint deferred while the stream was mid-sequence is owed as soon as
	// the stream is whole again.
	c.mu.Lock()
	owed := c.paintPending && !c.hostScan.MidSequence()
	c.mu.Unlock()
	if owed {
		c.paintNow()
	}
	c.flushDeferredNotifications()
	if attentionChanged {
		c.repaint()
	}
	// Derived state is consumed whether or not the child is on screen.
	if ch.batch.RowDirty {
		c.mu.Lock()
		p.rowDirty = true
		c.mu.Unlock()
	}
	if ch.batch.Bell {
		c.mu.Lock()
		// An actor the operator is already looking at is not "wanting" them.
		if !isActive {
			c.attention.Mark(p.thread, "")
			c.syncAttentionLocked()
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

// flushDeferredNotifications releases arrival-ordered batch suffixes once
// inserting another actor's OSC cannot corrupt a partial host sequence.
func (c *Console) flushDeferredNotifications() {
	c.mu.Lock()
	if c.flushingNotifications || c.hostScan.MidSequence() || len(c.deferredNotifications) == 0 {
		c.mu.Unlock()
		return
	}
	c.flushingNotifications = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.flushingNotifications = false
		c.mu.Unlock()
	}()

	for {
		c.mu.Lock()
		if c.hostScan.MidSequence() || len(c.deferredNotifications) == 0 {
			c.mu.Unlock()
			return
		}
		deferred := c.deferredNotifications[0]
		c.deferredNotifications = c.deferredNotifications[1:]
		c.mu.Unlock()
		c.onChunk(deferred)
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
			c.trace.record(chunk)
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

// onHotkey handles ctrl-space: OPEN THE SWITCHER, and nothing else.
//
// It used to mean "up one level" -- child to root actor, root actor to panel.
// That ladder is gone (#170), and with it the root-actor/home concept: one key
// now has one meaning wherever it is pressed from an actor. The panel keeps its
// own ctrl-space (the global start form), which is not a rung of the deleted
// ladder but the panel's own binding, and remains the only route to starting a
// thread.
//
// Runs on the Run goroutine.
func (c *Console) onHotkey() {
	c.mu.Lock()
	cur := c.focus
	c.mu.Unlock()
	if cur.IsPanel() {
		c.onMenuKey(PanelKey{Kind: KeyCtrlSpace})
		return
	}

	c.mu.Lock()
	c.focus = FocusPanel()
	// Open focused on whoever paged. This used to run only when the ladder
	// happened to land on the panel; now it is the point of the key, so it runs
	// on every ctrl-space from an actor.
	if len(c.menu.Frames) > 0 {
		focus := c.attention.NewestActor()
		if focus == (couchcore.ThreadAddress{}) {
			// The defined default with nothing paging: the thread being left.
			// Routed through reconcileRootSelection rather than assigned, because
			// ActiveAddress can name a thread that is no longer in the inventory
			// -- and that reconciler already means "preferred if present, else
			// the first visible row".
			focus = c.menu.ActiveAddress
		}
		c.menu.Frames = c.menu.Frames[:1]
		c.menu.Frames[0].Filter = ""
		reconcileRootSelection(&c.menu, focus)
	}
	c.mu.Unlock()

	c.requestMenuRefresh()
	c.showMenu()
}

// onPreviousHotkey handles ctrl+backspace: return to the actor the operator was
// working in before they were paged away.
//
// Runs on the Run goroutine. Resolving through the durable address rather than a
// remembered pane id is what lets `previous` survive a park/resume or
// detach/reattach cycle, which mints a new pane for the same thread.
func (c *Console) onPreviousHotkey() {
	c.mu.Lock()
	address, ok := c.tracker.Previous()
	target := ""
	if ok {
		target = c.switchTargetForAddressLocked(address)
	}
	c.mu.Unlock()

	if !ok {
		c.reportPrevious("previous: nowhere to return to")
		return
	}
	if target == "" {
		// The thread is durable but has no live pane here -- parked, detached,
		// or exited. Saying so beats blanking the screen or silently doing
		// nothing.
		c.reportPrevious("previous: that thread is no longer attached")
		return
	}
	c.switchTo(target, true, arrivalPrevious)
}

// reportPrevious puts a ctrl+backspace refusal where the operator is actually
// looking. The status row is behind the panel while the switcher owns the
// screen, so a setNotice there would make the key silently do nothing -- which
// is exactly what the operator would report as the bug.
func (c *Console) reportPrevious(text string) {
	c.mu.Lock()
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel {
		c.reduceMenu(MenuEvent{Kind: MenuEventNotice, Error: text})
		return
	}
	c.setNotice(text)
}

// reportLeave writes a final line to the operator's terminal on the way out.
//
// The console is being torn down, so this goes to stderr rather than through
// the menu: by the time it runs, the surface that would have rendered a notice
// is about to stop existing.
func (c *Console) reportLeave(result couchcore.LeaveResult) {
	if c.stderr == nil {
		return
	}
	if len(result.Detached) > 0 {
		fmt.Fprintf(c.stderr, "couch: detached %d thread(s); their agents keep running\n", len(result.Detached))
	}
	if len(result.Parked) > 0 {
		if result.Disposition == couchcore.LeavePark {
			fmt.Fprintf(c.stderr, "couch: parked %d thread(s); their agents were stopped\n", len(result.Parked))
		} else {
			fmt.Fprintf(c.stderr, "couch: parked %d thread(s) that were already shutting down\n", len(result.Parked))
		}
	}
	for _, address := range result.Skipped {
		fmt.Fprintf(c.stderr, "couch: left %s occupied — its state could not be proved detachable\n", address.Tag)
	}
}

// onDetachHotkey handles Pair's Alt+d chord at the Couch ownership boundary.
//
// No confirmation at either scope, unlike park: detach destroys nothing -- the
// agent keeps running behind its zellij session and only the client goes.
// Making the safe gesture cheap and the destructive one deliberate is the whole
// point of having both.
func (c *Console) onDetachHotkey() {
	c.mu.Lock()
	isPanel := c.focus.IsPanel()
	p := c.panes[c.active]
	if !isPanel && p != nil {
		// Detaching an actor lands the operator in the switcher, exactly as
		// park does. That is also what keeps couch alive when the LAST actor
		// detaches: an actor-focused console exits with its final child, and
		// the safe gesture must never be the one that ends the session by
		// accident.
		c.focus = FocusPanel()
		c.menu.ActiveAddress = p.thread
	}
	c.mu.Unlock()

	if isPanel {
		// The switcher IS couch, so the key applies to every live thread and
		// then leaves. Unconditional: a switcher with nothing live must still
		// have a way out, which is the trap this replaced (#170).
		c.reduceMenu(MenuEvent{
			Kind: MenuEventParkHotkey, Operation: "leave", Mode: string(couchcore.LeaveDetach),
		})
		return
	}
	if p == nil {
		c.setNotice("detach: no attached thread")
		return
	}
	c.reduceMenu(MenuEvent{Kind: MenuEventParkHotkey, Operation: "detach", Address: p.thread})
}

// onRelaunchHotkey handles Alt+n and Ctrl+Alt+n: replace this thread's Pair
// process with the current binary, keeping the agent conversation.
//
// Scope follows focus, WITH one deviation that is the point rather than an
// oversight. Alt+x and Alt+d mean "what you are looking at": one actor from an
// actor, every live thread from the switcher. Relaunch has no whole-couch form
// -- that is alt+d, rebuild, re-run couch, the symmetry this completes -- so
// from the panel it relaunches the HIGHLIGHTED ROW and leaves the operator in
// the switcher, and from an actor it relaunches that actor.
//
// Runs on the Run goroutine.
func (c *Console) onRelaunchHotkey() {
	c.mu.Lock()
	isPanel := c.focus.IsPanel()
	p := c.panes[c.active]
	target := couchcore.ThreadAddress{}
	switch {
	case isPanel:
		target = c.menu.CurrentFrame().SelectedAddress
	case p != nil:
		target = p.thread
		// An actor relaunch shows its progress on the panel, as park does, until
		// the holding surface exists to keep the operator in place.
		c.focus = FocusPanel()
		c.menu.ActiveAddress = p.thread
	}
	c.mu.Unlock()

	if target == (couchcore.ThreadAddress{}) {
		c.setNotice("relaunch: no thread selected")
		return
	}
	c.reduceMenu(MenuEvent{Kind: MenuEventParkHotkey, Operation: "relaunch", Address: target})
}

// onParkHotkey handles Pair's Alt+x chord at the Couch ownership boundary.
// It renders confirmation immediately; durable park work starts only after
// confirmation and runs off the terminal event loop.
func (c *Console) onParkHotkey() {
	c.mu.Lock()
	// Alt+x parks what you are looking at: one actor from an actor, every live
	// thread from couch's own switcher. Scope comes from the focus we already
	// have rather than being special-cased per key.
	isPanel := c.focus.IsPanel()
	p := c.panes[c.active]
	if !isPanel && p != nil {
		c.focus = FocusPanel()
		c.menu.ActiveAddress = p.thread
	}
	c.mu.Unlock()

	if isPanel {
		// Deliberately BEFORE the no-active-thread check: leaving couch needs
		// no live actor, and an all-detached couch is the normal state to quit
		// from. Park's whole-couch form stops every agent, so it keeps the
		// confirmation its per-thread form has -- the disposition carries the
		// confirmation, not the scope.
		c.reduceMenu(MenuEvent{
			Kind: MenuEventParkHotkey, Operation: "leave", Mode: string(couchcore.LeavePark),
		})
		return
	}
	if p == nil {
		c.setNotice("park: no active thread")
		return
	}
	c.reduceMenu(MenuEvent{Kind: MenuEventParkHotkey, Operation: "park", Address: p.thread})
}

func (c *Console) runMenuOperation(effect MenuEffect) {
	c.mu.Lock()
	fn := c.ops
	origin := c.menu.InFlight
	if origin.Operation == "switch" && origin.AttentionCapture == 0 {
		origin.AttentionCapture = c.attention.Capture(origin.Address)
		c.menu.InFlight.AttentionCapture = origin.AttentionCapture
	}
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
	// StartedChild, not a StartResult type assertion: relaunch returns its own
	// result struct around the same child, and asserting the concrete type left
	// that child spawned but never adopted.
	if child, ok := completed.value.(couchcore.StartedChild); ok {
		if started, hasChild := child.Started(); hasChild {
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
	}
	event := MenuEvent{
		Kind: MenuEventOperationResult, Operation: completed.origin.Operation,
		Attempt: completed.origin.Attempt, Address: address, Success: err == nil,
	}
	if err != nil {
		event.Error = err.Error()
	}
	c.mu.Lock()
	if completed.origin.Operation == "switch" {
		// Success is acknowledged by switchTo, which is the only place that
		// knows a landing actually happened -- two authorities for one rule is
		// how they drift. Failure still has to release the capture here,
		// because no landing occurred to do it.
		if !event.Success {
			c.attention.Cancel(completed.origin.AttentionCapture)
			c.syncAttentionLocked()
		}
	}
	if event.Success && operationNeedsProjectionRefresh(event.Operation) {
		event.ProjectionAfterGeneration = c.refreshSchedule.Sequence
	}
	// A lifecycle operation's child exit is expected in EITHER event order.
	// c.exited and c.operationQueue.results are separate select cases and Go
	// picks uniformly among ready ones, so roughly half the time the completion
	// wins the race, ReduceMenu clears InFlight, and the exit falls through to
	// consumeExpectedParkExitLocked's InFlight arm with nothing to match. This
	// bridge is the other half of that rule, and detach needs it as much as
	// park does -- both end their child deliberately.
	if err == nil && endsItsOwnChild(completed.origin.Operation) {
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
		// Report what leave actually did before the terminal goes. Skipped
		// threads are the ones that matter: Couch could not prove them
		// detachable, so it deliberately did NOT park them, and they stay
		// occupied. Told here, that is a fact; discovered later, it is a
		// mystery occupied thread.
		if result, ok := completed.value.(couchcore.LeaveResult); ok {
			c.reportLeave(result)
		}
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

// endsItsOwnChild names the operations whose child exit is EXPECTED, so the two
// sites that need the answer cannot disagree.
//
// They existed as two hand-written lists -- the expectedExits bridge and the
// switch below -- because the exit/completion race resolves in either order and
// each half needed the same fact. A third operation had to appear in both or the
// operator gets a spurious child-exited notice for work they asked for; deriving
// it is what stops the next one being added to one list only (ARCH-DRY).
func endsItsOwnChild(operation string) bool {
	switch operation {
	case "park", "detach", "relaunch":
		return true
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
	if origin.Attempt == 0 {
		return false
	}
	switch origin.Operation {
	case "park", "detach", "relaunch":
		return origin.Address == address
	case "leave":
		// Leave detaches every thread, so every child exit it causes is
		// expected. Without this the operator gets a burst of exit notices on
		// the way out -- exactly the noise the notification design exists to
		// keep meaningful.
		return true
	}
	return false
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
		// A notification hop is ctrl-space + Return on an actor that HAD a
		// pending notification. runMenuOperation captured that set before
		// dispatch, so a nonzero capture is exactly "the target was paging when
		// the operator chose it" -- the value that was true when they chose,
		// not after.
		how := arrivalOrdinary
		if c.menu.InFlight.Operation == "switch" && c.menu.InFlight.Address == address &&
			c.menu.InFlight.AttentionCapture != 0 {
			how = arrivalNotification
		}
		c.mu.Unlock()
		if target == "" {
			return nil, fmt.Errorf("thread %s/%s is not attached to this console", address.RepoScope, address.Tag)
		}
		c.switchTo(target, true, how)
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
