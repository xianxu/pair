package broadcast

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func protectedConfig() Cloudflared {
	return Cloudflared{Named: &NamedTunnel{Name: "remote", Hostname: "remote.example.com"}, Guard: []string{"couch", "__broadcast-guard"}, Records: "records", RunDir: "/tmp"}
}
func TestProtectedConfiguration(t *testing.T) {
	access := AccessEnforcement{TeamName: "team", Audience: strings.Repeat("a", 64)}
	for _, change := range []func(*Cloudflared){func(c *Cloudflared) { c.Named = nil }, func(c *Cloudflared) { c.Guard = nil }, func(c *Cloudflared) { c.Records = "" }, func(c *Cloudflared) { c.Named.Hostname = "evil\nfield: true" }, func(c *Cloudflared) { c.Named.Name = "../other" }} {
		c := protectedConfig()
		change(&c)
		if _, err := NewProtectedTunnel(c, access); err == nil {
			t.Fatal("unsafe config accepted")
		}
	}
	c := protectedConfig()
	p, err := NewProtectedTunnel(c, access)
	if err != nil {
		t.Fatal(err)
	}
	c.Named.Hostname = "changed.example.com"
	c.Guard[0] = "changed"
	l, err := p.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	pl := l.(*privateListener)
	for path, want := range map[string]os.FileMode{pl.dir: 0700, pl.socket: 0600} {
		s, err := os.Stat(path)
		if err != nil || s.Mode().Perm() != want {
			t.Fatalf("%s: %v %v", path, s, err)
		}
	}
	_, url, _, err := p.cloudflared.tunnelArgs(pl)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://remote.example.com" || p.cloudflared.Guard[0] != "couch" {
		t.Fatal("config was mutable")
	}
	body, err := os.ReadFile(filepath.Join(pl.dir, "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := "tunnel: \"remote\"\ningress:\n  - hostname: \"remote.example.com\"\n    service: \"unix:" + pl.socket + "\"\n    originRequest:\n      access:\n        required: true\n        teamName: \"team\"\n        audTag:\n          - \"" + access.Audience + "\"\n  - service: http_status:404\n"
	if string(body) != want {
		t.Fatalf("config:\n%s", body)
	}
	long := t.TempDir() + "/" + strings.Repeat("x", 80)
	if err := os.MkdirAll(long, 0700); err != nil {
		t.Fatal(err)
	}
	p.cloudflared.RunDir = long
	if l, err := p.Listen(); err == nil {
		l.Close()
		t.Fatal("long Unix path fell back")
	}
	if _, err := p.Open(context.Background(), mustTCP(t)); err == nil {
		t.Fatal("accepted TCP origin")
	}
}
func mustTCP(t *testing.T) net.Listener {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { l.Close() })
	return l
}
func TestProtectedHostAdmissionAndNoProbe(t *testing.T) {
	if _, e := NewProtectedHost(LocalOnly{}); e == nil {
		t.Fatal("local accepted")
	}
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{}}
	h, e := NewProtectedHost(f)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := h.Mount(context.Background(), tokenHandler("x"), "x"); e == nil {
		t.Fatal("ordinary mount accepted")
	}
	if _, e := NewHost(HostConfig{Tunnel: f}).MountProtected(context.Background(), tokenHandler("x")); e == nil {
		t.Fatal("ordinary host accepted protected mount")
	}
	calls := 0
	m, e := h.MountProtected(context.Background(), countHandler{&calls})
	if e != nil {
		t.Fatal(e)
	}
	if m.Base() != "https://remote.example.com" {
		t.Fatalf("public base %q", m.Base())
	}
	if calls != 0 {
		t.Fatal("public readiness probed handler")
	}
	if _, e := h.MountProtected(context.Background(), tokenHandler("y")); e == nil {
		t.Fatal("second mount accepted")
	}
	if e := m.UnmountConfirmed(); e != nil {
		t.Fatal(e)
	}
	if s := f.Stats(); s.Opens != 1 || s.Closes != 1 {
		t.Fatalf("%+v", s)
	}
}

type countHandler struct{ calls *int }

