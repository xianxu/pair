package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/lifetimelease"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"golang.org/x/sys/unix"
)

// messageAuthority separates live evidence from registry mutation. All probes
// are read-only; none call mutable Couch domain operations from request workers.
type messageAuthority struct {
	automation *hubAutomationOwner
	// now is the broker clock; nil uses wall time.
	now        func() time.Time
	thread     func(context.Context, couchmessage.Binding) (string, error)
	workspace  func(context.Context, string) (couchcore.WorkspaceIdentity, error)
	process    func(couchmessage.Binding) error
	wrapperPID func(couchmessage.Binding) (int, error)
	// launch proves the launch registration and live session ownership; its
	// production form runs the ps/zellij ownership probe, so it runs only at
	// admission (#365).
	launch func(context.Context, couchmessage.Binding) error
	// recorded is launch's file evidence alone — the ready file's nonce and
	// the session index's name — for use-time checks. It spawns nothing.
	recorded func(context.Context, couchmessage.Binding) error
	// child names the native child the caller's ready file records (launch
	// ordinal, process identity), fencing a native-exit report to it
	// (pair#657); nil refuses native-exit reports.
	child    func(context.Context, couchmessage.Binding) (uint64, string, error)
	endpoint func(couchmessage.Binding) couchmessage.DeliveryEndpoint
	branch   func(context.Context, string) (couchcore.SlotGitStatus, error)
	// families lists every enrolled message family with its alias ("" when
	// none); nil means routing sees live bindings only.
	families func(context.Context) (map[string]string, error)
	// agent names the agent a thread's record says it launched, so a slot
	// operation's caller can be checked against that launch's recorded files
	// without a messaging binding (pair#367).
	agent func(context.Context, couchcore.ThreadAddress) (string, error)
	// record reads a thread's record, so a tail read by slot (pair#429)
	// names its agent and working path as the typed peek does; nil names the
	// binding's agent only.
	record func(context.Context, couchcore.ThreadAddress) (couchcore.ThreadRecord, error)
	// hub is the worktree root Couch started from: the TL slot, which gets
	// the larger peer-message bucket (#433). "" when Couch started outside a
	// checkout.
	hub                    string
	hubs                   *hubRuntime
	hubTarget              func(context.Context, string) (couchcore.HubSlot, error)
	hubRepositoryFleetRoot string
	hubRepositoryNames     func(context.Context) ([]couchcore.RepositoryName, error)
	hubRepositorySlots     func(context.Context, string) (map[int]couchcore.HubSlot, error)
	// batchMaxChars is immutable startup policy; zero uses the qualified default.
	batchMaxChars int
}

// live is the full check, run once per wrapper admission: a lifecycle event, never a
// timer.
func (a messageAuthority) live(ctx context.Context, b couchmessage.Binding) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := b.Validate(); err != nil {
		return "", err
	}
	root, err := a.current(ctx, b)
	if err != nil {
		return "", err
	}
	if err = a.process(b); err != nil {
		return "", err
	}
	if err = a.launch(ctx, b); err != nil {
		return "", err
	}
	// External probes may yield while a terminal or its retained checkout changes.
	current, err := a.thread(ctx, b)
	if err != nil {
		return "", err
	}
	if current != root {
		return "", errors.New("message checkout changed during verification")
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	return root, nil
}

// current is the use-time check at Observe/Reserve/Deliver and for a sending
// caller: the exact terminal is still positively listed, the PID file still names
// this wrapper (a surviving old wrapper after replacement), and the launch's
// recorded files still name this nonce and session (an index rewritten while
// the old wrapper survives headless). Death and exec arrive as session close.
func (a messageAuthority) current(ctx context.Context, b couchmessage.Binding) (string, error) {
	root, err := a.thread(ctx, b)
	if err != nil {
		return "", err
	}
	pid, err := a.wrapperPID(b)
	if err != nil {
		return "", err
	}
	if pid != b.PID {
		return "", &couchmessage.RelaunchRequiredError{Cause: errors.New("wrapper does not own the current binding")}
	}
	if err = a.recorded(ctx, b); err != nil {
		return "", err
	}
	return root, ctx.Err()
}

// recordedMessageAuthority shares the use-time file checks with callers that
// supply their own artifact reader. Every invocation observes readiness afresh;
// sessionName owns any metadata cache, never live wrapper authority.
func recordedMessageAuthority(reader couchcore.OSReadyReader, sessionName func(context.Context, couchcore.ThreadAddress) (string, error)) func(context.Context, couchmessage.Binding) error {
	return func(ctx context.Context, b couchmessage.Binding) error {
		address := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
		ready, ok, e := reader.RecordedSession(ctx, address, b.Agent, b.Nonce)
		if e != nil {
			return &couchmessage.RelaunchRequiredError{Cause: fmt.Errorf("wrapper launch registration: %w", e)}
		}
		if !ok || ready != b.Session {
			return &couchmessage.RelaunchRequiredError{Cause: errors.New("wrapper launch registration unavailable")}
		}
		name, e := sessionName(ctx, address)
		if e != nil {
			return e
		}
		if name != b.Session {
			return &couchmessage.RelaunchRequiredError{Cause: errors.New("wrapper conversation changed")}
		}
		return nil
	}
}

// Admissions run concurrently up to this bound, so a broker restart with N
// surviving wrappers costs N full checks, a few at a time.
const (
	messageAdmissionWorkers = 4
	messageAdmissionTimeout = 10 * time.Second
)

// messageService turns registry effects into IO (#365). One goroutine owns
// the Registry and applies wrapper sessions, admission results and retry
// timers in arrival order. Request goroutines read
// only the published snapshot of connected bindings.
type messageService struct {
	automation *hubAutomationOwner
	lifecycle  atomic.Pointer[lifecyclePublisher]
	resources  atomic.Pointer[resourceSampler]
	hubMoves   atomic.Pointer[hubMoveService]
	facts      atomic.Pointer[supervisionFactLog]
	server     *couchmessage.Server
	sessions   *couchmessage.SessionServer
	broker     *couchmessage.Broker
	cancel     context.CancelFunc
	lifetime   context.Context
	workers    sync.WaitGroup
	authority  messageAuthority
	// hubTargets is admission's warm source for target slots (pair#726); nil
	// without hub ownership.
	hubTargets *hubTargetMemo
	panes      *couchmessage.PaneMailbox
	inbox      chan messageInput
	admitting  chan struct{}
	connected  atomic.Pointer[map[couchmessage.Binding]bool]

	// hostedIngest is installed once the Console is ready; requests take a fresh snapshot.
	hostedIngest atomic.Pointer[func() couchcore.HostedIngestSnapshot]

	// liveness is the registry's per-slot self-report snapshot (#421),
	// republished after every step like connected.
	liveness atomic.Pointer[map[string]couchmessage.SlotLiveness]
	// blocked hands the Console the threads behind a picker (pair#465). The
	// mutex orders each change with the view's installation, so a stale set
	// never lands after a newer one; the view itself never blocks.
	blocked struct {
		sync.Mutex
		view func(map[couchcore.ThreadAddress]bool)
		last map[couchcore.ThreadAddress]bool
	}
	// after schedules a retry; tests replace it to fire retries on demand.
	after func(time.Duration, func())
	// slotOps answers resume, reboot and operation-status (pair#367 M2);
	// nil answers them unsupported.
	slotOps       *slotOperations
	switches      atomic.Pointer[agentSwitchService]
	drives        atomic.Pointer[agentDriveService]
	desktopNotify atomic.Pointer[func(context.Context, string, string) error]
	// attention reports facts to the Console's attention list (pair#673).
	attention atomic.Pointer[func(couchcore.AttentionFact)]
	// Retained across disconnects; only a verified replacement supersedes a slot.
	stalls struct {
		sync.Mutex
		records map[string]automaticInputStallRecord
	}
	// exits remembers the native exit last reported per slot, so a repeat of
	// the same (binding, launch) posts nothing (pair#657).
	exits struct {
		sync.Mutex
		records map[string]nativeExitRecord
	}
	// exited hands a confirmed native exit, with the caller's verified slot,
	// to the lifecycle notices; nil when lifecycle notices are off.
	exited atomic.Pointer[func(couchmessage.Binding, couchcore.WorkspaceIdentity, couchmessage.NativeExit)]

	// broadcastStatus answers broadcast-status (pair#413). Installed after the
	// service is serving, so it is an atomic pointer; nil answers unsupported.
	broadcastStatus atomic.Pointer[func() (couchmessage.BroadcastStatus, bool)]
	// shares answers broadcast-status's list of every link out (pair#467).
	shares atomic.Pointer[func() []couchmessage.ShareStatus]
	// replay answers replay-share and replay-status (pair#477), like
	// broadcastStatus; nil answers them unsupported.
	replay     atomic.Pointer[func(op, id string) couchmessage.ReplayShare]
	submission atomic.Pointer[func(context.Context, string, string, core.Target) (couchmessage.SubmissionStatus, error)]
	// replays lists the shared recordings for broadcast-status; nil lists none.
	replays atomic.Pointer[func() []string]
	// deliveries records each landed delivery for session-sync owed events
	// (pair#437); nil records nothing.
	deliveries  atomic.Pointer[deliveryRecorder]
	supervision atomic.Pointer[supervisionCoordinator]

	mu sync.Mutex
	// workspaces holds each connected binding's verified workspace, for the
	// resting check; it shrinks with Disconnect, so it is bounded by MaxActors.
	workspaces map[couchmessage.Binding]couchcore.WorkspaceIdentity
	// prepared carries a passed admission's evidence to its Connect effect.
	// The loop drops each entry when it processes that admission's result, so
	// a superseded check leaves nothing behind.
	prepared map[preparedKey]preparedAdmission
}

