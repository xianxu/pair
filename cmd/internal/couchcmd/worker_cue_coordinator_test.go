package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type workerCueEndpoint struct {
	serviceEndpointFake
	mu        sync.Mutex
	messages  []couchmessage.Message
	uncertain bool
}

func (e *workerCueEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	e.messages = append(e.messages, m)
	uncertain := e.uncertain
	e.mu.Unlock()
	if uncertain && m.Body == couchmessage.WorkerStartNoticeBody {
		return couchmessage.Receipt{}, errors.New("controlled lost cue acknowledgement")
	}
	return couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}, ctx.Err()
}
func (e *workerCueEndpoint) counts() (cue, resend int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, m := range e.messages {
		if m.Body == couchmessage.WorkerStartNoticeBody {
			cue++
		}
		if m.Body == startupResendPrompt {
			resend++
		}
	}
	return
}

type workerCueRig struct {
	*serviceRig
	hub, worker couchmessage.Binding
	client      *couchmessage.SessionClient
	endpoint    *workerCueEndpoint
	coordinator *supervisionCoordinator
	proofs      atomic.Int32
	ordinal     atomic.Uint64
	stopHub     func()
}

func newWorkerCueRig(t *testing.T, report *couchmessage.WorkerCueReport) *workerCueRig {
	t.Helper()
	base := newServiceRig(t)
	base.world.hub = "/repo1"
	base.restart(nil)
	r := &workerCueRig{serviceRig: base, endpoint: &workerCueEndpoint{}}
	r.hub = base.world.add(1)
	r.worker = base.world.add(2)
	base.world.endpoints[r.worker] = r.endpoint
	base.attach(r.hub, "hub")
	_, r.stopHub = base.wrapper(r.hub)
	base.waitConnected(r.hub, true)
	r.ordinal.Store(7)
	base.s.authority.child = func(context.Context, couchmessage.Binding) (uint64, string, error) {
		r.proofs.Add(1)
		return r.ordinal.Load(), "native-child", nil
	}
	r.client = couchmessage.NewSessionClient()
	r.client.SetBuild(couchmessage.BuildIdentity{SHA256: strings.Repeat("a", 64)})
	r.client.WorkerCue(report)
	r.client.Update(couchmessage.Observation{LastActivity: time.Now(), Sequence: 1})
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.client.Run(ctx, base.regSock, r.worker, couchmessage.DefaultReconnectBackoff, time.Millisecond)
	}()
	t.Cleanup(func() { stop(); <-done })
	base.attach(r.worker, "worker")
	base.waitConnected(r.worker, true)
	r.coordinator = newSupervisionCoordinator(base.s, func(string) {})
	base.s.supervision.Store(r.coordinator)
	r.awaitReport(t, report)
	return r
}
func (r *workerCueRig) awaitReport(t *testing.T, want *couchmessage.WorkerCueReport) {
	t.Helper()
	supervisionEventually(t, "worker cue report missing from registry", func() bool {
		live, ok := r.s.LivenessForThread(r.worker.Scope, r.worker.Tag)
		if !ok {
			return false
		}
		if want == nil {
			return live.WorkerCue == nil
		}
		return live.WorkerCue != nil && *live.WorkerCue == *want
	})
}
func (r *workerCueRig) tick() { r.coordinator.tick(context.Background(), time.Now(), nil) }
func (r *workerCueRig) awaitCounts(t *testing.T, cue, resend int) {
	t.Helper()
	supervisionEventually(t, "worker cue/resend counts missing", func() bool { a, b := r.endpoint.counts(); return a >= cue && b >= resend })
	supervisionEventually(t, "worker delivery did not settle", func() bool { pending, known := r.s.broker.ResourceIdle(r.worker); return known && !pending })
	a, b := r.endpoint.counts()
	if a != cue || b != resend {
		t.Fatalf("cue=%d resend=%d want %d/%d", a, b, cue, resend)
	}
}

