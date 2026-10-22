package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// messageWorld is a stateful fake of everything the message authority reads,
// for any number of slots. It counts the probes that cost processes or git in
// production: launch (two ps/zellij ownership probes), process identity, and
// the resting-branch git status.
type messageWorld struct {
	mu                       sync.Mutex
	slots                    map[couchmessage.ThreadKey]*worldSlot
	launches, procs, branchs int
	recordeds                int
	// endpoints overrides a binding's delivery endpoint (default: one that
	// submits at once).
	endpoints map[couchmessage.Binding]couchmessage.DeliveryEndpoint
	hub       string
}

type worldSlot struct {
	binding   couchmessage.Binding
	root      string
	workspace couchcore.WorkspaceIdentity
	live      bool // Couch pane live and process alive
	pidFile   int
	launch    bool // launch registration + session ownership
	recorded  bool // ready file + session index
	branch    string
}

func newMessageWorld() *messageWorld {
	return &messageWorld{slots: map[couchmessage.ThreadKey]*worldSlot{}, endpoints: map[couchmessage.Binding]couchmessage.DeliveryEndpoint{}}
}

// add creates slot pair:n whose wrapper is this test process, as the session
// peer-PID check requires.
func (w *messageWorld) add(n int) couchmessage.Binding {
	slot, rest, root := fmt.Sprintf("pair:%d", n), "main-slot"+fmt.Sprint(n), fmt.Sprintf("/repo%d", n)
	b := couchmessage.Binding{Slot: slot, Repository: "/repo/.git", Scope: "scope", Tag: fmt.Sprintf("t%d", n), Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: os.Getpid(), Start: "start"}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.slots[b.Thread()] = &worldSlot{binding: b, root: root, workspace: couchcore.WorkspaceIdentity{Address: &slot, RestingBranch: &rest, RepoIdentity: b.Repository, WorktreeRoot: root}, live: true, pidFile: b.PID, launch: true, recorded: true, branch: rest}
	return b
}

func (w *messageWorld) set(b couchmessage.Binding, change func(*worldSlot)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	change(w.slots[b.Thread()])
}

func (w *messageWorld) counts() [4]int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return [4]int{w.launches, w.procs, w.branchs, w.recordeds}
}

func (w *messageWorld) authority() messageAuthority {
	slot := func(b couchmessage.Binding) *worldSlot { return w.slots[b.Thread()] }
	return messageAuthority{
		hub: w.hub,
		thread: func(ctx context.Context, b couchmessage.Binding) (string, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			if s := slot(b); s != nil && s.live {
				return s.root, ctx.Err()
			}
			return "", errors.New("no pane")
		},
		agent: func(ctx context.Context, address couchcore.ThreadAddress) (string, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			for _, s := range w.slots {
				if s.binding.Scope == address.RepoScope && s.binding.Tag == string(address.Tag) {
					return s.binding.Agent, ctx.Err()
				}
			}
			return "", errors.New("no such thread record")
		},
		workspace: func(ctx context.Context, root string) (couchcore.WorkspaceIdentity, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			for _, s := range w.slots {
				if s.root == root {
					return s.workspace, ctx.Err()
				}
			}
			return couchcore.WorkspaceIdentity{}, errors.New("unknown root")
		},
		process: func(b couchmessage.Binding) error {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.procs++
			if s := slot(b); s == nil || !s.live || s.binding.PID != b.PID || s.binding.Start != b.Start {
				return errors.New("dead process")
			}
			return nil
		},
		wrapperPID: func(b couchmessage.Binding) (int, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			if s := slot(b); s != nil {
				return s.pidFile, nil
			}
			return 0, errors.New("no PID file")
		},
		launch: func(ctx context.Context, b couchmessage.Binding) error {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.launches++
			if s := slot(b); s == nil || !s.launch || s.binding.Nonce != b.Nonce || s.binding.Session != b.Session {
				return errors.New("obsolete launch")
			}
			return ctx.Err()
		},
		recorded: func(ctx context.Context, b couchmessage.Binding) error {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.recordeds++
			// Production compares the ready file's nonce and the session
			// index's name with the binding's.
			if s := slot(b); s == nil || !s.recorded || s.binding.Nonce != b.Nonce || s.binding.Session != b.Session {
				return errors.New("launch record moved on")
			}
			return ctx.Err()
		},
		endpoint: func(b couchmessage.Binding) couchmessage.DeliveryEndpoint {
			w.mu.Lock()
			defer w.mu.Unlock()
			if e, ok := w.endpoints[b]; ok {
				return e
			}
			return serviceEndpointFake{}
		},
		branch: func(_ context.Context, root string) (couchcore.SlotGitStatus, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.branchs++
			for _, s := range w.slots {
				if s.root == root {
					return couchcore.SlotGitStatus{Branch: s.branch}, nil
				}
			}
			return couchcore.SlotGitStatus{}, errors.New("unknown root")
		},
	}
}