type preparedKey struct {
	token couchmessage.SessionToken
	pane  couchmessage.PaneHandle
}

type messageInput struct {
	event couchmessage.RegistryEvent
	reply chan error
}

type preparedAdmission struct {
	identity    couchcore.WorkspaceIdentity
	endpoint    couchmessage.DeliveryEndpoint
	observation couchmessage.Observation
}

func (s *messageService) Close() {
	s.cancel()
	_ = s.server.Close()
	_ = s.sessions.Close()
	_ = s.broker.Close()
	s.workers.Wait()
	if s.hubTargets != nil {
		s.hubTargets.wait()
	}
	if log := s.facts.Load(); log != nil {
		log.Close()
	}
}

// isHub reports whether a verified workspace is the TL hub's.
func (a messageAuthority) isHub(identity couchcore.WorkspaceIdentity) bool {
	if a.hubs != nil {
		return a.hubs.snapshot().IsHub(identity.WorktreeRoot)
	}
	return a.hub != "" && identity.WorktreeRoot == a.hub
}

func (a messageAuthority) owner(identity couchcore.WorkspaceIdentity) string {
	if a.hubs != nil {
		return a.hubs.owner(identity)
	}
	return a.hub
}

// hubWorktree is the worktree root Couch started in: the TL slot (#433). A
// start outside any checkout has no hub, so every slot gets the ordinary bucket.
func hubWorktree(getwd func() (string, error), resolve func(string) (couchcore.Worktree, error)) string {
	cwd, err := getwd()
	if err != nil {
		return ""
	}
	tree, err := resolve(cwd)
	if err != nil {
		return ""
	}
	return string(tree)
}

// startMessageService is called only while the existing supervisor lease is
// held. Socket cleanup and all child registration live inside that lifetime.
func startMessageService(console *couchtty.Console, c *couchcore.Couch, configuredHubs ...*hubRuntime) (*messageService, error) {
	return startMessageServiceWithStartup(console, c, nil, configuredHubs...)
}

