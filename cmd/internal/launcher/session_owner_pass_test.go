package launcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type ownerPassWorld struct {
	mu                                            sync.Mutex
	initial                                       map[string]ServerState
	live                                          map[string][]SessionServerIdentity
	captures, liveReads, paneReads, presenceReads int
	sourceErr                                     error
	paneHook                                      func(context.Context, string)
}

func newOwnerPassWorld(n int) *ownerPassWorld {
	w := &ownerPassWorld{initial: make(map[string]ServerState), live: make(map[string][]SessionServerIdentity)}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("session-%d", i)
		s := SessionServerIdentity{PID: i + 1, Identity: "original", Session: name, Socket: "/tmp/" + name}
		w.initial[name] = ServerState{Server: s, Verdict: ServerReachable}
		w.live[name] = []SessionServerIdentity{s}
	}
	return w
}

func (w *ownerPassWorld) ServerStates(context.Context) (map[string]ServerState, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.captures++
	return w.initial, w.sourceErr
}
func (w *ownerPassWorld) SessionServers(_ context.Context, name string) ([]SessionServerIdentity, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.liveReads++
	return append([]SessionServerIdentity(nil), w.live[name]...), nil
}
func (w *ownerPassWorld) SessionPresent(_ context.Context, name string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.presenceReads++
	return len(w.live[name]) > 0, nil
}
func (w *ownerPassWorld) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	w.mu.Lock()
	w.paneReads++
	hook := w.paneHook
	w.mu.Unlock()
	if hook != nil {
		hook(ctx, name)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []byte(`[{"id":1,"is_plugin":false,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"}]`), nil
}

type ownerPassSourceFunc func(context.Context) (map[string]ServerState, error)

func (f ownerPassSourceFunc) ServerStates(ctx context.Context) (map[string]ServerState, error) {
	return f(ctx)
}

func awaitOwnerPass[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(time.Second):
		t.Fatal("pass did not reach cancellation/completion barrier")
		var zero T
		return zero
	}
}

func TestSessionOwnerPassParentCancellationStopsLivePaneRead(t *testing.T) {
	w := newOwnerPassWorld(1)
	started := make(chan struct{})
	w.paneHook = func(ctx context.Context, _ string) { close(started); <-ctx.Done() }
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := NewSessionOwnerPass(parent, w, SessionOwnerProbe{IO: w})
	done := make(chan error, 1)
	go func() {
		_, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		done <- err
	}()
	awaitOwnerPass(t, started)
	cancel()
	if err := awaitOwnerPass(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("parent cancellation lost: %v", err)
	}
}
func (w *ownerPassWorld) Socket(string) SocketState { return SocketPresent }

func TestSessionOwnerPassSharesInitialCaptureFor21Callers(t *testing.T) {
	w := newOwnerPassWorld(21)
	p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
	var wg sync.WaitGroup
	for i := 0; i < 21; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got, err := p.Probe(t.Context(), fmt.Sprintf("session-%d", i), "/tmp/data", "0123456789abcdef", "1-repo-1")
			if err != nil || got.State != SessionOwnerOwned {
				t.Errorf("probe %d: got=%+v err=%v", i, got, err)
				return
			}
			if err := p.Revalidate(t.Context(), got); err != nil {
				t.Errorf("revalidate %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	if w.captures != 1 || w.liveReads != 42 || w.paneReads != 42 {
		t.Fatalf("captures=%d live reads=%d panes=%d, want 1 initial capture + 21 after-pair + 21 revalidate; 42 native pane reads", w.captures, w.liveReads, w.paneReads)
	}
}

func TestSessionOwnerPassMemoizesCaptureErrorFor21Callers(t *testing.T) {
	w := newOwnerPassWorld(21)
	w.sourceErr = errors.New("native inspection unavailable")
	p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
	var wg sync.WaitGroup
	for i := 0; i < 21; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for repeat := 0; repeat < 2; repeat++ {
				got, err := p.Probe(t.Context(), fmt.Sprintf("session-%d", i), "/tmp/data", "0123456789abcdef", "1-repo-1")
				if !errors.Is(err, w.sourceErr) || got.State != SessionOwnerUnknown {
					t.Errorf("capture error became authority: %+v %v", got, err)
				}
			}
		}(i)
	}
	wg.Wait()
	if w.captures != 1 || w.liveReads != 0 || w.paneReads != 0 || w.presenceReads != 0 {
		t.Fatalf("capture failure fell back to per-slot IO: %+v", w)
	}
}