type serviceEndpointFake struct{}

func (serviceEndpointFake) Observe(ctx context.Context) (couchmessage.Observation, error) {
	return couchmessage.Observation{LastActivity: time.Now(), Sequence: 1}, ctx.Err()
}
func (serviceEndpointFake) Reserve(ctx context.Context, _ string, _ uint64) error { return ctx.Err() }
func (serviceEndpointFake) Release(context.Context, string) error                 { return nil }
func (serviceEndpointFake) Deliver(ctx context.Context, _ couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}

// serviceRig is a running message service over real sockets, fed Console pane
// state through its mailbox and wrapper sessions through real clients.
type serviceRig struct {
	t                   *testing.T
	s                   *messageService
	world               *messageWorld
	panes               *couchmessage.PaneMailbox
	brokerSock, regSock string
	mu                  sync.Mutex
	slotGit             map[string]couchcore.SlotGitStatus
	retries             []func()
	wrapperStops        map[couchmessage.Binding]func()
	// slotOps answers resume/reboot/operation-status; nil = unsupported.
	slotOps           *slotOperations
	decorateAuthority func(messageAuthority) messageAuthority
}

func newServiceRig(t *testing.T, decorate ...func(messageAuthority) messageAuthority) *serviceRig {
	t.Helper()
	r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
	if len(decorate) > 0 {
		r.decorateAuthority = decorate[0]
	}
	r.init()
	return r
}

// init takes fresh sockets and starts the service.
func (r *serviceRig) init() {
	r.t.Helper()
	dir, err := os.MkdirTemp("", "pcms-")
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { _ = os.RemoveAll(dir) })
	r.brokerSock, r.regSock = filepath.Join(dir, "broker.sock"), filepath.Join(dir, "registry.sock")
	r.start()
}

// restart is a Couch restart: the old service goes, a new one takes the same
// sockets, and the Console replays its panes into the new mailbox.
func (r *serviceRig) restart(panes map[couchmessage.ThreadKey]couchmessage.PaneHandle) {
	r.s.Close()
	r.start()
	for thread, pane := range panes {
		r.panes.Post(thread, pane)
	}
}

func (r *serviceRig) start() {
	t := r.t
	var err error
	r.panes = couchmessage.NewPaneMailbox()
	authority := r.world.authority()
	if r.decorateAuthority != nil {
		authority = r.decorateAuthority(authority)
	}
	r.s, err = newMessageService(context.Background(), r.brokerSock, r.regSock, filepath.Dir(r.brokerSock), authority, r.panes, func(root string) (couchcore.SlotGitStatus, bool) {
		r.mu.Lock()
		defer r.mu.Unlock()
		v, ok := r.slotGit[root]
		return v, ok
	}, r.slotOps)
	if err != nil {
		t.Fatal(err)
	}
	// Retries fire only when a test says so.
	r.s.after = func(_ time.Duration, f func()) { r.mu.Lock(); r.retries = append(r.retries, f); r.mu.Unlock() }
	t.Cleanup(r.s.Close)
}

func (r *serviceRig) attach(b couchmessage.Binding, pane couchmessage.PaneHandle) {
	r.panes.Post(b.Thread(), pane)
}