func startMessageServiceWithStartup(console *couchtty.Console, c *couchcore.Couch, startup *couchcore.StartupObservation, configuredHubs ...*hubRuntime) (*messageService, error) {
	resolver, ok := c.Slots.(couchcore.SlotWorkspaceResolver)
	if !ok {
		return nil, nil
	}
	artifacts, ok := c.Artifacts.(couchcore.PairLifecycleEnvironment)
	if !ok || c.Proc == nil || c.Git == nil {
		return nil, nil
	}
	// Validated before any socket is taken: a bad knob refuses the start.
	resourceEnv := map[string]string{}
	for _, item := range os.Environ() {
		key, value, _ := strings.Cut(item, "=")
		if strings.HasPrefix(key, "COUCH_RESOURCE_") {
			resourceEnv[key] = value
		}
	}
	resourcePolicy, err := couchcore.ParseResourcePolicy(resourceEnv)
	if err != nil {
		return nil, err
	}
	threshold, err := waitNoticeThreshold(os.Getenv)
	if err != nil {
		return nil, err
	}
	lifecycleOn, err := lifecycleNoticesEnabled(os.Getenv)
	if err != nil {
		return nil, err
	}
	batchChars, err := peerBatchMaxChars(os.Getenv)
	if err != nil {
		return nil, err
	}
	reniceLevel, err := slotNiceSettings(os.Getenv)
	if err != nil {
		return nil, err
	}
	reniceEvery, err := reniceInterval(os.Getenv)
	if err != nil {
		return nil, err
	}
	switchGuard := &switchServiceGuard{}
	c.SwitchGuard = switchGuard
	namespace := c.Namespace.Dir()
	// Lifetime leases left by retirements that had no proof to remove them
	// (pair#695): only the free ones go, so a live tree keeps its lease, and
	// Hold reopens if one goes under it. Debris only; a failure keeps it.
	_, _ = lifetimelease.SweepFree(lifetimelease.Dir(namespace))
	brokerSocket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return nil, err
	}
	registrySocket, err := couchmessage.SocketPath(namespace, "registry")
	if err != nil {
		return nil, err
	}
	// The message authority reads zellij liveness from one polled snapshot
	// (pair#618): it runs on every admission and queued-message retry, and
	// listing per check starved admission under load.
	// Its loop lists at once when the service starts; until then a check refuses
	// as not yet known rather than blocking startup on zellij.
	listings := launcher.NewListingPoller(launcher.ZellijSource{}, messageListingInterval, messageListingMaxStale)
	reader := couchcore.OSReadyReader{DataDir: artifacts.PairLifecycleDataDir(), Session: artifacts.PairSession, Proc: c.Proc}
	session := func(ctx context.Context, address couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
		if err := ctx.Err(); err != nil {
			return couchcore.PairSessionBinding{}, err
		}
		v, e := artifacts.PairSession(address)
		if e == nil {
			e = ctx.Err()
		}
		return v, e
	}
	if source, ok := c.Artifacts.(interface {
		PairSessionContext(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
	}); ok {
		reader.SessionContext = source.PairSessionContext
		session = source.PairSessionContext
	}
	sessionName := func(ctx context.Context, address couchcore.ThreadAddress) (string, error) {
		return "", errors.New("recorded Pair session name unavailable")
	}
	if source, ok := c.Artifacts.(interface {
		PairSessionName(context.Context, couchcore.ThreadAddress) (string, error)
	}); ok {
		sessionName = source.PairSessionName
	}
	authority := messageAuthority{
		hub:           hubWorktree(os.Getwd, c.ResolveTree),
		batchMaxChars: batchChars,
		families: func(ctx context.Context) (map[string]string, error) {
			if c.Threads == nil {
				return nil, nil
			}
			names, err := c.Threads.RepositoryNamesContext(ctx)
			if err != nil {
				return nil, err
			}
			return messageFamilies(names), nil
		},
		thread: messageThreadResolver(c, listings), workspace: resolver.ResolveWorkspace,
		record: func(ctx context.Context, address couchcore.ThreadAddress) (couchcore.ThreadRecord, error) {
			if c.Threads == nil {
				return couchcore.ThreadRecord{}, errors.New("no thread store")
			}
			return c.Threads.PreviewThreadContext(ctx, address)
		},
		agent: func(ctx context.Context, address couchcore.ThreadAddress) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if c.Threads == nil {
				return "", errors.New("no thread store")
			}
			record, err := c.Threads.PreviewThreadContext(ctx, address)
			if err != nil {
				return "", err
			}
			if record.LatestLaunchProfile == nil || record.LatestLaunchProfile.Agent == "" {
				return "", errors.New("thread records no launched agent")
			}
			return record.LatestLaunchProfile.Agent, nil
		},
		process: func(b couchmessage.Binding) error {
			switch c.Liveness(couchcore.ActorRecord{PID: b.PID, Identity: b.Start}) {
			case couchcore.Dead:
				return errors.New("wrapper exited or was replaced")
			case couchcore.Unknown:
				return fmt.Errorf("cannot verify wrapper pid %d", b.PID)
			}
			return nil
		},
		wrapperPID: func(b couchmessage.Binding) (int, error) {
			paths, e := artifactpath.Resolve(artifactpath.Address{DataDir: artifacts.PairLifecycleDataDir(), RepoScope: b.Scope, Tag: b.Tag})
			if e != nil {
				return 0, e
			}
			return readMessageWrapperPID(paths.PairWrapPID())
		},
		launch: func(ctx context.Context, b couchmessage.Binding) error {
			address := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
			registered, e := reader.Registered(ctx, address, b.Agent, b.Nonce)
			if e != nil {
				return &couchmessage.RelaunchRequiredError{Cause: fmt.Errorf("wrapper launch registration: %w", e)}
			}
			if !registered {
				return &couchmessage.RelaunchRequiredError{Cause: errors.New("wrapper launch registration unavailable")}
			}
			current, e := session(ctx, address)
			if e != nil {
				return e
			}
			if !current.Present || current.Name != b.Session {
				return &couchmessage.RelaunchRequiredError{Cause: errors.New("wrapper conversation changed")}
			}
			return nil
		},
		recorded: recordedMessageAuthority(reader, sessionName),
		child: func(ctx context.Context, b couchmessage.Binding) (uint64, string, error) {
			ordinal, identity, ok, err := reader.RecordedChild(ctx, couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}, b.Agent, b.Nonce)
			if err != nil {
				return 0, "", err
			}
			if !ok {
				return 0, "", errors.New("wrapper launch registration unavailable")
			}
			return ordinal, identity, nil
		},
		endpoint: func(b couchmessage.Binding) couchmessage.DeliveryEndpoint {
			return couchmessage.RemoteEndpoint{Namespace: namespace, Binding: b}
		},
		branch: func(ctx context.Context, root string) (couchcore.SlotGitStatus, error) {
			return couchcore.ProbeSlotGit(ctx, c.Git, root)
		},
	}
	if len(configuredHubs) > 0 && configuredHubs[0] != nil {
		authority.hubs = configuredHubs[0]
		authority.hubTarget = hubTargetResolver(c)
		// Match the supervisor context used by fresh WorkspaceReferencePath.
		// If startup cannot establish it, retain the per-reference resolver rather
		// than guess sibling precedence from enrolled names.
		startup, stop := context.WithTimeout(context.Background(), messageAdmissionTimeout)
		identity, contextErr := resolver.ResolveWorkspace(startup, ".")
		stop()
		if contextErr == nil {
			authority.hubRepositoryFleetRoot = identity.FleetRoot
			authority.hubRepositoryNames = c.Threads.RepositoryNamesContext
			authority.hubRepositorySlots = c.RepositoryHubSlots
		}
		authority.hub = authority.hubs.snapshot().Main()
	}
	panes := couchmessage.NewPaneMailbox()
	probe := newLiveRestartProbe(c.Git, c.Proc)
	c.LiveRestart = probe // before the socket opens: no request can race this write
	// A reboot stops the slot's registered wrapper tree (pair#607); the
	// broker's binding names it by pid and kernel start identity.
	c.SlotWrapper = func(address couchcore.ThreadAddress) (couchcore.ProcessIdentity, bool) {
		live, ok := probe.service.Load().LivenessForThread(address.RepoScope, string(address.Tag))
		if !ok {
			return couchcore.ProcessIdentity{}, false
		}
		return couchcore.ProcessIdentity{PID: live.Binding.PID, Identity: live.Binding.Start}, true
	}
	c.ProcessTable = procutil.Table
	c.LeaseHolders = lifetimelease.Holders
	facts := openSupervisionFacts(artifacts.PairLifecycleDataDir(), namespace)
	initial := map[string]bool{}
	seed := func(root string) {
		initial[root] = startup != nil && startup.Hubs[root].Survived && !startup.Hubs[root].Unknown
	}
	if authority.hubs != nil {
		for _, hub := range authority.hubs.snapshot().Hubs() {
			seed(hub.Root)
		}
	} else if authority.hub != "" {
		seed(authority.hub)
	}
	kind := couchcore.StartupUnknown
	var missing []string
	if startup != nil {
		kind, missing = startup.Kind, startup.MissingHubs
	}
	authority.automation = newHubAutomationOwner(initial, kind, missing)
	service, err := newMessageService(context.Background(), brokerSocket, registrySocket, authority, panes, console.MessageSlotGit, consoleSlotOperations(console, c, probe), facts)
	if err != nil {
		facts.Close()
		return nil, err
	}
	probe.service.Store(service)
	console.SetAutomationControl(func(root, action string) (bool, error) {
		status, err := service.controlAutomation(root, action)
		return status.Enabled, err
	})
	service.installAgentSwitchService(console, c, switchGuard)
	service.installAgentDriveService(console)
	service.installHubMoves(console, c)
	service.SetDesktopNotify(console.NotifyDesktop)
	console.SetMessageBroker(service.broker)
	console.SetCopyObserver(facts.clipboardCopy)
	console.SetCopyKeyObserver(facts.clipboardKey)
	service.SetBroadcastStatus(console.BroadcastStatus)
	service.SetShares(console.Shares)
	service.SetBlockedView(console.SetBlockedThreads)
	service.SetAttention(console.ObserveAttention)
	console.SetWrapperActivity(service.WrapperActivity)
	service.showHub(console)
	service.SetReplay(console.ReplayShare, console.SharedReplays)
	service.SetSubmission(console.Submission)
	service.recordDeliveries(namespace)
	service.supervision.Store(newSupervisionCoordinator(service, func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) }))
	if lifecycleOn {
		publisher := startLifecyclePublisher(service, func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) })
		console.SetLifecycleObserver(publisher.observe)
		wireNativeExitNotices(service, console)
	}
	watcher := newWaitWatcher(service, threshold, namespace)
	stale := newSessionStaleWatcher(service, namespace, func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) })
	watcher.staleTick = stale.offer
	service.workers.Add(2)
	go func() { defer service.workers.Done(); stale.run(service.lifetime) }()
	go func() { defer service.workers.Done(); watcher.run(service.lifetime, waitTick) }()
	service.workers.Add(1)
	go func() { defer service.workers.Done(); listings.Run(service.lifetime) }()
	if c.ScratchMaintenance != nil {
		service.workers.Add(1)
		go func() { defer service.workers.Done(); c.ScratchMaintenance(service.lifetime) }()
	}
	// The slot renicer (#502): what each slot's agent spawns runs at
	// COUCH_SLOT_NICE; Couch, the wrapper, zellij and the agent stay at
	// normal priority.
	rn := newRenicer(osProcs{}, service.broker.Connected, func(b couchmessage.Binding) (int, error) {
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: artifacts.PairLifecycleDataDir(), RepoScope: b.Scope, Tag: b.Tag})
		if err != nil {
			return 0, err
		}
		return readMessageWrapperPID(paths.AgentPID()) // the same bare-decimal pid file
	}, reniceLevel, os.Getpid(), func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) })
	service.broker.SetNiceView(rn.view)
	service.workers.Add(1)
	go func() { defer service.workers.Done(); rn.run(service.lifetime, reniceEvery) }()
	inventory := newResourceInventory(console, c, service)
	persistence := newResourceHistoryPersistence(namespace)
	alerts := newResourceAlertDispatcher(service)
	sampler := &resourceSampler{reader: procutil.NativeResourceReader{}, slots: inventory.snapshot, policy: resourcePolicy, alerts: alerts.dispatch, persist: persistence.persist, history: persistence.historySummary, sweeps: make(chan chan resourceSweep), leftRunning: couchcore.NewLeftRunningRegistry(c.Namespace)}
	hostedIngest := console.HostedIngestSnapshot
	service.hostedIngest.Store(&hostedIngest)
	service.resources.Store(sampler)
	service.workers.Add(2)
	go func() { defer service.workers.Done(); inventory.run(service.lifetime) }()
	go func() { defer service.workers.Done(); sampler.run(service.lifetime) }()
	// History backup (#592): a daily continuous archive pass when configured.
	archive := newArchiveScheduler(artifacts.PairLifecycleDataDir(), func(text string) { console.Notify(couchtty.Notice{Kind: "status", Body: text}) })
	archive.warn = func(ctx context.Context, body string) (string, error) {
		return service.notifyArchiveWarning(ctx, body, archive.notify)
	}
	service.workers.Add(1)
	go func() { defer service.workers.Done(); archive.run(service.lifetime, archiveCheckEvery) }()
	return service, nil
}

