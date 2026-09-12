package couchtty

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// The pure pass, cell by cell (pair#206 plan, "The pass's transitions"). Each
// test names the cell it pins; the routing through ReduceMenu and the
// generated-sequence invariants live beside this once the reducer carries it.

func reattachRow(tag string, state couchcore.ActionableThreadState, activeMinutesAgo int) couchcore.ActionableThreadSummary {
	return couchcore.ActionableThreadSummary{
		Address:      menuAddress(tag),
		State:        state,
		LastActiveAt: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC).Add(-time.Duration(activeMinutesAgo) * time.Minute),
	}
}

func armedMenu(root string) MenuState {
	return armReattach(NewMenuState(nil, menuAddress(root)), menuAddress(root))
}

// Cell 1, and arming is once per console.
func TestReattachArmIsOnce(t *testing.T) {
	state := armedMenu("couch-root")
	if state.Reattach.Phase != ReattachArmed || state.Reattach.Root != menuAddress("couch-root") {
		t.Fatalf("pass = %+v, want armed with the root", state.Reattach)
	}
	state.Reattach.Phase = ReattachDone
	if again := armReattach(state, menuAddress("couch-other")); again.Reattach.Phase != ReattachDone || again.Reattach.Root != menuAddress("couch-root") {
		t.Fatalf("re-arm after the pass finished = %+v; it must not re-seed", again.Reattach)
	}
}

// Cell 3: what is seeded, and in what order.
func TestReattachSeedTakesWarmAndUnknownThreadsMostRecentFirst(t *testing.T) {
	inventory := []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadDetached, 0), // excluded: the startup root
		reattachRow("couch-old", couchcore.ThreadDetached, 90), // warm, least recent
		reattachRow("couch-live", couchcore.ThreadLive, 1),     // already hosted
		reattachRow("couch-parked", couchcore.ThreadParked, 2), // never resumed at startup
		reattachRow("couch-new", couchcore.ThreadDetached, 5),  // warm, most recent
		reattachRow("couch-gone", couchcore.ThreadUnusable, 3), // session-gone: not a candidate
	}
	unknown := reattachRow("couch-unknown", couchcore.ThreadUnusable, 30)
	unknown.Reason = couchcore.ReasonUnknown // "we could not ask", not "not detached"
	inventory = append(inventory, unknown)
	inventory[5].Reason = couchcore.ReasonSessionGone

	state := seedReattach(armedMenu("couch-root"), inventory)

	want := []couchcore.ThreadAddress{menuAddress("couch-new"), menuAddress("couch-unknown"), menuAddress("couch-old")}
	if !reflect.DeepEqual(state.Reattach.Queue, want) || state.Reattach.Phase != ReattachRunning {
		t.Fatalf("queue = %v (phase %d), want %v running", state.Reattach.Queue, state.Reattach.Phase, want)
	}
}

// Seeding happens once: a later inventory never extends the queue (cell 4), so
// a thread the operator detaches during the pass stays detached.
func TestReattachSeedsOnlyFromArmed(t *testing.T) {
	state := seedReattach(armedMenu("couch-root"), []couchcore.ActionableThreadSummary{reattachRow("couch-a", couchcore.ThreadDetached, 1)})
	reseeded := seedReattach(state, []couchcore.ActionableThreadSummary{
		reattachRow("couch-a", couchcore.ThreadDetached, 1),
		reattachRow("couch-detached-meanwhile", couchcore.ThreadDetached, 0),
	})
	if len(reseeded.Reattach.Queue) != 1 {
		t.Fatalf("queue = %v; the queue is seeded once and never extended", reseeded.Reattach.Queue)
	}
}

// Advancing starts exactly one attempt, as a warm-only background resume.
func TestReattachAdvanceStartsOneWarmOnlyBackgroundAttempt(t *testing.T) {
	state := seedReattach(armedMenu("couch-root"), []couchcore.ActionableThreadSummary{
		reattachRow("couch-a", couchcore.ThreadDetached, 1),
		reattachRow("couch-b", couchcore.ThreadDetached, 2),
	})
	state, effects := advanceReattach(state)
	if len(effects) != 1 {
		t.Fatalf("effects = %+v, want exactly one attempt", effects)
	}
	effect := effects[0]
	if effect.Operation != "resume" || !effect.Background || effect.Args["warm-only"] != "true" || effect.Args["tag"] != "couch-a" {
		t.Fatalf("effect = %+v, want a background warm-only resume of couch-a", effect)
	}
	if state.Reattach.Loading != menuAddress("couch-a") || state.Reattach.LoadingAttempt != effect.Attempt || effect.Attempt == 0 {
		t.Fatalf("pass = %+v, effect attempt %d", state.Reattach, effect.Attempt)
	}
	// One at a time: a second advance while couch-a is loading emits nothing.
	if _, more := advanceReattach(state); len(more) != 0 {
		t.Fatalf("a second advance mid-attempt emitted %+v", more)
	}
}

