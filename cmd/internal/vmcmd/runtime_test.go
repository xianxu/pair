package vmcmd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	f := newFakeRuntime()
	s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
	if !s.Done() || s.ExitCode() != 0 {
		t.Fatalf("result: %+v", s)
	}
	want := []Effect{Acquire, Prepare, Start, AwaitGuest, Execute, Collect, Stop, Release}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("calls %v", f.calls)
	}
	if f.acquired || f.prepared || f.running || f.child || !f.collected || !f.released {
		t.Fatalf("resources: %+v", f)
	}
}

func TestRunEveryEffectFailure(t *testing.T) {
	for _, stage := range []Effect{Acquire, Prepare, Start, AwaitGuest, Execute, Collect, Stop, Release} {
		t.Run(string(stage), func(t *testing.T) {
			f := newFakeRuntime()
			f.fail[stage] = errors.New("injected " + string(stage))
			s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
			if !s.Done() || s.ExitCode() == 0 || !strings.Contains(s.Detail(), "injected "+string(stage)) {
				t.Fatalf("result: %+v", s)
			}
			counts := map[Effect]int{}
			for _, e := range f.calls {
				counts[e]++
			}
			for e, n := range counts {
				if n != 1 {
					t.Fatalf("effect %s ran %d times", e, n)
				}
			}
			if stage != Acquire && counts[Release] != 1 {
				t.Fatalf("missing cleanup: %v", f.calls)
			}
			if stage == Acquire && len(f.calls) != 1 {
				t.Fatalf("cleanup without ownership: %v", f.calls)
			}
		})
	}
}

func waitForEffect(t *testing.T, f *fakeRuntime, stage Effect) {
	t.Helper()
	for {
		select {
		case e := <-f.entered:
			if e == stage {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("did not enter %s", stage)
		}
	}
}
func resultOfRun(t *testing.T, ch <-chan RunState) RunState {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(time.Second):
		t.Fatal("Run did not finish")
		return RunState{}
	}
}

func TestRunCancellationAtBarrier(t *testing.T) {
	for _, stage := range []Effect{Acquire, Prepare, Start, AwaitGuest, Execute, Collect, Stop, Release} {
		t.Run(string(stage), func(t *testing.T) {
			f := newFakeRuntime()
			gate := make(chan struct{})
			f.gates[stage] = gate
			f.ignoreCancellation = true
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan RunState, 1)
			go func() { done <- Run(ctx, RunRequest{Timeout: time.Second}, f) }()
			waitForEffect(t, f, stage)
			cancel()
			select {
			case s := <-done:
				t.Fatalf("returned before effect joined: %+v", s)
			default:
			}
			if stage == Stop {
				if !f.acquired || !f.prepared || !f.child || !f.running {
					t.Fatal("resources lost while Stop is blocked")
				}
				select {
				case e := <-f.entered:
					t.Fatalf("effect %s started before Stop joined", e)
				default:
				}
			}
			close(gate)
			s := resultOfRun(t, done)
			if !s.Done() || s.ExitCode() != 130 {
				t.Fatalf("late success accepted: %+v", s)
			}
			if !f.released || f.running || f.child {
				t.Fatalf("cleanup incomplete: %+v", f)
			}
		})
	}
}

func TestRunTimeoutRejectsLateSuccess(t *testing.T) {
	f := newFakeRuntime()
	gate := make(chan struct{})
	f.gates[Execute] = gate
	f.ignoreCancellation = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	done := make(chan RunState, 1)
	go func() { done <- Run(ctx, RunRequest{Timeout: time.Second}, f) }()
	waitForEffect(t, f, Execute)
	<-ctx.Done()
	close(gate)
	if s := resultOfRun(t, done); s.ExitCode() != 124 {
		t.Fatalf("late success accepted: %+v", s)
	}
}

func TestRunRequestTimeoutAppliesThroughExecute(t *testing.T) {
	for _, stage := range []Effect{Acquire, Prepare, Start, AwaitGuest, Execute} {
		t.Run(string(stage), func(t *testing.T) {
			f := newFakeRuntime()
			f.gates[stage] = make(chan struct{})
			s := Run(context.Background(), RunRequest{Timeout: time.Millisecond}, f)
			if !s.Done() || s.ExitCode() != 124 {
				t.Fatalf("timeout: %+v", s)
			}
			if stage != Acquire && !f.released {
				t.Fatal("cleanup inherited expired run context")
			}
		})
	}
}