// showHub hands the Console the hub's worktree root (pair#491): the same root
// the message authority resolves "hub" to, so the tab bar leads with the slot
// that messages to the hub reach.
func (s *messageService) showHub(view interface{ SetHub(root string) }) {
	view.SetHub(s.authority.hub)
}

func newMessageService(parent context.Context, brokerSocket, registrySocket string, authority messageAuthority, panes *couchmessage.PaneMailbox, slotGit func(string) (couchcore.SlotGitStatus, bool), slotOps *slotOperations, factLogs ...*supervisionFactLog) (*messageService, error) {
	if authority.batchMaxChars == 0 {
		authority.batchMaxChars = couchmessage.DefaultPeerBatchMaxChars
	}
	if authority.batchMaxChars < 1 || authority.batchMaxChars > couchmessage.MaxPeerBatchChars {
		return nil, errors.New("COUCH_PEER_BATCH_MAX_CHARS must be from 1 through 32768")
	}
	if authority.thread == nil || authority.agent == nil || authority.workspace == nil || authority.process == nil || authority.wrapperPID == nil || authority.launch == nil || authority.recorded == nil || authority.endpoint == nil || authority.branch == nil || panes == nil || slotGit == nil {
		return nil, errors.New("message authority is incomplete")
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	for _, socket := range []string{brokerSocket, registrySocket} {
		if err := prepareMessageSocket(socket); err != nil {
			return nil, err
		}
	}
	lifetime, cancel := context.WithCancel(parent)
	s := &messageService{cancel: cancel, lifetime: lifetime, authority: authority, panes: panes,
		inbox: make(chan messageInput), admitting: make(chan struct{}, messageAdmissionWorkers),
		after: func(d time.Duration, f func()) { time.AfterFunc(d, f) }, slotOps: slotOps,
		workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{},
		prepared:   map[preparedKey]preparedAdmission{}}
	s.automation = authority.automation
	if s.automation == nil {
		initial := map[string]bool{}
		if authority.hubs != nil {
			for _, hub := range authority.hubs.snapshot().Hubs() {
				initial[hub.Root] = false
			}
		} else if authority.hub != "" {
			initial[authority.hub] = false
		}
		s.automation = newHubAutomationOwner(initial, couchcore.StartupUnknown, nil)
	}
	if len(factLogs) > 0 {
		s.facts.Store(factLogs[0])
	}
	s.connected.Store(&map[couchmessage.Binding]bool{})
	if slotOps != nil {
		slotOps.fleetRoot = authority.hubRepositoryFleetRoot
		slotOps.authorizePark = s.authorizeParkAdmission
	}
	if authority.hubTarget != nil {
		// Every fresh resolution refreshes admission's memo (pair#726).
		s.hubTargets = newHubTargetMemo(lifetime, authority.hubTarget, authority.now)
		s.hubTargets.fleetRoot = authority.hubRepositoryFleetRoot
		s.hubTargets.names = authority.hubRepositoryNames
		s.hubTargets.discover = authority.hubRepositorySlots
		s.authority.hubTarget = s.hubTargets.Fresh
	}
	// Family admission's resting check is on the request path (one git status
	// per candidate per send); listings read the Console's slot-git cache.
	now := authority.now
	if now == nil {
		now = time.Now
	}
	s.broker = couchmessage.NewBroker(lifetime, now, func(ctx context.Context, b couchmessage.Binding) (bool, error) {
		identity, err := s.connectedWorkspace(ctx, b)
		if err != nil {
			return false, err
		}
		status, err := authority.branch(ctx, identity.WorktreeRoot)
		return err == nil && !status.Detached && status.Branch == *identity.RestingBranch, err
	})
	s.broker.SetSubmissionObserver(func(b couchmessage.Binding, n uint64) {
		if log := s.facts.Load(); log != nil {
			log.submitted(b, n)
		}
	})
	s.broker.SetFamilies(authority.families)
	if authority.hubs != nil {
		s.broker.SetFamilyEligibility(s.sameHubBindings)
	}
	s.broker.SetRestingView(func(b couchmessage.Binding) (bool, bool) {
		s.mu.Lock()
		identity, ok := s.workspaces[b]
		s.mu.Unlock()
		if !ok || identity.RestingBranch == nil {
			return false, false
		}
		status, ok := slotGit(identity.WorktreeRoot)
		return ok && !status.Detached && status.Branch == *identity.RestingBranch, ok
	})
	s.workers.Add(1)
	go s.loop()
	server, err := couchmessage.StartServer(lifetime, brokerSocket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return json.Marshal(couchmessage.Response{Code: "invalid-request", Error: err.Error()})
		}
		return json.Marshal(s.handle(ctx, request))
	})
	if err != nil {
		cancel()
		_ = s.broker.Close()
		s.workers.Wait()
		return nil, err
	}
	s.server = server
	sessions, err := couchmessage.StartSessionServer(lifetime, registrySocket, couchmessage.SessionHandler{
		Open: func(t couchmessage.SessionToken, h couchmessage.SessionHello) error {
			return s.post(couchmessage.RegistryEvent{Kind: couchmessage.SessionOpened, Token: t, Binding: h.Binding, Build: h.Build, WorkerCue: h.WorkerCue}, true)
		},
		Frame: func(t couchmessage.SessionToken, f couchmessage.SessionFrame) {
			kind := couchmessage.SessionActivity
			if f.Op == couchmessage.FrameSubmit {
				kind = couchmessage.SessionSubmit
			}
			_ = s.post(couchmessage.RegistryEvent{Kind: kind, Token: t, Observation: *f.Observation, Settled: f.Settled, Picker: f.Picker, BackgroundWork: f.BackgroundWork, WorkerCue: f.WorkerCue}, false)
		},
		Closed: func(t couchmessage.SessionToken) {
			_ = s.post(couchmessage.RegistryEvent{Kind: couchmessage.SessionClosed, Token: t}, false)
		},
	})
	if err != nil {
		cancel()
		_ = server.Close()
		_ = s.broker.Close()
		s.workers.Wait()
		return nil, err
	}
	s.sessions = sessions
	if s.hubTargets != nil && authority.hubs != nil {
		var primaries []string
		for _, hub := range authority.hubs.snapshot().Hubs() {
			primaries = append(primaries, hub.Slot.Repository)
		}
		s.hubTargets.Prewarm(primaries)
	}
	return s, nil
}

// post hands an event to the loop; wait returns the registry's verdict. After
// shutdown it drops the event: nothing is left to act on it.
func (s *messageService) post(e couchmessage.RegistryEvent, wait bool) error {
	return s.postContext(s.lifetime, e, wait)
}

// postContext bounds request-owned events by both caller and service lifetime.
func (s *messageService) postContext(ctx context.Context, e couchmessage.RegistryEvent, wait bool) error {
	in := messageInput{event: e}
	if wait {
		in.reply = make(chan error, 1)
	}
	select {
	case s.inbox <- in:
	case <-ctx.Done():
		return ctx.Err()
	case <-s.lifetime.Done():
		return s.lifetime.Err()
	}
	if !wait {
		return nil
	}
	select {
	case err := <-in.reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-s.lifetime.Done():
		return s.lifetime.Err()
	}
}