func TestWorkerCueCoordinatorOncePerLaunchAndSeparateResendLifetime(t *testing.T) {
	report := &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded}
	r := newWorkerCueRig(t, report)
	setTestAutomation(t, r.s, "/repo1", "off") // Worker cues and resend stay independent.
	r.tick()
	r.awaitCounts(t, 1, 1)
	for i := 0; i < 4; i++ {
		r.tick()
	}
	r.awaitCounts(t, 1, 1)
	if r.proofs.Load() != 1 {
		t.Fatalf("repeated child proof: %d", r.proofs.Load())
	}
	// An unchanged needed report cannot cause a second attempt in this supervisor.
	r.ordinal.Store(8)
	next := &couchmessage.WorkerCueReport{LaunchOrdinal: 8, State: couchmessage.WorkerCueNeeded}
	r.client.WorkerCue(next)
	r.awaitReport(t, next)
	r.tick()
	r.awaitCounts(t, 2, 1)
	if r.proofs.Load() != 2 {
		t.Fatal("new launch did not get one new proof")
	}
	// A stale older ordinal must not evict the latest attempt and make the
	// current launch eligible again when its report returns.
	r.client.WorkerCue(report)
	r.awaitReport(t, report)
	r.tick()
	r.client.WorkerCue(next)
	r.awaitReport(t, next)
	r.tick()
	if r.proofs.Load() != 2 {
		t.Fatal("stale ordinal reopened a previously attempted launch")
	}
	r.awaitCounts(t, 2, 1)

	// Wrapper witness survives a coordinator replacement, while the independent
	// Couch-restart resend policy has a new supervisor lifetime.
	submitted := &couchmessage.WorkerCueReport{LaunchOrdinal: 8, State: couchmessage.WorkerCueSubmitted, ID: couchmessage.WorkerStartNoticeID(r.worker, 8)}
	r.client.WorkerCue(submitted)
	r.awaitReport(t, submitted)
	r.coordinator = newSupervisionCoordinator(r.s, func(string) {})
	r.s.supervision.Store(r.coordinator)
	r.tick()
	r.awaitCounts(t, 2, 2)
}

func TestWorkerCueCoordinatorUnknownAndUncertainNeverRetry(t *testing.T) {
	for _, state := range []couchmessage.WorkerCueState{"unknown", couchmessage.WorkerCueCommitted, couchmessage.WorkerCueSubmitted, couchmessage.WorkerCueUncertain} {
		t.Run(string(state), func(t *testing.T) {
			var report *couchmessage.WorkerCueReport
			r := newWorkerCueRig(t, nil)
			if state != "unknown" {
				report = &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: state, ID: couchmessage.WorkerStartNoticeID(r.worker, 7)}
				r.client.WorkerCue(report)
				r.awaitReport(t, report)
			}
			r.tick()
			r.awaitCounts(t, 0, 1)
			if r.proofs.Load() != 0 {
				t.Fatal("non-needed cue cost a child proof")
			}
		})
	}
	r := newWorkerCueRig(t, &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded})
	r.endpoint.mu.Lock()
	r.endpoint.uncertain = true
	r.endpoint.mu.Unlock()
	r.tick()
	r.awaitCounts(t, 1, 1)
	for i := 0; i < 3; i++ {
		r.tick()
	}
	r.awaitCounts(t, 1, 1)
	if r.proofs.Load() != 1 {
		t.Fatal("uncertain admission retried child proof")
	}
}

func TestWorkerCueCoordinatorDefersWithoutOwnerAndRequiresCurrentChild(t *testing.T) {
	r := newWorkerCueRig(t, &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded})
	r.stopHub()
	r.waitConnected(r.hub, false)
	r.tick()
	if a, b := r.endpoint.counts(); a != 0 || b != 0 || r.proofs.Load() != 0 {
		t.Fatalf("missing owner admitted/probed cue: %d/%d proofs=%d", a, b, r.proofs.Load())
	}
	r.wrapper(r.hub)
	r.waitConnected(r.hub, true)
	r.ordinal.Store(8) // Stale launch evidence never authorizes current input.
	r.tick()
	r.awaitCounts(t, 0, 1)
	for i := 0; i < 3; i++ {
		r.tick()
	}
	if r.proofs.Load() != 1 {
		t.Fatal("failed child proof retried every tick")
	}
}

