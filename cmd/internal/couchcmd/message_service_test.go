package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
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
	return &messageWorld{slots: map[couchmessage.ThreadKey]*worldSlot{}}
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
		thread: func(ctx context.Context, b couchmessage.Binding) (string, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			if s := slot(b); s != nil && s.live {
				return s.root, ctx.Err()
			}
			return "", errors.New("no pane")
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
			if s := slot(b); s == nil || !s.recorded {
				return errors.New("launch record moved on")
			}
			return ctx.Err()
		},
		endpoint: func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return serviceEndpointFake{} },
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
func (serviceEndpointFake) Deliver(ctx context.Context, _ couchmessage.Message) (couchmessage.Receipt, error) {
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
}

func newServiceRig(t *testing.T) *serviceRig {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pair-message-service-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	r := &serviceRig{t: t, world: newMessageWorld(), panes: couchmessage.NewPaneMailbox(), brokerSock: filepath.Join(dir, "broker.sock"), regSock: filepath.Join(dir, "registry.sock"), slotGit: map[string]couchcore.SlotGitStatus{}}
	r.s, err = newMessageService(context.Background(), r.brokerSock, r.regSock, r.world.authority(), r.panes, func(root string) (couchcore.SlotGitStatus, bool) {
		r.mu.Lock()
		defer r.mu.Unlock()
		v, ok := r.slotGit[root]
		return v, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	// Retries fire only when a test says so.
	r.s.after = func(_ time.Duration, f func()) { r.mu.Lock(); r.retries = append(r.retries, f); r.mu.Unlock() }
	t.Cleanup(r.s.Close)
	return r
}

func (r *serviceRig) attach(b couchmessage.Binding, pane couchmessage.PaneHandle) {
	r.panes.Post(b.Thread(), pane)
}

// wrapper runs b's session client until the returned stop is called.
func (r *serviceRig) wrapper(b couchmessage.Binding) (*couchmessage.SessionClient, func()) {
	c := couchmessage.NewSessionClient()
	c.Update(couchmessage.Observation{LastActivity: time.Now(), Sequence: 1})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Run(ctx, r.regSock, b, couchmessage.DefaultReconnectBackoff, 10*time.Millisecond)
	}()
	stop := func() { cancel(); <-done }
	r.t.Cleanup(stop)
	return c, stop
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

// Detach and reattach are lifecycle events: detach costs nothing, each
// reattach costs exactly one full check.
func TestMessageDetachAndReattachCheckOncePerEvent(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	before := r.world.counts()[0]
	r.attach(b, "")
	r.waitConnected(b, false)
	if _, err := r.s.broker.Caller(b.Scope, b.Tag, b.Session, b.Nonce); err == nil {
		t.Fatal("detached slot still a broker caller")
	}
	r.attach(b, "h1-again")
	r.waitConnected(b, true)
	if got := r.world.counts()[0]; got != before+1 {
		t.Fatalf("detach+reattach ran %d launch checks, want 1", got-before)
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
	r.s.mu.Lock()
	workspaces, prepared := len(r.s.workspaces), len(r.s.prepared)
	r.s.mu.Unlock()
	if workspaces != 0 || prepared != 0 {
		t.Fatalf("exited wrapper left workspaces=%d prepared=%d", workspaces, prepared)
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
	r := newServiceRig(t)
	from := r.connect(1)
	to := r.world.add(2)
	r.world.set(to, func(s *worldSlot) { s.launch = false })
	r.attach(to, "h2")
	r.wrapper(to)
	for fired := 0; fired < len(couchmessage.RetryDelays); {
		fired += r.fireRetries()
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	r.world.set(to, func(s *worldSlot) { s.launch = true })
	send := callerRequest(from, "send")
	send.ID, send.Target, send.Body = "id-1", to.Slot, "work"
	if resp := r.s.handle(context.Background(), send); resp.Code != "unavailable" {
		t.Fatalf("send to dormant slot %+v", resp)
	}
	r.waitConnected(to, true)
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
		"pane":     func(s *worldSlot) { s.live = false },
	} {
		r.world.set(b, change)
		if _, err := endpoint.Observe(context.Background()); err == nil {
			t.Fatalf("%s: observed a moved-on wrapper", name)
		}
		if err := endpoint.Reserve(context.Background(), "id", 1); err == nil {
			t.Fatalf("%s: reserved a moved-on wrapper", name)
		}
		if receipt, err := endpoint.Deliver(context.Background(), couchmessage.Message{ID: "id"}); err != nil || receipt.Status != couchmessage.Cancelled {
			t.Fatalf("%s: delivery %+v %v", name, receipt, err)
		}
		r.world.set(b, func(s *worldSlot) { s.pidFile, s.recorded, s.live = b.PID, true, true })
	}
	// Not connected in the registry: refused without consulting anything.
	r.attach(b, "")
	r.waitConnected(b, false)
	if _, err := endpoint.Observe(context.Background()); !errors.Is(err, couchmessage.ErrUnavailable) {
		t.Fatalf("detached observe %v", err)
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
	dir, err := os.MkdirTemp("/tmp", "pair-message-service-")
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
	// Fill the broker's actor table with disconnected tombstones of other
	// slots, so Register refuses for capacity.
	for i := 0; i < couchmessage.MaxActors; i++ {
		b := couchmessage.Binding{Slot: fmt.Sprintf("other:%d", i), Repository: "/other/.git", Scope: "s", Tag: fmt.Sprintf("x%d", i), Session: "s", Nonce: "n", Agent: "codex", Version: "1", PID: 1, Start: "s"}
		if err := r.s.broker.Register(b, serviceEndpointFake{}); err != nil {
			t.Fatal(err)
		}
		r.s.broker.Disconnect(b)
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