func (s *messageService) loop() {
	defer s.workers.Done()
	registry := couchmessage.NewRegistry()
	var step func(e couchmessage.RegistryEvent) error
	step = func(e couchmessage.RegistryEvent) error {
		if e.Kind == couchmessage.SessionActivity || e.Kind == couchmessage.SessionSubmit {
			e.At = time.Now()
			if s.authority.now != nil {
				e.At = s.authority.now()
			}
		}
		var before map[couchmessage.Binding]factSlotState
		log := s.facts.Load()
		if log != nil {
			before = factStates(registry.Connected(), registry.Liveness())
		}
		fx, err := registry.Advance(e)
		// Publish first: endpoint checks then never admit a binding the broker
		// is about to drop, nor refuse one it is about to register.
		connected := registry.Connected()
		s.connected.Store(&connected)
		liveness := registry.Liveness()
		s.liveness.Store(&liveness)
		s.publishBlocked(liveness)
		if log != nil {
			log.registry(before, factStates(connected, liveness))
		}
		var followUps []couchmessage.RegistryEvent
		for _, effect := range fx {
			if next := s.execute(effect); next != nil {
				followUps = append(followUps, *next)
			}
		}
		if publisher := s.lifecycle.Load(); publisher != nil {
			publisher.signal()
		}
		if e.Kind == couchmessage.AdmissionDone {
			s.mu.Lock()
			delete(s.prepared, preparedKey{e.Token, e.Pane})
			s.mu.Unlock()
		}
		// An effect that failed reports back before any other event, so the
		// registry never runs ahead of what the broker holds.
		for _, next := range followUps {
			_ = step(next)
		}
		return err
	}
	for {
		select {
		case <-s.lifetime.Done():
			return
		case in := <-s.inbox:
			err := step(in.event)
			if in.reply != nil {
				in.reply <- err
			}
		case <-s.panes.Wake():
			for thread, pane := range s.panes.Drain() {
				_ = step(couchmessage.RegistryEvent{Kind: couchmessage.PaneChanged, Thread: thread, Pane: pane})
			}
		}
	}
}

// execute runs on the loop goroutine; anything slow goes to a worker. An
// effect that fails synchronously returns the event reporting it (ARCH-ORDER):
// Admit reports through AdmissionDone, Connect through ConnectFailed;
// Disconnect and ScheduleRetry cannot fail, and a refused Observe only drops a
// stale observation.
func (s *messageService) execute(e couchmessage.RegistryEffect) *couchmessage.RegistryEvent {
	switch e.Kind {
	case couchmessage.EffectAdmit:
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			err := s.admit(e)
			_ = s.post(couchmessage.RegistryEvent{Kind: couchmessage.AdmissionDone, Token: e.Token, Pane: e.Pane, Err: err}, false)
		}()
	case couchmessage.EffectConnect:
		s.mu.Lock()
		p, ok := s.prepared[preparedKey{e.Token, e.Pane}]
		s.mu.Unlock()
		if !ok {
			// Unreachable: Connect follows the AdmissionDone the worker prepared.
			return &couchmessage.RegistryEvent{Kind: couchmessage.ConnectFailed, Token: e.Token, Pane: e.Pane, Err: errors.New("admission evidence missing")}
		}
		allowance := couchmessage.InboundAllowance
		if s.authority.isHub(p.identity) {
			allowance = couchmessage.HubAllowance
		}
		endpoint := messageEndpoint{service: s, binding: e.Binding, endpoint: p.endpoint}
		var err error
		if s.authority.isHub(p.identity) && e.Binding.Agent == "claude" {
			err = s.broker.RegisterWithBatching(e.Binding, endpoint, allowance, s.authority.batchMaxChars)
		} else {
			err = s.broker.RegisterWithAllowance(e.Binding, endpoint, allowance)
		}
		if err != nil {
			return &couchmessage.RegistryEvent{Kind: couchmessage.ConnectFailed, Token: e.Token, Pane: e.Pane, Err: err}
		}
		s.mu.Lock()
		s.workspaces[e.Binding] = p.identity
		s.mu.Unlock()
		_ = s.broker.ReconcileObservation(e.Binding, p.observation)
	case couchmessage.EffectDisconnect:
		s.broker.Disconnect(e.Binding)
		s.mu.Lock()
		delete(s.workspaces, e.Binding)
		s.mu.Unlock()
	case couchmessage.EffectScheduleRetry:
		s.after(e.Delay, func() {
			_ = s.post(couchmessage.RegistryEvent{Kind: couchmessage.RetryDue, Token: e.Token, Attempt: e.Attempt}, false)
		})
	case couchmessage.EffectObserve:
		_ = s.broker.ReconcileObservation(e.Binding, e.Observation)
	}
	return nil
}