func TestSessionOwnerPassCanceledFirstCallerDoesNotPoisonSibling(t *testing.T) {
	w := newOwnerPassWorld(1)
	started, release := make(chan struct{}), make(chan struct{})
	source := ownerPassSourceFunc(func(ctx context.Context) (map[string]ServerState, error) {
		states, err := w.ServerStates(ctx)
		deadline, bounded := ctx.Deadline()
		if !bounded || time.Until(deadline) > zellijQueryTimeout {
			t.Error("shared capture lacks query timeout")
		}
		close(started)
		select {
		case <-release:
			return states, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	p := NewSessionOwnerPass(t.Context(), source, SessionOwnerProbe{IO: w})
	firstCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first := make(chan error, 1)
	go func() {
		_, err := p.Probe(firstCtx, "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		first <- err
	}()
	awaitOwnerPass(t, started)
	cancel()
	if err := awaitOwnerPass(t, first); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled caller: %v", err)
	}
	sibling := make(chan error, 1)
	go func() {
		got, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		if err == nil && got.State != SessionOwnerOwned {
			err = fmt.Errorf("sibling lost authority: %+v", got)
		}
		sibling <- err
	}()
	close(release)
	if err := awaitOwnerPass(t, sibling); err != nil {
		t.Fatal(err)
	}
	if w.captures != 1 || w.liveReads != 1 {
		t.Fatalf("capture restarted/extra reads: captures=%d live=%d", w.captures, w.liveReads)
	}
}

func TestSessionOwnerPassParentCancellationStopsCapture(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	source := ownerPassSourceFunc(func(ctx context.Context) (map[string]ServerState, error) {
		close(started)
		<-ctx.Done()
		close(stopped)
		return nil, ctx.Err()
	})
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	w := newOwnerPassWorld(1)
	p := NewSessionOwnerPass(parent, source, SessionOwnerProbe{IO: w})
	done := make(chan error, 1)
	go func() {
		_, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		done <- err
	}()
	awaitOwnerPass(t, started)
	cancel()
	awaitOwnerPass(t, stopped)
	if err := awaitOwnerPass(t, done); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if w.liveReads != 0 || w.paneReads != 0 {
		t.Fatal("canceled capture reached live IO")
	}
}

func TestSessionOwnerPassPreCanceledCallerDoesNotCapture(t *testing.T) {
	w := newOwnerPassWorld(1)
	p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := p.Probe(ctx, "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if !errors.Is(err, context.Canceled) || got.State != SessionOwnerUnknown || w.captures != 0 {
		t.Fatalf("pre-canceled probe: %+v %v captures=%d", got, err, w.captures)
	}
}

func TestSessionOwnerPassClonesMapAndIsolatesPasses(t *testing.T) {
	w := newOwnerPassWorld(1)
	p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
	probe := func(p *SessionOwnerPass) SessionOwnerObservation {
		got, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := probe(p); got.State != SessionOwnerOwned {
		t.Fatalf("original: %+v", got)
	}
	w.mu.Lock()
	s := w.initial["session-0"]
	s.Server.Identity = "changed-after-return"
	w.initial["session-0"] = s
	w.mu.Unlock()
	if got := probe(p); got.State != SessionOwnerOwned {
		t.Fatalf("capture aliased returned source map: %+v", got)
	}
	second := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
	if got := probe(second); got.State != SessionOwnerUnknown {
		t.Fatalf("independent pass reused old evidence: %+v", got)
	}
	if w.captures != 2 {
		t.Fatalf("captures=%d want 2", w.captures)
	}
}

func TestSessionOwnerPassRefusesContestedUnresolvedAndInvalidInitial(t *testing.T) {
	for _, verdict := range []ServerVerdict{ServerContested, ServerUnresolved, ServerReachable} {
		w := newOwnerPassWorld(1)
		s := w.initial["session-0"]
		s.Verdict = verdict
		if verdict == ServerReachable {
			s.Server.Identity = ""
		}
		w.initial["session-0"] = s
		p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
		got, _ := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		if got.State != SessionOwnerUnknown || w.liveReads != 0 || w.paneReads != 0 || w.presenceReads != 0 {
			t.Fatalf("initial %v became usable: %+v reads=%d/%d/%d", verdict, got, w.liveReads, w.paneReads, w.presenceReads)
		}
	}
}

func TestSessionOwnerPassAbsentEvidenceRequiresFreshDiscovery(t *testing.T) {
	for _, born := range []bool{false, true} {
		w := newOwnerPassWorld(1)
		delete(w.initial, "session-0")
		if !born {
			delete(w.live, "session-0")
		}
		p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
		got, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
		want := SessionOwnerAbsent
		if born {
			want = SessionOwnerUnknown
		}
		if err != nil || got.State != want || w.liveReads != 1 || w.presenceReads != 1 || w.paneReads != 0 {
			t.Fatalf("born=%v got=%+v err=%v live=%d presence=%d panes=%d", born, got, err, w.liveReads, w.presenceReads, w.paneReads)
		}
		if err := p.Revalidate(t.Context(), got); err == nil {
			t.Fatal("initial absence became effect authority")
		}
	}
}

func TestSessionOwnerPassFreshFencesRefuseChangedClaimants(t *testing.T) {
	for _, change := range []string{"competitor", "generation", "socket", "disappear"} {
		t.Run(change, func(t *testing.T) {
			w := newOwnerPassWorld(1)
			original := w.live["session-0"][0]
			mutate := func() {
				w.mu.Lock()
				defer w.mu.Unlock()
				s := original
				switch change {
				case "competitor":
					s.PID++
					s.Identity = "competitor"
					w.live["session-0"] = []SessionServerIdentity{original, s}
				case "generation":
					s.Identity = "recycled"
					w.live["session-0"] = []SessionServerIdentity{s}
				case "socket":
					s.Socket = "/different/session-0"
					w.live["session-0"] = []SessionServerIdentity{s}
				case "disappear":
					delete(w.live, "session-0")
				}
			}
			p := NewSessionOwnerPass(t.Context(), w, SessionOwnerProbe{IO: w})
			owned, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
			if err != nil || owned.State != SessionOwnerOwned {
				t.Fatalf("healthy: %+v %v", owned, err)
			}
			mutate()
			if err := p.Revalidate(t.Context(), owned); err == nil {
				t.Fatal("cached equality replaced fresh effect fence")
			}
			w.mu.Lock()
			w.live["session-0"] = []SessionServerIdentity{original}
			w.paneHook = func(context.Context, string) { mutate() }
			w.mu.Unlock()
			got, err := p.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
			if err != nil || got.State != SessionOwnerUnknown || got.Diagnostic != "session server changed during observation" {
				t.Fatalf("cached equality replaced after-pane exclusivity: %+v %v", got, err)
			}
			if w.captures != 1 || w.liveReads != 3 {
				t.Fatalf("captures=%d live=%d", w.captures, w.liveReads)
			}
		})
	}
}
