package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type sessionStaleRig struct {
	*waitRig
	watcher      *sessionStaleWatcher
	endpoint     *resourceBrokerEndpoint
	client       *couchmessage.SessionClient
	file         string
	checks, logs atomic.Int32
	now          time.Time
	output       string
	failure      error
}

func newSessionStaleRig(t *testing.T) *sessionStaleRig {
	w := &waitRig{serviceRig: newServiceRig(t)}
	w.world.hub = "/repo1"
	w.restart(nil)
	w.hub = w.world.add(1)
	endpoint := &resourceBrokerEndpoint{deliveries: make(chan couchmessage.Message, 32)}
	w.world.endpoints[w.hub] = endpoint
	w.serviceRig.mu.Lock()
	w.slotGit["/repo1"] = couchcore.SlotGitStatus{Branch: "main-slot1"}
	w.serviceRig.mu.Unlock()
	client := w.v2wrapper(w.hub, "hub", true)
	client.Picker(false)
	testwait.Until(t, "hub picker evidence", func() bool {
		live, ok := w.s.LivenessForThread(w.hub.Scope, w.hub.Tag)
		return ok && live.Picker != nil && !*live.Picker
	})
	r := &sessionStaleRig{waitRig: w, endpoint: endpoint, client: client, file: filepath.Join(t.TempDir(), "session.md"), now: time.Now(), output: "stale: ## Threads (unstamped)\n"}
	r.watcher = newSessionStaleWatcher(w.s, t.TempDir(), func(string) { r.logs.Add(1) })
	r.watcher.now = func() time.Time { return r.now }
	r.watcher.check = func(context.Context, string) (string, error) { r.checks.Add(1); return r.output, r.failure }
	w.s.supervision.Store(newSupervisionCoordinator(w.s, func(string) {}))
	doc := "---\ntype: session-sync\ncouch-owner: " + w.hub.Slot + "\n---\n\n## Threads\n\nunread body\n"
	if err := os.WriteFile(r.file, []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.watcher.sync.Register(r.file, w.hub.Slot); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *sessionStaleRig) waitMessage(t *testing.T) couchmessage.Message {
	t.Helper()
	select {
	case m := <-r.endpoint.deliveries:
		testwait.Until(t, "reminder delivery settled", func() bool { pending, known := r.s.broker.ResourceIdle(r.hub); return known && !pending })
		return m
	case <-time.After(3 * time.Second):
		t.Fatal("no reminder")
		return couchmessage.Message{}
	}
}

func TestSessionStaleWorkerCadenceAndReadonly(t *testing.T) {
	r := newSessionStaleRig(t)
	ctx := context.Background()
	original, err := os.ReadFile(r.file)
	if err != nil {
		t.Fatal(err)
	}
	r.watcher.tick(ctx)
	m := r.waitMessage(t)
	if m.From != r.hub || m.To != r.hub || !strings.Contains(m.Body, r.output[:len(r.output)-1]) {
		t.Fatalf("message %+v", m)
	}
	r.watcher.tick(ctx)
	if r.checks.Load() != 1 {
		t.Fatal("checked within allowance")
	}
	r.now = r.now.Add(30 * time.Minute)
	r.watcher.tick(ctx)
	r.waitMessage(t)
	r.now = r.now.Add(30 * time.Minute)
	r.output = "fresh: every budgeted section\n"
	r.watcher.tick(ctx)
	if r.checks.Load() != 3 {
		t.Fatal("wrong cadence")
	}
	select {
	case <-r.endpoint.deliveries:
		t.Fatal("fresh reminder")
	default:
	}
	after, err := os.ReadFile(r.file)
	if err != nil || string(after) != string(original) {
		t.Fatal("document changed")
	}
	if _, err := os.Stat(sessionsync.PathsOf(r.file).Shadow); !os.IsNotExist(err) {
		t.Fatal("inspection created a legacy shadow")
	}
}

func TestSessionStaleWorkerLockBusyFailureAndRelease(t *testing.T) {
	r := newSessionStaleRig(t)
	ctx := context.Background()
	if err := os.WriteFile(r.file+".lock", []byte("holder: operator\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r.watcher.tick(ctx)
	if r.checks.Load() != 0 {
		t.Fatal("checked held file")
	}
	if err := os.Remove(r.file + ".lock"); err != nil {
		t.Fatal(err)
	}
	r.client.Settle(false)
	r.waitSettled(r.hub, false)
	r.watcher.tick(ctx)
	if r.checks.Load() != 0 {
		t.Fatal("checked busy hub")
	}
	r.client.Settle(true)
	r.waitSettled(r.hub, true)
	r.failure = errors.New("tl-lead missing")
	r.watcher.tick(ctx)
	r.watcher.tick(ctx)
	r.now = r.now.Add(30 * time.Minute)
	r.watcher.tick(ctx)
	if r.checks.Load() != 2 || r.logs.Load() != 1 {
		t.Fatalf("checks %d logs %d", r.checks.Load(), r.logs.Load())
	}
	select {
	case <-r.endpoint.deliveries:
		t.Fatal("failed check sent")
	default:
	}
	r.failure = nil
	r.now = r.now.Add(30 * time.Minute)
	r.watcher.tick(ctx)
	r.waitMessage(t)
}

func TestSessionStaleWorkerRechecksLockAfterCommand(t *testing.T) {
	r := newSessionStaleRig(t)
	r.watcher.check = func(context.Context, string) (string, error) {
		if err := os.WriteFile(r.file+".lock", []byte("holder: operator\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return r.output, nil
	}
	r.watcher.tick(context.Background())
	select {
	case <-r.endpoint.deliveries:
		t.Fatal("message after lock acquired")
	default:
	}
	if err := os.Remove(r.file + ".lock"); err != nil {
		t.Fatal(err)
	}
	r.watcher.check = func(context.Context, string) (string, error) { return r.output, nil }
	r.watcher.tick(context.Background())
	r.waitMessage(t)
}

func TestSessionStaleWorkerDoesNotBlockWaitTicksAndCancels(t *testing.T) {
	r := newSessionStaleRig(t)
	started := make(chan struct{})
	var active, maximum atomic.Int32
	r.watcher.check = func(ctx context.Context, _ string) (string, error) {
		n := active.Add(1)
		maximum.Store(n)
		defer active.Add(-1)
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}
	r.s.workers.Add(1)
	go func() { defer r.s.workers.Done(); r.watcher.run(r.s.lifetime) }()
	waiter := newWaitWatcher(r.s, time.Hour, t.TempDir())
	waiter.staleTick = r.watcher.offer
	waiter.tick(context.Background(), r.now)
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("check not started")
	}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			waiter.tick(context.Background(), r.now)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("status command blocked wait watcher")
	}
	r.s.cancel()
	r.s.workers.Wait()
	if maximum.Load() != 1 || active.Load() != 0 || r.logs.Load() != 0 {
		t.Fatalf("max %d active %d diagnostics %d", maximum.Load(), active.Load(), r.logs.Load())
	}
	select {
	case <-r.endpoint.deliveries:
		t.Fatal("shutdown sent a message")
	default:
	}
}
