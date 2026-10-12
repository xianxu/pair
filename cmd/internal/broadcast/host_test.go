package broadcast

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

// tokenHandler is a Mountable answering one token with its own name.
type tokenHandler string

func (t tokenHandler) Owns(segment string) bool { return segment == string(t) }
func (t tokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, string(t))
}

func hostGet(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(url)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func mount(t *testing.T, h *Host, token string) *Mount {
	t.Helper()
	m, err := h.Mount(context.Background(), tokenHandler(token), token+"/")
	if err != nil {
		t.Fatalf("mount %s: %v", token, err)
	}
	return m
}

// assertServes checks the stated invariant from the outside: a mounted
// handler is reachable iff its mount is live, and at most one tunnel was
// ever open at once.
func assertServes(t *testing.T, f *FakeTunnel, m *Mount, token string, want bool) {
	t.Helper()
	code, body := hostGet(t, m.Base()+"/"+token+"/")
	if got := code == 200 && body == token; got != want {
		t.Fatalf("%s reachable=%v (code %d %q), want %v", token, got, code, body, want)
	}
	if s := f.Stats(); s.MaxOpen > 1 {
		t.Fatalf("%d tunnels open at once", s.MaxOpen)
	}
}

func TestHostSharesOneTunnelAndClosesAfterTheLast(t *testing.T) {
	f := &FakeTunnel{}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	a, b := mount(t, h, "aa"), mount(t, h, "bb")
	if a.Base() != b.Base() || f.Stats().Opens != 1 {
		t.Fatalf("bases %s %s, stats %+v", a.Base(), b.Base(), f.Stats())
	}
	assertServes(t, f, a, "aa", true)
	assertServes(t, f, b, "bb", true)
	if code, _ := hostGet(t, a.Base()+"/cc/"); code != 404 {
		t.Fatalf("unknown token: %d", code)
	}
	a.Unmount()
	assertServes(t, f, a, "aa", false)
	assertServes(t, f, b, "bb", true)
	b.Unmount()
	b.Unmount()
	if s := f.Stats(); s.Closes != 1 || s.DoubleCloses != 0 {
		t.Fatalf("stats %+v", s)
	}
	if _, err := net.DialTimeout("tcp", f.LastAddr(), 200*time.Millisecond); err == nil {
		t.Fatal("listener still accepting after the last unmount")
	}
}

func TestHostMountsDuringOpeningShareIt(t *testing.T) {
	f := &FakeTunnel{OpenDelay: 200 * time.Millisecond}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	var wg sync.WaitGroup
	mounts := make([]*Mount, 3)
	for i := range mounts {
		wg.Go(func() { mounts[i] = mount(t, h, string(rune('a'+i))+"x") })
	}
	wg.Wait()
	if s := f.Stats(); s.Opens != 1 {
		t.Fatalf("stats %+v", s)
	}
	for i, m := range mounts {
		assertServes(t, f, m, string(rune('a'+i))+"x", true)
	}
}

func TestHostMountDuringClosingReopens(t *testing.T) {
	block := make(chan struct{})
	f := &FakeTunnel{CloseBlock: block}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	a := mount(t, h, "aa")
	unmounted := make(chan struct{})
	go func() { a.Unmount(); close(unmounted) }()
	waitFor(t, func() bool { h.mu.Lock(); defer h.mu.Unlock(); return h.state == hostClosing })
	got := make(chan *Mount)
	go func() { got <- mount(t, h, "bb") }()
	select {
	case <-got:
		t.Fatal("mounted on a closing host")
	case <-time.After(100 * time.Millisecond):
	}
	close(block)
	<-unmounted
	b := <-got
	if s := f.Stats(); s.Opens != 2 || s.Closes != 1 {
		t.Fatalf("stats %+v", s)
	}
	assertServes(t, f, b, "bb", true)
}

// failingTunnel fails its first Open after a delay, then behaves.
type failingTunnel struct {
	LocalOnly
	mu     sync.Mutex
	failed bool
}

var errOpen = errors.New("open failed")

func (f *failingTunnel) Open(ctx context.Context, l net.Listener) (Handle, error) {
	f.mu.Lock()
	first := !f.failed
	f.failed = true
	f.mu.Unlock()
	if first {
		time.Sleep(100 * time.Millisecond)
		return nil, errOpen
	}
	return f.LocalOnly.Open(ctx, l)
}

func TestHostFailedOpenReachesEveryWaiter(t *testing.T) {
	ft := &failingTunnel{}
	h := NewHost(HostConfig{Tunnel: ft, ProbeTimeout: time.Second})
	errs := make(chan error, 2)
	for _, tok := range []string{"aa", "bb"} {
		go func() {
			_, err := h.Mount(context.Background(), tokenHandler(tok), tok+"/")
			errs <- err
		}()
	}
	for range 2 {
		if err := <-errs; !errors.Is(err, errOpen) {
			t.Fatalf("waiter got %v", err)
		}
	}
	m := mount(t, h, "cc") // the next Mount opens afresh
	if code, body := hostGet(t, m.Base()+"/cc/"); code != 200 || body != "cc" {
		t.Fatalf("after a failed open: %d %q", code, body)
	}
	m.Unmount()
}

func TestHostCancelledWaiterLeavesTheOpenToOthers(t *testing.T) {
	f := &FakeTunnel{OpenDelay: 200 * time.Millisecond}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cancelled := make(chan error)
	go func() {
		_, err := h.Mount(ctx, tokenHandler("aa"), "aa/")
		cancelled <- err
	}()
	time.Sleep(10 * time.Millisecond)
	b := mount(t, h, "bb")
	if err := <-cancelled; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelled waiter: %v", err)
	}
	if s := f.Stats(); s.Opens != 1 || s.Closes != 0 {
		t.Fatalf("stats %+v", s)
	}
	assertServes(t, f, b, "bb", true)
}