// wrapper runs b's session client until the returned stop is called.
func (r *serviceRig) wrapper(b couchmessage.Binding, initial ...couchmessage.Observation) (*couchmessage.SessionClient, func()) {
	c := couchmessage.NewSessionClient()
	observation := couchmessage.Observation{LastActivity: time.Now(), Sequence: 1}
	if len(initial) > 0 {
		observation = initial[0]
	}
	c.Update(observation)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Run(ctx, r.regSock, b, couchmessage.DefaultReconnectBackoff, 10*time.Millisecond)
	}()
	stop := func() { cancel(); <-done }
	r.mu.Lock()
	if r.wrapperStops == nil {
		r.wrapperStops = map[couchmessage.Binding]func(){}
	}
	r.wrapperStops[b] = stop
	r.mu.Unlock()
	r.t.Cleanup(stop)
	return c, stop
}

// disconnect ends the wrapper's actual Unix session, independent of its pane.
func (r *serviceRig) disconnect(b couchmessage.Binding) {
	r.t.Helper()
	r.mu.Lock()
	stop := r.wrapperStops[b]
	r.mu.Unlock()
	if stop == nil {
		r.t.Fatal("no live wrapper session to disconnect")
	}
	stop()
	r.waitConnected(b, false)
}

func (r *serviceRig) waitConnected(b couchmessage.Binding, want bool) {
	r.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for r.s.isConnected(b) != want {
		if time.Now().After(deadline) {
			r.t.Fatalf("binding %s connected=%v never reached %v", b.Slot, !want, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// waitWorkspace waits for b's verified workspace in the service cache, the
// evidence the lifecycle publisher's refresh reads. connected is published
// before the cache is written (pair#693), so it is not that evidence.
func (r *serviceRig) waitWorkspace(b couchmessage.Binding) {
	r.t.Helper()
	testwait.Until(r.t, "verified workspace of "+b.Slot, func() bool {
		r.s.mu.Lock()
		defer r.s.mu.Unlock()
		_, ok := r.s.workspaces[b]
		return ok
	})
}

func (r *serviceRig) fireRetries() int {
	r.mu.Lock()
	pending := r.retries
	r.retries = nil
	r.mu.Unlock()
	for _, f := range pending {
		f()
	}
	return len(pending)
}

func (r *serviceRig) connect(n int) couchmessage.Binding {
	b := r.world.add(n)
	r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("h%d", n)))
	r.wrapper(b)
	r.waitConnected(b, true)
	return b
}

func callerRequest(b couchmessage.Binding, op string) couchmessage.Request {
	return couchmessage.Request{Op: op, Scope: b.Scope, Tag: b.Tag, Session: b.Session, Nonce: b.Nonce}
}

// TestMessageIdleMultiSlotRunsNoProbes is #365's Done-when 1. Before #365 one
// idle wrapper cost 12 full checks (24 ownership probes), 12 process checks
// and 6 git status per minute; now an idle Couch runs none, whatever the slot
// count, because nothing in messaging runs on a timer.
func TestMessageIdleMultiSlotRunsNoProbes(t *testing.T) {
	r := newServiceRig(t)
	slots := []couchmessage.Binding{r.connect(1), r.connect(2), r.connect(3)}
	settled := r.world.counts()
	if settled[0] != len(slots) {
		t.Fatalf("admission ran %d launch checks for %d slots", settled[0], len(slots))
	}
	time.Sleep(2500 * time.Millisecond) // longer than any former 1 s heartbeat or reconcile tick
	if got := r.world.counts(); got != settled {
		t.Fatalf("idle probes: launch/process/branch/recorded %v -> %v", settled, got)
	}
	for _, b := range slots {
		if !r.s.isConnected(b) {
			t.Fatalf("%s dropped while idle", b.Slot)
		}
	}
}

// Client presentation changes do not spend admission probes or disconnect
// an authenticated wrapper whose listed terminal and process remain live.
func TestMessageDetachAndReattachKeepWrapperMembership(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	before := r.world.counts()[0]
	for _, pane := range []couchmessage.PaneHandle{"", "h1-again"} {
		r.attach(b, pane)
		if err := r.s.post(couchmessage.RegistryEvent{Kind: couchmessage.PaneChanged, Thread: b.Thread(), Pane: pane}, true); err != nil {
			t.Fatal(err)
		}
		if !r.s.isConnected(b) {
			t.Fatal("client presentation change disconnected live wrapper")
		}
		if _, err := r.s.broker.Caller(b.Scope, b.Tag, b.Session, b.Nonce); err != nil {
			t.Fatal(err)
		}
	}
	if got := r.world.counts()[0]; got != before {
		t.Fatalf("client presentation spent %d admission probes", got-before)
	}
}

func TestMessageWrapperExitDisconnectsAndForgetsWorkspace(t *testing.T) {
	r := newServiceRig(t)
	b := r.world.add(1)
	r.attach(b, "h1")
	_, stop := r.wrapper(b)
	r.waitConnected(b, true)
	stop()
	r.waitConnected(b, false)
	// The loop publishes the connected set before it runs the Disconnect effect
	// (message_service.go: "Publish first"), so the forget lands just after.
	deadline := time.Now().Add(3 * time.Second)
	for {
		r.s.mu.Lock()
		workspaces, prepared := len(r.s.workspaces), len(r.s.prepared)
		r.s.mu.Unlock()
		if workspaces == 0 && prepared == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("exited wrapper left workspaces=%d prepared=%d", workspaces, prepared)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestMessageAdmissionRejectsMismatchesThenGoesDormant(t *testing.T) {
	for _, which := range []string{"slot", "repository", "root", "pid", "launch"} {
		t.Run(which, func(t *testing.T) {
			r := newServiceRig(t)
			b := r.world.add(1)
			r.world.set(b, func(s *worldSlot) {
				switch which {
				case "slot":
					v := "pair:2"
					s.workspace.Address = &v
				case "repository":
					s.workspace.RepoIdentity = "other"
				case "root":
					s.workspace.WorktreeRoot = "/other"
				case "pid":
					s.pidFile++
				case "launch":
					s.launch = false
				}
			})
			r.attach(b, "h1")
			r.wrapper(b)
			// The bounded ladder: each failure schedules exactly one retry.
			fired := 0
			deadline := time.Now().Add(3 * time.Second)
			for fired < len(couchmessage.RetryDelays) && time.Now().Before(deadline) {
				fired += r.fireRetries()
				time.Sleep(5 * time.Millisecond)
			}
			time.Sleep(100 * time.Millisecond)
			if extra := r.fireRetries(); fired != len(couchmessage.RetryDelays) || extra != 0 {
				t.Fatalf("retries fired %d (+%d), want %d", fired, extra, len(couchmessage.RetryDelays))
			}
			if r.s.isConnected(b) {
				t.Fatal("mismatch connected")
			}
		})
	}
}

// A send to a slot whose session went dormant gets one more bounded attempt,
// so an idle healthy agent is not unreachable until someone touches its pane.
func TestMessageSendToDormantSlotReadmits(t *testing.T) {
	awakened, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	calls := 0
	r := newServiceRig(t, func(a messageAuthority) messageAuthority {
		launch := a.launch
		a.launch = func(ctx context.Context, b couchmessage.Binding) error {
			if b.Slot == "pair:2" {
				calls++
				if calls == len(couchmessage.RetryDelays)+2 {
					close(awakened)
					select {
					case <-release:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return launch(ctx, b)
		}
		return a
	})
	from := r.connect(1)
	retries := make(chan func(), 1)
	r.s.after = func(_ time.Duration, f func()) { retries <- f }
	to := r.world.add(2)
	r.world.set(to, func(s *worldSlot) { s.launch = false })
	r.attach(to, "h2")
	r.wrapper(to)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for range couchmessage.RetryDelays {
		select {
		case retry := <-retries:
			retry()
		case <-ctx.Done():
			t.Fatal("retry ladder did not finish")
		}
	}
	send := callerRequest(from, "send")
	send.ID, send.Target, send.Body = "id-1", to.Slot, "work"
	// Firing the final retry does not mean its failure reached Dormant.
	// Keep launch unhealthy until a send starts the next admission, which
	// can happen only from Dormant. Hold that admission so even a fast
	// worker cannot connect before the triggering request refuses.
	for {
		if resp := r.s.handle(ctx, send); resp.Code != "unavailable" {
			t.Fatalf("send before successful readmission %+v", resp)
		}
		select {
		case <-awakened:
			r.world.set(to, func(s *worldSlot) { s.launch = true })
			releaseOnce.Do(func() { close(release) })
			r.waitConnected(to, true)
			return
		case <-ctx.Done():
			t.Fatal("send never awakened dormant session")
		default:
		}
	}
}

func TestMessageSendingCallerMustRemainCurrent(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	r.world.set(b, func(s *worldSlot) { s.pidFile++ })
	send := callerRequest(b, "send")
	send.ID, send.Target, send.Body = "id", "pair", "work"
	if resp := r.s.handle(context.Background(), send); resp.Code != "unavailable" {
		t.Fatalf("replaced caller sent: %+v", resp)
	}
}

// The use-time check runs on every Observe/Reserve/Deliver: it must refuse a
// moved-on wrapper and spawn nothing (no launch or process probe).
func TestMessageEndpointUseTimeCheckIsCheapAndRefusesMovedOn(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	endpoint := messageEndpoint{service: r.s, binding: b, endpoint: serviceEndpointFake{}}
	before := r.world.counts()
	if _, err := endpoint.Observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := endpoint.Reserve(context.Background(), "id", 1); err != nil {
		t.Fatal(err)
	}
	after := r.world.counts()
	if after[0] != before[0] || after[1] != before[1] || after[3] != before[3]+2 {
		t.Fatalf("use-time checks: launch/process/branch/recorded %v -> %v", before, after)
	}
	for name, change := range map[string]func(*worldSlot){
		"pid":      func(s *worldSlot) { s.pidFile++ },
		"recorded": func(s *worldSlot) { s.recorded = false },
		"terminal": func(s *worldSlot) { s.live = false },
	} {
		r.world.set(b, change)
		if _, err := endpoint.Observe(context.Background()); err == nil {
			t.Fatalf("%s: observed a moved-on wrapper", name)
		}
		if err := endpoint.Reserve(context.Background(), "id", 1); err == nil {
			t.Fatalf("%s: reserved a moved-on wrapper", name)
		}
		if receipt, err := endpoint.Deliver(context.Background(), couchmessage.Message{ID: "id"}, nil); err != nil || receipt.Status != couchmessage.Cancelled {
			t.Fatalf("%s: delivery %+v %v", name, receipt, err)
		}
		r.world.set(b, func(s *worldSlot) { s.pidFile, s.recorded, s.live = b.PID, true, true })
	}
	// Not connected in the registry: refused without consulting anything.
	if err := r.s.post(couchmessage.RegistryEvent{Kind: couchmessage.SessionClosed, Token: (*r.s.liveness.Load())[b.Slot].Session}, true); err != nil {
		t.Fatal(err)
	}
	r.waitConnected(b, false)
	if _, err := endpoint.Observe(context.Background()); !errors.Is(err, couchmessage.ErrUnavailable) {
		t.Fatalf("detached observe %v", err)
	}
}

// #433: the slot Couch started from is the TL hub; it registers with the
// larger bucket, every other slot with the ordinary one.
func TestMessageHubSlotGetsTheLargerAllowance(t *testing.T) {
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub, worker := r.connect(1), r.connect(2)
	resp := r.s.handle(context.Background(), callerRequest(worker, "actors"))
	if resp.Code != "ok" || len(resp.Actors) != 2 {
		t.Fatalf("listing %+v", resp)
	}
	for _, a := range resp.Actors {
		want := couchmessage.InboundAllowance
		if a.Binding == hub {
			want = couchmessage.HubAllowance
		}
		if a.Capacity != want || a.Remaining != want {
			t.Fatalf("%s allowance %d/%d, want %d", a.Binding.Slot, a.Remaining, a.Capacity, want)
		}
	}
}

// #433 BR-2: the hub is the worktree Couch started in, found from a
// subdirectory too, and absent outside any checkout.
func TestHubWorktreeIsTheStartupCheckout(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	sub := filepath.Join(repo, "a", "b")
	outside := filepath.Join(root, "outside")
	for _, dir := range []string{sub, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	resolve := func(path string) (couchcore.Worktree, error) {
		return couchcore.Resolve(path, couchcore.ExecGit{}, couchcore.OSPathOps{})
	}
	want, err := resolve(repo)
	if err != nil {
		t.Fatal(err)
	}
	for dir, expect := range map[string]string{repo: string(want), sub: string(want), outside: ""} {
		if got := hubWorktree(func() (string, error) { return dir, nil }, resolve); got != expect {
			t.Fatalf("hub from %s = %q, want %q", dir, got, expect)
		}
	}
	if got := hubWorktree(func() (string, error) { return "", errors.New("no cwd") }, resolve); got != "" {
		t.Fatalf("hub without cwd = %q", got)
	}
}

func TestMessageActorsListingIsMemoryOnly(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	r.mu.Lock()
	r.slotGit["/repo1"] = couchcore.SlotGitStatus{Branch: "main-slot1"}
	r.mu.Unlock()
	before := r.world.counts()
	actors := callerRequest(b, "actors")
	for i := 0; i < 3; i++ {
		resp := r.s.handle(context.Background(), actors)
		if resp.Code != "ok" || len(resp.Actors) != 1 || !resp.Actors[0].Known || !resp.Actors[0].Resting {
			t.Fatalf("listing %+v", resp)
		}
	}
	if after := r.world.counts(); after != before {
		t.Fatalf("listing probed: %v -> %v", before, after)
	}
	// Without a Console slot-git observation the row is unknown, not probed.
	r.mu.Lock()
	delete(r.slotGit, "/repo1")
	r.mu.Unlock()
	if resp := r.s.handle(context.Background(), actors); resp.Code != "ok" || resp.Actors[0].Known {
		t.Fatalf("listing without slot git %+v", resp)
	}
}

func TestMessageServicePrivateSocketsLegacyRegisterAndShutdown(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	for _, socket := range []string{r.brokerSock, r.regSock} {
		info, err := os.Lstat(socket)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("%s: %v %v", socket, info, err)
		}
	}
	before := r.world.counts()
	var resp couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, couchmessage.Request{Op: "register", Binding: &b}, &resp); err != nil || resp.Code != "unsupported" {
		t.Fatalf("legacy register %+v %v", resp, err)
	}
	if after := r.world.counts(); after != before {
		t.Fatal("legacy register ran checks")
	}
	if err := couchmessage.Call(context.Background(), r.brokerSock, callerRequest(b, "actors"), &resp); err != nil || resp.Code != "ok" || len(resp.Actors) != 1 {
		t.Fatalf("actors %+v %v", resp, err)
	}
	r.s.Close()
	for _, socket := range []string{r.brokerSock, r.regSock} {
		if _, err := os.Lstat(socket); !os.IsNotExist(err) {
			t.Fatalf("owned socket %s not removed: %v", socket, err)
		}
	}
}

func TestMessageWrapperPIDBoundedRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pid")
	for _, raw := range []string{"42", "42\n", "0", string(make([]byte, 128))} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		pid, err := readMessageWrapperPID(path)
		if raw == "42" {
			if err != nil || pid != 42 {
				t.Fatalf("valid %d %v", pid, err)
			}
		} else if err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readMessageWrapperPID(link); err == nil {
		t.Fatal("followed PID symlink")
	}
}

func TestMessageSocketCleanupRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := prepareMessageSocket(filepath.Join(link, "broker.sock")); err == nil {
		t.Fatal("accepted symlink socket parent")
	}
}

func TestMessageSocketCleanupPreservesRegularFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "pcmf-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "broker.sock")
	if err := os.WriteFile(socket, []byte("unrelated"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareMessageSocket(socket); err == nil {
		t.Fatal("accepted regular socket path")
	}
	raw, err := os.ReadFile(socket)
	if err != nil || string(raw) != "unrelated" {
		t.Fatalf("destroyed unrelated file %q %v", raw, err)
	}
}

// BR-7: a broker that refuses the actor must not leave the registry believing
// the binding is connected; the refusal retries on the bounded ladder.
func TestMessageBrokerRefusalIsNotConnectedAndRetries(t *testing.T) {
	r := newServiceRig(t)
	// Fill the broker's actor table with connected actors of other slots, so
	// Register refuses for capacity (tombstones alone would be evicted).
	for i := 0; i < couchmessage.MaxActors; i++ {
		b := couchmessage.Binding{Slot: fmt.Sprintf("other:%d", i), Repository: "/other/.git", Scope: "s", Tag: fmt.Sprintf("x%d", i), Session: "s", Nonce: "n", Agent: "codex", Version: "1", PID: 1, Start: "s"}
		if err := r.s.broker.Register(b, serviceEndpointFake{}); err != nil {
			t.Fatal(err)
		}
	}
	b := r.world.add(1)
	r.attach(b, "h1")
	r.wrapper(b)
	deadline := time.Now().Add(3 * time.Second)
	for r.fireRetries() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("broker refusal scheduled no retry")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if r.s.isConnected(b) {
		t.Fatal("registry reports a binding the broker refused")
	}
	if _, err := r.s.broker.Caller(b.Scope, b.Tag, b.Session, b.Nonce); err == nil {
		t.Fatal("broker holds the refused binding")
	}
}

// pair#463: `hub` resolves at send time to the slot Couch started from and
// delivers as an ordinary exact-slot message.
func TestMessageSendToHubReachesTheHubSlot(t *testing.T) {
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub, worker := r.connect(1), r.connect(2)
	send := callerRequest(worker, "send")
	send.ID, send.Target, send.Body = "id-hub", couchmessage.HubTarget, "done: pair#463"
	resp := r.s.handle(context.Background(), send)
	if resp.Code != "accepted" || resp.Receipt == nil || resp.Receipt.Message.To != hub || resp.Receipt.Message.From != worker {
		t.Fatalf("send to hub %+v", resp)
	}
}

// pair#463: each refusal names its reason.
func TestMessageSendToHubRefusalsNameTheReason(t *testing.T) {
	send := func(r *serviceRig, from couchmessage.Binding) couchmessage.Response {
		req := callerRequest(from, "send")
		req.ID, req.Target, req.Body = "id-hub", couchmessage.HubTarget, "report"
		return r.s.handle(context.Background(), req)
	}
	t.Run("no hub", func(t *testing.T) {
		r := newServiceRig(t)
		r.world.hub = ""
		r.restart(nil)
		r.connect(1)
		if resp := send(r, r.connect(2)); resp.Code != "refused" || !strings.Contains(resp.Error, "no hub") {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("hub not connected", func(t *testing.T) {
		r := newServiceRig(t)
		r.world.hub = "/repo1"
		r.restart(nil)
		if resp := send(r, r.connect(2)); resp.Code != "unavailable" || !strings.Contains(resp.Error, "hub isn't connected") {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("sender is the hub", func(t *testing.T) {
		r := newServiceRig(t)
		r.world.hub = "/repo1"
		r.restart(nil)
		hub := r.connect(1)
		r.connect(2)
		if resp := send(r, hub); resp.Code != "refused" || !strings.Contains(resp.Error, "you are the hub") {
			t.Fatalf("%+v", resp)
		}
	})
}

// pair#463: `hub` is reserved ahead of repository names; a repository named
// hub keeps its slots reachable as hub:N.
func TestMessageHubTargetIsReservedAheadOfRepositoryNames(t *testing.T) {
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub, worker := r.connect(1), r.connect(2)
	other := r.world.add(3)
	r.world.set(other, func(s *worldSlot) {
		slot := "hub:3"
		s.binding.Slot, s.workspace.Address = slot, &slot
	})
	other.Slot = "hub:3"
	r.attach(other, "h3")
	r.wrapper(other)
	r.waitConnected(other, true)
	for target, want := range map[string]couchmessage.Binding{couchmessage.HubTarget: hub, "hub:3": other} {
		send := callerRequest(worker, "send")
		send.ID, send.Target, send.Body = "id-"+target, target, "report"
		if resp := r.s.handle(context.Background(), send); resp.Code != "accepted" || resp.Receipt == nil || resp.Receipt.Message.To != want {
			t.Fatalf("send to %s: %+v", target, resp)
		}
	}
}

type hubRecorder []string

func (h *hubRecorder) SetHub(root string) { *h = append(*h, root) }

// pair#491: the Console's hub is the authority's, the slot "hub" reaches.
func TestShowHubHandsTheConsoleTheAuthorityHub(t *testing.T) {
	var got hubRecorder
	(&messageService{authority: messageAuthority{hub: "/src/ops"}}).showHub(&got)
	if len(got) != 1 || got[0] != "/src/ops" {
		t.Fatalf("SetHub calls %q, want [/src/ops]", got)
	}
}
