package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

// messageAuthority separates live evidence from registry mutation. All probes
// are read-only; none call mutable Couch domain operations from request workers.
type messageAuthority struct {
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
	endpoint func(couchmessage.Binding) couchmessage.DeliveryEndpoint
	branch   func(context.Context, string) (couchcore.SlotGitStatus, error)
	// families lists every enrolled message family with its alias ("" when
	// none); nil means routing sees live bindings only.
	families func(context.Context) (map[string]string, error)
	// agent names the agent a thread's record says it launched, so a slot
	// operation's caller can be checked against that launch's recorded files
	// without a messaging binding (pair#367).
	agent func(context.Context, couchcore.ThreadAddress) (string, error)
}

// live is the full check, run once per admission: a lifecycle event, never a
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
	// External probes may yield while a pane is replaced or detached.
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
// caller: the Couch pane is still live (in memory), the PID file still names
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
		return "", errors.New("wrapper does not own the current binding")
	}
	if err = a.recorded(ctx, b); err != nil {
		return "", err
	}
	return root, ctx.Err()
}

// Admissions run concurrently up to this bound, so a broker restart with N
// surviving wrappers costs N full checks, a few at a time.
const (
	messageAdmissionWorkers = 4
	messageAdmissionTimeout = 10 * time.Second
)

// messageService turns registry effects into IO (#365). One goroutine owns
// the Registry and applies events in arrival order: wrapper sessions, Console
// pane state, admission results and retry timers. Request goroutines read
// only the published snapshot of connected bindings.
type messageService struct {
	server    *couchmessage.Server
	sessions  *couchmessage.SessionServer
	broker    *couchmessage.Broker
	cancel    context.CancelFunc
	lifetime  context.Context
	workers   sync.WaitGroup
	authority messageAuthority
	panes     *couchmessage.PaneMailbox
	inbox     chan messageInput
	admitting chan struct{}
	connected atomic.Pointer[map[couchmessage.Binding]bool]
	// liveness is the registry's per-slot self-report snapshot (#421),
	// republished after every step like connected.
	liveness atomic.Pointer[map[string]couchmessage.SlotLiveness]
	// after schedules a retry; tests replace it to fire retries on demand.
	after func(time.Duration, func())
	// slotOps answers resume, reboot and operation-status (pair#367 M2);
	// nil answers them unsupported.
	slotOps *slotOperations
	// broadcastStatus answers broadcast-status (pair#413). Installed after the
	// service is serving, so it is an atomic pointer; nil answers unsupported.
	broadcastStatus atomic.Pointer[func() (couchmessage.BroadcastStatus, bool)]

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
}

// startMessageService is called only while the existing supervisor lease is
// held. Socket cleanup and all child registration live inside that lifetime.
func startMessageService(console *couchtty.Console, c *couchcore.Couch) (*messageService, error) {
	resolver, ok := c.Slots.(couchcore.SlotWorkspaceResolver)
	if !ok {
		return nil, nil
	}
	artifacts, ok := c.Artifacts.(couchcore.PairLifecycleEnvironment)
	if !ok || c.Proc == nil || c.Git == nil {
		return nil, nil
	}
	namespace := c.Namespace.Dir()
	brokerSocket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return nil, err
	}
	registrySocket, err := couchmessage.SocketPath(namespace, "registry")
	if err != nil {
		return nil, err
	}
	reader := couchcore.OSOrientationStatusReader{DataDir: artifacts.PairLifecycleDataDir(), Session: artifacts.PairSession, Proc: c.Proc}
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
		thread: console.MessageBinding, workspace: resolver.ResolveWorkspace,
		agent: func(ctx context.Context, address couchcore.ThreadAddress) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if c.Threads == nil {
				return "", errors.New("no thread store")
			}
			record, err := c.Threads.GetThread(address)
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
				return fmt.Errorf("wrapper launch registration: %w", e)
			}
			if !registered {
				return errors.New("wrapper launch registration unavailable")
			}
			current, e := session(ctx, address)
			if e != nil {
				return e
			}
			if !current.Present || current.Name != b.Session {
				return errors.New("wrapper conversation changed")
			}
			return nil
		},
		recorded: func(ctx context.Context, b couchmessage.Binding) error {
			address := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
			ready, ok, e := reader.RecordedSession(ctx, address, b.Agent, b.Nonce)
			if e != nil {
				return fmt.Errorf("wrapper launch registration: %w", e)
			}
			if !ok || ready != b.Session {
				return errors.New("wrapper launch registration unavailable")
			}
			name, e := sessionName(ctx, address)
			if e != nil {
				return e
			}
			if name != b.Session {
				return errors.New("wrapper conversation changed")
			}
			return nil
		},
		endpoint: func(b couchmessage.Binding) couchmessage.DeliveryEndpoint {
			return couchmessage.RemoteEndpoint{Namespace: namespace, Binding: b}
		},
		branch: func(ctx context.Context, root string) (couchcore.SlotGitStatus, error) {
			return couchcore.ProbeSlotGit(ctx, c.Git, root)
		},
	}
	panes := couchmessage.NewPaneMailbox()
	probe := newLiveRestartProbe(c.Git, c.Proc)
	c.LiveRestart = probe // before the socket opens: no request can race this write
	service, err := newMessageService(context.Background(), brokerSocket, registrySocket, authority, panes, console.MessageSlotGit, consoleSlotOperations(console, c, probe))
	if err != nil {
		return nil, err
	}
	probe.service.Store(service)
	console.SetMessageBroker(service.broker)
	service.SetBroadcastStatus(console.BroadcastStatus)
	// After the loop runs: the replay of already-attached panes lands in the
	// mailbox the loop drains.
	console.SubscribeMessageLifecycle(panes)
	return service, nil
}

