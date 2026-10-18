package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestRelaunchBeforeBeginFailureDoesNotInventAnOpenPark(t *testing.T) {
	env, live := envWithLiveThread(t)
	env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
		if old.Park == nil && next.Park != nil {
			return errors.New("begin refused")
		}
		return nil
	}
	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if result.Outcome != RefusedBeforePark || err == nil {
		t.Fatalf("result=%+v err=%v; want unchanged refusal", result, err)
	}
	state, getErr := env.Couch.Threads.GetThread(live.Address)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if state.Park != nil || state.Revision != live.Revision || env.Proc.Exists(42) != Live {
		t.Fatalf("unexpected effects: %+v", state)
	}
	for _, lie := range []string{"Pair was stopped", "transaction is still open", "park's own modes"} {
		if strings.Contains(err.Error(), lie) {
			t.Fatalf("error invents effects: %v", err)
		}
	}
}

func TestRelaunchFinalizedJournalWithLostReplyReportsParked(t *testing.T) {
	env, live := envWithLiveThread(t)
	armed := false
	env.Couch.Threads.hooks.AfterJournal = func() error {
		if armed {
			return errors.New("lost finalization reply")
		}
		return nil
	}
	env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
		if old.Park != nil && next.Park == nil && next.VerifiedPark != nil {
			armed = true
		}
		return nil
	}
	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if result.Outcome != ParkedNotResumed || err == nil {
		t.Fatalf("result=%+v err=%v; want durable parked outcome", result, err)
	}
	state, getErr := env.Couch.Threads.GetThread(live.Address)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if state.Park != nil || state.VerifiedPark == nil || len(state.Incarnations) != 0 {
		t.Fatalf("not finalized: %+v", state)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("errored park must not resume: %v", env.Runner.Ops)
	}
	if !strings.Contains(err.Error(), "parked") || strings.Contains(err.Error(), "transaction is still open") {
		t.Fatalf("misleading guidance: %v", err)
	}
}

func TestRelaunchCanceledWaiterDoesNotClaimActiveWorkSettled(t *testing.T) {
	env, live := envWithLiveThread(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
		close(entered)
		<-release
		return errors.New("trigger uncertain")
	}
	type response struct {
		result RelaunchResult
		err    error
	}
	replies := make(chan response, 1)
	go func() { result, err := env.Couch.Relaunch(ctx, live.Address); replies <- response{result, err} }()
	<-entered
	w := env.Couch.PairLifecycle.worker
	w.mu.Lock()
	future := w.active[live.Address].future
	w.mu.Unlock()
	defer func() {
		close(release)
		wait, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_, _ = future.Await(wait)
	}()
	cancel()
	reply := <-replies
	if string(reply.result.Outcome) != "park-in-progress" || reply.err == nil {
		t.Fatalf("result=%+v err=%v; active work must remain explicit", reply.result, reply.err)
	}
	if strings.Contains(reply.err.Error(), "Pair was stopped") || strings.Contains(reply.err.Error(), "retry    ") {
		t.Fatalf("settled claim while worker active: %v", reply.err)
	}
}

func TestClassifyRelaunchParkOutcomeUsesIndependentDurableAndActiveFacts(t *testing.T) {
	original := ThreadRecord{Revision: 7, Incarnations: []ThreadIncarnation{{PID: 42, Identity: "original", State: IncarnationLive}}}
	identity := ParkIdentity{Nonce: "new-park", PID: 42, ProcessIdentity: "original"}
	open := original
	open.Revision++
	open.Park = &ParkTransaction{Identity: identity, BaseRevision: 7, Phase: ParkRequested}
	finalized := original
	finalized.Revision += 3
	finalized.Incarnations = nil
	finalized.VerifiedPark = &VerifiedPark{Identity: identity, Attempt: 1}
	rollback := original
	rollback.Revision += 2
	rollback.ParkHistory = []ParkTransaction{{Identity: identity, BaseRevision: 7, Closed: true, Tombstoned: true}}
	unrelated := original
	unrelated.Revision++
	staleReceipt := original
	staleReceipt.VerifiedPark = &VerifiedPark{Identity: ParkIdentity{Nonce: "old"}, Attempt: 1}
	for _, tc := range []struct {
		name    string
		current ThreadRecord
		want    string
	}{
		{"unchanged", original, "refused-before-park"},
		{"open", open, "park-incomplete"},
		{"finalized", finalized, "park-ok-resume-failed"},
		{"rollback", rollback, "park-rolled-back"},
		{"unrelated mutation", unrelated, "park-state-unknown"},
		{"old verified receipt beside live owner", staleReceipt, "park-state-unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(ClassifyRelaunchParkOutcome(original, tc.current, false)); got != tc.want {
				t.Fatalf("settled outcome=%s want %s", got, tc.want)
			}
			if got := string(ClassifyRelaunchParkOutcome(original, tc.current, true)); got != "park-in-progress" {
				t.Fatalf("active outcome=%s; cannot claim settled state", got)
			}
		})
	}
}

func TestRelaunchUnreadableOutcomeDoesNotInventOpenOrCompletedPark(t *testing.T) {
	env, live := envWithLiveThread(t)
	env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
		if old.Park == nil && next.Park != nil {
			env.Couch.Threads.hooks.AfterGetThread = func(ThreadAddress) error { return errors.New("outcome observation failed") }
			return errors.New("begin refused")
		}
		return nil
	}
	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if string(result.Outcome) != "park-state-unknown" || err == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !strings.Contains(err.Error(), "outcome observation failed") || !strings.Contains(err.Error(), env.Couch.ParkRecoveryAdvice(context.Background(), live.Address)) {
		t.Fatalf("missing uncertainty/recovery: %v", err)
	}
	for _, lie := range []string{"Pair was stopped", "transaction is still open", "durably complete", "thread is unchanged"} {
		if strings.Contains(err.Error(), lie) {
			t.Fatalf("unreadable state became a claim: %v", err)
		}
	}
}
