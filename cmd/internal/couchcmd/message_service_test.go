package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type messageAuthorityFake struct {
	mu        sync.Mutex
	binding   couchmessage.Binding
	root      string
	workspace couchcore.WorkspaceIdentity
	live      bool
	pid       int
	launch    bool
	branch    string
}

func newMessageAuthorityFake() *messageAuthorityFake {
	slot, rest := "pair:1", "main-slot1"
	b := couchmessage.Binding{Slot: slot, Repository: "/repo/.git", Scope: "scope", Tag: "thread", Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 42, Start: "start"}
	return &messageAuthorityFake{binding: b, root: "/repo", workspace: couchcore.WorkspaceIdentity{Address: &slot, RestingBranch: &rest, RepoIdentity: b.Repository, WorktreeRoot: "/repo"}, live: true, pid: b.PID, launch: true, branch: rest}
}
func (f *messageAuthorityFake) authority() messageAuthority {
	return messageAuthority{
		thread: func(ctx context.Context, b couchmessage.Binding) (string, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if !f.live || b.Scope != f.binding.Scope || b.Tag != f.binding.Tag {
				return "", errors.New("no pane")
			}
			return f.root, nil
		},
		workspace: func(ctx context.Context, root string) (couchcore.WorkspaceIdentity, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.workspace, ctx.Err()
		},
		process: func(b couchmessage.Binding) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if !f.live || b.PID != f.binding.PID || b.Start != f.binding.Start {
				return errors.New("dead process")
			}
			return nil
		},
		wrapperPID: func(couchmessage.Binding) (int, error) { f.mu.Lock(); defer f.mu.Unlock(); return f.pid, nil },
		launch: func(ctx context.Context, b couchmessage.Binding) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if !f.launch || b.Session != f.binding.Session || b.Nonce != f.binding.Nonce {
				return errors.New("obsolete launch")
			}
			return ctx.Err()
		},
		endpoint: func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return serviceEndpointFake{} },
		branch: func(context.Context, string) (couchcore.SlotGitStatus, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			return couchcore.SlotGitStatus{Branch: f.branch}, nil
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
func serviceFixture(t *testing.T) (*messageService, *messageAuthorityFake) {
	t.Helper()
	f := newMessageAuthorityFake()
	dir, err := os.MkdirTemp("/tmp", "pair-message-service-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	s, err := newMessageService(context.Background(), filepath.Join(dir, "broker.sock"), f.authority())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, f
}
func TestMessageServiceKnownBindingRechecksLaunch(t *testing.T) {
	s, f := serviceFixture(t)
	if err := s.register(context.Background(), f.binding); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.launch = false
	f.mu.Unlock()
	if err := s.register(context.Background(), f.binding); err == nil {
		t.Fatal("cached workspace bypassed changed launch")
	}
	// Within the verification window delivery still re-checks at use.
	guarded := s.guard(f.binding, serviceEndpointFake{})
	if err := guarded.Reserve(context.Background(), "id", 1); err == nil {
		t.Fatal("stale launch reserved within the verification window")
	}
	advanceMessageClock(s, messageVerificationWindow)
	s.reconcile(context.Background())
	if _, err := s.broker.Caller(f.binding.Scope, f.binding.Tag, f.binding.Session, f.binding.Nonce); err == nil {
		t.Fatal("stale registered launch survived reconciliation")
	}
}
func TestMessageServiceRejectsWrongWorkspaceAndPID(t *testing.T) {
	for _, which := range []string{"slot", "repository", "root", "pid"} {
		t.Run(which, func(t *testing.T) {
			s, f := serviceFixture(t)
			f.mu.Lock()
			switch which {
			case "slot":
				v := "pair:2"
				f.workspace.Address = &v
			case "repository":
				f.workspace.RepoIdentity = "other"
			case "root":
				f.workspace.WorktreeRoot = "/other"
			case "pid":
				f.pid++
			}
			f.mu.Unlock()
			if err := s.register(context.Background(), f.binding); err == nil {
				t.Fatal("mismatch accepted")
			}
		})
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
func TestMessageServiceCallerMustRemainCurrent(t *testing.T) {
	s, f := serviceFixture(t)
	if err := s.register(context.Background(), f.binding); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.pid = f.binding.PID + 1
	f.mu.Unlock()
	caller := couchmessage.Request{Scope: f.binding.Scope, Tag: f.binding.Tag, Session: f.binding.Session, Nonce: f.binding.Nonce}
	send := caller
	send.Op, send.ID, send.Target, send.Body = "send", "id", "pair", "work"
	if response := s.handle(context.Background(), send); response.Code != "unavailable" {
		t.Fatalf("stale caller sent within the verification window: %+v", response)
	}
	advanceMessageClock(s, messageVerificationWindow)
	actors := caller
	actors.Op = "actors"
	if response := s.handle(context.Background(), actors); response.Code == "ok" {
		t.Fatal("stale caller queried broker")
	}
}

func TestMessageServicePrivateSocketRegistrationAndShutdown(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "pair-message-service-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "broker.sock")
	f := newMessageAuthorityFake()
	s, err := newMessageService(context.Background(), socket, f.authority())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), socket, couchmessage.Request{Op: "register", Binding: &f.binding}, &response); err != nil || response.Code != "ok" {
		t.Fatalf("register %+v %v", response, err)
	}
	request := couchmessage.Request{Op: "actors", Scope: f.binding.Scope, Tag: f.binding.Tag, Session: f.binding.Session, Nonce: f.binding.Nonce}
	if err := couchmessage.Call(context.Background(), socket, request, &response); err != nil || response.Code != "ok" || len(response.Actors) != 1 {
		t.Fatalf("actors %+v %v", response, err)
	}
	s.Close()
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("owned socket not removed: %v", err)
	}
}
func TestMessageEndpointRevalidatesBeforeDelivery(t *testing.T) {
	f := newMessageAuthorityFake()
	endpoint := messageEndpoint{authority: f.authority(), binding: f.binding, endpoint: serviceEndpointFake{}}
	f.mu.Lock()
	f.pid++
	f.mu.Unlock()
	if _, err := endpoint.Observe(context.Background()); err == nil {
		t.Fatal("observed replaced wrapper")
	}
	if err := endpoint.Reserve(context.Background(), "id", 1); err == nil {
		t.Fatal("reserved replaced wrapper")
	}
	r, err := endpoint.Deliver(context.Background(), couchmessage.Message{ID: "id"})
	if err != nil || r.Status != couchmessage.Cancelled {
		t.Fatalf("replacement delivery %+v %v", r, err)
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

// advanceMessageClock moves the service's clock forward from real time.
func advanceMessageClock(s *messageService, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	base := s.now()
	s.now = func() time.Time { return base.Add(d) }
}

func (f *messageAuthorityFake) countingLaunch(calls *int) messageAuthority {
	a := f.authority()
	launch := a.launch
	a.launch = func(ctx context.Context, b couchmessage.Binding) error {
		f.mu.Lock()
		*calls++
		f.mu.Unlock()
		return launch(ctx, b)
	}
	return a
}

func TestMessageHeartbeatSkipsFullCheckWithinWindow(t *testing.T) {
	f := newMessageAuthorityFake()
	calls := 0
	dir, err := os.MkdirTemp("/tmp", "pair-message-service-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	s, err := newMessageService(context.Background(), filepath.Join(dir, "broker.sock"), f.countingLaunch(&calls))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	heartbeat := couchmessage.Request{Op: "register", Binding: &f.binding}
	count := func() int { f.mu.Lock(); defer f.mu.Unlock(); return calls }
	if r := s.handle(context.Background(), heartbeat); r.Code != "ok" {
		t.Fatalf("first registration %+v", r)
	}
	first := count()
	if first == 0 {
		t.Fatal("first registration skipped the full check")
	}
	for i := 0; i < 5; i++ {
		if r := s.handle(context.Background(), heartbeat); r.Code != "ok" {
			t.Fatalf("heartbeat %+v", r)
		}
	}
	s.reconcile(context.Background())
	if got := count(); got != first {
		t.Fatalf("heartbeats and reconcile ran %d full checks within the window", got-first)
	}
	submit := couchmessage.Request{Op: "operator-submit", Binding: &f.binding}
	if r := s.handle(context.Background(), submit); r.Code != "ok" || count() == first {
		t.Fatalf("operator-submit skipped the full check: %+v", r)
	}
	afterSubmit := count()
	advanceMessageClock(s, messageVerificationWindow)
	if r := s.handle(context.Background(), heartbeat); r.Code != "ok" || count() == afterSubmit {
		t.Fatalf("heartbeat after the window skipped the full check: %+v", r)
	}
}

func TestMessageReconcileForgetsDeadBindings(t *testing.T) {
	s, f := serviceFixture(t)
	if err := s.register(context.Background(), f.binding); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.live = false
	f.mu.Unlock()
	advanceMessageClock(s, messageVerificationWindow)
	s.reconcile(context.Background())
	s.mu.Lock()
	_, workspace := s.workspaces[f.binding]
	_, verified := s.verified[f.binding]
	s.mu.Unlock()
	if workspace || verified {
		t.Fatalf("dead binding retained: workspace %v verified %v", workspace, verified)
	}
}

func TestMessageEndpointObserveReusesRecentCheckButReserveNever(t *testing.T) {
	f := newMessageAuthorityFake()
	fresh := true
	endpoint := messageEndpoint{authority: f.authority(), binding: f.binding, endpoint: serviceEndpointFake{}, fresh: func() bool { return fresh }}
	f.mu.Lock()
	f.pid++
	f.mu.Unlock()
	if _, err := endpoint.Observe(context.Background()); err != nil {
		t.Fatalf("fresh observation re-checked: %v", err)
	}
	if err := endpoint.Reserve(context.Background(), "id", 1); err == nil {
		t.Fatal("reserve trusted a recent check")
	}
	fresh = false
	if _, err := endpoint.Observe(context.Background()); err == nil {
		t.Fatal("stale observation skipped the check")
	}
}

func TestMessageActorsListingIsMemoryOnly(t *testing.T) {
	f := newMessageAuthorityFake()
	calls := 0
	dir, err := os.MkdirTemp("/tmp", "pair-message-service-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	s, err := newMessageService(context.Background(), filepath.Join(dir, "broker.sock"), f.countingLaunch(&calls))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if r := s.handle(context.Background(), couchmessage.Request{Op: "register", Binding: &f.binding}); r.Code != "ok" {
		t.Fatalf("register %+v", r)
	}
	f.mu.Lock()
	before := calls
	f.mu.Unlock()
	actors := couchmessage.Request{Op: "actors", Scope: f.binding.Scope, Tag: f.binding.Tag, Session: f.binding.Session, Nonce: f.binding.Nonce}
	for i := 0; i < 3; i++ {
		r := s.handle(context.Background(), actors)
		if r.Code != "ok" || len(r.Actors) != 1 || !r.Actors[0].Known || !r.Actors[0].Resting {
			t.Fatalf("listing %+v", r)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if calls != before {
		t.Fatalf("listing ran %d full checks", calls-before)
	}
}
