package couchtty

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// boundedPass is a running pass over n detached threads with the given limit.
func boundedPass(n, limit int) MenuState {
	rows := make([]couchcore.ActionableThreadSummary, n)
	for i := range rows {
		rows[i] = reattachRow(fmt.Sprintf("couch-%02d", i), couchcore.ThreadDetached, i+1)
	}
	state := seedReattach(armedMenu("couch-root"), rows)
	state.Reattach.Limit = limit
	return state
}

// The pass keeps up to its limit in flight, and none while the operator has an
// operation in flight (cell 10 holds for the bounded pass too).
func TestReattachAdvanceStartsUpToTheLimit(t *testing.T) {
	state, effects := advanceReattach(boundedPass(5, 3))
	if len(effects) != 3 || len(state.Reattach.InFlight) != 3 || len(state.Reattach.Queue) != 2 {
		t.Fatalf("effects %d, in flight %d, queued %d; want 3/3/2", len(effects), len(state.Reattach.InFlight), len(state.Reattach.Queue))
	}
	if again, more := advanceReattach(state); len(more) != 0 || len(again.Reattach.InFlight) != 3 {
		t.Fatalf("advance at the limit started more: %+v", more)
	}
	held := boundedPass(5, 3)
	held.InFlight = MenuOperationOrigin{Operation: "relaunch", Attempt: 99}
	if _, effects := advanceReattach(held); len(effects) != 0 {
		t.Fatalf("the pass advanced under the operator's operation: %+v", effects)
	}
}

// Completions resolve by attempt number, in any order, and a stale or unknown
// attempt changes nothing (cell 8).
func TestReattachFinishResolvesOutOfOrderCompletions(t *testing.T) {
	state, effects := advanceReattach(boundedPass(3, 3))
	for i := len(effects) - 1; i >= 0; i-- {
		event := passResult(effects[i].Attempt, string(state.Reattach.InFlight[effects[i].Attempt].Tag))
		event.Success = true
		state = finishReattach(state, event)
	}
	if len(state.Reattach.InFlight) != 0 || len(state.Reattach.Attached) != 3 {
		t.Fatalf("pass after reverse-order completions = %+v", state.Reattach)
	}
	before := state.Reattach
	state = finishReattach(state, passResult(12345, "couch-00"))
	if len(state.Reattach.Attached) != len(before.Attached) || len(state.Reattach.InFlight) != 0 {
		t.Fatal("an unknown attempt changed the pass")
	}
}

// Placeholder chips are in attempt order however the map iterates, so they
// never jitter between frames.
func TestReattachPlaceholdersAreInAttemptOrder(t *testing.T) {
	state, effects := advanceReattach(boundedPass(6, 4))
	want := make([]couchcore.ThreadAddress, 0, len(effects))
	for _, effect := range effects {
		want = append(want, state.Reattach.InFlight[effect.Attempt])
	}
	want = append(want, state.Reattach.Queue...)
	for frame := 0; frame < 20; frame++ {
		got := pendingPlaceholders(state.Reattach)
		if len(got) != len(want) {
			t.Fatalf("frame %d: %d placeholders, want %d", frame, len(got), len(want))
		}
		for i := range want {
			if got[i].Address != want[i] || got[i].Loading != (i < len(effects)) {
				t.Fatalf("frame %d: placeholder %d = %+v, want %v (loading %v)", frame, i, got[i], want[i], i < len(effects))
			}
		}
	}
}

// Generated sequences: random interleavings of advance, completions (success,
// failure, busy) in random order, and the operator's in-flight slot toggling,
// under fixed seeds. Invariants stated independently of the implementation:
// in flight never exceeds the limit; no thread is attempted twice; the pass
// is Done exactly when nothing is queued or in flight; every seeded thread
// ends attached, failed or skipped.
func TestReattachBoundedPassGeneratedSequences(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		n, limit := 1+rng.Intn(9), 1+rng.Intn(4)
		state := boundedPass(n, limit)
		attempted := map[couchcore.ThreadAddress]int{}
		resolved := map[couchcore.ThreadAddress]bool{}
		var pending []MenuEffect
		for step := 0; step < 400 && state.Reattach.Phase != ReattachDone; step++ {
			switch rng.Intn(4) {
			case 0, 1:
				var effects []MenuEffect
				state, effects = advanceReattach(state)
				for _, e := range effects {
					address := state.Reattach.InFlight[e.Attempt]
					attempted[address]++
				}
				pending = append(pending, effects...)
			case 2:
				if len(pending) == 0 {
					continue
				}
				i := rng.Intn(len(pending))
				e := pending[i]
				pending = append(pending[:i], pending[i+1:]...)
				address := state.Reattach.InFlight[e.Attempt]
				event := passResult(e.Attempt, string(address.Tag))
				switch rng.Intn(3) {
				case 0:
					event.Success = true
				case 1:
					event.Error = "spawn failed"
				default:
					event.Busy = true
				}
				state = finishReattach(state, event)
				resolved[address] = true
			case 3:
				if state.InFlight.Operation == "" {
					state.InFlight = MenuOperationOrigin{Operation: "detach", Attempt: 1 << 40}
				} else {
					state.InFlight = MenuOperationOrigin{}
				}
			}
			if got := len(state.Reattach.InFlight); got > limit {
				t.Fatalf("seed %d step %d: %d in flight, limit %d", seed, step, got, limit)
			}
			for address, count := range attempted {
				if count > 1 {
					t.Fatalf("seed %d: %v attempted %d times", seed, address, count)
				}
			}
			done := state.Reattach.Phase == ReattachDone
			empty := len(state.Reattach.Queue) == 0 && len(state.Reattach.InFlight) == 0
			if done && !empty {
				t.Fatalf("seed %d step %d: Done with work left: %+v", seed, step, state.Reattach)
			}
		}
		// Drain: release the operator and finish everything.
		state.InFlight = MenuOperationOrigin{}
		for guard := 0; state.Reattach.Phase != ReattachDone && guard < 100; guard++ {
			var effects []MenuEffect
			state, effects = advanceReattach(state)
			pending = append(pending, effects...)
			for _, e := range pending {
				address := state.Reattach.InFlight[e.Attempt]
				event := passResult(e.Attempt, string(address.Tag))
				event.Success = true
				state = finishReattach(state, event)
				resolved[address] = true
			}
			pending = nil
		}
		if state.Reattach.Phase != ReattachDone {
			t.Fatalf("seed %d: pass never finished: %+v", seed, state.Reattach)
		}
		if len(resolved) != n {
			t.Fatalf("seed %d: %d of %d threads resolved", seed, len(resolved), n)
		}
	}
}