// admit is one full authority check plus the workspace and a first
// observation. It records its evidence for the Connect effect only on success.
func (s *messageService) admit(e couchmessage.RegistryEffect) error {
	select {
	case s.admitting <- struct{}{}:
		defer func() { <-s.admitting }()
	case <-s.lifetime.Done():
		return s.lifetime.Err()
	}
	ctx, cancel := context.WithTimeout(s.lifetime, messageAdmissionTimeout)
	defer cancel()
	b := e.Binding
	root, err := s.authority.live(ctx, b)
	if err != nil {
		return err
	}
	identity, err := s.authority.workspace(ctx, root)
	if err != nil {
		return err
	}
	if identity.Address == nil || *identity.Address != b.Slot || identity.RepoIdentity != b.Repository || identity.WorktreeRoot != root || identity.RestingBranch == nil {
		return errors.New("message slot does not match verified workspace")
	}
	endpoint := s.authority.endpoint(b)
	if endpoint == nil {
		return couchmessage.ErrUnsupported
	}
	observation, err := endpoint.Observe(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.prepared[preparedKey{e.Token, e.Pane}] = preparedAdmission{identity: identity, endpoint: endpoint, observation: observation}
	s.mu.Unlock()
	return nil
}

func (s *messageService) isConnected(b couchmessage.Binding) bool {
	return (*s.connected.Load())[b]
}

func (s *messageService) connectedWorkspace(ctx context.Context, b couchmessage.Binding) (couchcore.WorkspaceIdentity, error) {
	root, err := s.authority.thread(ctx, b)
	if err != nil {
		return couchcore.WorkspaceIdentity{}, err
	}
	s.mu.Lock()
	identity, ok := s.workspaces[b]
	s.mu.Unlock()
	if !ok || identity.WorktreeRoot != root || identity.RestingBranch == nil {
		return couchcore.WorkspaceIdentity{}, errors.New("unknown message workspace")
	}
	return identity, nil
}

func (s *messageService) handle(ctx context.Context, request couchmessage.Request) (response couchmessage.Response) {
	if request.Op == "send" {
		// Sender authority and recipient admission spend one budget, leaving
		// the exchange's existing margin for the broker's verdict (#634).
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, couchmessage.AdmissionTimeout)
		defer cancel()
	}
	switch {
	case request.Op == "resources":
		if err := couchmessage.ValidateRequest(request); err != nil {
			return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
		}
		if sampler := s.resources.Load(); sampler != nil {
			view := sampler.view()
			view.SupervisionLog = s.facts.Load().health()
			if snapshot := s.hostedIngest.Load(); snapshot != nil {
				hosted := (*snapshot)()
				hosted.ServerPID = os.Getpid()
				hosted.ServerStartedAt = processStartedAt
				view.HostedIngest = &hosted
			}
			view = projectResourceView(view)
			return couchmessage.Response{Code: "ok", Resources: &view}
		}
		return couchmessage.Response{Code: "unsupported", Error: "resource sampler unavailable"}
	case request.Op == orphanReapOp:
		// The reap's plan is a classification of its own, never the report the
		// last cadence left behind: an operator confirmed a kill, and the world
		// is allowed to have moved since. ValidateRequest is the confirmation
		// gate -- refusing here is what makes a caller's --confirm presentation
		// rather than the only authority (#583).
		if err := couchmessage.ValidateRequest(request); err != nil {
			return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
		}
		sampler := s.resources.Load()
		if sampler == nil {
			return couchmessage.Response{Code: "unsupported", Error: "resource sampler unavailable"}
		}
		view, err := sampler.sweep(ctx)
		if err != nil {
			return couchmessage.Response{Code: "unavailable", Error: "resource sweep: " + err.Error()}
		}
		return couchmessage.Response{Code: "ok", Resources: &view}
	case request.Op == "refusals":
		// A memory read for `couch --list` (pair#618); no caller, like resources.
		if err := couchmessage.ValidateRequest(request); err != nil {
			return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
		}
		return couchmessage.Response{Code: "ok", Refusals: s.broker.LatestRefusals()}
	case request.Op == "resume-candidates":
		return s.handleResumeCandidates(ctx, request)
	case request.Op == "move-slot":
		return s.handleHubMove(ctx, request)
	case request.Op == "session-write":
		return s.handleSessionWrite(ctx, request)
	case request.Op == "automatic-input-stall":
		return s.handleAutomaticInputStall(ctx, request)
	case request.Op == "native-exit":
		return s.handleNativeExit(ctx, request)
	case request.Op == "notify":
		return s.handleNotify(ctx, request)
	case request.Op == "automation":
		return s.handleAutomation(ctx, request)
	case request.Op == "switch-agent":
		return s.handleAgentSwitch(ctx, request)
	case couchmessage.IsSubmissionOp(request.Op):
		return s.handleSubmission(ctx, request)
	case couchmessage.IsDriveOp(request.Op):
		return s.handleAgentDrive(ctx, request)
	case request.Op == "recovery-confirmation" || couchcore.IsSlotOperation(request.Op) || request.Op == "operation-status":
		return s.handleSlotOperation(ctx, request)
	case request.Op == "broadcast-status":
		return s.handleBroadcastStatus(request)
	case couchmessage.IsReplayOp(request.Op):
		return s.handleReplay(request)
	case request.Op == "tail":
		return s.handleTail(ctx, request)
	}
	if request.Binding == nil && couchmessage.ValidateRequest(request) == nil {
		binding, err := s.broker.Caller(request.Scope, request.Tag, request.Session, request.Nonce)
		phase := "caller lookup"
		// A send acts as this caller, so it proves the caller current; the
		// cheap check spawns nothing.
		if err == nil && request.Op == "send" {
			phase = "caller " + binding.Slot + " current"
			_, err = s.authority.current(ctx, binding)
		}
		if err != nil {
			return couchmessage.Response{Code: "unavailable", Error: phase + ": " + err.Error()}
		}
		if request.Op == "send" {
			if c := s.supervision.Load(); c != nil {
				if c.sending(binding, request) {
					defer func() { c.admitted(request.ID, response.Receipt) }()
				}
			}
		}
		if request.Op == "send" && request.Target == couchmessage.HubTarget {
			hub, refusal := s.hubRecipient(binding)
			if refusal != nil {
				return *refusal
			}
			request.Target = hub
		}
		// An exact target whose session went dormant after failed admissions
		// gets one more bounded attempt; this send does not wait for its
		// result. Posting only waits for the loop to take the event (the loop
		// never waits on a request), so nothing outlives this request. A target
		// spelled by repository alias names no binding's Slot and wakes
		// nothing; the slot's own pane, submit or reconnect still does.
		if request.Op == "send" && strings.Contains(request.Target, ":") {
			if err := s.postContext(ctx, couchmessage.RegistryEvent{Kind: couchmessage.SendTargeted, Slot: request.Target}, false); err != nil {
				return couchmessage.Response{Code: "refused", Error: "not admitted: targeted registry post: " + err.Error()}
			}
		}
	}
	return couchmessage.Handle(ctx, s.broker, request)
}

// hubRecipient resolves the reserved `hub` target (pair#463) to the hub's
// exact slot at send time, so a resumed thread never carries a stale address.
// It runs before repository-name resolution: a repository named hub stays
// reachable as hub:N.
func (s *messageService) hubRecipient(from couchmessage.Binding) (string, *couchmessage.Response) {
	if s.authority.hub == "" {
		return "", &couchmessage.Response{Code: "refused", Error: "no hub: this Couch started outside a checkout"}
	}
	hub, ok := s.hubBindingFor(from)
	if !ok {
		detail := "the hub isn't connected: no live wrapper in " + s.authority.hub
		if s.authority.hubs != nil {
			s.mu.Lock()
			identity, known := s.workspaces[from]
			s.mu.Unlock()
			if known {
				detail = "the hub isn't connected: no live wrapper in " + s.authority.owner(identity)
			}
		}
		return "", &couchmessage.Response{Code: "unavailable", Error: "recipient hub lookup: " + detail}
	}
	if hub.Slot == from.Slot {
		return "", &couchmessage.Response{Code: "refused", Error: "you are the hub (" + hub.Slot + "): a message to hub would reach your own slot"}
	}
	return hub.Slot, nil
}

// handleBroadcastStatus answers the operator's `couch --broadcast-list`
// (pair#413). It needs no slot identity: the broker socket lives in the
// operator-only store directory, and the answer is read-only memory.
func (s *messageService) handleBroadcastStatus(request couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(request); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	status := s.broadcastStatus.Load()
	if status == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this Couch reports no broadcast status"}
	}
	resp := couchmessage.Response{Code: "ok"}
	if snapshot, running := (*status)(); running {
		resp.Broadcast = &snapshot
	}
	if shares := s.shares.Load(); shares != nil {
		resp.Shares = (*shares)()
	}
	if list := s.replays.Load(); list != nil {
		resp.Replays = (*list)()
	}
	return resp
}

// handleReplay answers the operator's or an agent's replay request
// (pair#477). Like broadcast-status it needs no slot identity: the broker
// socket lives in the operator-only store, as the recordings do, and a
// request never waits on the tunnel: it starts a share or reads one.
func (s *messageService) handleReplay(request couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(request); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	replay := s.replay.Load()
	if replay == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this Couch shares no replays"}
	}
	share := (*replay)(request.Op, request.Target)
	return couchmessage.Response{Code: "ok", Replay: &share}
}

// handleTail answers a peek's tail read (pair#425) from the connected
// wrapper running the thread: named by scope + exact tag, as
// LivenessForThread matches, or by its slot as a send names one (pair#429).
// Like broadcast-status it needs no slot identity: the socket lives in the
// operator-only store, and the answer is read-only.
func (s *messageService) handleTail(ctx context.Context, request couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(request); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	var bindings []couchmessage.Binding
	if connected := s.connected.Load(); connected != nil {
		for b, live := range *connected {
			if live {
				bindings = append(bindings, b)
			}
		}
	}
	binding, code, err := s.tailBinding(ctx, request, bindings)
	if err != nil {
		return couchmessage.Response{Code: code, Error: err.Error()}
	}
	var reader couchmessage.TailReader
	if s.authority.endpoint != nil {
		reader, _ = s.authority.endpoint(binding).(couchmessage.TailReader)
	}
	if reader == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this wrapper's endpoint reads no tail"}
	}
	tail, err := reader.Tail(ctx, request.Lines)
	if err != nil {
		return couchmessage.Response{Code: "unavailable", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok", Tail: &tail, TailThread: s.tailThread(ctx, binding)}
}

// tailThread names whose tail it is with what the typed peek reads from the
// thread record -- its agent (couchcore.RecordAgent) and working path -- so a
// fast peek answers in the same shape (pair#429). Without a readable record
// it names the wrapper's own agent and no path.
func (s *messageService) tailThread(ctx context.Context, b couchmessage.Binding) *couchmessage.TailThread {
	thread := &couchmessage.TailThread{Slot: b.Slot, Tag: b.Tag, Agent: b.Agent}
	if s.authority.record == nil {
		return thread
	}
	record, err := s.authority.record(ctx, couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)})
	if err != nil {
		return thread
	}
	if agent := couchcore.RecordAgent(record); agent != "" {
		thread.Agent = agent
	}
	thread.WorkingPath = record.WorkingPath
	return thread
}

