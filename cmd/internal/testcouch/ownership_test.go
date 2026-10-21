package testcouch

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"testing"
)

func TestUncertainCompletionKeepsFailureUntilRootsAreRemoved(t *testing.T) {
	r := Resource{ID: "cli", Generation: "one"}
	s := ReduceRun(OwnedRun{}, RunEvent{Kind: "own", Resource: r})
	s = ReduceRun(s, RunEvent{Kind: "unknown", Error: "CLI cleanup uncertain"})
	s = ReduceRun(s, RunEvent{Kind: "gone", Resource: r})
	if s.Error == "" {
		t.Fatal("resource absence silently reopened failed fleet")
	}
	s = ReduceRun(s, RunEvent{Kind: "own", Resource: r})
	if s.Error == "" {
		t.Fatal("late ownership cleared uncertainty")
	}
	s = ReduceRun(s, RunEvent{Kind: "closing"})
	s = ReduceRun(s, RunEvent{Kind: "gone", Resource: r})
	if s.Error == "" {
		t.Fatal("closing cleared uncertainty before root removal")
	}
	s = ReduceRun(s, RunEvent{Kind: "roots-removed"})
	if s.Error != "" || s.Phase != "cleaned" {
		t.Fatalf("completed cleanup retained failure: %+v", s)
	}
}

func TestUncertainFleetRefusesNewWork(t *testing.T) {
	f := &Fleet{ctx: t.Context(), model: ReduceRun(OwnedRun{}, RunEvent{Kind: "unknown", Error: "CLI cleanup uncertain"})}
	_, done, err := f.begin(context.Background())
	if err == nil {
		done()
		t.Fatal("uncertain fleet admitted new work")
	}
}

func TestOwnershipReconcilesLateStartAndStaleOutcome(t *testing.T) {
	s := OwnedRun{}
	s = ReduceRun(s, RunEvent{Kind: "own", Resource: Resource{ID: "child", Generation: "old"}})
	s = ReduceRun(s, RunEvent{Kind: "closing"})
	s = ReduceRun(s, RunEvent{Kind: "own", Resource: Resource{ID: "late", Generation: "new"}})
	s = ReduceRun(s, RunEvent{Kind: "gone", Resource: Resource{ID: "child", Generation: "reused"}})
	if s.Phase != "closing" || len(s.Resources) != 2 {
		t.Fatalf("lost owned work: %+v", s)
	}
	s = ReduceRun(s, RunEvent{Kind: "unknown", Error: "identity unreadable"})
	if s.Phase != "stuck" {
		t.Fatalf("unknown claimed cleanup: %+v", s)
	}
	s = ReduceRun(s, RunEvent{Kind: "gone", Resource: Resource{ID: "child", Generation: "old"}})
	s = ReduceRun(s, RunEvent{Kind: "gone", Resource: Resource{ID: "late", Generation: "new"}})
	if s.Phase != "quiescent" || len(s.Resources) != 0 {
		t.Fatalf("owned work not joined: %+v", s)
	}
	s = ReduceRun(s, RunEvent{Kind: "roots-removed"})
	late := ReduceRun(s, RunEvent{Kind: "gone", Resource: Resource{ID: "child", Generation: "old"}})
	if late.Phase != "cleaned" {
		t.Fatal("stale completion reopened cleaned run")
	}
}

func TestOwnershipBoundedEffects(t *testing.T) {
	for _, n := range []int{8, 16} {
		s := OwnedRun{}
		for i := 0; i < n; i++ {
			s = ReduceRun(s, RunEvent{Kind: "own", Resource: Resource{ID: fmt.Sprint(i), Generation: "one"}})
		}
		before := s
		s = ReduceRun(s, RunEvent{Kind: "closing"})
		if before.Phase != "ready" || s.Phase != "closing" || len(before.Resources) != n {
			t.Fatal("reducer mutated prior value")
		}
		for i := 0; i < n; i++ {
			s = ReduceRun(s, RunEvent{Kind: "gone", Resource: Resource{ID: fmt.Sprint(i), Generation: "one"}})
		}
		if len(s.Resources) != 0 || s.Phase != "quiescent" {
			t.Fatalf("n=%d: %+v", n, s)
		}
	}
}

func FuzzOwnedRunStaleCompletion(f *testing.F) {
	f.Add("generation-one", "generation-two")
	f.Fuzz(func(t *testing.T, owned, stale string) {
		if owned == "" || stale == owned {
			return
		}
		s := ReduceRun(OwnedRun{}, RunEvent{Kind: "own", Resource: Resource{ID: "child", Generation: owned}})
		s = ReduceRun(s, RunEvent{Kind: "closing"})
		s = ReduceRun(s, RunEvent{Kind: "gone", Resource: Resource{ID: "child", Generation: stale}})
		if len(s.Resources) != 1 || s.Phase == "cleaned" {
			t.Fatal("stale outcome released ownership")
		}
	})
}

func TestUncertainFleetRefusesConsoleMutations(t *testing.T) {
	f := &Fleet{ctx: t.Context(), model: ReduceRun(OwnedRun{}, RunEvent{Kind: "unknown", Error: "CLI cleanup uncertain"})}
	for name, run := range map[string]func() error{
		"input":  func() error { return f.Input([]byte("x")) },
		"resize": func() error { return f.Resize(24, 80) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); !errors.Is(err, agentfixture.ErrCleanupUncertain) {
				t.Fatalf("uncertain console mutation: %v", err)
			}
		})
	}
}
