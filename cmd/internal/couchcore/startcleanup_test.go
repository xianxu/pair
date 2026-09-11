package couchcore

import "testing"

// Every combination of the rule's inputs, enumerated rather than sampled: 3
// shapes x 2 helper x 3 presence x 2 phase = 36 rows (pair#230).
//
// The expected values are WRITTEN OUT, not recomputed from the same conditions
// the implementation branches on. A table that derives its own expectations
// mirrors whatever the code does and asserts nothing -- the shape
// TestOperationArityMatchesExpectation guards against by declaring its numbers
// in the test.
//
// Reading the key: shape / helper / presence / phase.
//
//	sp = spawn, cr = cold resume, wr = warm reattach
//	dead / live   = the helper is proven gone / may still be running
//	?  - +        = presence unobserved / absent / present
//	claim / rec   = a start claim / a live incarnation
func TestDecideStartCleanupTable(t *testing.T) {
	tests := []struct {
		name string
		in   StartCleanupInput
		want DurableAction
		why  string
	}{
		// A spawn reconciles in every cell, in both phases. Unchanged by
		// pair#230.
		{"sp/live/?/claim", StartCleanupInput{StartSpawn, false, PresenceUnobserved, false}, DurableReconcile, "spawn tail"},
		{"sp/live/-/claim", StartCleanupInput{StartSpawn, false, PresenceAbsent, false}, DurableReconcile, "spawn tail"},
		{"sp/live/+/claim", StartCleanupInput{StartSpawn, false, PresencePresent, false}, DurableReconcile, "spawn tail"},
		{"sp/dead/?/claim", StartCleanupInput{StartSpawn, true, PresenceUnobserved, false}, DurableReconcile, "spawn tail"},
		{"sp/dead/-/claim", StartCleanupInput{StartSpawn, true, PresenceAbsent, false}, DurableReconcile, "spawn tail"},
		{"sp/dead/+/claim", StartCleanupInput{StartSpawn, true, PresencePresent, false}, DurableReconcile, "spawn tail"},
		{"sp/live/?/rec", StartCleanupInput{StartSpawn, false, PresenceUnobserved, true}, DurableReconcile, "spawn tail"},
		{"sp/live/-/rec", StartCleanupInput{StartSpawn, false, PresenceAbsent, true}, DurableReconcile, "spawn tail"},
		{"sp/live/+/rec", StartCleanupInput{StartSpawn, false, PresencePresent, true}, DurableReconcile, "spawn tail"},
		{"sp/dead/?/rec", StartCleanupInput{StartSpawn, true, PresenceUnobserved, true}, DurableReconcile, "spawn tail"},
		{"sp/dead/-/rec", StartCleanupInput{StartSpawn, true, PresenceAbsent, true}, DurableReconcile, "spawn tail"},
		{"sp/dead/+/rec", StartCleanupInput{StartSpawn, true, PresencePresent, true}, DurableReconcile, "spawn tail"},

		// A cold resume owns its session too. At claim phase it rolls back only
		// once that session is gone and the helper with it; at record phase it
		// takes the reconcile tail. Unchanged by pair#230.
		{"cr/live/?/claim", StartCleanupInput{StartColdResume, false, PresenceUnobserved, false}, DurableMarkUnknown, "helper unaccounted for"},
		{"cr/live/-/claim", StartCleanupInput{StartColdResume, false, PresenceAbsent, false}, DurableMarkUnknown, "helper unaccounted for"},
		{"cr/live/+/claim", StartCleanupInput{StartColdResume, false, PresencePresent, false}, DurableMarkUnknown, "helper unaccounted for"},
		{"cr/dead/?/claim", StartCleanupInput{StartColdResume, true, PresenceUnobserved, false}, DurableMarkUnknown, "presence unproven"},
		{"cr/dead/-/claim", StartCleanupInput{StartColdResume, true, PresenceAbsent, false}, DurableRollback, "its session is gone"},
		{"cr/dead/+/claim", StartCleanupInput{StartColdResume, true, PresencePresent, false}, DurableMarkUnknown, "a session it created survived"},
		{"cr/live/?/rec", StartCleanupInput{StartColdResume, false, PresenceUnobserved, true}, DurableReconcile, "live-record tail"},
		{"cr/live/-/rec", StartCleanupInput{StartColdResume, false, PresenceAbsent, true}, DurableReconcile, "live-record tail"},
		{"cr/live/+/rec", StartCleanupInput{StartColdResume, false, PresencePresent, true}, DurableReconcile, "live-record tail"},
		{"cr/dead/?/rec", StartCleanupInput{StartColdResume, true, PresenceUnobserved, true}, DurableReconcile, "live-record tail"},
		{"cr/dead/-/rec", StartCleanupInput{StartColdResume, true, PresenceAbsent, true}, DurableReconcile, "live-record tail"},
		{"cr/dead/+/rec", StartCleanupInput{StartColdResume, true, PresencePresent, true}, DurableReconcile, "live-record tail"},

		// The warm column: never quiesce. This is what pair#230 changes.
		{"wr/live/?/claim", StartCleanupInput{StartWarmReattach, false, PresenceUnobserved, false}, DurableMarkUnknown, "helper unaccounted for"},
		{"wr/live/-/claim", StartCleanupInput{StartWarmReattach, false, PresenceAbsent, false}, DurableMarkUnknown, "helper unaccounted for"},
		{"wr/live/+/claim", StartCleanupInput{StartWarmReattach, false, PresencePresent, false}, DurableMarkUnknown, "helper unaccounted for"},
		{"wr/dead/?/claim", StartCleanupInput{StartWarmReattach, true, PresenceUnobserved, false}, DurableRollback, "the claim is all this start wrote"},
		{"wr/dead/-/claim", StartCleanupInput{StartWarmReattach, true, PresenceAbsent, false}, DurableRollback, "session died on its own; reads session-gone"},
		{"wr/dead/+/claim", StartCleanupInput{StartWarmReattach, true, PresencePresent, false}, DurableRollback, "session survived; reads detached"},
		{"wr/live/?/rec", StartCleanupInput{StartWarmReattach, false, PresenceUnobserved, true}, DurableMarkUnknown, "helper unaccounted for"},
		{"wr/live/-/rec", StartCleanupInput{StartWarmReattach, false, PresenceAbsent, true}, DurableMarkUnknown, "helper unaccounted for"},
		{"wr/live/+/rec", StartCleanupInput{StartWarmReattach, false, PresencePresent, true}, DurableMarkUnknown, "helper unaccounted for"},
		{"wr/dead/?/rec", StartCleanupInput{StartWarmReattach, true, PresenceUnobserved, true}, DurableMarkUnknown, "cannot prove the session to retire onto"},
		{"wr/dead/-/rec", StartCleanupInput{StartWarmReattach, true, PresenceAbsent, true}, DurableMarkUnknown, "nothing to reattach to"},
		{"wr/dead/+/rec", StartCleanupInput{StartWarmReattach, true, PresencePresent, true}, DurableRetire, "give the thread back to detached"},
	}
	if len(tests) != 36 {
		t.Fatalf("table has %d rows, want 36 -- the input space is 3x2x3x2", len(tests))
	}
	seen := map[StartCleanupInput]string{}
	for _, tt := range tests {
		if prior, dup := seen[tt.in]; dup {
			t.Fatalf("%s duplicates %s: the table must cover each input once", tt.name, prior)
		}
		seen[tt.in] = tt.name
		if got := DecideStartCleanup(tt.in); got != tt.want {
			t.Errorf("%s: DecideStartCleanup(%+v) = %+v, want %+v (%s)", tt.name, tt.in, got, tt.want, tt.why)
		}
	}
}

