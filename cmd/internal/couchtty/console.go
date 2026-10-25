package couchtty

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"image/color"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
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
	// messageHandle names this pane incarnation to the message service; a
	// reused handleID still gets a new one (#365).
	messageHandle couchmessage.PaneHandle
	// Counts acknowledged child notifications for this attachment only.
	suppressedNotifications uint64
}

// pendingInstall reserves routing identity while terminal preparation runs
// without the Console mutex. The child remains caller-owned until commit.
type pendingInstall struct {
	thread couchcore.ThreadAddress
	child  *ptychild.Child
}

// Console routes the operator's terminal to one child at a time.
//
// It is the integration controller: reusable decisions live in pure functions
// in this package, while Console owns event ordering and transient UI
// transitions as it drives hostty.Host. It never calls x/term or os/signal
// directly, which keeps resize and teardown testable without a terminal.
type Console struct {
	messageBroker  *couchmessage.Broker
	messagePanes   *couchmessage.PaneMailbox
	messagePaneGen uint64
	host           hostty.Host
	stdin          io.Reader
	stderr         io.Writer
	capture        *terminalcapture.Recorder
	presenter      *terminal.Presenter
	// publishMu orders presentation (pair#484). Each paint snapshots console
	// state, publishes it to the presenter, then commits its hit-test row; paints
	// run on Run and on other goroutines (a notice from a worker). Unordered, an
	// older snapshot could publish after a newer one and stay on screen until
	// something else repainted -- on an idle console, never. Held across
	// snapshot-to-commit, the last publication is always the latest state. Never
	// take it while holding c.mu.
	publishMu sync.Mutex
	// publishHook runs inside publishMu, inside a site's snapshot-to-commit span
	// (chrome, panel, select, resize); tests use it to park a paint there while
	// another tries to overtake it.
	publishHook func(site string)
	// codes is what the operator-only layer shows of cast link codes
	// (pair#495); its mutex is a leaf.
	codes            castCodes
	terminalCommands chan terminalCommand
	terminalFailure  error
	drivePending     *driveOperation // guarded by mu; survives forgetDrive/cast replacement
	// exitFailure is teardown's one classification of why the terminal failed
	// (shutdown cancellations filtered out), read by TerminalFailure.
	exitFailure     error
	leaveReport     string
	ownedChildren   map[*ptychild.Child]struct{}
	pendingInstalls map[string]pendingInstall

	mu     sync.Mutex
	panes  map[string]*pane
	order  []string
	active string

	// focus is what the terminal is pointed at. It is not the same as `active`:
	// the switcher is a focus with no actor behind it.
	focus           Focus
	shortcutFocused func(context.Context, couchcore.ThreadAddress) (workbenchshortcut.PaneRole, error)

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
	ops            func(couchcore.OperationCall) (any, error)
	beginAttempt   func(couchcore.OperationCall) (*couchcore.OperationAttempt, error)
	forget         func(couchcore.Worktree, couchcore.ActorID) error
	feed           *Feed
	noticeSequence uint64
	attention      AttentionLedger
	size           ptychild.Size
	// expectedExits are exact child handles whose successful Park already
	// authorized shutdown. They bridge the race between the child-exit channel
	// and the asynchronous operation-completion channel. Marks are counted per
	// owner (pair#205), so an operation that undoes its own mark never removes
	// one that another operation set on the same pane.
	expectedExits     exitMarks
	lifecycleObserver func(LifecycleObservation)
	lifecycleSequence uint64
	lifecyclePassDone bool
	lifecycleNeeds    map[string]bool

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
	statusChips    []ChipSpan
	tabStyle       TabStyle
	statusGroups   []GroupCountSpan
	statusRows     int
	statusRevision uint64
	// statusControl is the broadcast cell's span on the last drawn row.
	statusControl ColumnSpan
	// broadcastCfg is nil when broadcasting is not configured; bcast is the
	// running broadcast's state (console_broadcast.go). Both under mu.
	broadcastCfg *broadcast.Config
	bcast        broadcastState
	// webcastCfg is nil when web casting is not configured; wcast is the
	// cast Chrome's state (console_webcast.go). Both under mu.
	webcastCfg *webcast.CastConfig
	wcast      webcastState
	// replays are the recordings shared for replay on the Host (#477); nil
	// when Couch has no store. Set before Run, closed at teardown.
	replays     *webcast.ReplayShares
	submissions *submissionManager
	// statusWebcast is the Chrome cell's span on the last drawn row.
	statusWebcast ColumnSpan
	// statusWebNotes is the web cast's 💭N span (#470).
	statusWebNotes ColumnSpan
	// statusAttention is the 🔔N span (pair#673); attn owns the list.
	statusAttention ColumnSpan
	attn            attentionState
	// statusWebFocus, statusWebPointerLink and statusWebDriveLink are the
	// Chrome group's ↗ 👆 👽 spans (#501).
	statusWebFocus, statusWebPointerLink, statusWebDriveLink ColumnSpan
	// statusWebDrive is remote driving's span (#467).
	statusWebDrive ColumnSpan
	// overlayUsers is who needs the collaboration overlay (syncOverlay).
	overlayUsers overlayUser
	// pointer is the pointer link's phase (#412), under mu; pmarks is the
	// marks state the Presenter's overlay reads, behind its own leaf lock.
	pointer pointerPhase
	pmarks  pointerMarks
	// pnotes is the sticky notes the overlay and status row read (#447),
	// behind its own leaf lock.
	pnotes pointerNotes
	// anchors holds the public frame used for synchronous note anchoring.
	anchors noteAnchors
	// statusPointer and statusNotes are the 👆 and 💭N spans on the last
	// drawn row.
	statusPointer ColumnSpan
	statusNotes   ColumnSpan
	// ansiPalette is the terminal's 16 ANSI colours from OSC 4 replies, for
	// broadcast viewers; ansiKnown marks which arrived. Under mu.
	ansiPalette [16]color.RGBA
	ansiKnown   [16]bool
	// mouseHit is the payload of the hit currently being dispatched.
	mouseHit MouseHit
	// started reports that Run owns the terminal, so a notice may paint itself.
	// Its own field rather than something inferred from another: "is it safe to
	// write to the operator's screen yet" is its own question.
	started            bool
	noticeRouter       func(Notice)
	noticeDelivery     NoticeDelivery
	exited             chan childExit
	operationQueue     *operationQueue
	resourceRows       []couchcore.ActionableThreadSummary
	resourceGeneration uint64
	resourceRowsKnown  bool
	resourceChanges    chan struct{}
	refreshRequests    chan struct{}
	refreshResults     chan menuRefreshResult
	refreshSchedule    RefreshSchedule
	// Slot quick-status glyph refresh (pair#317); see console_slotgit.go.
	slotGitProbe    SlotGitProbe
	slotGitRequests chan struct{}
	slotGitResults  chan slotGitResult
	slotGitSchedule RefreshSchedule
	slotGitInterval time.Duration
	// Idle-fading activity refresh (pair#247); see console_activity.go.
	activityProbe          ActivityProbe
	activityPass           ActivityPass
	activityCancel         context.CancelFunc
	activityRowsGeneration uint64
	activityRowsKnown      bool
	wrapperActivity        func(couchcore.ActionableThreadSummary) time.Time
	automationControl      func(string, string) (bool, error)
	copyObserver           func(CopyObservation)
	copyKeyObserver        func(CopyKeyObservation)
	activityRequests       chan struct{}
	activityResults        chan activityResult
	activitySchedule       RefreshSchedule
	activityInterval       time.Duration
	// Per-probe bound from console_activity.go; the New default is the package
	// const and tests shrink it to exercise the deadline path (#628). Only
	// tests write it, before any worker starts; the worker reads it without
	// c.mu.
	probeTimeout time.Duration
	// blockedUpdates carries the latest picker set (pair#465); see
	// SetBlockedThreads.
	blockedUpdates chan map[couchcore.ThreadAddress]bool
	// paletteFG and paletteBG record which of the host terminal's colour
	// replies have arrived (pair#247); the palette is Known once both have.
	paletteFG, paletteBG bool
	// now is the console's clock for idle fading, so tests can cross the day
	// and three-day bands without waiting.
	now                   func() time.Time
	continuationProvider  ContinuationProvider
	continuationResults   chan continuationScanResult
	continuationScanError string
	continuations         map[couchcore.ThreadAddress]continuationWatch
	previewResults        chan menuPreviewResult
	previewSchedule       PreviewSchedule
	previewCancel         context.CancelFunc
	previewRunning        uint64
	directoryReader       DirectoryBatchReader
	completionResults     chan menuCompletionResult
	completionSchedule    latestSchedule[CompletionRequest]
	completionCancel      context.CancelFunc
	completionRunning     CompletionIdentity
	lifetime              context.Context
	cancelLifetime        context.CancelFunc
	sessionOwnerPass      func(context.Context) (context.Context, context.CancelFunc)
	reattachArmOnce       sync.Once
	reattachContext       context.Context
	reattachCancel        context.CancelFunc
	stop                  chan struct{}
	once                  sync.Once
	workers               sync.WaitGroup
	// queueWorkers retains the existing total. All but the final worker own
	// lifecycle/adoption; the final worker owns hosted navigation only (#576).
	queueWorkers         int
	navigationGeneration uint64
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
		tabStyle:            DefaultTabStyle,
		presenter:           terminal.NewPresenter(host, terminal.AnyMotion),
		terminalCommands:    make(chan terminalCommand, 16),
		stdin:               stdin,
		panes:               map[string]*pane{},
		chunks:              make(chan chunk, 256),
		resized:             make(chan struct{}, 1),
		switching:           make(chan string, 8),
		input:               make(chan []byte, 64),
		exited:              make(chan childExit, 64),
		operationQueue:      newOperationQueue(16),
		queueWorkers:        max(1, couchcore.LifecycleParallelism) + 1,
		refreshRequests:     make(chan struct{}, 1),
		refreshResults:      make(chan menuRefreshResult, 1),
		slotGitRequests:     make(chan struct{}, 1),
		slotGitResults:      make(chan slotGitResult, 1),
		slotGitInterval:     defaultSlotGitInterval,
		activityRequests:    make(chan struct{}, 1),
		activityResults:     make(chan activityResult, 1),
		activityInterval:    defaultActivityInterval,
		probeTimeout:        activityProbeTimeout,
		blockedUpdates:      make(chan map[couchcore.ThreadAddress]bool, 1),
		now:                 time.Now,
		continuationResults: make(chan continuationScanResult, 1),
		continuations:       make(map[couchcore.ThreadAddress]continuationWatch),
		previewResults:      make(chan menuPreviewResult, 1),
		directoryReader:     OSDirectoryBatchReader{},
		completionResults:   make(chan menuCompletionResult, 1),
		expectedExits:       exitMarks{},
		lifetime:            lifetime,
		cancelLifetime:      cancelLifetime,
		stop:                make(chan struct{}),
		feed:                NewFeed(8, time.Now, NoticeLifetime),
	}
	c.operationQueue.prepare = c.prepareOperationCompletion
	c.operationQueue.publish = c.publishOperationCompletion
	c.operationQueue.abandon = c.abandonOperationCompletion
	if s, err := host.Size(); err == nil {
		c.size = s
	}
	// Cast link codes draw on the operator's screen only (pair#495).
	c.terminalError(c.installCodesLayer())
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
		c.Warn(Notice{Class: couchcore.NoticeInputTraceFailed, Kind: "trace", Control: true, Body: err.Error()})
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
		c.Warn(Notice{Class: couchcore.NoticeMouseTraceFailed, Kind: "trace", Control: true, Body: err.Error()})
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
// rendered chrome. Handed to PtyRunner so the FIRST frame is already right --
// spawning at the host height and reflowing is a whole redraw for a full-screen
// agent harness.
func (c *Console) ChildSize() ptychild.Size {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.childSizeLocked()
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
//
// Its tree is the label: a tab reads its row's working directory, never a
// pane label transported through the stored name (pair#363), so a pane
// attached as "brain" must sit in a tree whose name is brain.
func (c *Console) Attach(id, label string, child *ptychild.Child) {
	tree := couchcore.Worktree(id)
	if label != "" {
		tree = couchcore.Worktree(label)
	}
	c.AttachActor(id, couchcore.ActorID(id), tree, label, child)
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
// It commits a complete pane or no pane. Preparation reserves identity and a
// worker under the teardown mutex, then releases it across resize/registration.
// Commit transfers that worker to the exit watcher; rejection retires only the
// registration, leaving child disposal with the caller's typed rollback.
func (c *Console) installObservedThreadActor(ctx context.Context, handleID string, actorID couchcore.ActorID, thread couchcore.ThreadAddress, tree couchcore.Worktree, label string, child *ptychild.Child, process couchcore.ProcessIdentity, background bool) (err error) {
	if ctx == nil {
		ctx = c.lifetime
	}
	if handleID == "" || actorID == "" || child == nil {
		return errors.New("attach requires complete handle, actor, and terminal identities")
	}
	if child.Endpoint().InputEnded() {
		return captureAttachFailure("attach-preflight", child, errors.New("attach terminal input has ended"))
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
	if _, exists := c.ownedChildren[child]; exists {
		c.mu.Unlock()
		return errors.New("terminal child is already attached")
	}
	for pendingHandle, pending := range c.pendingInstalls {
		if pendingHandle == handleID || pending.thread == thread || pending.child == child {
			c.mu.Unlock()
			return fmt.Errorf("terminal installation for %s/%s is already pending", pending.thread.RepoScope, pending.thread.Tag)
		}
	}
	for _, installed := range c.panes {
		if installed.thread == thread && !installed.child.Endpoint().InputEnded() {
			c.mu.Unlock()
			return fmt.Errorf("thread %s/%s is already attached", thread.RepoScope, thread.Tag)
		}
	}
	if c.pendingInstalls == nil {
		c.pendingInstalls = make(map[string]pendingInstall)
	}
	c.pendingInstalls[handleID] = pendingInstall{thread: thread, child: child}
	c.workers.Add(1)
	desired := c.childSizeLocked()
	c.mu.Unlock()
	registered, committed := false, false
	defer func() {
		if committed {
			return
		}
		if registered {
			// Stop cancels the operation context before teardown joins us. A
			// fresh cleanup budget lets the still-live presenter retire our
			// uncommitted registration before it is released.
			cleanup, cancel := context.WithTimeout(context.Background(), terminal.WriteTimeout)
			err = errors.Join(err, c.presenter.Retire(cleanup, child.Endpoint()))
			cancel()
		}
		c.mu.Lock()
		delete(c.pendingInstalls, handleID)
		c.mu.Unlock()
		c.workers.Done()
	}()
	for {
		if child.Size() != desired {
			if err := child.Resize(desired); err != nil {
				return err
			}
		}
		if !registered {
			if err := c.presenter.Register(ctx, child.Endpoint()); err != nil {
				return err
			}
			registered = true
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
		if child.Endpoint().InputEnded() {
			c.mu.Unlock()
			return captureAttachFailure("attach-precommit", child, errors.New("attach terminal input has ended"))
		}
		pending, reserved := c.pendingInstalls[handleID]
		if !reserved || pending.child != child || pending.thread != thread {
			c.mu.Unlock()
			return errors.New("terminal installation reservation changed")
		}
		current := c.childSizeLocked()
		if desired != current {
			desired = current
			c.mu.Unlock()
			continue
		}
		break // The short routing commit below retains c.mu.
	}
	if c.ownedChildren == nil {
		c.ownedChildren = make(map[*ptychild.Child]struct{})
	}
	c.ownedChildren[child] = struct{}{}
	delete(c.pendingInstalls, handleID)
	committed = true
	c.messagePaneGen++
	c.panes[handleID] = &pane{
		tree: tree, thread: thread, process: process, actorID: actorID,
		label: label, child: child,
		messageHandle: couchmessage.PaneHandle(fmt.Sprintf("%s#%d", handleID, c.messagePaneGen)),
	}
	if c.capture != nil {
		c.capture.Record(terminalcapture.Record{Kind: "thread-bind", EndpointID: child.Endpoint().ID(), Scope: thread.RepoScope, Tag: string(thread.Tag), Actor: string(actorID)})
	}
	c.order = append(c.order, handleID)
	c.postMessagePaneLocked(thread)
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
	c.ensureMenuLocked(thread)
	c.resourcesChangedLocked()
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

type menuOperationContextKey struct{}
type menuOperationContext struct {
	origin MenuOperationOrigin
	hosted string
}

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
	// paging row, or an attention item. Defined by that property rather than by the
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
	// Arriving at a slot is when its glyph is read; refresh off the render path.
	// A switch is also operator activity, so the fades refresh too (pair#247).
	c.requestSlotGit()
	c.requestActivity()
	return stayed
}

// Stop tears the console down. Safe to call more than once, and from any
// goroutine, including while an attach holds c.mu. Lifetime cancellation also
// cancels pass workers; teardown releases their scoped state after the barrier.
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
	// Before any frame: ask the terminal for its colours (pair#247). A write
	// failure only leaves the palette unknown, which fading already handles.
	paletteCtx, cancelPalette := context.WithTimeout(c.lifetime, terminal.WriteTimeout)
	_, _ = c.host.WriteContext(paletteCtx, []byte(paletteQuery))
	cancelPalette()
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

	queueWorkers := max(1, c.queueWorkers)
	c.workers.Add(3 + queueWorkers)
	go func() { defer c.workers.Done(); c.watchContinuations() }()
	go func() { defer c.workers.Done(); c.pumpStdin() }()
	go func() { defer c.workers.Done(); c.watchResize() }()
	// Lifecycle/adoption workers and the reserved hosted-navigation worker
	// share the bounded queue. Only Run reduces their final outcomes.
	for i := 0; i < max(1, queueWorkers-1); i++ {
		go func() { defer c.workers.Done(); c.operationQueue.Run(c.stop) }()
	}
	if queueWorkers > 1 {
		go func() { defer c.workers.Done(); c.operationQueue.RunNavigation(c.stop) }()
	}
	slotGitTicker := time.NewTicker(c.slotGitInterval)
	defer slotGitTicker.Stop()
	activityTicker := time.NewTicker(c.activityInterval)
	defer activityTicker.Stop()
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
		loading := len(c.menu.Reattach.InFlight) > 0
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
		if notice.Level != MenuNoticeProgress {
			for _, job := range c.menu.Operations {
				if job.Phase == MenuOperationRunning && (notice.Level != MenuNoticeProgress || job.Origin.Attempt < notice.Owner.OperationAttempt) {
					notice = MenuNotice{Level: MenuNoticeProgress, Owner: MenuProgressOwner{OperationAttempt: job.Origin.Attempt}}
				}
			}
		}
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
	// Notifications are coalesced wakeups; chrome reads the current snapshot.
	// The recorder owns this channel and never closes it. Nil disables capture.
	captureChanges := c.capture.Changes()

	for {
		select {
		case <-captureChanges:
			c.repaint()
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
		case <-slotGitTicker.C:
			c.advanceSlotGit(RefreshScheduleEvent{Kind: RefreshRequested})
		case <-c.slotGitRequests:
			c.advanceSlotGit(RefreshScheduleEvent{Kind: RefreshRequested})
		case result := <-c.slotGitResults:
			c.finishSlotGit(result)
		case <-activityTicker.C:
			c.expireAttention()
			c.advanceActivity(RefreshScheduleEvent{Kind: RefreshRequested})
		case <-c.activityRequests:
			c.advanceActivity(RefreshScheduleEvent{Kind: RefreshRequested})
		case result := <-c.activityResults:
			c.finishActivity(result)
		case blocked := <-c.blockedUpdates:
			c.mu.Lock()
			c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventBlocked, Blocked: blocked})
			c.mu.Unlock()
			c.repaintVisible()
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
			// Joined teardown classifies the failure independently of whether
			// this notification or Stop wins the select.
			return 0
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

type contextualInput interface {
	ReadContext(context.Context, []byte) (int, error)
}

// teardown cancels and joins event sources before releasing the presenter and
// restoring raw state. Diagnostics are emitted only after ownership ends.
func (c *Console) teardown(restore func() error) error {
	c.Stop()
	c.CloseNoticeDelivery()
	// Viewers are told before the presenter goes, and the tunnel is closed
	// rather than left running (#395).
	c.endBroadcastForShutdown()
	c.endWebcastForShutdown()
	if c.replays != nil {
		c.replays.Close()
	}
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
	cancelPass := c.releaseReattachPassLocked()
	c.mu.Unlock()
	if cancelPass != nil {
		cancelPass()
	}
	c.workers.Wait()
	c.operationQueue.abandonPending()
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
	// Release has joined the presenter: a concurrent paint failure can no
	// longer arrive after this observation. A failure latched while Run was
	// live remains fatal even when its leaf was context.Canceled; only a new
	// presenter cancellation caused by owner shutdown is expected here.
	presenterFailure := c.presenter.Failure()
	if c.lifetime.Err() != nil && shutdownCancellation(presenterFailure) {
		presenterFailure = nil
	}
	err := errors.Join(failure, presenterFailure, cleanupErr)
	c.mu.Lock()
	c.exitFailure = err
	c.menu.CopyFeedback = CopyFeedback{}
	c.mu.Unlock()
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
	c.resourcesChangedLocked()
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
	c.postMessagePaneLocked(p.thread)
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
		n := ExitNotice(p.actorID, p.label, event.code)
		n.Class = couchcore.NoticeChildExited
		n.Thread, n.Label = p.thread, p.label
		c.Warn(n)
	}

	if forget != nil {
		if err := forget(p.tree, p.actorID); err != nil {
			c.Warn(Notice{Class: couchcore.NoticeExitCleanupFailed, Kind: "forget", Thread: p.thread, Label: p.label, Body: fmt.Sprintf("forget %s: %v", p.label, err)})
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
// Presenter owns the release budget (terminal.WriteTimeout); a shorter one
// here would cut off a host that resumes within it (#383).
func (c *Console) release() error {
	err := c.presenter.Release(context.Background())
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

func (c *Console) activeChild() *ptychild.Child {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.panes[c.active]; ok {
		return p.child
	}
	return nil
}

// applyLayout sizes every child to fit above the rendered chrome. Chrome itself
// is drawn by the paint, so there is one gated path to the screen rather than
// two.
func (c *Console) applyLayout() {
	c.mu.Lock()
	size := c.childSizeLocked()
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
func (c *Console) repaint() { c.paintNow() }

// paintNow publishes chrome independently of child parsing and cursor state.
func (c *Console) paintNow() {
	c.publishMu.Lock()
	defer c.publishMu.Unlock()
	c.mu.Lock()
	panel, started := c.focus.IsPanel(), c.started
	c.mu.Unlock()
	if !started {
		return
	}
	if panel {
		c.showMenuPublishing()
		return
	}
	c.paintActorPublishing("chrome")
}

// beforePublish runs the test hook, if any; the caller holds publishMu.
func (c *Console) beforePublish(site string) {
	c.mu.Lock()
	hook := c.publishHook
	c.mu.Unlock()
	if hook != nil {
		hook(site)
	}
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
	var effectErr error
	if len(ch.batch.Terminal.Effects) > 0 {
		effects, effectErr = c.presenter.EmitEffects(c.lifetime, ch.batch.Terminal.Effects, terminal.EffectPolicy{Bell: active, Title: active, Clipboard: active, Notifications: false})
	}
	changed := false
	var suppressed uint64
	c.mu.Lock()
	for _, effect := range effects {
		switch effect.Kind {
		case terminal.NotificationEffect, terminal.SuppressedNotificationEffect:
			suppressed++
			if p.suppressedNotifications < ^uint64(0) {
				p.suppressedNotifications++
			}
		case terminal.BellEffect:
			if !active {
				c.attention.Mark(p.thread, "")
				changed = true
			}
		}
	}
	if changed {
		c.syncAttentionLocked()
	}
	totalSuppressed := p.suppressedNotifications
	c.mu.Unlock()
	if suppressed > 0 {
		c.traceEvent(traceNotifySuppressed, p.thread, fmt.Sprintf("endpoint=%s count=%d", p.child.Endpoint().ID(), totalSuppressed))
	}
	if effectErr != nil {
		c.terminalError(effectErr)
		return
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
	c.publishMu.Lock()
	defer c.publishMu.Unlock()
	c.mu.Lock()
	c.size = size
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel {
		c.showMenuPublishing()
	} else {
		c.paintActorPublishing("resize")
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
	c.openSwitcher()
}

// openSwitcher opens the root without interpreting a second Ctrl+Space as Start.
func (c *Console) openSwitcher() {
	c.mu.Lock()
	c.menu.LandingAttempt = 0
	c.navigationGeneration++
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
		c.menu.Frames[0].GroupScope = nil
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
	// capture below addresses an operator attempt, which the pass never owns.
	if effect.Background {
		c.runBackgroundOperation(effect)
		return
	}
	c.mu.Lock()
	fn := c.ops
	origin := menuOriginForAttempt(c.menu, effect.Attempt)

	if effect.Operation == "switch-agent" {
		origin.PanelOrigin = c.focus.IsPanel()
		c.menu = replaceMenuOperationOrigin(c.menu, origin)
	}
	if origin.Operation == "switch" && origin.AttentionCapture == 0 && !origin.Manual {
		origin.AttentionCapture = c.attention.Capture(origin.Address)
		c.menu = replaceMenuOperationOrigin(c.menu, origin)
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
	attempt, attemptErr := c.beginLifecycleAttempt(effect.Operation, requestArgs, origin.Address, origin.RowKey)
	if attemptErr != nil {
		c.finishOperation(operationCompletion{name: effect.Operation, origin: origin, err: attemptErr})
		return
	}
	request := operationRequest{attempt: attempt, key: key, name: effect.Operation, origin: origin, run: func() (any, error) {
		operationContext, cancelOperation := context.WithCancel(c.lifetime)
		defer cancelOperation()
		if attempt != nil {
			operationContext = couchcore.WithOperationAttempt(operationContext, attempt)
		}
		// A long operation reports its current step onto its own progress
		// notice (#399); the reducer ignores a line that arrives after the
		// notice became a result.
		operationContext = context.WithValue(operationContext, menuOperationContextKey{}, menuOperationContext{origin: origin})
		operationContext = couchcore.WithOperationProgress(operationContext, func(detail string) {
			c.mu.Lock()
			c.menu = updateOperationProgress(c.menu, origin.Attempt, detail, time.Now())
			c.mu.Unlock()
			c.repaint()
		})
		return fn(couchcore.OperationCall{Name: effect.Operation, Args: requestArgs, TypedPayload: effect.TypedPayload, Implicit: true, Context: operationContext})
	}}
	if effect.Operation == "switch" {
		c.mu.Lock()
		hosted := c.switchTargetForAddressLocked(origin.Address)
		c.mu.Unlock()
		if hosted != "" {
			request.navigation = func() (any, error) {
				ctx := context.WithValue(c.lifetime, menuOperationContextKey{}, menuOperationContext{origin: origin, hosted: hosted})
				if attempt != nil {
					ctx = couchcore.WithOperationAttempt(ctx, attempt)
				}
				return fn(couchcore.OperationCall{Name: effect.Operation, Args: requestArgs, Implicit: true, Context: ctx})
			}
		}
	}
	_, err := c.operationQueue.Enqueue(request)
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
	var admission context.Context
	route(before)
	if hit == HitNone {
		return
	}
	c.mu.Lock()
	actorFocused := !c.focus.IsPanel()
	c.mu.Unlock()
	if actorFocused && hit == HitRelaunch {
		c.mu.Lock()
		probe := c.shortcutFocused
		var address couchcore.ThreadAddress
		observedActor := c.active
		var observedEndpoint *terminal.Endpoint
		if p := c.panes[observedActor]; p != nil {
			address = p.thread
			observedEndpoint = p.child.Endpoint()
		}
		c.mu.Unlock()
		if probe != nil {
			ctx, cancel := context.WithTimeout(c.lifetime, time.Second)
			defer cancel()
			admission = ctx
			role, err := probe(ctx, address)
			if err == nil {
				err = ctx.Err()
			}
			if err != nil {
				c.setNotice("Cannot determine restart-key focus: " + err.Error())
				return
			}
			c.mu.Lock()
			current := c.panes[c.active]
			unchanged := !c.focus.IsPanel() && c.active == observedActor && current != nil && current.thread == address && current.child.Endpoint() == observedEndpoint && !observedEndpoint.InputEnded()
			c.mu.Unlock()
			if !unchanged {
				c.setNotice("Cannot determine restart-key focus: focus changed during observation")
				return
			}
			chord, _ := workbenchshortcut.DecodeChord(rawHit)
			if role == workbenchshortcut.PaneRoleRightTerminal || role == workbenchshortcut.PaneRoleReview && chord == workbenchshortcut.ChordAltN {
				if err := zellijreply.Effect(ctx, func() error { route(rawHit); return nil }); err != nil {
					c.setNotice("Cannot determine restart-key focus: " + err.Error())
				}
				return
			}
		}
	}
	if actorFocused && hit != HitMouse && !hit.actorReserved() {
		route(rawHit)
		return
	}
	// The declared handler table is checked against AllInterceptorHits.
	if handle := c.hitHandlers()[hit]; handle != nil {
		if admission != nil {
			if err := zellijreply.Effect(admission, func() error { handle(); return nil }); err != nil {
				c.setNotice("Cannot determine restart-key focus: " + err.Error())
			}
		} else {
			handle()
		}
	} else {
		c.setNotice(fmt.Sprintf("chord %d is intercepted but has no handler", hit))
	}
}

// SetShortcutFocusProbe supplies inner-pane ownership. The outer child is
// Zellij, whose alternate screen says nothing about its focused inner pane.
// Only restart candidates query it; ordinary input never pays for discovery.
func (c *Console) SetShortcutFocusProbe(probe func(context.Context, couchcore.ThreadAddress) (workbenchshortcut.PaneRole, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shortcutFocused = probe
}

// hitHandlers maps every intercepted chord to what the console does about it.
// A method rather than a package var because the handlers are bound to this
// Console; the point is that the mapping is DATA a test can walk, not control
// flow it can only execute.
func (c *Console) hitHandlers() map[InterceptorHit]func() {
	return map[InterceptorHit]func(){
		HitSwitch:    c.onHotkey,
		HitPark:      c.onParkHotkey,
		HitPrevious:  c.onPreviousHotkey,
		HitDetach:    c.onDetachHotkey,
		HitRelaunch:  c.onRelaunchHotkey,
		HitBroadcast: c.toggleBroadcast,
		HitWebcast:   c.toggleWebcast,
		HitAttention: c.toggleAttention,
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
	if completed.owner != nil && !completed.owner.CompareAndSwap(false, true) {
		return false
	}
	err := completed.err
	defer func() {
		c.finishContinuationOperation(completed, err)
	}()
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
			// A failed start may carry an empty typed result. Keep its exact
			// attempt origin so the reducer can finish it, while retaining a
			// genuine partial launch's nonempty address.
			if started.Record.Thread != (couchcore.ThreadAddress{}) {
				address = started.Record.Thread
			}
			if started.Handle != nil {
				startedHandleID = started.Handle.ID()
			}
		}
	}
	c.observeLifecycleCompletion(completed, address, err)
	event := MenuEvent{
		Kind: MenuEventOperationResult, Operation: completed.origin.Operation,
		Attempt: completed.origin.Attempt, Address: address, Success: err == nil,
		Background: completed.origin.Background,
	}
	if err != nil {
		var recoveryUnconfirmed *couchcore.RecoveryLaunchUnconfirmed
		var reloadUnconfirmed *couchcore.ReloadUnconfirmed
		event.Unconfirmed = errors.As(err, &recoveryUnconfirmed) || errors.As(err, &reloadUnconfirmed)
		var required *couchcore.RecoveryConfirmationRequired
		if errors.As(err, &required) {
			event.RecoveryConfirmation = &required.Offer
		}
		event.Error = couchcore.OperatorError(err)
		// A code, so the pass tells a skip from a failure without matching text.
		event.Diagnostic = couchcore.ResumeDiagnosticOf(err)
		event.Busy = couchcore.IsThreadBusy(err)
	}
	if completed.origin.Background {
		c.traceEvent(traceReattachDone, address, reattachDoneDetail(event.Success, event.Diagnostic))
	}
	c.mu.Lock()
	if remoteResumeCompletion(completed.origin) {
		// A remote resume clears its row's reattach-failure mark, as the
		// switcher's resume does at dispatch (pair#206 cell 9).
		if row, ok := remoteOperationAddress(c.menu, completed.remote); ok {
			c.menu = clearReattachFailure(c.menu, row)
		}
	}
	if completed.origin.Operation == "switch" {
		// Only the selection owner acknowledges. Finishing always releases this
		// job's capture, including a successful switch superseded before landing.
		c.attention.Cancel(completed.origin.AttentionCapture)
		c.syncAttentionLocked()
	}
	if event.Success && operationNeedsProjectionRefresh(event.Operation) {
		event.ProjectionAfterGeneration = c.refreshSchedule.Sequence
	}
	// A lifecycle operation's child exit is expected in EITHER event order.
	// c.exited and c.operationQueue.results are separate select cases and Go
	// picks uniformly among ready ones, so roughly half the time the completion
	// wins the race, ReduceMenu completes the row job, and the exit falls through to
	// consumeExpectedParkExitLocked's running-job arm with nothing to match. This
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
				c.expectedExits.mark(id)
			}
		}
	}
	var menuEffects []MenuEffect
	if c.menuReady {
		c.menu, menuEffects = ReduceMenu(c.menu, event)
	}
	panelFocused := c.focus.IsPanel()
	land := completed.origin.Attempt != 0 && c.menu.LandingAttempt == completed.origin.Attempt && len(c.menu.Frames) == 1
	continuationLand := completed.origin.ContinuationID != "" && !completed.origin.PreserveFocus && completed.origin.NavigationGeneration == c.navigationGeneration
	cancelPass := c.completedReattachPassLocked()
	c.mu.Unlock()
	c.observeLifecyclePass()
	// A completion can start the reattach pass's next attempt (pair#206).
	if cancelPass != nil {
		cancelPass()
	}
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
	if warned, ok := completed.value.(interface{ Warning() string }); ok && err == nil {
		if message := warned.Warning(); message != "" {
			c.reduceMenu(MenuEvent{Kind: MenuEventNotice, Error: message})
			if source, actionable := lifecycleCompletionWarningSource(completed.value, completed.origin.Address); actionable {
				c.Warn(Notice{Class: couchcore.NoticeLifecycleResultWarning, Kind: "status", Thread: source, Body: message})
			} else {
				c.setNotice(message)
			}
		}
	}
	if err == nil && startedHandleID != "" && !completed.origin.Background && !completed.origin.PreserveFocus && (land || continuationLand) {
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
	if c.expectedExits.consume(id) {
		return true
	}
	for _, job := range c.menu.Operations {
		if job.Phase != MenuOperationRunning {
			continue
		}
		origin := job.Origin
		if endsItsOwnChild(origin.Operation) && origin.Address == address {
			return true
		}
		if origin.Operation == "leave" {
			// Leave detaches every thread, so every child exit it causes is
			// expected. Without this the operator gets a burst of exit notices on
			// the way out -- exactly the noise the notification design exists to
			// keep meaningful.
			return true
		}
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
		if call.Context == nil {
			call.Context = c.lifetime
		}
		menuCall, _ := call.Context.Value(menuOperationContextKey{}).(menuOperationContext)
		c.mu.Lock()
		target := c.switchTargetForAddressLocked(address)
		// A notification hop is a landing on an actor that HAD a pending
		// notification when the operator chose it; on this path that means
		// Return on a paging row. runMenuOperation captured that set before
		// dispatch, so a nonzero capture is exactly "the target was paging when
		// the operator chose it" -- the value that was true when they chose,
		// not after.
		how := arrivalOrdinary
		if job, ok := menuRunningForAddress(c.menu, address); ok && job.Origin.Operation == "switch" &&
			job.Origin.AttentionCapture != 0 {
			how = arrivalNotification
		}
		c.mu.Unlock()
		if menuCall.hosted != "" && target != menuCall.hosted {
			return nil, errNavigationNeedsLifecycle
		}
		if target == "" {
			// A restored registry entry does not restore its PTY handle. Route
			// through resume's fresh terminal proof and let normal completion
			// install the returned helper, rather than treating the row as lost.
			c.mu.Lock()
			fn := c.ops
			listed := false
			for _, row := range menuRows(c.menu) {
				if row.Address == address && couchcore.ContinuityOutcome(row) == couchcore.Continue {
					listed = true
					break
				}
			}
			c.mu.Unlock()
			if fn == nil || !listed {
				return nil, fmt.Errorf("thread %s/%s is not attached to this console", address.RepoScope, address.Tag)
			}
			return fn(couchcore.OperationCall{Name: "resume", Context: call.Context, Implicit: true,
				Args: map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag), "warm-only": "true"}})
		}
		err := c.runTerminalCommand(call.Context, func() error {
			c.mu.Lock()
			current := c.switchTargetForAddressLocked(address)
			allowed := menuCall.origin.Attempt == 0 || c.menu.LandingAttempt == menuCall.origin.Attempt
			c.mu.Unlock()
			if menuCall.hosted != "" && current != menuCall.hosted {
				return errNavigationNeedsLifecycle
			}
			if !allowed {
				return nil
			}
			_, err := c.selectActorContext(call.Context, target, true, how)
			return err
		})
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

// Notify publishes a notice from outside the console, e.g. the composition
// root reporting a previous incarnation's crash (#397).
func (c *Console) Notify(n Notice) { c.publishNotice(n) }

// GoTracked runs f off the console goroutine, on a worker the console joins at
// shutdown (pair#205): for work that may wait on a thread's gate and so must
// never block rendering, input, or the operation results the queue workers
// deliver. Once the console has stopped, f runs inline -- there is no event
// loop left to block, and joining has already begun.
func (c *Console) GoTracked(f func()) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		f()
		return
	}
	c.workers.Add(1)
	c.mu.Unlock()
	go func() {
		defer c.workers.Done()
		f()
	}()
}

func (c *Console) setNotice(text string) {
	c.publishNotice(Notice{Kind: "status", Body: text})
}

// Warn publishes an actionable asynchronous event to the existing attention list
// and feed. Routine progress, success and key feedback use Notify/setNotice.
func (c *Console) Warn(n Notice) {
	c.mu.Lock()
	started := c.publishWarningLocked(n)
	c.mu.Unlock()
	if started {
		c.repaint()
	}
}

func (c *Console) setWarning(text string) {
	c.Warn(Notice{Class: couchcore.NoticeUnknown, Kind: "warning", Body: text})
}

// publishWarningLocked uses the existing console-to-attention lock order. No IO
// or repaint occurs while either owner is held.
func (c *Console) publishWarningLocked(n Notice) bool {
	if c.lifetime.Err() != nil {
		return false
	}
	if n.SourceSlot == nil && n.Thread != (couchcore.ThreadAddress{}) {
		for _, row := range c.resourceRows {
			if row.Address == n.Thread {
				if n.SourceRoot == "" {
					n.SourceRoot = row.StartingPath
				}
				if row.Target.Kind == couchcore.ThreadTargetSlot {
					slot := couchcore.HubSlot{Repository: row.Target.Slot.PrimaryRoot, Number: row.Target.Slot.Number}
					n.SourceSlot = &slot
				}
				break
			}
		}
	}
	started := c.started
	if !n.Cleared {
		started = c.publishNoticeLocked(n)
	}
	if c.noticeRouter != nil {
		c.noticeRouter(n) // bounded memory handoff only; never re-enters Console
		return started
	}
	if n.Cleared {
		return false
	}
	if !couchcore.RouteNotice(couchcore.NoticeRouteInput{Class: n.Class, Origin: n.Origin, Owner: couchcore.NoticeOwnerUnavailable, HubHandled: n.HubHandled}).Bell {
		return started
	}
	c.noticeSequence++
	if item, ok := noticeAttention(n, c.noticeSequence); ok {
		c.attn.mu.Lock()
		c.attn.list = c.attn.list.Apply(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: item}, c.now())
		c.layoutAttentionLocked(int(c.size.Cols), int(c.size.Rows)-c.statusRows+2)
		c.attn.mu.Unlock()
	}
	return started
}

// publishNotice retains feedback and repaints the current surface. The switcher
// derives its latest message from this feed; general messages do not use chrome.
//
// The repaint waits for Run: before it, the terminal is not in raw mode or the
// alt screen. Pre-run messages remain retained for the first switcher paint.
func (c *Console) publishNotice(n Notice) {
	c.mu.Lock()
	started := c.publishNoticeLocked(n)
	c.mu.Unlock()
	if started {
		c.repaint()
	}
}

// publishNoticeLocked publishes while the caller holds c.mu; repaint after unlocking.
func (c *Console) publishNoticeLocked(n Notice) bool {
	c.feed.Push(n)
	return c.started
}
