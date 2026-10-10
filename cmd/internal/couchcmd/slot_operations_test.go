package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// fakeSlotRunner stands for the console queue: like operationQueue it
// refuses a key already pending (ErrRemotePending) until that job finishes,
// records every job and keeps its outcome hooks for the test to fire.
type fakeSlotRunner struct {
	mu      sync.Mutex
	jobs    []fakeSlotJob
	pending map[string]bool
	err     error // what the next enqueue returns, overriding the model
}

type fakeSlotJob struct {
	key, op, target string
	opts            couchcore.LiveRestartOptions
	started         func()
	finished        func(any, error)
}

func (f *fakeSlotRunner) run(key, op, target string, opts couchcore.LiveRestartOptions, started func(), finished func(any, error)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if f.pending == nil {
		f.pending = map[string]bool{}
	}
	if f.pending[key] {
		return couchtty.ErrRemotePending
	}
	f.pending[key] = true
	done := func(value any, err error) {
		f.mu.Lock()
		delete(f.pending, key)
		f.mu.Unlock()
		finished(value, err)
	}
	f.jobs = append(f.jobs, fakeSlotJob{key, op, target, opts, started, done})
	return nil
}

func (f *fakeSlotRunner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.jobs)
}

func (f *fakeSlotRunner) job(i int) fakeSlotJob {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jobs[i]
}

// slotRig is a serviceRig whose slot operations run on a fakeSlotRunner, over
// the enrolled repository "pair" (alias "pa").
func slotRig(t *testing.T) (*serviceRig, *fakeSlotRunner, *fakeClock) {
	t.Helper()
	runner, clock := &fakeSlotRunner{}, &fakeClock{now: time.Unix(10000, 0)}
	r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
	r.slotOps = newSlotOperations(runner.run, func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/src/pair", Dir: "pair", Alias: "pa"}, {Key: "/src/brain", Dir: "brain"}}, nil
	}, clock.Now)
	r.init()
	return r, runner, clock
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func slotRequest(b couchmessage.Binding, op, id, target string) couchmessage.Request {
	r := callerRequest(b, op)
	r.ID, r.Target = id, target
	if op == "reboot" {
		r.Confirmed = true
	}
	return r
}

func statusRequest(b couchmessage.Binding, id string) couchmessage.Request {
	r := callerRequest(b, "operation-status")
	r.ID = id
	return r
}

// TestSlotOperationCallerRule: only a live Couch slot may run a slot
// operation, judged by Couch's own records, not by messaging registration
// (operator decision after the #367 smoke test): the thread named by the
// request's scope and tag has a live Couch pane, and its recorded launch
// names this shell's session and launch nonce. One strategy per fault; every
// refusal enqueues nothing and says the caller is not a live Couch slot.
func TestSlotOperationCallerRule(t *testing.T) {
	for _, c := range []struct {
		name  string
		code  string
		fault func(*serviceRig, couchmessage.Binding) couchmessage.Request
	}{
		// Without the identity fields the request is not a caller request at
		// all: ValidateRequest refuses it before the caller check.
		{"no identity", "invalid-request", func(_ *serviceRig, b couchmessage.Binding) couchmessage.Request {
			return couchmessage.Request{Op: "resume", ID: "id", Target: "pair:1"}
		}},
		{"wrong launch nonce", "unavailable", func(_ *serviceRig, b couchmessage.Binding) couchmessage.Request {
			r := slotRequest(b, "resume", "id", "pair:1")
			r.Nonce = "forged"
			return r
		}},
		{"wrong session", "unavailable", func(_ *serviceRig, b couchmessage.Binding) couchmessage.Request {
			r := slotRequest(b, "resume", "id", "pair:1")
			r.Session = "another-session"
			return r
		}},
		{"thread not live", "unavailable", func(r *serviceRig, b couchmessage.Binding) couchmessage.Request {
			r.world.set(b, func(s *worldSlot) { s.live = false })
			return slotRequest(b, "resume", "id", "pair:1")
		}},
		{"recorded launch moved on", "unavailable", func(r *serviceRig, b couchmessage.Binding) couchmessage.Request {
			r.world.set(b, func(s *worldSlot) { s.recorded = false })
			return slotRequest(b, "resume", "id", "pair:1")
		}},
		{"unknown tag", "unavailable", func(_ *serviceRig, b couchmessage.Binding) couchmessage.Request {
			r := slotRequest(b, "resume", "id", "pair:1")
			r.Tag = "no-such-thread"
			return r
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, runner, _ := slotRig(t)
			b := r.world.add(0)
			resp := r.s.handle(context.Background(), c.fault(r, b))
			if resp.Code != c.code || resp.Operation != nil {
				t.Fatalf("response %+v, want %s", resp, c.code)
			}
			if c.code == "unavailable" && (!strings.HasPrefix(resp.Error, "caller is not a live Couch slot") || strings.Contains(resp.Error, "recipient")) {
				t.Fatalf("refusal text %q", resp.Error)
			}
			if runner.count() != 0 {
				t.Fatal("a refused caller enqueued a job")
			}
		})
	}
}

