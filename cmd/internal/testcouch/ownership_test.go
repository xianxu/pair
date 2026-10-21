package testcouch

import (
	"fmt"
	"testing"
)

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
	if s.Phase != "cleaned" || len(s.Resources) != 0 {
		t.Fatalf("owned work not joined: %+v", s)
	}
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
		if len(s.Resources) != 0 || s.Phase != "cleaned" {
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
