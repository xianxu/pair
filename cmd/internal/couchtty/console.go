package couchtty

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
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
}

// Console routes the operator's terminal to one child at a time.
//
// It is the integration controller: reusable decisions live in pure functions
// in this package, while Console owns event ordering and transient UI
// transitions as it drives hostty.Host. It never calls x/term or os/signal
// directly, which keeps resize and teardown testable without a terminal.
type Console struct {
	host             hostty.Host
	stdin            io.Reader
	stderr           io.Writer
	presenter        *terminal.Presenter
	terminalCommands chan terminalCommand
	terminalFailure  error
	leaveReport      string
	ownedChildren    map[*ptychild.Child]struct{}

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
	// statusSpinner is the status row's spinner frame, advanced while the
	// reattach pass has a thread loading (pair#206).
	statusSpinner uint8

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

	// Run orders product input, output notifications, and focus transitions.
	// Presenter alone owns the host writer.
	//
	chunks    chan chunk
	resized   chan struct{}
	switching chan string
	input     chan []byte
	// trace is nil unless COUCH_INPUT_TRACE names a file; see inputtrace.go.
	trace *inputTracer
	// events is the COUCH_TRACE timing trace, nil unless the composition root
	// named a file; see trace.go. framePainted gates its first-frame event.
	events *eventTracer
	// mouseTrace is nil unless COUCH_MOUSE_TRACE names a file; see
	// mousetrace.go. It records couch's host mouse-mode decisions for #207.
	mouseTrace   *mouseTracer
	framePainted bool
	// menuExtents is where each actor was drawn by the LAST menu paint, so a
	// click resolves against what the operator saw rather than a re-render,
	// which a refresh or a notice could have changed in between.
	menuExtents []ActorExtent
	// statusChips is the same for the reserved row.
	statusChips []ChipSpan
	// mouseHit is the payload of the hit currently being dispatched.
	mouseHit MouseHit
	// started reports that Run owns the terminal, so a notice may paint itself.
	// Its own field rather than something inferred from another: "is it safe to
	// write to the operator's screen yet" is its own question.
	started              bool
	exited               chan childExit
	operationQueue       *operationQueue
	refreshRequests      chan struct{}
	refreshResults       chan menuRefreshResult
	refreshSchedule      RefreshSchedule
	orientationResults   chan orientationWatchResult
	orientationWatches   map[couchcore.ThreadAddress]orientationWatch
	continuationProvider ContinuationProvider
	continuationResults  chan continuationScanResult
	continuations        map[couchcore.ThreadAddress]continuationWatch
	previewResults       chan menuPreviewResult
	previewSchedule      PreviewSchedule
	previewCancel        context.CancelFunc
	previewRunning       uint64
	directoryReader      DirectoryBatchReader
	completionResults    chan menuCompletionResult
	completionSchedule   latestSchedule[CompletionRequest]
	completionCancel     context.CancelFunc
	completionRunning    CompletionIdentity
	lifetime             context.Context
	cancelLifetime       context.CancelFunc
	stop                 chan struct{}
	once                 sync.Once
	workers              sync.WaitGroup
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
		host:                host,
		presenter:           terminal.NewPresenter(host, terminal.CouchAnyMotion),
		terminalCommands:    make(chan terminalCommand, 16),
		stdin:               stdin,
		panes:               map[string]*pane{},
		chunks:              make(chan chunk, 256),
		resized:             make(chan struct{}, 1),
		switching:           make(chan string, 8),
		input:               make(chan []byte, 64),
		exited:              make(chan childExit, 64),
		operationQueue:      newOperationQueue(16),
		refreshRequests:     make(chan struct{}, 1),
		refreshResults:      make(chan menuRefreshResult, 1),
		orientationResults:  make(chan orientationWatchResult, 8),
		continuationResults: make(chan continuationScanResult, 1),
		continuations:       make(map[couchcore.ThreadAddress]continuationWatch),
		previewResults:      make(chan menuPreviewResult, 1),
		directoryReader:     OSDirectoryBatchReader{},
		completionResults:   make(chan menuCompletionResult, 1),
		expectedExits:       map[string]bool{},
		lifetime:            lifetime,
		cancelLifetime:      cancelLifetime,
		stop:                make(chan struct{}),
		feed:                NewFeed(8, time.Now, NoticeLifetime),
	}
	if s, err := host.Size(); err == nil {
		c.size = s
	}
	return c
}

// SetInputTrace opens the operator-keystroke probe at path, or turns it off when
// path is empty. Called by the composition root with the environment's value, so
// that a constructor never reaches for ambient env and a test never opens a real
// file it did not ask for.
//
// A failed open is REPORTED, at control priority, rather than silently tracing
// nothing: an empty trace would otherwise read as "no bytes arrived", the exact
// ambiguity the probe exists to remove.
func (c *Console) SetInputTrace(path string, options ...diagnosticlog.Options) error {
	tracer, err := newInputTracer(path, options...)
	c.mu.Lock()
	previous := c.trace
	c.trace = tracer
	c.mu.Unlock()
	_ = previous.Close()
	if err != nil {
		c.publishNotice(Notice{Kind: "trace", Control: true, Body: err.Error()})
	}
	return err
}