func TestHostTunnelExitReachesEveryMount(t *testing.T) {
	f := &FakeTunnel{}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	a, b := mount(t, h, "aa"), mount(t, h, "bb")
	f.Exit()
	for _, m := range []*Mount{a, b} {
		select {
		case <-m.Exited():
		case <-time.After(2 * time.Second):
			t.Fatal("mount not told the tunnel exited")
		}
		if !errors.Is(m.Err(), ErrTunnelExited) {
			t.Fatalf("err %v", m.Err())
		}
	}
	a.Unmount() // already gone: no-op
	c := mount(t, h, "cc")
	if s := f.Stats(); s.Opens != 2 {
		t.Fatalf("stats %+v", s)
	}
	assertServes(t, f, c, "cc", true)
}

// The collision PQ-1 named: a screen broadcast and a web cast on a named
// tunnel. Two private hosts collide on the claim; one shared Host serves both.
func TestHostSharedNamedTunnelServesBothCasts(t *testing.T) {
	f := &FakeTunnel{Exclusive: true}
	s1, err := Start(context.Background(), Config{Tunnel: f, ProbeTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Start(context.Background(), Config{Tunnel: f, ProbeTimeout: time.Second}); !errors.Is(err, ErrTunnelBusy) {
		t.Fatalf("second private host: %v, want ErrTunnelBusy", err)
	}
	s1.Stop(nil)
	<-s1.Done()

	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	screen, err := Start(context.Background(), Config{Host: h})
	if err != nil {
		t.Fatal(err)
	}
	web := mount(t, h, "web")
	if code, _ := hostGet(t, screen.Link()); code != 200 {
		t.Fatalf("screen broadcast page: %d", code)
	}
	assertServes(t, f, web, "web", true)
	screen.Stop(nil)
	<-screen.Done()
	assertServes(t, f, web, "web", true)
	web.Unmount()
	if s := f.Stats(); s.Opens-s.Closes != 0 {
		t.Fatalf("stats %+v", s)
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// BR-14: an open completes with one waiter still counted, and that waiter
// then gives up. Nothing is mounted and nobody waits, so the Host must close.
func TestHostLastWaiterGivingUpAfterTheOpenCloses(t *testing.T) {
	f := &FakeTunnel{}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	a := mount(t, h, "aa")
	h.mu.Lock()
	h.waiting++ // a waiter registered during the open, not yet woken
	h.mu.Unlock()
	a.Unmount() // the waiter keeps the Host open
	if s := f.Stats(); s.Closes != 0 {
		t.Fatalf("closed under a waiter: %+v", s)
	}
	h.abandonWait(true)
	if s := f.Stats(); s.Opens != s.Closes {
		t.Fatalf("tunnel left open: %+v", s)
	}
}

// The reviewer's reproduction: cancels landing around the open's completion
// never leave a tunnel open.
func TestHostCancelRacingOpenNeverLeaks(t *testing.T) {
	f := &FakeTunnel{OpenDelay: 2 * time.Millisecond, IgnoreCancel: true}
	h := NewHost(HostConfig{Tunnel: f, ProbeTimeout: time.Second})
	for i := range 300 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1+i%4)*time.Millisecond)
		m, err := h.Mount(ctx, tokenHandler("aa"), "aa/")
		cancel()
		if err == nil {
			m.Unmount()
		}
		if s := f.Stats(); s.Opens != s.Closes {
			t.Fatalf("iteration %d: tunnel left open: %+v", i, s)
		}
	}
}
