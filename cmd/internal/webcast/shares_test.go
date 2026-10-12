package webcast

import (
	"errors"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func sharesOn(t *testing.T, f *broadcast.FakeTunnel, names ...string) *ReplayShares {
	t.Helper()
	h := broadcast.NewHost(broadcast.HostConfig{Tunnel: f, ProbeTimeout: 2 * time.Second})
	s := NewReplayShares(h, replayRoot(t, names...))
	t.Cleanup(s.Close)
	return s
}

// awaitShare asks for id and polls its Status as the CLI does, until it
// leaves opening.
func awaitShare(t *testing.T, s *ReplayShares, id string) ShareStatus {
	t.Helper()
	s.Share(id)
	deadline := time.Now().Add(5 * time.Second)
	for {
		st := s.Status(id)
		if st.State != ShareOpening || time.Now().After(deadline) {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func shareGet(t *testing.T, url string) (int, string) {
	t.Helper()
	res, err := (&http.Client{Timeout: 2 * time.Second}).Get(url)
	if err != nil {
		return 0, err.Error()
	}
	defer res.Body.Close()
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := res.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return res.StatusCode, b.String()
}

// A share opens in the background, then serves its recording through the
// tunnel, and only that one; a second recording gets its own token on the
// same tunnel.
func TestReplaySharesShareOneRecordingPerToken(t *testing.T) {
	f := &broadcast.FakeTunnel{OpenDelay: 50 * time.Millisecond}
	s := sharesOn(t, f, "segments", "blog")
	if st := s.Share("segments"); st.State != ShareOpening {
		t.Fatalf("first Share: %+v", st)
	}
	a := awaitShare(t, s, "segments")
	if a.State != ShareReady || a.Base == "" {
		t.Fatalf("segments: %+v", a)
	}
	link := ReplayLink(a.Base, "segments", 1500)
	if code, body := shareGet(t, strings.TrimSuffix(link, "#t=1500")); code != 200 || !strings.Contains(body, `id="timeline"`) {
		t.Fatalf("viewer %d", code)
	}
	if code, body := shareGet(t, a.Base+"/r/segments/records"); code != 200 || !strings.Contains(body, `"k":"head"`) {
		t.Fatalf("records %d %.80s", code, body)
	}
	b := awaitShare(t, s, "blog")
	if b.State != ShareReady || b.Base == a.Base {
		t.Fatalf("blog: %+v (segments %s)", b, a.Base)
	}
	for _, u := range []string{a.Base + "/r/blog/", a.Base + "/", b.Base + "/r/segments/records"} {
		if code, _ := shareGet(t, u); code != 404 {
			t.Errorf("%s: %d, want 404", u, code)
		}
	}
	if st := s.Share("segments"); st != a {
		t.Errorf("a ready share's Share answers it again: %+v, want %+v", st, a)
	}
	if st := f.Stats(); st.Opens != 1 {
		t.Errorf("tunnel opens %d, want 1 (shared)", st.Opens)
	}
}

// Two Shares while opening start one mount, under one token.
func TestReplaySharesOneMountWhileOpening(t *testing.T) {
	f := &broadcast.FakeTunnel{OpenDelay: 100 * time.Millisecond}
	s := sharesOn(t, f, "segments")
	for range 5 {
		if st := s.Share("segments"); st.State != ShareOpening {
			t.Fatalf("%+v", st)
		}
	}
	a := awaitShare(t, s, "segments")
	s.mu.Lock()
	n := len(s.shares)
	s.mu.Unlock()
	if a.State != ShareReady || n != 1 || f.Stats().Opens != 1 || f.Stats().Listens != 1 {
		t.Fatalf("%+v, %d shares, %+v", a, n, f.Stats())
	}
}

// A share whose tunnel died is lost; the next Share mounts it again under
// a new token.
func TestReplaySharesLostThenReshared(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	s := sharesOn(t, f, "segments")
	a := awaitShare(t, s, "segments")
	f.Exit()
	deadline := time.Now().Add(2 * time.Second)
	var lost ShareStatus
	for {
		lost = s.Status("segments")
		if lost.State == ShareLost || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if lost.State != ShareLost || !errors.Is(lost.Err, broadcast.ErrTunnelExited) {
		t.Fatalf("after exit: %+v", lost)
	}
	b := awaitShare(t, s, "segments")
	if b.State != ShareReady || b.Base == a.Base {
		t.Fatalf("reshared %+v (was %s)", b, a.Base)
	}
}

// An unknown recording fails at once and opens nothing; a failed mount
// (no tunnel to open) is failed, and a later Share tries again.
func TestReplaySharesFailures(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	s := sharesOn(t, f, "segments")
	for _, id := range []string{"nope", "../segments", ""} {
		if st := s.Share(id); st.State != ShareFailed || !errors.Is(st.Err, ErrNoRecording) {
			t.Errorf("%q: %+v", id, st)
		}
	}
	if st := s.Status("segments"); st.State != ShareFailed || !errors.Is(st.Err, ErrNotShared) {
		t.Errorf("never asked for: %+v", st)
	}
	if f.Stats().Listens != 0 {
		t.Errorf("an unknown recording opened the tunnel: %+v", f.Stats())
	}
	none := NewReplayShares(broadcast.NewHost(broadcast.HostConfig{}), replayRoot(t, "segments"))
	defer none.Close()
	if st := awaitShare(t, none, "segments"); st.State != ShareFailed || st.Err == nil {
		t.Fatalf("no tunnel: %+v", st)
	}
	if st := none.Status("segments"); st.State != ShareFailed {
		t.Fatalf("Status restarted a failed share: %+v", st)
	}
	if st := none.Share("segments"); st.State != ShareOpening {
		t.Fatalf("a failed share isn't retried: %+v", st)
	}
}

// Close while a share is opening abandons it: the tunnel that finishes
// opening is closed again, every goroutine joins, and Share answers closed.
func TestReplaySharesCloseWhileOpening(t *testing.T) {
	f := &broadcast.FakeTunnel{OpenDelay: 150 * time.Millisecond, IgnoreCancel: true}
	h := broadcast.NewHost(broadcast.HostConfig{Tunnel: f, ProbeTimeout: 2 * time.Second})
	s := NewReplayShares(h, replayRoot(t, "segments", "blog"))
	before := runtime.NumGoroutine()
	s.Share("segments")
	s.Share("blog")
	time.Sleep(20 * time.Millisecond) // both mounts are waiting on the open
	s.Close()
	if st := f.Stats(); st.Opens != st.Closes {
		t.Fatalf("tunnel left open after Close: %+v", st)
	}
	if st, st2 := s.Share("segments"), s.Status("segments"); st.State != ShareClosed || st2.State != ShareClosed {
		t.Fatalf("after Close: Share %+v, Status %+v", st, st2)
	}
	s.Close() // idempotent
	waitGoroutines(t, before)
}

// Close unmounts ready shares and closes the tunnel.
func TestReplaySharesCloseReady(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	h := broadcast.NewHost(broadcast.HostConfig{Tunnel: f, ProbeTimeout: 2 * time.Second})
	s := NewReplayShares(h, replayRoot(t, "segments"))
	a := awaitShare(t, s, "segments")
	s.Close()
	if st := f.Stats(); st.Opens != 1 || st.Closes != 1 {
		t.Fatalf("%+v", st)
	}
	if code, _ := shareGet(t, a.Base+"/r/segments/"); code == 200 {
		t.Fatalf("a closed share still serves")
	}
}

func waitGoroutines(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > want {
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutines, want at most %d", runtime.NumGoroutine(), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A mount that completes just after Close is unmounted at once: nobody
// serves it, and the tunnel it held closes. The seam holds the mount
// between completing and being recorded until Close has begun.
func TestReplaySharesMountCompletingAfterClose(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	h := broadcast.NewHost(broadcast.HostConfig{Tunnel: f, ProbeTimeout: 2 * time.Second})
	s := NewReplayShares(h, replayRoot(t, "segments"))
	entered, release := make(chan struct{}), make(chan struct{})
	s.mounted = func() { close(entered); <-release }
	var ready []string
	s.OnReady(func(id string, _ ShareStatus) { ready = append(ready, id) })
	s.Share("segments")
	<-entered // mounted: the tunnel is open and the share is about to be ready
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	for s.Status("segments").State != ShareClosed {
		time.Sleep(time.Millisecond)
	}
	close(release)
	<-closed
	if st := f.Stats(); st.Opens != 1 || st.Closes != 1 {
		t.Fatalf("tunnel %+v: the late mount was left open", st)
	}
	if len(ready) != 0 || len(s.Shared()) != 0 {
		t.Fatalf("a share after Close was reported: %q %q", ready, s.Shared())
	}
}

// Shared lists the ready shares, and OnReady says when each becomes so.
func TestReplaySharesSharedAndOnReady(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	s := sharesOn(t, f, "segments", "blog")
	var mu sync.Mutex
	var ready []string
	s.OnReady(func(id string, _ ShareStatus) { mu.Lock(); ready = append(ready, id); mu.Unlock() })
	awaitShare(t, s, "segments")
	awaitShare(t, s, "blog")
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(s.Shared(), ",") != "blog,segments" || len(ready) != 2 {
		t.Fatalf("shared %q, ready %q", s.Shared(), ready)
	}
}