// A live slot whose wrapper never registered for messaging (its one-shot
// peer setup failed) is still a live Couch slot and may call (smoke test,
// pair:0 under codex).
func TestSlotOperationCallerNeedsNoMessagingRegistration(t *testing.T) {
	r, runner, _ := slotRig(t)
	b := r.world.add(0) // live pane and recorded launch, no wrapper session
	if r.s.isConnected(b) {
		t.Fatal("fixture registered the caller for messaging")
	}
	resp := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1"))
	if resp.Code != "accepted" || resp.Operation == nil || resp.Operation.Status != couchmessage.ReceiptQueued || runner.count() != 1 {
		t.Fatalf("unregistered live caller: %+v, %d jobs", resp, runner.count())
	}
	if job := runner.job(0); job.op != "resume" || job.target != "pair:1" {
		t.Fatalf("job = %+v", job)
	}
	// A connected messaging caller is still a live slot.
	connected := r.connect(2)
	if resp := r.s.handle(context.Background(), slotRequest(connected, "resume", "id", "pair:2")); resp.Code != "accepted" {
		t.Fatalf("connected caller: %+v", resp)
	}
}

func TestSlotOperationRebootNeedsConfirmation(t *testing.T) {
	r, runner, _ := slotRig(t)
	b := r.connect(0)
	request := slotRequest(b, "reboot", "id", "pair:1")
	request.Confirmed = false
	if resp := r.s.handle(context.Background(), request); resp.Code != "confirmation-required" || runner.count() != 0 {
		t.Fatalf("unconfirmed reboot: %+v, %d jobs", resp, runner.count())
	}
	request.Confirmed = true
	if resp := r.s.handle(context.Background(), request); resp.Code != "accepted" || runner.count() != 1 {
		t.Fatalf("confirmed reboot: %+v", resp)
	}
}