// tailBinding is the one connected wrapper a tail request names, or the
// response code and reason it names none.
func (s *messageService) tailBinding(ctx context.Context, request couchmessage.Request, bindings []couchmessage.Binding) (couchmessage.Binding, string, error) {
	if request.Target != "" {
		var families map[string]string
		if s.authority.families != nil {
			var err error
			if families, err = s.authority.families(ctx); err != nil {
				return couchmessage.Binding{}, "unavailable", fmt.Errorf("repository names: %w", err)
			}
		}
		binding, err := couchmessage.ResolveTailSlot(request.Target, bindings, families)
		switch {
		case errors.Is(err, couchmessage.ErrAmbiguous):
			return couchmessage.Binding{}, "ambiguous", err
		case err != nil:
			return couchmessage.Binding{}, "unavailable", err
		}
		return binding, "", nil
	}
	var matches []couchmessage.Binding
	for _, b := range bindings {
		if b.Scope == request.TailScope && b.Tag == request.TailTag {
			matches = append(matches, b)
		}
	}
	switch len(matches) {
	case 0:
		return couchmessage.Binding{}, "unavailable", errors.New("no wrapper is connected for this thread")
	case 1:
		return matches[0], "", nil
	}
	return couchmessage.Binding{}, "ambiguous", fmt.Errorf("%d wrappers are connected for this thread", len(matches))
}

// SetBlockedView installs the Console's picker view and hands it the current
// set at once.
func (s *messageService) SetBlockedView(view func(map[couchcore.ThreadAddress]bool)) {
	s.blocked.Lock()
	defer s.blocked.Unlock()
	s.blocked.view = view
	view(s.blocked.last)
}

// publishBlocked derives the threads whose wrapper reports a picker up and
// hands the view the set when it changed.
func (s *messageService) publishBlocked(liveness map[string]couchmessage.SlotLiveness) {
	set := map[couchcore.ThreadAddress]bool{}
	prompts := map[couchcore.ThreadAddress]couchcore.AttentionCondition{}
	for _, l := range liveness {
		if l.Picker != nil && *l.Picker {
			thread := bindingThread(l.Binding)
			set[thread] = true
			prompts[thread] = couchcore.AttentionCondition{Label: l.Binding.Slot, Title: attentionPromptTitle}
		}
	}
	s.blocked.Lock()
	defer s.blocked.Unlock()
	if maps.Equal(set, s.blocked.last) {
		return
	}
	s.blocked.last = set
	if s.blocked.view != nil {
		s.blocked.view(set)
	}
	s.observeAttention(couchcore.AttentionFact{Kind: couchcore.AttentionPrompt, Live: prompts})
}

// SetReplay installs the console's replay shares: the requests, and the
// listing broadcast-status carries.
func (s *messageService) SetReplay(replay func(op, id string) couchmessage.ReplayShare, list func() []string) {
	s.replay.Store(&replay)
	s.replays.Store(&list)
}

// SetBroadcastStatus installs the console's broadcast snapshot.
func (s *messageService) SetBroadcastStatus(status func() (couchmessage.BroadcastStatus, bool)) {
	s.broadcastStatus.Store(&status)
}

// SetShares installs the console's list of links out (pair#467).
func (s *messageService) SetShares(shares func() []couchmessage.ShareStatus) {
	s.shares.Store(&shares)
}

// handleSlotOperation authenticates a slot operation's caller by Couch's own
// terminal and launch identity, not by messaging registration (#367 smoke
// test: a live slot whose wrapper's peer setup failed must still recover
// others). The thread named by the request's scope and tag must have a live
// listed terminal (authority.thread), and its recorded launch must name this
// shell's session and launch nonce (authority.recorded, for the agent its
// record launched). Any failure refuses with nothing enqueued.
func (s *messageService) handleSlotOperation(ctx context.Context, request couchmessage.Request) couchmessage.Response {
	ctx, cancel := context.WithTimeout(ctx, slotAdmissionBudget)
	defer cancel()
	if err := couchmessage.ValidateRequest(request); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if s.slotOps == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this Couch runs no slot operations"}
	}
	caller, err := s.liveCaller(ctx, request)
	if ctx.Err() != nil {
		return admissionRefusal("caller", ctx.Err())
	}
	if err != nil {
		return couchmessage.Response{Code: "unavailable", Error: fmt.Sprintf(
			"caller is not a live Couch slot (thread %s not listed, or this shell's session/launch does not match its record): %v", request.Tag, err)}
	}
	if authenticated, authErr := s.switchCaller(ctx, request); authErr == nil {
		caller = authenticated
	} else if request.Op == "recovery-confirmation" || request.Op == couchcore.OpRecoverPark {
		if ctx.Err() != nil {
			return admissionRefusal("caller", ctx.Err())
		}
		return couchmessage.Response{Code: "refused", Error: request.Op + " requires an authenticated live hub binding"}
	}
	if ctx.Err() != nil {
		return admissionRefusal("caller", ctx.Err())
	}
	if request.Op == "recovery-confirmation" {
		identity, err := s.connectedWorkspace(ctx, caller)
		if ctx.Err() != nil {
			return admissionRefusal("caller workspace", ctx.Err())
		}
		if err != nil || !s.authority.isHub(identity) {
			return couchmessage.Response{Code: "refused", Error: "only the owning hub may accept a recovery confirmation"}
		}
	}
	if request.Op == "operation-status" {
		if authenticated, err := s.switchCaller(ctx, request); err == nil {
			if identity, err := s.connectedWorkspace(ctx, authenticated); err == nil && s.authority.isHub(identity) {
				if response, ok := s.slotOps.switchStatus(request.ID, s.hubReceiptFilter(ctx, identity)); ok {
					if ctx.Err() != nil {
						return admissionRefusal("status", ctx.Err())
					}
					return response
				}
			}
		}
		if ctx.Err() != nil {
			return admissionRefusal("status", ctx.Err())
		}
	}
	return s.handleHubSlotOperation(ctx, caller, request)
}

// liveCaller is the request's identity proved against Couch's records.
func (s *messageService) liveCaller(ctx context.Context, request couchmessage.Request) (couchmessage.Binding, error) {
	caller := couchmessage.Binding{Scope: request.Scope, Tag: request.Tag, Session: request.Session, Nonce: request.Nonce}
	agent, err := s.authority.agent(ctx, couchcore.ThreadAddress{RepoScope: request.Scope, Tag: couchcore.ThreadTag(request.Tag)})
	if err != nil {
		return couchmessage.Binding{}, err
	}
	caller.Agent = agent
	if _, err := s.authority.thread(ctx, caller); err != nil {
		return couchmessage.Binding{}, err
	}
	if err := s.authority.recorded(ctx, caller); err != nil {
		return couchmessage.Binding{}, err
	}
	return caller, nil
}

// messageEndpoint repeats the use-time check, so a wrapper the registry no
// longer connects — or whose pane, PID file or recorded launch has moved on —
// cannot receive.
type messageEndpoint struct {
	service  *messageService
	binding  couchmessage.Binding
	endpoint couchmessage.DeliveryEndpoint
}

func (e messageEndpoint) check(ctx context.Context) error {
	if !e.service.isConnected(e.binding) {
		return couchmessage.ErrUnavailable
	}
	_, err := e.service.authority.current(ctx, e.binding)
	return err
}
func (e messageEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	if err := e.check(ctx); err != nil {
		return couchmessage.Observation{}, err
	}
	return e.endpoint.Observe(ctx)
}
func (e messageEndpoint) Reserve(ctx context.Context, id string, seq uint64) error {
	if err := e.check(ctx); err != nil {
		return err
	}
	return e.endpoint.Reserve(ctx, id, seq)
}