func (countHandler) Owns(string) bool                               { return true }
func (c countHandler) ServeHTTP(http.ResponseWriter, *http.Request) { *c.calls++ }
func (c countHandler) LockFor(string) *castlock.Lock                { return nil }
func TestProtectedCloseFailureStaysBlocked(t *testing.T) {
	failure := errors.New("uncertain close")
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{CloseError: failure}}
	h, _ := NewProtectedHost(f)
	m, e := h.MountProtected(context.Background(), tokenHandler("x"))
	if e != nil {
		t.Fatal(e)
	}
	if e := m.UnmountConfirmed(); !errors.Is(e, failure) {
		t.Fatalf("close %v", e)
	}
	if e := m.UnmountConfirmed(); !errors.Is(e, failure) {
		t.Fatalf("repeat close %v", e)
	}
	if _, e := h.MountProtected(context.Background(), tokenHandler("y")); e == nil {
		t.Fatal("reopened uncertain host")
	}
}

func TestProtectedExclusiveDuringOpeningAndLateCancelJoin(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	closeStarted := make(chan struct{}, 1)
	closeRelease := make(chan struct{})
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{OpenStarted: started, OpenRelease: release, IgnoreCancel: true, CloseStarted: closeStarted, CloseBlock: closeRelease}}
	h, _ := NewProtectedHost(f)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := h.MountProtected(ctx, tokenHandler("first")); result <- err }()
	<-started
	if _, err := h.MountProtected(context.Background(), tokenHandler("second")); err == nil {
		t.Fatal("mounted during opening")
	}
	cancel()
	select {
	case err := <-result:
		t.Fatalf("returned before late open was reclaimed: %v", err)
	default:
	}
	close(release)
	<-closeStarted
	select {
	case err := <-result:
		t.Fatalf("returned before handle close joined: %v", err)
	default:
	}
	if _, err := h.MountProtected(context.Background(), tokenHandler("third")); err == nil {
		t.Fatal("mounted during cancellation teardown")
	}
	close(closeRelease)
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if s := f.Stats(); s.Opens != 1 || s.Closes != 1 {
		t.Fatalf("late open leaked: %+v", s)
	}
}
func TestProtectedUnmountJoinsHeldCloseAndListener(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{CloseStarted: started, CloseBlock: release}}
	h, _ := NewProtectedHost(f)
	m, err := h.MountProtected(context.Background(), tokenHandler("x"))
	if err != nil {
		t.Fatal(err)
	}
	addr := f.LastAddr()
	done := make(chan error, 1)
	go func() { done <- m.UnmountConfirmed() }()
	<-started
	if l, err := net.Dial("unix", addr); err == nil {
		l.Close()
		t.Fatal("origin still listening at connector close")
	}
	select {
	case err := <-done:
		t.Fatalf("returned before close joined: %v", err)
	default:
	}
	if _, err := h.MountProtected(context.Background(), tokenHandler("y")); err == nil {
		t.Fatal("mounted during close")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(addr)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("origin residue: %v", err)
	}
}
func TestProtectedRejectsInvalidAccessAndOrdinaryMount(t *testing.T) {
	for _, a := range []AccessEnforcement{{}, {TeamName: "team", Audience: "wrong"}, {TeamName: "bad\nteam", Audience: strings.Repeat("a", 64)}} {
		if _, err := NewProtectedTunnel(protectedConfig(), a); err == nil {
			t.Fatalf("accepted %+v", a)
		}
	}
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{}}
	if _, err := NewHost(HostConfig{Tunnel: f}).Mount(context.Background(), tokenHandler("x"), "x"); err == nil {
		t.Fatal("ordinary path admitted protected tunnel")
	}
}
func TestProtectedCancelledOpenPreservesCleanupFailure(t *testing.T) {
	failure := errors.New("close failed")
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{OpenStarted: started, OpenRelease: release, IgnoreCancel: true, CloseError: failure}}
	h, _ := NewProtectedHost(f)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := h.MountProtected(ctx, tokenHandler("x")); done <- err }()
	<-started
	cancel()
	close(release)
	if err := <-done; !errors.Is(err, failure) || !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cleanup failure: %v", err)
	}
	if _, err := h.MountProtected(context.Background(), tokenHandler("next")); err == nil {
		t.Fatal("reopened after failed disposal")
	}
}