// consoleSlotOperations runs slot operations on the console's queue;
// PrepareSlotOperation runs on it, so admission is judged against the
// inventory at execution time, and the queue key resolves repository names
// from the thread store.
func consoleSlotOperations(console *couchtty.Console, c *couchcore.Couch, probe *liveRestartProbe) *slotOperations {
	return newSlotOperations(func(key, op, target string, opts couchcore.LiveRestartOptions, started func(), finished func(any, error)) error {
		// The admission note and readiness wait are written on the queue
		// goroutine by prepare and read by finished after the job; the queue
		// orders the two.
		var note string
		var await func(context.Context) error
		return console.EnqueueRemoteOperation(key, op, func(ctx context.Context) (couchcore.OperationCall, error) {
			call, n, err := c.PrepareSlotOperation(ctx, op, target, opts)
			note = n
			if err == nil {
				await = probe.readinessAfter(op, call.Args)
			}
			return call, err
		}, started, awaitReadiness(&await, withAdmissionNote(&note, finished)))
	}, func(ctx context.Context) ([]couchcore.RepositoryName, error) {
		if c.Threads == nil {
			return nil, errors.New("no thread store")
		}
		return c.Threads.RepositoryNamesContext(ctx)
	}, time.Now)
}

func newMessageService(parent context.Context, brokerSocket, registrySocket string, authority messageAuthority, panes *couchmessage.PaneMailbox, slotGit func(string) (couchcore.SlotGitStatus, bool), slotOps *slotOperations) (*messageService, error) {
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
	s.connected.Store(&map[couchmessage.Binding]bool{})
	// Family admission's resting check is on the request path (one git status
	// per candidate per send); listings read the Console's slot-git cache.
	s.broker = couchmessage.NewBroker(lifetime, time.Now, func(ctx context.Context, b couchmessage.Binding) (bool, error) {
		identity, err := s.connectedWorkspace(ctx, b)
		if err != nil {
			return false, err
		}
		status, err := authority.branch(ctx, identity.WorktreeRoot)
		return err == nil && !status.Detached && status.Branch == *identity.RestingBranch, err
	})
	s.broker.SetFamilies(authority.families)
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
			return s.post(couchmessage.RegistryEvent{Kind: couchmessage.SessionOpened, Token: t, Binding: h.Binding, Build: h.Build}, true)
		},
		Frame: func(t couchmessage.SessionToken, f couchmessage.SessionFrame) {
			kind := couchmessage.SessionActivity
			if f.Op == couchmessage.FrameSubmit {
				kind = couchmessage.SessionSubmit
			}
			_ = s.post(couchmessage.RegistryEvent{Kind: kind, Token: t, Observation: *f.Observation, Settled: f.Settled}, false)
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
	return s, nil
}

// post hands an event to the loop; wait returns the registry's verdict. After
// shutdown it drops the event: nothing is left to act on it.
func (s *messageService) post(e couchmessage.RegistryEvent, wait bool) error {
	in := messageInput{event: e}
	if wait {
		in.reply = make(chan error, 1)
	}
	select {
	case s.inbox <- in:
	case <-s.lifetime.Done():
		return s.lifetime.Err()
	}
	if !wait {
		return nil
	}
	select {
	case err := <-in.reply:
		return err
	case <-s.lifetime.Done():
		return s.lifetime.Err()
	}
}