// Cell 10: the pass holds while the OPERATOR has an operation in flight.
func TestReattachHoldsWhileAnOperatorOperationIsInFlight(t *testing.T) {
	state := seedReattach(armedMenu("couch-root"), []couchcore.ActionableThreadSummary{reattachRow("couch-a", couchcore.ThreadDetached, 1)})
	state.InFlight = MenuOperationOrigin{Operation: "leave", Attempt: 7}
	if held, effects := advanceReattach(state); len(effects) != 0 || held.Reattach.Loading != (couchcore.ThreadAddress{}) {
		t.Fatalf("advanced under an operator operation: effects %+v, pass %+v", effects, held.Reattach)
	}
	state.InFlight = MenuOperationOrigin{}
	if _, effects := advanceReattach(state); len(effects) != 1 {
		t.Fatalf("did not resume once the slot cleared: %+v", effects)
	}
}

// An empty queue finishes the pass rather than idling in Running forever.
func TestReattachDrainsToDone(t *testing.T) {
	state := seedReattach(armedMenu("couch-root"), nil)
	state, effects := advanceReattach(state)
	if len(effects) != 0 || state.Reattach.Phase != ReattachDone {
		t.Fatalf("empty pass: effects %+v, phase %d; want no effect and Done", effects, state.Reattach.Phase)
	}
}

func loadingPass(t *testing.T) (MenuState, uint64) {
	t.Helper()
	state := seedReattach(armedMenu("couch-root"), []couchcore.ActionableThreadSummary{
		reattachRow("couch-a", couchcore.ThreadDetached, 1),
		reattachRow("couch-b", couchcore.ThreadDetached, 2),
	})
	state, effects := advanceReattach(state)
	if len(effects) != 1 {
		t.Fatalf("setup: %+v", effects)
	}
	return state, effects[0].Attempt
}

func passResult(attempt uint64, tag string) MenuEvent {
	return MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true, Attempt: attempt, Address: menuAddress(tag)}
}

// Cells 5-7: success, a skip, and a failure.
func TestReattachFinishResolvesTheAttempt(t *testing.T) {
	t.Run("cell 5: success is attached at its generation", func(t *testing.T) {
		state, attempt := loadingPass(t)
		event := passResult(attempt, "couch-a")
		event.Success, event.ProjectionAfterGeneration = true, 41
		state = finishReattach(state, event)
		if state.Reattach.Loading != (couchcore.ThreadAddress{}) || state.Reattach.Attached[menuAddress("couch-a")] != 41 {
			t.Fatalf("pass = %+v", state.Reattach)
		}
	})
	for _, code := range []couchcore.ResumeDiagnosticCode{couchcore.ResumeNotDetached, couchcore.ResumeSessionGone} {
		t.Run("cell 6: "+string(code)+" is a silent skip", func(t *testing.T) {
			state, attempt := loadingPass(t)
			event := passResult(attempt, "couch-a")
			event.Diagnostic = code
			state = finishReattach(state, event)
			if view, owned := passViewOf(state.Reattach, menuAddress("couch-a")); owned {
				t.Fatalf("a thread that stopped being warm is still owned by the pass: %+v", view)
			}
		})
	}
	t.Run("cell 7: a failure is marked with its code", func(t *testing.T) {
		state, attempt := loadingPass(t)
		event := passResult(attempt, "couch-a")
		event.Diagnostic = couchcore.ResumePathMissing
		state = finishReattach(state, event)
		if got := state.Reattach.Failed[menuAddress("couch-a")]; got != string(couchcore.ResumePathMissing) {
			t.Fatalf("failed = %v", state.Reattach.Failed)
		}
	})
	t.Run("cell 7: a failure with no code shows its error's first line", func(t *testing.T) {
		state, attempt := loadingPass(t)
		event := passResult(attempt, "couch-a")
		event.Error = "spawn pair: exec: \"pair\": executable file not found in $PATH\nwhile starting couch-a"
		state = finishReattach(state, event)
		if got, want := state.Reattach.Failed[menuAddress("couch-a")], `spawn pair: exec: "pair": executable file not found in $PATH`; got != want {
			t.Fatalf("failed = %q, want the error's first line %q", got, want)
		}
	})
	t.Run("cell 7: a failure with neither a code nor text still says it failed", func(t *testing.T) {
		state, attempt := loadingPass(t)
		event := passResult(attempt, "couch-a")
		event.Error = ""
		state = finishReattach(state, event)
		if got := state.Reattach.Failed[menuAddress("couch-a")]; got != reattachFailedCode {
			t.Fatalf("failed = %v; an empty mark would render as \"reattach failed: \"", state.Reattach.Failed)
		}
	})
}