// Retained is read-only, so it needs only the registry's connection, not
// the use-time check.
func (e messageEndpoint) Retained(ctx context.Context, id string) (couchmessage.Receipt, error) {
	if !e.service.isConnected(e.binding) {
		return couchmessage.Receipt{}, couchmessage.ErrUnavailable
	}
	h, ok := e.endpoint.(couchmessage.ReceiptHolder)
	if !ok {
		return couchmessage.Receipt{}, couchmessage.ErrUnknownDelivery
	}
	return h.Retained(ctx, id)
}
func (e messageEndpoint) Release(ctx context.Context, id string) error {
	return e.endpoint.Release(ctx, id)
}
func (e messageEndpoint) Deliver(ctx context.Context, m couchmessage.Message, progress func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	if err := e.check(ctx); err != nil {
		return couchmessage.Receipt{Status: couchmessage.Cancelled, Detail: err.Error()}, nil
	}
	log := e.service.facts.Load()
	// Classify before IO: delivery may finish after this hub disconnects.
	// Explicit cross-hub messages are still deliveries to an active hub.
	toHub := e.service.isHubBinding(m.To)
	receipt, err := e.endpoint.Deliver(ctx, m, progress)
	e.accountDelivery(m, receipt, err, log, toHub)
	return receipt, err
}

func (e messageEndpoint) DeliverBatch(ctx context.Context, head couchmessage.Message, freeze func() ([]couchmessage.Message, error), progress func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	endpoint, ok := e.endpoint.(couchmessage.BatchDeliveryEndpoint)
	if !ok {
		return e.Deliver(ctx, head, progress)
	}
	if err := e.check(ctx); err != nil {
		return couchmessage.Receipt{Status: couchmessage.Cancelled, Detail: err.Error()}, nil
	}
	log, toHub := e.service.facts.Load(), e.service.isHubBinding(head.To)
	group := []couchmessage.Message{head}
	receipt, err := endpoint.DeliverBatch(ctx, head, func() ([]couchmessage.Message, error) {
		members, err := freeze()
		if err == nil {
			group = append([]couchmessage.Message(nil), members...)
		}
		return members, err
	}, progress)
	for _, member := range group {
		projected := receipt
		projected.Message = member
		e.accountDelivery(member, projected, err, log, toHub)
	}
	return receipt, err
}

// accountDelivery preserves every original message's supervision and document
// ledger provenance, even when several messages share one physical input turn.
func (e messageEndpoint) accountDelivery(m couchmessage.Message, receipt couchmessage.Receipt, err error, log *supervisionFactLog, toHub bool) {
	system := m.Origin == couchmessage.OriginSystem
	if publisher := e.service.lifecycle.Load(); publisher != nil {
		publisher.signal()
	}
	if c := e.service.supervision.Load(); c != nil {
		c.delivered(m, receipt, err)
	}
	if log != nil && toHub {
		log.delivery(m, receipt, err, system)
	}
	if r := e.service.deliveries.Load(); r != nil && !system {
		r.record(m, receipt, err)
	}
}

func readMessageWrapperPID(path string) (int, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return 0, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > 32 {
		return 0, errors.New("invalid wrapper PID binding file")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 33))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid <= 0 || len(raw) > 32 || strconv.Itoa(pid) != string(raw) {
		return 0, errors.New("invalid wrapper PID binding")
	}
	return pid, nil
}

// The supervisor lease establishes stale-listener authority. Validate the
// private parent before removing anything beneath it, including on restart.
func prepareMessageSocket(socket string) error {
	if !filepath.IsAbs(socket) || socket != filepath.Clean(socket) {
		return errors.New("invalid message socket path")
	}
	parent := filepath.Dir(socket)
	if err := os.Mkdir(parent, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	dir, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	stat, ok := dir.Sys().(*syscall.Stat_t)
	if !dir.IsDir() || dir.Mode().Perm() != 0700 || !ok || int(stat.Uid) != os.Getuid() {
		return errors.New("message socket parent is not a private owned directory")
	}
	info, err := os.Lstat(socket)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if info.Mode()&os.ModeSocket == 0 || !ok || int(owner.Uid) != os.Getuid() {
		return errors.New("message socket path is not an owned socket")
	}
	return os.Remove(socket)
}

// messageFamilies keys every enrolled repository by message family
// (directory name). Two enrolled repositories sharing a directory name are one
// ambiguous family: it stays known, so its name never prefix-routes elsewhere,
// but neither alias is published for it.
func messageFamilies(names []couchcore.RepositoryName) map[string]string {
	families, shared := map[string]string{}, map[string]bool{}
	for _, name := range names {
		if _, seen := families[name.Dir]; seen {
			shared[name.Dir] = true
		}
		families[name.Dir] = name.Alias
	}
	for family := range shared {
		families[family] = ""
	}
	return families
}

// LivenessForThread reads the latest self-report of the admitted wrapper
// running the thread (scope + exact tag, which its Binding carries from
// COUCH_THREAD_SCOPE/TAG). Matching the thread rather than the address string
// avoids alias and prefix spellings of repo:N. False: no admitted session.
func (s *messageService) LivenessForThread(scope, tag string) (couchmessage.SlotLiveness, bool) {
	if s == nil {
		return couchmessage.SlotLiveness{}, false
	}
	m := s.liveness.Load()
	if m == nil {
		return couchmessage.SlotLiveness{}, false
	}
	for _, l := range *m {
		if l.Binding.Scope == scope && l.Binding.Tag == tag {
			return l, true
		}
	}
	return couchmessage.SlotLiveness{}, false
}

// WrapperActivity reads the existing admitted-session snapshot, never probes.
// Known-unsettled or active background work includes a silent command wait;
// otherwise sessions age from their last work/completion, not idle refreshes.
func (s *messageService) WrapperActivity(row couchcore.ActionableThreadSummary) time.Time {
	live, admitted := s.LivenessForThread(row.Address.RepoScope, string(row.Address.Tag))
	if !admitted || live.Binding.Agent != row.Agent {
		return time.Time{}
	}
	activeBackground := live.BackgroundWork != nil && *live.BackgroundWork
	if live.Settled == nil && !activeBackground {
		return time.Time{}
	}
	work := live.LastWorkAt
	if (live.Settled != nil && !*live.Settled) || activeBackground {
		now := time.Now()
		if s.authority.now != nil {
			now = s.authority.now()
		}
		if now.After(work) {
			work = now
		}
	}
	return work
}

// Liveness behind the message authority: a fresh listing at most this often,
// and none older than this counts as evidence (pair#618).
const (
	messageListingInterval = time.Second
	messageListingMaxStale = 5 * time.Second
)

type listedSessionObserver interface {
	ListedPairSessionContext(context.Context, couchcore.ThreadAddress, string) (couchcore.PairSessionBinding, error)
}

// messageListedObserver is the artifact checker reading zellij through the
// message authority's poller. Other supervisor paths keep their own reads.
func messageListedObserver(artifacts couchcore.ThreadArtifactController, listings *launcher.ListingPoller) (listedSessionObserver, bool) {
	switch checker := artifacts.(type) {
	case couchcore.ScopedThreadArtifactCollisionChecker:
		checker.Zellij.Listings = listings
		return checker, true
	case *couchcore.ScopedThreadArtifactCollisionChecker:
		copied := *checker
		copied.Zellij.Listings = listings
		return copied, true
	}
	observer, ok := artifacts.(listedSessionObserver)
	return observer, ok
}

// messageThreadResolver proves the retained thread and its positively listed
// terminal without requiring this Couch to attach a local client pane.
func messageThreadResolver(c *couchcore.Couch, listings *launcher.ListingPoller) func(context.Context, couchmessage.Binding) (string, error) {
	observer, observable := messageListedObserver(c.Artifacts, listings)
	return func(ctx context.Context, binding couchmessage.Binding) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if c.Threads == nil {
			return "", errors.New("no thread store")
		}
		address := couchcore.ThreadAddress{RepoScope: binding.Scope, Tag: couchcore.ThreadTag(binding.Tag)}
		record, err := c.Threads.PreviewThreadContext(ctx, address)
		if err != nil {
			return "", err
		}
		if record.WorkingPath == "" {
			return "", &couchmessage.RelaunchRequiredError{Cause: errors.New("message thread has no retained checkout")}
		}
		if !observable {
			return "", errors.New("listed Pair session observer unavailable")
		}
		session, err := observer.ListedPairSessionContext(ctx, address, binding.Session)
		if err != nil {
			return "", err
		}
		if !session.Present || session.Name != binding.Session {
			return "", &couchmessage.RelaunchRequiredError{Cause: errors.New("message terminal is not positively listed")}
		}
		return record.WorkingPath, ctx.Err()
	}
}