// TestSlotOperationTypedOutcomes: every outcome reaches the caller as a typed
// receipt or response code.
func TestSlotOperationTypedOutcomes(t *testing.T) {
	finishWith := func(value any, err error) func(*testing.T) couchmessage.OperationReceipt {
		return func(t *testing.T) couchmessage.OperationReceipt {
			r, runner, _ := slotRig(t)
			b := r.connect(0)
			if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1")); resp.Code != "accepted" {
				t.Fatalf("admit %+v", resp)
			}
			job := runner.job(0)
			if got := r.s.handle(context.Background(), statusRequest(b, "id")); got.Operation == nil || got.Operation.Status != couchmessage.ReceiptQueued {
				t.Fatalf("queued status %+v", got)
			}
			job.started()
			if got := r.s.handle(context.Background(), statusRequest(b, "id")); got.Operation.Status != couchmessage.ReceiptRunning {
				t.Fatalf("running status %+v", got)
			}
			job.finished(value, err)
			got := r.s.handle(context.Background(), statusRequest(b, "id"))
			if got.Code != "ok" || got.Operation == nil {
				t.Fatalf("status %+v", got)
			}
			return *got.Operation
		}
	}
	t.Run("not-offered", func(t *testing.T) {
		got := finishWith(nil, &couchcore.SlotOperationError{Code: couchcore.SlotOpNotOffered, Detail: "pair:1 is live"})(t)
		if got.Status != couchmessage.ReceiptRefused || got.Code != couchcore.SlotOpNotOffered || got.Detail != "pair:1 is live" {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("no-thread", func(t *testing.T) {
		got := finishWith(nil, &couchcore.SlotOperationError{Code: couchcore.SlotOpNoThread, Detail: "none"})(t)
		if got.Status != couchmessage.ReceiptRefused || got.Code != couchcore.SlotOpNoThread {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("failed with diagnostic", func(t *testing.T) {
		got := finishWith(nil, &couchcore.ResumeRefusal{Code: couchcore.ResumeNotDetached, Diagnostic: "not detached"})(t)
		if got.Status != couchmessage.ReceiptFailed || got.Diagnostic != string(couchcore.ResumeNotDetached) || got.Detail == "" {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("succeeded with tag", func(t *testing.T) {
		started := couchcore.StartResult{Record: couchcore.ActorRecord{Thread: couchcore.ThreadAddress{RepoScope: "scope", Tag: "fresh"}}}
		got := finishWith(couchcore.RebootResult{Start: started, Archived: couchcore.ThreadAddress{RepoScope: "scope", Tag: "old"}}, nil)(t)
		if got.Status != couchmessage.ReceiptSucceeded || got.Tag != "fresh" || got.Archived != "old" {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("busy", func(t *testing.T) {
		r, runner, _ := slotRig(t)
		b := r.connect(0)
		runner.err = couchtty.ErrRemotePending
		if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1")); resp.Code != "busy" {
			t.Fatalf("%+v", resp)
		}
		// The reservation is dropped: the same ID admits once the slot frees.
		runner.err = nil
		if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1")); resp.Code != "accepted" {
			t.Fatalf("after busy %+v", resp)
		}
	})
	t.Run("overloaded queue", func(t *testing.T) {
		r, runner, _ := slotRig(t)
		b := r.connect(0)
		runner.err = couchtty.ErrOperationQueueOverloaded
		if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1")); resp.Code != "overloaded" {
			t.Fatalf("%+v", resp)
		}
		if got := r.s.handle(context.Background(), statusRequest(b, "id")); got.Operation == nil || got.Operation.Status != couchmessage.ReceiptUnknown {
			t.Fatalf("refused admission left a receipt: %+v", got)
		}
	})
	t.Run("id-conflict", func(t *testing.T) {
		r, runner, _ := slotRig(t)
		b := r.connect(0)
		r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1"))
		if resp := r.s.handle(context.Background(), slotRequest(b, "reboot", "id", "pair:1")); resp.Code != "id-conflict" || runner.count() != 1 {
			t.Fatalf("%+v, %d jobs", resp, runner.count())
		}
	})
	t.Run("duplicate admit", func(t *testing.T) {
		r, runner, _ := slotRig(t)
		b := r.connect(0)
		first := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1"))
		second := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "pair:1"))
		if second.Code != "accepted" || second.Operation == nil || *second.Operation != *first.Operation || runner.count() != 1 {
			t.Fatalf("duplicate %+v, %d jobs", second, runner.count())
		}
	})
	t.Run("unknown repository", func(t *testing.T) {
		r, runner, _ := slotRig(t)
		b := r.connect(0)
		if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "id", "nosuch:1")); resp.Code != couchcore.SlotOpUnknownSlot || runner.count() != 0 {
			t.Fatalf("%+v", resp)
		}
	})
}

func TestOperationStatusIsCallerScoped(t *testing.T) {
	r, _, _ := slotRig(t)
	a, other := r.connect(0), r.connect(2)
	r.s.handle(context.Background(), slotRequest(a, "resume", "id", "pair:1"))
	if got := r.s.handle(context.Background(), statusRequest(other, "id")); got.Operation == nil || got.Operation.Status != couchmessage.ReceiptUnknown {
		t.Fatalf("another slot read the receipt: %+v", got)
	}
	if got := r.s.handle(context.Background(), statusRequest(a, "id")); got.Operation.Status != couchmessage.ReceiptQueued {
		t.Fatalf("own receipt: %+v", got)
	}
}

func TestReceiptsExpireAndCap(t *testing.T) {
	r, runner, clock := slotRig(t)
	b := r.connect(0)
	r.s.handle(context.Background(), slotRequest(b, "resume", "done", "pair:1"))
	job := runner.job(0)
	job.started()
	job.finished(nil, &couchcore.SlotOperationError{Code: couchcore.SlotOpNotOffered, Detail: "live"})
	clock.Add(couchmessage.ReceiptRetention - time.Second)
	if got := r.s.handle(context.Background(), statusRequest(b, "done")); got.Operation.Status != couchmessage.ReceiptRefused {
		t.Fatalf("expired early: %+v", got)
	}
	clock.Add(time.Second)
	if got := r.s.handle(context.Background(), statusRequest(b, "done")); got.Operation.Status != couchmessage.ReceiptUnknown {
		t.Fatalf("kept past retention: %+v", got)
	}
	for i := range maxSlotOperationReceipts {
		if resp := r.s.handle(context.Background(), slotRequest(b, "resume", fmt.Sprintf("id-%d", i), fmt.Sprintf("pair:%d", i+1))); resp.Code != "accepted" {
			t.Fatalf("admission %d: %+v", i, resp)
		}
	}
	before := runner.count()
	if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "one-more", "pair:99")); resp.Code != "overloaded" || runner.count() != before {
		t.Fatalf("65th admission: %+v", resp)
	}
}

// TestSlotOperationReceiptsConcurrent: admit, status, started and finished
// from separate goroutines (socket handlers, queue runner, console loop).
// Run under -race; every receipt ends terminal exactly once.
func TestSlotOperationReceiptsConcurrent(t *testing.T) {
	r, runner, _ := slotRig(t)
	b := r.connect(0)
	const n = 32
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("id-%d", i)
			if resp := r.s.handle(context.Background(), slotRequest(b, "resume", id, fmt.Sprintf("pair:%d", i+1))); resp.Code != "accepted" {
				t.Errorf("%s: %+v", id, resp)
			}
			for range 5 {
				r.s.handle(context.Background(), statusRequest(b, id))
			}
		}()
	}
	wg.Wait()
	if runner.count() != n {
		t.Fatalf("%d jobs, want %d", runner.count(), n)
	}
	for i := range n {
		job := runner.job(i)
		wg.Add(2)
		go func() { defer wg.Done(); job.started() }()
		go func() { defer wg.Done(); job.finished(couchcore.StartResult{}, nil) }()
	}
	wg.Wait()
	for i := range n {
		got := r.s.handle(context.Background(), statusRequest(b, fmt.Sprintf("id-%d", i)))
		if got.Operation == nil || got.Operation.Status != couchmessage.ReceiptSucceeded {
			t.Errorf("id-%d: %+v", i, got)
		}
	}
	// A second finish is no second terminal transition.
	runner.job(0).finished(nil, errors.New("late"))
	if got := r.s.handle(context.Background(), statusRequest(b, "id-0")); got.Operation.Status != couchmessage.ReceiptSucceeded {
		t.Fatalf("a second finish changed the receipt: %+v", got)
	}
}