// Cell 8: a result for any other attempt changes nothing.
func TestReattachFinishIgnoresOtherAttempts(t *testing.T) {
	state, attempt := loadingPass(t)
	for _, event := range []MenuEvent{
		passResult(attempt+1, "couch-a"), // a different attempt
		passResult(attempt, "couch-b"),   // the right attempt, the wrong thread
		{Kind: MenuEventOperationResult, Attempt: attempt, Address: menuAddress("couch-a")}, // not a pass result
	} {
		if after := finishReattach(state, event); !reflect.DeepEqual(after.Reattach, state.Reattach) {
			t.Fatalf("event %+v changed the pass: %+v -> %+v", event, state.Reattach, after.Reattach)
		}
	}
}

// Cell 11: only an inventory NEWER than the attach is authoritative again.
func TestReattachAttachedExpiresOnANewerGeneration(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state.Reattach.Attached = map[couchcore.ThreadAddress]uint64{menuAddress("couch-a"): 10}
	if kept := expireAttached(state, 10); len(kept.Reattach.Attached) != 1 {
		t.Fatalf("the generation the attach landed at expired it: %+v", kept.Reattach.Attached)
	}
	if expired := expireAttached(state, 11); len(expired.Reattach.Attached) != 0 {
		t.Fatalf("a newer generation kept it: %+v", expired.Reattach.Attached)
	}
}

// Cell 9: resuming a failed thread by hand clears its mark.
func TestReattachFailureClearsOnManualResume(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state.Reattach.Failed = map[couchcore.ThreadAddress]string{menuAddress("couch-a"): "resume-path-missing"}
	if cleared := clearReattachFailure(state, menuAddress("couch-a")); len(cleared.Reattach.Failed) != 0 {
		t.Fatalf("failed = %v", cleared.Reattach.Failed)
	}
}

// The view: pending is exactly Queued and Loading.
func TestPassViewMarksExactlyQueuedAndLoadingPending(t *testing.T) {
	state, _ := loadingPass(t) // couch-a loading, couch-b queued
	state.Reattach.Failed = map[couchcore.ThreadAddress]string{menuAddress("couch-f"): "resume-path-missing"}
	state.Reattach.Attached = map[couchcore.ThreadAddress]uint64{menuAddress("couch-t"): 3}
	for tag, want := range map[string]struct {
		state   PassState
		pending bool
	}{
		"couch-a": {PassLoading, true},
		"couch-b": {PassQueued, true},
		"couch-f": {PassFailed, false},
		"couch-t": {PassAttached, false},
	} {
		view, owned := passViewOf(state.Reattach, menuAddress(tag))
		if !owned || view.State != want.state || view.Pending() != want.pending {
			t.Fatalf("%s: view %+v owned %v, want state %d pending %v", tag, view, owned, want.state, want.pending)
		}
	}
	if _, owned := passViewOf(state.Reattach, menuAddress("couch-stranger")); owned {
		t.Fatal("the pass claims a thread it never held")
	}
}

// The status bar's placeholders: the loading thread first, then the queue --
// the order that keeps a chip in place when it resolves.
func TestPendingPlaceholdersPutTheLoadingThreadFirst(t *testing.T) {
	state, _ := loadingPass(t)
	got := pendingPlaceholders(state.Reattach)
	want := []ReattachPlaceholder{{Address: menuAddress("couch-a"), Loading: true}, {Address: menuAddress("couch-b")}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("placeholders = %+v, want %+v", got, want)
	}
}

// MenuState is immutable-by-copy, and the pass's maps and slice must be too:
// tests and the console both hold prior states and compare them.
func TestCloneMenuStateCopiesThePassDeeply(t *testing.T) {
	state, _ := loadingPass(t)
	state.Reattach.Attached = map[couchcore.ThreadAddress]uint64{menuAddress("couch-t"): 3}
	state.Reattach.Failed = map[couchcore.ThreadAddress]string{menuAddress("couch-f"): "x"}
	clone := cloneMenuState(state)
	clone.Reattach.Queue[0] = menuAddress("couch-mutated")
	clone.Reattach.Attached[menuAddress("couch-new")] = 9
	clone.Reattach.Failed[menuAddress("couch-new")] = "y"
	if state.Reattach.Queue[0] == menuAddress("couch-mutated") || len(state.Reattach.Attached) != 1 || len(state.Reattach.Failed) != 1 {
		t.Fatalf("mutating the clone reached the original: %+v", state.Reattach)
	}
	if empty := cloneMenuState(NewMenuState(nil, couchcore.ThreadAddress{})); empty.Reattach.Attached != nil || empty.Reattach.Failed != nil {
		t.Fatal("a nil map must stay nil, or DeepEqual comparisons of whole states break")
	}
}