func TestRunCommandFailureSurvivesCleanupFailure(t *testing.T) {
	f := newFakeRuntime()
	f.commandExit = 42
	f.fail[Collect] = errors.New("collection broke")
	f.fail[Stop] = errors.New("stop broke")
	s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
	if !s.Done() || s.ExitCode() != 42 || f.released || !f.prepared {
		t.Fatalf("result: %+v", s)
	}
}

func TestRunUnconfirmedStopRetainsVMInputs(t *testing.T) {
	f := newFakeRuntime()
	f.fail[Stop] = errors.New("termination unconfirmed")
	s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
	if !s.Done() || s.ExitCode() == 0 {
		t.Fatalf("result: %+v", s)
	}
	if f.released || !f.running || !f.prepared || f.child {
		t.Fatalf("unsafe reuse after unknown VM outcome: %+v", f)
	}
}

func TestRunCleanupSharesBoundedIndependentBudget(t *testing.T) {
	f := newFakeRuntime()
	before := time.Now()
	s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
	if s.ExitCode() != 0 {
		t.Fatalf("result: %+v", s)
	}
	cleanup := f.contexts[Collect]
	deadline, ok := cleanup.Deadline()
	if !ok || deadline.Before(before.Add(9*time.Second)) || deadline.After(time.Now().Add(10*time.Second)) {
		t.Fatalf("cleanup deadline %v", deadline)
	}
	if f.contexts[Stop] != cleanup || f.contexts[Release] != cleanup {
		t.Fatal("cleanup budget reset between effects")
	}
	runDeadline, _ := f.contexts[Execute].Deadline()
	if !runDeadline.Before(deadline) {
		t.Fatal("cleanup reused run budget")
	}
}

func TestRunRejectsInvalidRuntimeExit(t *testing.T) {
	for _, exit := range []int{-1, 256} {
		f := newFakeRuntime()
		f.commandExit = exit
		s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
		if !s.Done() || s.ExitCode() == 0 || !f.released {
			t.Fatalf("invalid exit %d: %+v", exit, s)
		}
	}
}

func TestRunCleanupBudgetExpiresWithoutReleasingLiveVM(t *testing.T) {
	f := newFakeRuntime()
	f.gates[Collect] = make(chan struct{})
	before := time.Now()
	s := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
	if !s.Done() || s.ExitCode() != 124 {
		t.Fatalf("cleanup timeout: %+v", s)
	}
	if time.Since(before) > 12*time.Second {
		t.Fatal("cleanup exceeded total budget")
	}
	if f.acquired {
		t.Fatal("expired cleanup retained local lease")
	}
	if f.released || !f.prepared || !f.running || f.child {
		t.Fatalf("expired cleanup lost VM inputs: %+v", f)
	}
	want := []Effect{Acquire, Prepare, Start, AwaitGuest, Execute, Collect, Stop, Release}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("cleanup did not finish once: %v", f.calls)
	}
}

type nestedCleanupRuntime struct {
	*fakeRuntime
	nested context.Context
}

func (f *nestedCleanupRuntime) Execute(ctx context.Context) (int, error) {
	f.nested, _ = cleanupContext(ctx)
	return f.fakeRuntime.Execute(ctx)
}
func TestRunNestedProcessCleanupSharesFinalBudget(t *testing.T) {
	f := &nestedCleanupRuntime{fakeRuntime: newFakeRuntime()}
	state := Run(context.Background(), RunRequest{Timeout: time.Second}, f)
	if state.ExitCode() != 0 {
		t.Fatal(state)
	}
	if f.nested != f.contexts[Collect] || f.nested != f.contexts[Stop] || f.nested != f.contexts[Release] {
		t.Fatal("nested cleanup renewed the final ten-second budget")
	}
	if f.nested.Err() != context.Canceled {
		t.Fatal("terminal run leaked cleanup context", f.nested.Err())
	}
}