// The queue key comes from the resolved slot: a repository named by its
// alias shares the pending key with its directory name, so pa:1 while
// pair:1 is pending is busy.
func TestSlotOperationAliasSharesKey(t *testing.T) {
	r, runner, _ := slotRig(t)
	b := r.connect(0)
	if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "first", "pair:1")); resp.Code != "accepted" {
		t.Fatalf("first %+v", resp)
	}
	if resp := r.s.handle(context.Background(), slotRequest(b, "resume", "second", "pa:1")); resp.Code != "busy" || runner.count() != 1 {
		t.Fatalf("alias while pending: %+v, %d jobs", resp, runner.count())
	}
	if key := runner.job(0).key; key != "remote\x00/src/pair:1" {
		t.Fatalf("key %q", key)
	}
}

// recover never asks (#399, 2026-10-07): the socket admits it without
// --confirm, and a held row's refusal reaches the caller typed, naming the hold.
func TestSlotOperationRecoverNeedsNoConfirmation(t *testing.T) {
	r, runner, _ := slotRig(t)
	b := r.connect(0)
	if resp := r.s.handle(context.Background(), slotRequest(b, "recover", "id0", "pair:1")); resp.Code != "accepted" {
		t.Fatalf("recover: %+v", resp)
	}
	if job := runner.job(0); job.op != "recover" || job.target != "pair:1" {
		t.Fatalf("job = %+v", job)
	}
	outcome := slotOperationOutcome(nil, &couchcore.RecoverRefusal{Code: couchcore.RecoverHeld, Detail: "conflict:claim-elsewhere: claimed in pair:3"})
	if outcome.Status != couchmessage.ReceiptRefused || outcome.Code != couchcore.RecoverHeld || !strings.Contains(outcome.Detail, "claim-elsewhere") {
		t.Fatalf("outcome = %+v", outcome)
	}
}

// pair#421: relaunch's overrides travel from the request to the queued job.
func TestSlotOperationRelaunchCarriesOverrides(t *testing.T) {
	r, runner, _ := slotRig(t)
	b := r.world.add(0)
	req := slotRequest(b, "relaunch", "id", "pair:1")
	req.Confirmed, req.SameBinary, req.ForceUnknown = true, true, true
	if resp := r.s.handle(context.Background(), req); resp.Code != "accepted" {
		t.Fatalf("relaunch: %+v", resp)
	}
	if job := runner.job(0); job.op != "relaunch" || !job.opts.SameBinary || !job.opts.ForceUnknown {
		t.Fatalf("job = %+v", job)
	}
	unconfirmed := slotRequest(b, "relaunch", "id2", "pair:1")
	if resp := r.s.handle(context.Background(), unconfirmed); resp.Code != "confirmation-required" {
		t.Fatalf("unconfirmed relaunch: %+v", resp)
	}
}