// The property pair#230 exists to establish, and the single question the whole
// fix turns on: a start may end only a session it created. It is asserted here
// over every shape, including values no constant names -- an unrecognised shape
// must answer NO, because guessing wrong in that direction kills an agent.
func TestOnlyAnOwningShapeMayEndItsSession(t *testing.T) {
	for _, tt := range []struct {
		shape StartShape
		want  bool
	}{
		{StartSpawn, true},
		{StartColdResume, true},
		{StartWarmReattach, false},
		{StartShape(""), false},
		{StartShape("something-a-later-version-wrote"), false},
	} {
		if got := tt.shape.OwnsSession(); got != tt.want {
			t.Errorf("StartShape(%q).OwnsSession() = %v, want %v", string(tt.shape), got, tt.want)
		}
	}
}

// Nothing durable is undone while the helper may still be writing. Unknown is
// recoverable; a rolled-back claim or a retired incarnation is not.
func TestDecideStartCleanupUndoesNothingWhileTheHelperIsUnaccountedFor(t *testing.T) {
	forEachStartCleanupInput(t, func(t *testing.T, in StartCleanupInput) {
		if in.HelperDead || in.Shape == StartSpawn {
			return // a spawn's reconcile reads registration evidence, not the helper
		}
		switch got := DecideStartCleanup(in); got {
		case DurableRollback, DurableRetire:
			t.Errorf("DecideStartCleanup(%+v) = %q with a live helper", in, got)
		}
	})
}