type closeErrorListener struct {
	net.Listener
	failure error
	closed  bool
}

func (l *closeErrorListener) Close() error {
	_ = l.Listener.Close()
	if l.closed {
		return nil
	}
	l.closed = true
	return l.failure
}
func TestProtectedListenerCloseFailureIsRetained(t *testing.T) {
	l, err := listenProtected("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("listener close uncertain")
	pl := l.(*privateListener)
	pl.Listener = &closeErrorListener{Listener: pl.Listener, failure: failure}
	if err := pl.Close(); !errors.Is(err, failure) {
		t.Fatalf("first close: %v", err)
	}
	if err := pl.Close(); !errors.Is(err, failure) {
		t.Fatalf("repeat close: %v", err)
	}
}

type heldProtectedHandler struct{ started, release chan struct{} }

func (heldProtectedHandler) Owns(string) bool              { return true }
func (heldProtectedHandler) LockFor(string) *castlock.Lock { return nil }
func (h heldProtectedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	close(h.started)
	<-h.release
}
func TestProtectedShutdownTimeoutIsUncertain(t *testing.T) {
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{}}
	h, _ := NewProtectedHost(f)
	handler := heldProtectedHandler{make(chan struct{}), make(chan struct{})}
	defer close(handler.release)
	m, err := h.MountProtected(context.Background(), handler)
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", f.LastAddr())
	}}
	defer transport.CloseIdleConnections()
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := (&http.Client{Transport: transport}).Get("http://origin/x/")
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-handler.started
	if err := m.UnmountConfirmed(); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unjoined active handler reported confirmed cleanup: %v", err)
	}
	<-done
}

func TestProtectedUsesExistingGuardAndNamedLease(t *testing.T) {
	bin, _ := fakeCloudflared(t, namedReady)
	c := protectedConfig()
	c.Binary = bin
	c.Guard = guardArgv(t)
	c.RunDir = shortDir(t)
	c.Records = t.TempDir()
	p, err := NewProtectedTunnel(c, AccessEnforcement{TeamName: "team", Audience: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	l, err := p.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	handle, err := p.Open(context.Background(), l)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	other, err := p.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if second, err := p.Open(context.Background(), other); !errors.Is(err, ErrTunnelBusy) {
		if second != nil {
			second.Close()
		}
		t.Fatalf("duplicate named lease: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handle.Exited():
	default:
		t.Fatal("Close did not join connector")
	}
	if _, err := os.Stat(l.(*privateListener).dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("guard left private origin: %v", err)
	}
}

type exitedProtectedFixture struct{ *FakeProtectedTunnel }

func (f exitedProtectedFixture) Open(ctx context.Context, l net.Listener) (Handle, error) {
	h, err := f.FakeProtectedTunnel.Open(ctx, l)
	if err == nil {
		f.Exit()
	}
	return h, err
}
func TestProtectedDoesNotReopenAnAlreadyLostOwner(t *testing.T) {
	for range 100 {
		f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{}}
		h, _ := NewProtectedHost(f)
		// Exercise the existing Host machinery with a connector that exits before
		// returning readiness. External callers cannot replace the private config.
		h.cfg.Tunnel = exitedProtectedFixture{f}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		m, err := h.MountProtected(ctx, tokenHandler("x"))
		cancel()
		if m != nil {
			_ = m.UnmountConfirmed()
		}
		_ = err
		if s := f.Stats(); s.Opens > 1 {
			t.Fatalf("old owner reopened: %+v", s)
		}
	}
}
func TestProtectedLostOpeningCannotRestartBeforeOwnerReturns(t *testing.T) {
	f := &FakeProtectedTunnel{FakeTunnel: &FakeTunnel{}}
	h, _ := NewProtectedHost(f)
	// A successful open advanced gen, and its watcher finished closing before
	// the waiting owner resumed. That owner has not returned a Mount yet.
	h.protectedBusy = true
	h.protectedGeneration = 0
	h.gen = 1
	if m, err := h.mount(context.Background(), tokenHandler("x"), ""); err == nil {
		_ = m.UnmountConfirmed()
		t.Fatal("lost opening restarted under its original owner")
	}
	if s := f.Stats(); s.Opens != 0 {
		t.Fatalf("reopened lost owner: %+v", s)
	}
}