func TestWorkerCueCoordinatorFencesRoleChangeDuringChildProof(t *testing.T) {
	r := newWorkerCueRig(t, &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded})
	r.s.authority.child = func(context.Context, couchmessage.Binding) (uint64, string, error) {
		r.proofs.Add(1)
		r.s.mu.Lock()
		identity := r.s.workspaces[r.worker]
		identity.WorktreeRoot = "/repo1"
		r.s.workspaces[r.worker] = identity
		r.s.mu.Unlock()
		return 7, "native-child", nil
	}
	r.tick()
	// Role changes fence both the generic cue and the separate startup resend.
	r.coordinator.mu.Lock()
	attempts := len(r.coordinator.sends)
	r.coordinator.mu.Unlock()
	if attempts != 0 {
		t.Fatal("a worker that became a hub received startup input")
	}
	if a, b := r.endpoint.counts(); a != 0 || b != 0 {
		t.Fatalf("role change admitted cue/resend: %d/%d", a, b)
	}
}

func TestWorkerCueCoordinatorFencesCommittedReportDuringChildProof(t *testing.T) {
	r := newWorkerCueRig(t, &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded})
	r.s.authority.child = func(context.Context, couchmessage.Binding) (uint64, string, error) {
		r.proofs.Add(1)
		committed := &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueCommitted, ID: couchmessage.WorkerStartNoticeID(r.worker, 7)}
		r.client.WorkerCue(committed)
		r.awaitReport(t, committed)
		return 7, "native-child", nil
	}
	r.tick()
	r.awaitCounts(t, 0, 1)
	if r.proofs.Load() != 1 {
		t.Fatal("changed report reprobed")
	}
}

func TestWorkerCueCoordinatorBoundsReadyRecordProof(t *testing.T) {
	r := newWorkerCueRig(t, &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded})
	bounded := false
	r.s.authority.child = func(ctx context.Context, _ couchmessage.Binding) (uint64, string, error) {
		_, bounded = ctx.Deadline()
		return 0, "", errors.New("controlled unavailable proof")
	}
	r.tick()
	if !bounded {
		t.Fatal("ready-record proof has no admission deadline")
	}
}

func TestWorkerCueCoordinatorCountedTwentyFortyWorkers(t *testing.T) {
	for _, n := range []int{20, 40} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			r := newServiceRig(t)
			r.world.hub = "/repo1"
			r.restart(nil)
			hub := r.world.add(1)
			connected := map[couchmessage.Binding]bool{hub: true}
			lives := map[string]couchmessage.SlotLiveness{hub.Slot: {Binding: hub}}
			endpoints := map[couchmessage.Binding]*workerCueEndpoint{}
			for i := 2; i <= n+1; i++ {
				b := r.world.add(i)
				e := &workerCueEndpoint{}
				r.world.endpoints[b] = e
				endpoints[b] = e
				connected[b] = true
				lives[b.Slot] = couchmessage.SlotLiveness{Binding: b, WorkerCue: &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded}}
			}
			// Populate already-admitted registry/actor memory directly; private socket
			// tests above separately qualify authentication, publication and delivery.
			for b := range connected {
				r.world.mu.Lock()
				identity := r.world.slots[b.Thread()].workspace
				r.world.mu.Unlock()
				r.s.mu.Lock()
				r.s.workspaces[b] = identity
				r.s.mu.Unlock()
				if err := r.s.broker.Register(b, r.s.authority.endpoint(b)); err != nil {
					t.Fatal(err)
				}
			}
			r.s.connected.Store(&connected)
			r.s.liveness.Store(&lives)
			var proofs atomic.Int32
			var ordinal atomic.Uint64
			ordinal.Store(7)
			r.s.authority.child = func(context.Context, couchmessage.Binding) (uint64, string, error) {
				proofs.Add(1)
				return ordinal.Load(), "native-child", nil
			}
			c := newSupervisionCoordinator(r.s, func(string) {})
			r.s.supervision.Store(c)
			for i := 0; i < 5; i++ {
				c.tick(context.Background(), time.Now(), nil)
			}
			supervisionEventually(t, "counted fleet deliveries did not settle", func() bool {
				for b := range endpoints {
					pending, known := r.s.broker.ResourceIdle(b)
					if !known || pending {
						return false
					}
				}
				return true
			})
			if got := proofs.Load(); got != int32(n) {
				t.Fatalf("%d workers cost %d child proofs", n, got)
			}
			for b, e := range endpoints {
				if cue, resend := e.counts(); cue != 1 || resend != 1 {
					t.Fatalf("%s cue/resend=%d/%d", b.Slot, cue, resend)
				}
			}
			before := r.world.counts()
			for i := 0; i < 5; i++ {
				c.tick(context.Background(), time.Now(), nil)
			}
			if proofs.Load() != int32(n) || before != r.world.counts() {
				t.Fatal("unchanged ticks repeated child/ready/process/git work")
			}
			next := make(map[string]couchmessage.SlotLiveness, len(lives))
			for slot, live := range lives {
				next[slot] = live
			}
			changed := next["pair:2"]
			changed.WorkerCue = &couchmessage.WorkerCueReport{LaunchOrdinal: 8, State: couchmessage.WorkerCueNeeded}
			next["pair:2"] = changed
			ordinal.Store(8)
			r.s.liveness.Store(&next)
			c.tick(context.Background(), time.Now(), nil)
			supervisionEventually(t, "new launch did not get one new opportunity", func() bool { cue, _ := endpoints[changed.Binding].counts(); return cue == 2 })
			for i := 0; i < 5; i++ {
				c.tick(context.Background(), time.Now(), nil)
			}
			if proofs.Load() != int32(n+1) {
				t.Fatalf("new launch proof count=%d", proofs.Load())
			}
		})
	}
}