// Retire means "hand the thread back to the session it was reattaching to", so
// it belongs to the warm shape alone.
func TestDecideStartCleanupRetiresOnlyOnTheWarmPath(t *testing.T) {
	forEachStartCleanupInput(t, func(t *testing.T, in StartCleanupInput) {
		if got := DecideStartCleanup(in); got == DurableRetire && in.Shape.OwnsSession() {
			t.Errorf("DecideStartCleanup(%+v) = %q: an owning start has no surviving session to go back to", in, got)
		}
	})
}

func forEachStartCleanupInput(t *testing.T, check func(*testing.T, StartCleanupInput)) {
	t.Helper()
	for _, shape := range []StartShape{StartSpawn, StartColdResume, StartWarmReattach} {
		for _, dead := range []bool{false, true} {
			for _, presence := range []SessionPresence{PresenceUnobserved, PresenceAbsent, PresencePresent} {
				for _, live := range []bool{false, true} {
					check(t, StartCleanupInput{Shape: shape, HelperDead: dead, Presence: presence, LiveRecord: live})
				}
			}
		}
	}
}

// The shell asks zellij about the session only where the decision reads it.
// This pins the shell's predicate against the decider itself: for every input,
// flipping presence must change the answer exactly when the predicate says the
// observation is worth making. Otherwise the shell either pays for a round trip
// nobody reads, or skips one the decision depends on.
func TestPresenceIsObservedExactlyWhereTheDecisionReadsIt(t *testing.T) {
	for _, shape := range []StartShape{StartSpawn, StartColdResume, StartWarmReattach} {
		for _, liveRecord := range []bool{false, true} {
			for _, helperDead := range []bool{false, true} {
				base := StartCleanupInput{Shape: shape, HelperDead: helperDead, LiveRecord: liveRecord}
				matters := false
				for _, presence := range []SessionPresence{PresenceAbsent, PresencePresent} {
					probe := base
					probe.Presence = presence
					unobserved := base
					unobserved.Presence = PresenceUnobserved
					if DecideStartCleanup(probe) != DecideStartCleanup(unobserved) {
						matters = true
					}
				}
				if matters && !startCleanupReadsPresence(shape, liveRecord) {
					t.Errorf("%+v: the decision reads presence, but the shell does not observe it", base)
				}
				// The converse holds only where a dead helper makes presence
				// decisive; with a live helper every shape answers mark-unknown,
				// and the shell still observes because it cannot know that
				// without duplicating the decider.
				if !matters && helperDead && startCleanupReadsPresence(shape, liveRecord) {
					t.Errorf("%+v: the shell observes presence the decision ignores", base)
				}
			}
		}
	}
}