func (s *messageService) loop() {
	defer s.workers.Done()
	registry := couchmessage.NewRegistry()
	var step func(e couchmessage.RegistryEvent) error
	step = func(e couchmessage.RegistryEvent) error {
		fx, err := registry.Advance(e)
		// Publish first: endpoint checks then never admit a binding the broker
		// is about to drop, nor refuse one it is about to register.
		connected := registry.Connected()
		s.connected.Store(&connected)
		liveness := registry.Liveness()
		s.liveness.Store(&liveness)
		var followUps []couchmessage.RegistryEvent
		for _, effect := range fx {
			if next := s.execute(effect); next != nil {
				followUps = append(followUps, *next)
			}
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
		if err := s.broker.Register(e.Binding, messageEndpoint{service: s, binding: e.Binding, endpoint: p.endpoint}); err != nil {
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

func (s *messageService) handle(ctx context.Context, request couchmessage.Request) couchmessage.Response {
	switch {
	case couchcore.IsSlotOperation(request.Op) || request.Op == "operation-status":
		return s.handleSlotOperation(ctx, request)
	case request.Op == "broadcast-status":
		return s.handleBroadcastStatus(request)
	case request.Op == "tail":
		return s.handleTail(ctx, request)
	}
	if request.Binding == nil && couchmessage.ValidateRequest(request) == nil {
		binding, err := s.broker.Caller(request.Scope, request.Tag, request.Session, request.Nonce)
		// A send acts as this caller, so it proves the caller current; the
		// cheap check spawns nothing.
		if err == nil && request.Op == "send" {
			_, err = s.authority.current(ctx, binding)
		}
		if err != nil {
			return couchmessage.Response{Code: "unavailable", Error: err.Error()}
		}
		// An exact target whose session went dormant after failed admissions
		// gets one more bounded attempt; this send does not wait for its
		// result. Posting only waits for the loop to take the event (the loop
		// never waits on a request), so nothing outlives this request. A target
		// spelled by repository alias names no binding's Slot and wakes
		// nothing; the slot's own pane, submit or reconnect still does.
		if request.Op == "send" && strings.Contains(request.Target, ":") {
			_ = s.post(couchmessage.RegistryEvent{Kind: couchmessage.SendTargeted, Slot: request.Target}, false)
		}
	}
	return couchmessage.Handle(ctx, s.broker, request)
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
	snapshot, running := (*status)()
	if !running {
		return couchmessage.Response{Code: "ok"}
	}
	return couchmessage.Response{Code: "ok", Broadcast: &snapshot}
}

// handleTail answers a peek's tail read (pair#425) from the connected
// wrapper running the thread (scope + exact tag, as LivenessForThread
// matches). Like broadcast-status it needs no slot identity: the socket lives
// in the operator-only store, and the answer is read-only.
func (s *messageService) handleTail(ctx context.Context, request couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(request); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	var matches []couchmessage.Binding
	if connected := s.connected.Load(); connected != nil {
		for b, live := range *connected {
			if live && b.Scope == request.TailScope && b.Tag == request.TailTag {
				matches = append(matches, b)
			}
		}
	}
	switch {
	case len(matches) == 0:
		return couchmessage.Response{Code: "unavailable", Error: "no wrapper is connected for this thread"}
	case len(matches) > 1:
		return couchmessage.Response{Code: "ambiguous", Error: fmt.Sprintf("%d wrappers are connected for this thread", len(matches))}
	}
	var reader couchmessage.TailReader
	if s.authority.endpoint != nil {
		reader, _ = s.authority.endpoint(matches[0]).(couchmessage.TailReader)
	}
	if reader == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this wrapper's endpoint reads no tail"}
	}
	tail, err := reader.Tail(ctx, request.Lines)
	if err != nil {
		return couchmessage.Response{Code: "unavailable", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok", Tail: &tail}
}

// SetBroadcastStatus installs the console's broadcast snapshot.
func (s *messageService) SetBroadcastStatus(status func() (couchmessage.BroadcastStatus, bool)) {
	s.broadcastStatus.Store(&status)
}

// handleSlotOperation authenticates a slot operation's caller by Couch's own
// liveness, not by messaging registration (operator decision, #367 smoke
// test: a live slot whose wrapper's peer setup failed must still recover
// others). The thread named by the request's scope and tag must have a live
// Couch pane (authority.thread), and its recorded launch must name this
// shell's session and launch nonce (authority.recorded, for the agent its
// record launched). Any failure refuses with nothing enqueued.
func (s *messageService) handleSlotOperation(ctx context.Context, request couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(request); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if s.slotOps == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this Couch runs no slot operations"}
	}
	caller, err := s.liveCaller(ctx, request)
	if err != nil {
		return couchmessage.Response{Code: "unavailable", Error: fmt.Sprintf(
			"caller is not a live Couch slot (thread %s not live, or this shell's session/launch does not match its record): %v", request.Tag, err)}
	}
	return s.slotOps.handle(ctx, caller, request)
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
func (e messageEndpoint) Deliver(ctx context.Context, m couchmessage.Message) (couchmessage.Receipt, error) {
	if err := e.check(ctx); err != nil {
		return couchmessage.Receipt{Status: couchmessage.Cancelled, Detail: err.Error()}, nil
	}
	return e.endpoint.Deliver(ctx, m)
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