// SetMouseTrace opens the host mouse-mode probe at path, or turns it off when
// path is empty (#207). Same shape as SetInputTrace: the composition root
// passes the environment's value so a constructor never reaches for ambient env
// and a test never opens a file it did not ask for.
func (c *Console) SetMouseTrace(path string, options ...diagnosticlog.Options) error {
	tracer, err := newMouseTracer(path, options...)
	c.mu.Lock()
	previous := c.mouseTrace
	c.mouseTrace = tracer
	c.mu.Unlock()
	_ = previous.Close()
	if err != nil {
		c.publishNotice(Notice{Kind: "trace", Control: true, Body: err.Error()})
	}
	return err
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
	return ptychild.Size{Rows: bottomReservation(c.size.Rows).ChildRows(), Cols: c.size.Cols}
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
func (c *Console) Deliver(ctx context.Context, id string, batch ptychild.OutputBatch) error {
	c.mu.Lock()
	focused := c.focus == FocusActor(id)
	c.mu.Unlock()
	ack := make(chan struct{})
	select {
	case c.chunks <- chunk{id: id, batch: batch, focusedAtDelivery: focused, ack: ack}:
		select {
		case <-ack:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stop:
			return context.Canceled
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return context.Canceled
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
	if err := c.installObservedThreadActor(c.lifetime, handleID, actorID, thread, tree, label, child, process, false); err != nil && child != nil {
		// This compatibility entry point has no StartResult rollback owner. It
		// consumes a rejected fresh child, but never an already accepted lifetime.
		c.mu.Lock()
		_, accepted := c.ownedChildren[child]
		c.mu.Unlock()
		if !accepted {
			_ = child.Close()
		}
	}
}

// installObservedThreadActor transfers terminal disposal ownership only on
// success. A rejected typed start remains owned by its AbortStarted rollback.
// It commits a complete pane or no pane. The worker
// count is reserved under the same mutex as routing state, so teardown's mutex
// barrier cannot begin its final Wait between a partial map insertion and the
// exit watcher becoming owned.
func (c *Console) installObservedThreadActor(ctx context.Context, handleID string, actorID couchcore.ActorID, thread couchcore.ThreadAddress, tree couchcore.Worktree, label string, child *ptychild.Child, process couchcore.ProcessIdentity, background bool) error {
	if ctx == nil {
		ctx = c.lifetime
	}
	if handleID == "" || actorID == "" || child == nil {
		return errors.New("attach requires complete handle, actor, and terminal identities")
	}
	if child.Endpoint().InputEnded() {
		return errors.New("attach terminal input has ended")
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
		if installed.thread == thread && !installed.child.Endpoint().InputEnded() {
			c.mu.Unlock()
			return fmt.Errorf("thread %s/%s is already attached", thread.RepoScope, thread.Tag)
		}
	}
	desired := ptychild.Size{Rows: bottomReservation(c.size.Rows).ChildRows(), Cols: c.size.Cols}
	if child.Size() != desired {
		if err := child.Resize(desired); err != nil {
			c.mu.Unlock()
			return err
		}
	}
	if err := c.presenter.Register(ctx, child.Endpoint()); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.ownedChildren == nil {
		c.ownedChildren = make(map[*ptychild.Child]struct{})
	}
	c.ownedChildren[child] = struct{}{}
	c.workers.Add(1)
	c.panes[handleID] = &pane{
		tree: tree, thread: thread, process: process, actorID: actorID,
		label: label, child: child,
	}
	c.order = append(c.order, handleID)
	if c.active == "" {
		c.active = handleID
		// A BACKGROUND attach -- the reattach pass (pair#206) -- never moves focus
		// or seeds the tracker. c.active is empty whenever the last pane exited
		// while the switcher was focused, and a background completion arriving
		// then would otherwise hand the keyboard to a pane the operator never
		// chose, with no screen takeover, so their typing reached an invisible
		// agent.
		if !background {
			c.focus = FocusActor(handleID)
			// The first attach lands the operator on an actor WITHOUT going
			// through switchTo, so the tracker has to be seeded here or the actor
			// they started in is never recorded -- and the first notification hop
			// would then pin nothing instead of pinning it.
			c.tracker.Switch(thread, false)
		}
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
	// arrivalNotification is a landing on an actor that HAD a pending
	// notification when the operator chose it -- today ctrl-space + Return on a
	// paging row, or ctrl+return. Defined by that property rather than by the
	// keys, so the next gesture that produces one does not make this false.
	// Only this one is non-pinning.
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
//
// stayed reports that the actor was already current and force was false: the
// landing was recorded and acknowledged, and the screen was left alone. It is
// returned rather than left for a caller to re-derive, because this function
// decides it under its own lock, and a caller's separate read of c.active is a
// second authority that another goroutine's switch can falsify in between.
func (c *Console) switchTo(id string, force bool, how arrival) (stayed bool) {
	c.mu.Lock()
	known := c.panes[id] != nil
	c.mu.Unlock()
	if !known {
		return false
	}
	stayed, err := c.selectActor(id, force, how)
	c.terminalError(err)
	return stayed
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
func (c *Console) Run() (code int) {
	restore, err := c.host.MakeRaw()
	if err != nil {
		// Accepted children and the presenter still need disposal even when
		// acquisition fails. An untouched presenter releases without writes.
		c.terminalError(fmt.Errorf("cannot take the terminal: %w", err))
		_ = c.teardown(func() error { return nil })
		return 1
	}
	defer func() {
		if err := c.teardown(restore); err != nil {
			code = 1
		}
	}()
	c.mu.Lock()
	c.started = true
	c.mu.Unlock()
	c.applyLayout()
	c.mu.Lock()
	initial := c.active
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel || initial == "" {
		c.showMenu()
	} else {
		c.switchTo(initial, true, arrivalOrdinary)
	}

	c.workers.Add(4)
	go func() { defer c.workers.Done(); c.watchContinuations() }()
	go func() { defer c.workers.Done(); c.pumpStdin() }()
	go func() { defer c.workers.Done(); c.watchResize() }()
	go func() { defer c.workers.Done(); c.operationQueue.Run(c.stop) }()
	var terminated <-chan os.Signal
	if h, ok := c.host.(hostty.TerminationHost); ok {
		terminated = h.Terminated()
	}

	var decoder terminal.Decoder
	var inputEscapeTimer *time.Timer
	var inputEscapeC, panelEscapeC <-chan time.Time
	var spinnerTimer *time.Timer
	var noticeTimer *time.Timer
	var noticeC <-chan time.Time
	var noticeExpiry time.Time
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
		if !decoder.PendingEscape() {
			inputEscapeC = nil
			return
		}
		if inputEscapeTimer == nil {
			inputEscapeTimer = time.NewTimer(workbenchshortcut.EscapeAmbiguity)
		} else {
			inputEscapeTimer.Reset(workbenchshortcut.EscapeAmbiguity)
		}
		inputEscapeC = inputEscapeTimer.C
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
	// The status row's own spinner (pair#206). The one above runs only while the
	// switcher is focused AND a progress notice shows, but the operator spends
	// the reattach pass in their own thread -- so this is a second timer, armed
	// only while a thread is loading, that repaints the status row. It is the
	// seam #231's clock would extend rather than adding another.
	var statusTimer *time.Timer
	var statusC <-chan time.Time
	syncStatusTick := func() {
		c.mu.Lock()
		loading := c.menu.Reattach.Loading != (couchcore.ThreadAddress{})
		c.mu.Unlock()
		if !loading {
			ticking := statusC != nil
			if statusTimer != nil {
				stopTimer(statusTimer)
			}
			statusC = nil
			if ticking {
				// While a thread loads, this tick is the only thing that repaints
				// the status row, so the frame it last painted still shows that
				// thread's spinning placeholder. A stopping tick owes the frame
				// without it (pair#206, found in the operator's smoke test: the
				// last reattached thread kept spinning until something else
				// repainted).
				c.repaint()
			}
			return
		}
		if statusC != nil {
			return
		}
		if statusTimer == nil {
			statusTimer = time.NewTimer(statusSpinnerInterval)
		} else {
			statusTimer.Reset(statusSpinnerInterval)
		}
		statusC = statusTimer.C
	}
	defer func() {
		if statusTimer != nil {
			stopTimer(statusTimer)
		}
	}()
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
	processEvents := func(events []terminal.InputEvent) {
		for _, event := range events {
			c.routeInputEvent(event)
		}
	}
	processInput := func(raw []byte) {
		events, err := decoder.Feed(raw)
		processEvents(events)
		c.terminalError(err)
	}

	// A transient notice stops being true on its own, and nothing else is
	// guaranteed to happen when it does -- an idle console would keep painting a
	// stale sentence forever. syncNoticeExpiry arms one repaint for the moment
	// the row changes, the same shape syncSpinner uses for its animation.
	syncNoticeExpiry := func() {
		c.mu.Lock()
		row := c.feed.Row()
		c.mu.Unlock()
		if row.Expires.IsZero() {
			stopTimer(noticeTimer)
			noticeC = nil
			noticeExpiry = time.Time{}
			return
		}
		// Same deadline, same timer: an OPTIMISATION, not a correctness rule.
		// Re-arming every iteration would still fire at the right moment, since
		// Standing shrinks as the deadline approaches -- it would only churn the
		// timer. Said plainly because the first version of this comment claimed
		// the notice "would never actually go" without it, which is false, and a
		// false rationale is worse than none: it invites the next reader to
		// preserve a line for a reason that was never true.
		if noticeC != nil && row.Expires.Equal(noticeExpiry) {
			return
		}
		stopTimer(noticeTimer)
		noticeExpiry = row.Expires
		if noticeTimer == nil {
			noticeTimer = time.NewTimer(row.Standing)
		} else {
			noticeTimer.Reset(row.Standing)
		}
		noticeC = noticeTimer.C
	}
	syncNoticeExpiry()

	for {
		select {
		case command := <-c.terminalCommands:
			if err := command.ctx.Err(); err != nil {
				command.done <- err
			} else {
				command.done <- command.run()
			}
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
			events, err := decoder.FlushEscape()
			processEvents(events)
			c.terminalError(err)
		case <-panelEscapeC:
			panelEscapeC = nil
			c.menuHeld = nil
			c.onMenuKey(PanelKey{Kind: KeyEscape})
		case <-noticeC:
			// The row's own deadline: repaint so the expired sentence goes.
			// Clearing the remembered deadline too, so the dedup above compares
			// against nothing stale. Correct without it today -- an expired row
			// reports a zero Expires -- but only by a coincidence two files
			// apart, and an invariant that holds by coincidence is one edit from
			// not holding.
			noticeC = nil
			noticeExpiry = time.Time{}
			c.repaint()
		case <-statusC:
			statusC = nil
			c.mu.Lock()
			c.statusSpinner++
			c.mu.Unlock()
			c.repaint()
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
		case result := <-c.orientationResults:
			c.finishOrientation(result)
		case result := <-c.continuationResults:
			c.acceptContinuationRequests(result)
		case result := <-c.previewResults:
			c.finishMenuPreview(result)
		case result := <-c.completionResults:
			c.finishMenuCompletion(result)
		case completed := <-c.operationQueue.results:
			if c.finishOperation(completed) {
				continue
			}
		case <-c.presenter.Failed():
			c.terminalError(c.presenter.Failure())
			return 1
		case <-terminated:
			return 0
		case <-c.stop:
			c.mu.Lock()
			failed := c.terminalFailure != nil
			c.mu.Unlock()
			if failed {
				return 1
			}
			return 0
		}
		syncSpinner()
		syncStatusTick()
		syncNoticeExpiry()
	}
}

func (c *Console) drainChunks() {
	for {
		select {
		case command := <-c.terminalCommands:
			if err := command.ctx.Err(); err != nil {
				command.done <- err
			} else {
				command.done <- command.run()
			}
		case ch := <-c.chunks:
			c.onChunk(ch)
		default:
			return
		}
	}
}

// teardown cancels and joins event sources before releasing the presenter and
// restoring raw state. Diagnostics are emitted only after ownership ends.
type contextualInput interface {
	ReadContext(context.Context, []byte) (int, error)
}

func (c *Console) teardown(restore func() error) error {
	c.Stop()
	// Contextual readers retain their fd lease until after the input pump joins
	// and Presenter has restored the parent terminal. Plain fixture pipes need
	// explicit close to interrupt Read.
	if _, ok := c.stdin.(contextualInput); !ok {
		if closer, ok := c.stdin.(io.Closer); ok {
			_ = closer.Close()
		}
	}
	c.mu.Lock()
	c.started = false
	c.mu.Unlock()
	c.workers.Wait()
	releaseErr := c.release()
	c.mu.Lock()
	tracer, events, mouseTrace := c.trace, c.events, c.mouseTrace
	c.trace, c.events, c.mouseTrace = nil, nil, nil
	c.mu.Unlock()
	// Complete every cleanup before writing to stderr, which may refer to the
	// same physical terminal. A release failure must still restore raw state.
	cleanupErr := errors.Join(releaseErr, tracer.Close(), events.Close(), mouseTrace.Close(), c.host.Close(), restore())
	c.mu.Lock()
	failure, leaveReport := c.terminalFailure, c.leaveReport
	c.leaveReport = ""
	c.mu.Unlock()
	err := errors.Join(failure, cleanupErr)
	if err != nil {
		fmt.Fprintf(c.errw(), "couch: terminal: %v\n", err)
	}
	if leaveReport != "" && c.stderr != nil {
		fmt.Fprint(c.stderr, leaveReport)
	}
	return err
}

// onExit removes a dead child from the console and registry. An active exit
// lands on the panel; an inactive exit only repaints the notice so it cannot
// steal the operator from the child they are typing in. The final child ends
// an actor-focused console, but an already panel-focused console stays up so a
// completed Park can expose the durable row that Enter resumes.
func (c *Console) onExit(event childExit) bool {
	c.terminalError(c.presenter.Flush(c.lifetime))
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
	_, continuationPending := c.continuations[p.thread]
	forget := c.forget
	last := len(c.panes) == 0
	c.mu.Unlock()

	if !expected {
		// Through publishNotice like every other notice, rather than a bare
		// push: one path decides that a notice is shown, and this one is the
		// operator's only explanation for a pane that just vanished.
		c.publishNotice(ExitNotice(p.actorID, p.label, event.code))
	}

	if forget != nil {
		if err := forget(p.tree, p.actorID); err != nil {
			c.setNotice(fmt.Sprintf("forget %s: %v", p.label, err))
		}
	}
	c.requestMenuRefresh()
	defer func() {
		if c.presenter.View().Selected == p.child.Endpoint().ID() {
			return
		}
		if err := c.presenter.Retire(c.lifetime, p.child.Endpoint()); err != nil {
			c.terminalError(err)
			return
		}
		c.terminalError(c.disposeChild(p.child))
	}()
	if last && !panelFocused && !(expected && continuationPending) {
		return true
	}
	if wasFocused || panelFocused {
		c.showMenu()
	} else {
		c.paintNow()
	}
	return false
}

// release restores the parent terminal modes before raw state is restored.
func (c *Console) release() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := c.presenter.Release(ctx)
	c.traceTerminal("release", err)
	// Accepted children remain owned after their panes disappear. In particular,
	// the final selected endpoint stays readable until its presenter releases.
	c.mu.Lock()
	children := make([]*ptychild.Child, 0, len(c.ownedChildren))
	for child := range c.ownedChildren {
		children = append(children, child)
	}
	c.mu.Unlock()
	for _, child := range children {
		err = errors.Join(err, c.disposeChild(child))
	}
	return err
}

// disposeChild follows retirement (or complete presenter release). Exited
// children have drained Sink acknowledgements; teardown cancels delivery first.
func (c *Console) disposeChild(child *ptychild.Child) error {
	err := child.Close()
	c.mu.Lock()
	delete(c.ownedChildren, child)
	c.mu.Unlock()
	return err
}

// bottomReservation is couch's row: always the host's bottom one.
//
// A plain function of rows, NOT a method that reads c.size under the lock. Most
// callers here already hold c.mu, so a locking accessor deadlocks -- which it
// did, in ChildSize and applyLayout, on the first version of this lift. Taking
// rows as an argument makes every call site obviously safe and leaves reading
// c.size where the locking discipline already is.
func bottomReservation(rows uint16) hostty.Reservation {
	return hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom}
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
	size := ptychild.Size{Rows: bottomReservation(c.size.Rows).ChildRows(), Cols: c.size.Cols}
	children := make([]*ptychild.Child, 0, len(c.panes))
	for _, p := range c.panes {
		children = append(children, p.child)
	}
	c.mu.Unlock()

	// The resize always happens; only the SCREEN write is gated, and it goes
	// through the paint below so there is one gated path rather than two.
	for _, child := range children {
		if err := child.Resize(size); err != nil {
			c.terminalError(err)
			return
		}
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
// statusModelLocked builds the status row's model: the attached chips, then a
// placeholder for each thread the reattach pass has not attached yet
// (pair#206). Callers hold c.mu. Separate from paintNow so the model -- which is
// where a placeholder either appears or silently does not -- is testable without
// rendering to a terminal.
func (c *Console) statusModelLocked() StatusModel {
	model := StatusModel{Notice: c.feed.Row().Body}
	for _, id := range c.order {
		p := c.panes[id]
		model.Actors = append(model.Actors, StatusActor{
			Label:  p.label,
			Thread: p.thread,
			Active: id == c.active,
			Bell:   len(c.attention.Projection(p.thread)) > 0,
		})
	}
	// Placeholders for threads the reattach pass has not attached yet, after
	// the attached chips and in pass order (pair#206), so a thread that attaches
	// takes the column its placeholder held. The label is the one its chip will
	// carry -- the repository of the thread's starting path -- so it does not
	// change when the thread arrives.
	model.Spinner = c.statusSpinner
	for _, pending := range pendingPlaceholders(c.menu.Reattach) {
		label := string(pending.Address.Tag)
		if row, ok := menuThread(c.menu, pending.Address); ok {
			label = couchcore.Worktree(row.StartingPath).Repo()
		}
		model.Actors = append(model.Actors, StatusActor{
			Label: label, Thread: pending.Address, Placeholder: true, Loading: pending.Loading,
		})
	}
	return model
}

func (c *Console) repaint() { c.paintNow() }

// paintNow publishes chrome independently of child parsing and cursor state.
func (c *Console) paintNow() {
	c.mu.Lock()
	panel, started := c.focus.IsPanel(), c.started
	c.mu.Unlock()
	if !started {
		return
	}
	if panel {
		c.showMenu()
		return
	}
	row, cells, err := c.chrome()
	if err == nil {
		err = c.presenter.UpdateChrome(c.lifetime, cells)
	}
	if err != nil {
		c.terminalError(err)
		return
	}
	c.commitChrome(row)
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
	if ch.ack != nil {
		defer close(ch.ack)
	}
	c.mu.Lock()
	p := c.panes[ch.id]
	active := c.focus == FocusActor(ch.id)
	c.mu.Unlock()
	if p == nil {
		return
	}
	if ch.batch.Err != nil {
		c.terminalError(ch.batch.Err)
		return
	}
	var effects []terminal.Effect
	if len(ch.batch.Terminal.Effects) > 0 {
		var err error
		effects, err = c.presenter.EmitEffects(c.lifetime, ch.batch.Terminal.Effects, terminal.EffectPolicy{Bell: active, Title: active, Clipboard: active, Notifications: true})
		if err != nil {
			c.terminalError(err)
			return
		}
	}
	changed := false
	for _, effect := range effects {
		c.mu.Lock()
		switch {
		case effect.Kind == terminal.NotificationEffect && effect.Selection == "pair" && !ch.focusedAtDelivery:
			c.attention.Mark(p.thread, effect.Text)
			changed = true
		case effect.Kind == terminal.BellEffect && !active:
			c.attention.Mark(p.thread, "")
			changed = true
		}
		c.syncAttentionLocked()
		c.mu.Unlock()
	}
	c.terminalError(c.presenter.Present(c.lifetime, p.child.Endpoint()))
	if changed {
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
	size, err := c.host.Size()
	if err != nil {
		c.terminalError(err)
		return
	}
	c.mu.Lock()
	c.size = size
	panel := c.focus.IsPanel()
	selected := c.panes[c.active]
	children := make([]*ptychild.Child, 0, len(c.panes))
	for _, p := range c.panes {
		children = append(children, p.child)
	}
	c.mu.Unlock()
	if !panel && selected != nil {
		err = c.presenter.Resize(c.lifetime, terminal.Geometry{Cols: int(size.Cols), Rows: int(size.Rows)}, func(g terminal.Geometry) error {
			return selected.child.ResizePTY(ptychild.Size{Cols: uint16(g.Cols), Rows: uint16(g.Rows)})
		})
		if err != nil {
			c.terminalError(err)
			return
		}
	}
	childSize := c.ChildSize()
	for _, child := range children {
		if (!panel && selected != nil && child == selected.child) || child.Endpoint().InputEnded() {
			continue
		}
		if err := child.Resize(childSize); err != nil {
			c.terminalError(err)
			return
		}
	}
	if panel {
		c.showMenu()
	} else {
		c.paintNow()
	}
}

// pumpStdin is the one blocking reader. It hands raw chunks to Run, which owns
// framing, ambiguity timers, focus transitions, and routing in one event loop.
func (c *Console) pumpStdin() {
	buf := make([]byte, 4096)
	for {
		var n int
		var err error
		if reader, ok := c.stdin.(contextualInput); ok {
			n, err = reader.ReadContext(c.lifetime, buf)
		} else {
			n, err = c.stdin.Read(buf)
		}
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			// Under the mutex the field is WRITTEN under. record is nil- and
			// closed-safe, so the hazard is the unsynchronised field read
			// itself: teardown nils c.trace before the workers are joined, so
			// this can still be in flight. Latent today, a data race under -race.
			c.mu.Lock()
			tracer := c.trace
			c.mu.Unlock()
			tracer.record(chunk)
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

// onNewestPageHotkey handles ctrl+return: land on the thread ctrl-space would
// have opened the switcher on, without the switcher.
//
// onPreviousHotkey's mirror image rather than a switcher gesture: a target
// computed from console-local state, then straight into switchTo. The menu's
// `switch` operation, which a status-chip click dispatches, would add the
// operation-queue hop and a dependency on the inventory having loaded, and this
// key has no inventory row to act on -- the answer is a live pane.
//
// Runs on the Run goroutine.
func (c *Console) onNewestPageHotkey() {
	c.mu.Lock()
	panel := c.focus.IsPanel()
	// The SAME call ctrl-space focuses the switcher with, so the two answers to
	// "who paged most recently" cannot drift. No ActiveAddress fallback: that is
	// a browser's default, and a jump to where you already are is a no-op that
	// reads as a dropped key.
	newest := c.attention.NewestActor()
	target := ""
	if newest != (couchcore.ThreadAddress{}) {
		target = c.switchTargetForAddressLocked(newest)
	}
	c.mu.Unlock()

	switch {
	case panel:
		// Not claimed here: the switcher already has Return. Whatever the
		// panel's own decoder makes of the chord is what it does -- Return,
		// today, since decodeCSIu drops modifiers for codepoint 13 -- so the
		// meaning is derived rather than restated. Decoded directly instead of
		// through onMenuInput, which would consume the panel's held partial
		// without stopping Run's escape timer.
		keys, _ := DecodePanelKeys([]byte(newestPageSequence))
		for _, key := range keys {
			c.onMenuKey(key)
		}
	case newest == (couchcore.ThreadAddress{}):
		c.setNotice("nothing is paging")
	case target == "":
		// Durable but without a live pane: its child is done and the exit has not
		// been reduced yet. ctrl+backspace refuses the same state the same way.
		c.setNotice("the paging thread is no longer attached")
	default:
		// arrivalNotification because the target is paging by construction --
		// the value the switcher's Return derives from a non-zero capture -- so
		// ctrl+backspace still goes back to where the operator was working.
		//
		// force=false: for another actor it is identical to force=true; for the
		// active one switchTo acknowledges and stays, with no takeover of a
		// screen that did not change -- and says which it did, so the notice
		// below keys on switchTo's own decision rather than a second read.
		if c.switchTo(target, false, arrivalNotification) {
			// The acknowledgement alone is invisible from here: the status row
			// never draws the active actor's bell. Without a word the key would
			// look dropped.
			c.setNotice("already on the paging thread")
		}
	}
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

// reportLeave retains the final product report until terminal ownership ends.
func (c *Console) reportLeave(result couchcore.LeaveResult) {
	var report strings.Builder
	if len(result.Detached) > 0 {
		fmt.Fprintf(&report, "couch: detached %d thread(s); their agents keep running\n", len(result.Detached))
	}
	if len(result.Parked) > 0 {
		if result.Disposition == couchcore.LeavePark {
			fmt.Fprintf(&report, "couch: parked %d thread(s); their agents were stopped\n", len(result.Parked))
		} else {
			fmt.Fprintf(&report, "couch: parked %d thread(s) that were already shutting down\n", len(result.Parked))
		}
	}
	for _, address := range result.Skipped {
		fmt.Fprintf(&report, "couch: left %s occupied — its state could not be proved detachable\n", address.Tag)
	}
	c.mu.Lock()
	c.leaveReport += report.String()
	c.mu.Unlock()
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
		target = c.menu.SelectedThreadAddress()
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
	// A background effect -- the reattach pass (pair#206) -- goes FIRST, before
	// anything that addresses the operator's in-flight slot: the attention
	// capture below reads InFlight, and the match further down would drop this
	// effect, because the pass never holds InFlight.
	if effect.Background {
		c.runBackgroundOperation(effect)
		return
	}
	c.mu.Lock()
	fn := c.ops
	origin := c.menu.InFlight
	if effect.Operation == "retry-continuation" {
		origin.ContinuationID = effect.Args["request-id"]
		origin.PreserveFocus = c.focus.IsPanel()
		if origin.ContinuationID != "" {
			watch := c.continuations[origin.Address]
			watch.status.Address, watch.status.RequestID = origin.Address, origin.ContinuationID
			watch.queued, watch.handled = true, true
			c.continuations[origin.Address] = watch
			for id, p := range c.panes {
				if p.thread == origin.Address {
					c.expectedExits[id] = true
				}
			}
			c.menu.InFlight = origin
		}
	}
	if effect.Operation == "switch-agent" {
		if previous, ok := c.orientationWatches[origin.Address]; ok {
			previous.cancel()
			delete(c.orientationWatches, origin.Address)
		}
		delete(c.menu.Orientation, origin.Address)
		origin.PanelOrigin = c.focus.IsPanel()
		c.menu.InFlight.PanelOrigin = origin.PanelOrigin
	}
	if origin.Operation == "switch" && origin.AttentionCapture == 0 && !origin.Manual {
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

// onMouse routes one decoded mouse report.
//
// The ONE place the report's 1-based coordinates meet the render's 0-based
// geometry, converted here so no other site has to know two bases exist.
//
// Focus decides which surface a non-row click means, and the Interceptor is why
// no new input path was needed: it sees every byte before focus is considered,
// so the panel needs no PanelKey kind and panelkeys.go is untouched.
func (c *Console) onMouse(hit MouseHit) {
	var decoder terminal.Decoder
	events, err := decoder.Feed(hit.Raw)
	if err != nil {
		c.terminalError(err)
		return
	}
	for _, event := range events {
		c.routeMouseEvent(event)
	}
}

// switchToThread dispatches the SAME declared switch operation ctrl-space+Return
// dispatches, marked MANUAL.
//
// Return on a PAGING actor is a notification hop and therefore non-pinning; a
// click never is, so ctrl+backspace always undoes it. That flag is the only
// difference -- the operation, the queue, the projection refresh and the notice
// bookkeeping are all Return's.
func (c *Console) switchToThread(thread couchcore.ThreadAddress) {
	c.reduceMenu(MenuEvent{Kind: MenuEventMouseSwitch, Address: thread})
}

// dispatchInputCandidate resolves ownership after prefix delivery. The route
// callback is the input loop's existing panel/child writer, not a focus cache.
func (c *Console) dispatchInputCandidate(before []byte, hit InterceptorHit, rawHit []byte, route func([]byte)) {
	route(before)
	if hit == HitNone {
		return
	}
	c.mu.Lock()
	actorFocused := !c.focus.IsPanel()
	c.mu.Unlock()
	if actorFocused && hit != HitMouse && !hit.actorReserved() {
		route(rawHit)
		return
	}
	// The declared handler table is checked against AllInterceptorHits.
	if handle := c.hitHandlers()[hit]; handle != nil {
		handle()
	} else {
		c.setNotice(fmt.Sprintf("chord %d is intercepted but has no handler", hit))
	}
}

// hitHandlers maps every intercepted chord to what the console does about it.
// A method rather than a package var because the handlers are bound to this
// Console; the point is that the mapping is DATA a test can walk, not control
// flow it can only execute.
func (c *Console) hitHandlers() map[InterceptorHit]func() {
	return map[InterceptorHit]func(){
		HitSwitch:     c.onHotkey,
		HitPark:       c.onParkHotkey,
		HitPrevious:   c.onPreviousHotkey,
		HitNewestPage: c.onNewestPageHotkey,
		HitDetach:     c.onDetachHotkey,
		HitRelaunch:   c.onRelaunchHotkey,
		// HitMouse carries coordinates, which func() cannot, so it is dispatched
		// from processInput with the payload rather than through this table. The
		// entry is the CONSOLE's handler for it -- a real call, not a placeholder
		// the dispatcher skips, which is what an empty func() here would be.
		HitMouse: func() { c.onMouse(c.pendingMouse()) },
	}
}

// pendingMouse is the payload of the HitMouse being dispatched. Stored by
// processInput immediately after the Interceptor returns it, so the handler
// table can carry a real function for HitMouse like every other hit.
func (c *Console) pendingMouse() MouseHit {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mouseHit
}

// finishOperation returns true when the completion requested Console exit.
func (c *Console) finishOperation(completed operationCompletion) bool {
	err := completed.err
	defer func() { c.finishContinuationOperation(completed, err) }()
	if completed.name == "continuation-status" {
		return false
	}
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
					args := map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag)}
					if completed.origin.PreserveFocus || completed.origin.Background || completed.origin.Operation == "switch-agent" && completed.origin.PanelOrigin {
						// The reattach pass's child is adopted without taking
						// focus (pair#206); only the declared arg carries that
						// across the operation table to the installer.
						args["background"] = "true"
					}
					_, err = fn(couchcore.OperationCall{
						Name: "attach", Context: c.lifetime, Implicit: true, TypedPayload: started,
						Args: args,
					})
				}
			}
		}
	}
	event := MenuEvent{
		Kind: MenuEventOperationResult, Operation: completed.origin.Operation,
		Attempt: completed.origin.Attempt, Address: address, Success: err == nil,
		Background: completed.origin.Background,
	}
	if err != nil {
		event.Error = err.Error()
		// A code, so the pass tells a skip from a failure without matching text.
		event.Diagnostic = couchcore.ResumeDiagnosticOf(err)
	}
	if completed.origin.Background {
		c.traceEvent(traceReattachDone, address, reattachDoneDetail(event.Success, event.Diagnostic))
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
			// Excluding the child this operation just STARTED. A relaunch has
			// already adopted its replacement by the time it gets here, so
			// marking every pane on the address marked the new one -- and its
			// first real death would then be swallowed as expected, which is the
			// exact spurious-notice bug this bridge exists to prevent, inverted.
			// The child a relaunch expects to exit is the one it replaced.
			if p.thread == address && id != startedHandleID {
				c.expectedExits[id] = true
			}
		}
	}
	var menuEffects []MenuEffect
	if c.menuReady {
		c.menu, menuEffects = ReduceMenu(c.menu, event)
	}
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	// A completion can start the reattach pass's next attempt (pair#206).
	c.dispatchMenuEffects(menuEffects)
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
	// Never for a background completion: the pass reattaches behind the
	// operator, and there is no adoption, so no background completion is ever
	// the operator's own landing (pair#206).
	if (completed.origin.Operation == "resume" || completed.origin.Operation == "recover-thread" || completed.origin.Operation == "recover-checkpoint") && err == nil && startedHandleID != "" && !completed.origin.Background && !completed.origin.PreserveFocus {
		c.requestMenuRefresh()
		c.forceSwitch(startedHandleID)
		return false
	}
	if result, ok := completed.value.(couchcore.SwitchAgentResult); ok && err == nil {
		if result.Warning != "" {
			c.reduceMenu(MenuEvent{Kind: MenuEventNotice, Error: result.Warning})
		}
		if result.Orientation != nil {
			c.watchOrientation(address, startedHandleID, *result.Orientation)
		}
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
	if origin.Attempt == 0 {
		return false
	}
	if endsItsOwnChild(origin.Operation) {
		return origin.Address == address
	}
	switch origin.Operation {
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
		// A notification hop is a landing on an actor that HAD a pending
		// notification when the operator chose it; on this path that means
		// Return on a paging row. runMenuOperation captured that set before
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
		err := c.runTerminalCommand(call.Context, func() error { _, err := c.selectActorContext(call.Context, target, true, how); return err })
		return address, err
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
			couchcore.ProcessIdentity{PID: start.Handle.PID(), Identity: start.Handle.Identity()},
			call.Args["background"] == "true"); err != nil {
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
		if pane != nil && pane.thread == address && !pane.child.Endpoint().InputEnded() {
			return id
		}
	}
	return ""
}

func (c *Console) setNotice(text string) {
	c.publishNotice(Notice{Kind: "status", Body: text})
}

// publishNotice is the one way a notice is PUBLISHED: it pushes to the feed and
// repaints the row that shows it.
//
// Pushing without repainting used to be merely late -- the sentence appeared
// whenever something else next painted, and stayed forever once it did. Giving
// transients a lifetime (pair#185) turned that into a correctness bug: on an
// idle console with no child producing output, nothing repaints, so a refusal
// could expire entirely UNSEEN. A message the operator never saw is worse than
// one that overstayed. Push and paint are therefore one operation rather than
// two things to remember in the right order.
//
// The repaint waits for Run: before it, the terminal is not in raw mode or the
// alt screen, and painting a status row into the operator's shell is not a
// message, it is damage. Whatever is queued by then is shown by the first paint.
func (c *Console) publishNotice(n Notice) {
	c.mu.Lock()
	c.feed.Push(n)
	started := c.started
	c.mu.Unlock()
	if started {
		c.repaint()
	}
}