func TestWorkerCueCoordinatorCapacityVisibleAndWrapperWitnessReclaimsSlot(t *testing.T) {
	r := newWorkerCueRig(t, &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueNeeded})
	c := r.coordinator
	var notices []string
	c.notice = func(text string) { notices = append(notices, text) }
	c.mu.Lock()
	c.workerCues = map[string]workerCueAttempt{}
	for i := 0; i < couchmessage.MaxActors; i++ {
		c.workerCues[fmt.Sprintf("historical:%d", i)] = workerCueAttempt{ordinal: 1}
	}
	c.mu.Unlock()
	live, _ := r.s.LivenessForThread(r.worker.Scope, r.worker.Tag)
	for i := 0; i < 3; i++ {
		c.cueWorker(context.Background(), live)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "tracking is full") || r.proofs.Load() != 0 {
		t.Fatalf("silent/unbounded capacity failure: proofs=%d notices=%v", r.proofs.Load(), notices)
	}
	// A positive current wrapper witness is sufficient to release supervisor
	// dedup bookkeeping, while attempted-but-still-needed launches remain owned.
	c.mu.Lock()
	delete(c.workerCues, "historical:0")
	c.workerCues[r.worker.Slot] = workerCueAttempt{binding: r.worker, ordinal: 7}
	c.mu.Unlock()
	committed := &couchmessage.WorkerCueReport{LaunchOrdinal: 7, State: couchmessage.WorkerCueCommitted, ID: couchmessage.WorkerStartNoticeID(r.worker, 7)}
	r.client.WorkerCue(committed)
	r.awaitReport(t, committed)
	live, _ = r.s.LivenessForThread(r.worker.Scope, r.worker.Tag)
	c.cueWorker(context.Background(), live)
	c.mu.Lock()
	size := len(c.workerCues)
	c.mu.Unlock()
	if size != couchmessage.MaxActors-1 {
		t.Fatal("wrapper witness did not reclaim attempt slot")
	}
	r.ordinal.Store(8)
	next := &couchmessage.WorkerCueReport{LaunchOrdinal: 8, State: couchmessage.WorkerCueNeeded}
	r.client.WorkerCue(next)
	r.awaitReport(t, next)
	live, _ = r.s.LivenessForThread(r.worker.Scope, r.worker.Tag)
	c.cueWorker(context.Background(), live)
	if r.proofs.Load() != 1 {
		t.Fatal("reclaimed capacity did not admit one new proof")
	}
}
