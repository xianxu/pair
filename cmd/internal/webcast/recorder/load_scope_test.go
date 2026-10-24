package recorder

import (
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func scopeTestEvidence() loadEvidence {
	return loadEvidence{Selection: loadSelection{Session: "session", Target: "target", Frame: "frame"}, Load: &recording.PageLoad{Target: "target", Loader: "loader", Root: math.MaxInt64}}
}

func scopeTestPending() loadState {
	s, _ := nextLoadScope(loadState{}, loadInvalidate{Selection: scopeTestEvidence().Selection})
	return s
}

func scopeTestReserve(t *testing.T, s loadState, evidence loadEvidence) (loadState, *loadReservation) {
	t.Helper()
	next, effects := nextLoadScope(s, loadSnapshotReady{Generation: s.generation, Evidence: evidence})
	if next.phase != loadCommitting || len(effects) != 1 {
		t.Fatalf("ready did not reserve: phase=%v effects=%#v", next.phase, effects)
	}
	commit, ok := effects[0].(loadCommitSnapshot)
	if !ok || commit.Reservation == nil {
		t.Fatalf("ready lacks opaque commit reservation: %#v", effects)
	}
	return next, commit.Reservation
}

func TestLoadScopeReservationConfirmsExactlyOnce(t *testing.T) {
	evidence := scopeTestEvidence()
	pending, effects := nextLoadScope(loadState{}, loadInvalidate{Selection: evidence.Selection})
	if pending.phase != loadPending || pending.generation != 1 || !reflect.DeepEqual(effects, []loadEffect{loadCancelSnapshot{Generation: 0}, loadPublishPending{Generation: 1}}) {
		t.Fatalf("invalidation must publish pending before caller starts IO: %#v %#v", pending, effects)
	}
	committing, reservation := scopeTestReserve(t, pending, evidence)
	duplicate, effects := nextLoadScope(committing, loadSnapshotReady{Generation: 1, Evidence: evidence})
	if !reflect.DeepEqual(duplicate, committing) || len(effects) != 0 {
		t.Fatal("duplicate ready reserved a second commit")
	}
	// Equality of fields does not confer a reservation's authority.
	forged := &loadReservation{generation: 1, evidence: reservation.evidenceCopy()}
	unchanged, effects := nextLoadScope(committing, loadCommitDone{Reservation: forged, Success: true})
	if !reflect.DeepEqual(unchanged, committing) || len(effects) != 0 {
		t.Fatal("forged reservation confirmed a load")
	}
	confirmed, effects := nextLoadScope(committing, loadCommitDone{Reservation: reservation, Success: true})
	if confirmed.phase != loadConfirmed || !recording.SamePageLoad(confirmed.load, evidence.Load) || len(effects) != 1 {
		t.Fatalf("reserved successful commit did not confirm: %#v %#v", confirmed, effects)
	}
	publication, ok := effects[0].(loadPublishConfirmed)
	if !ok || publication.Generation != 1 || !recording.SamePageLoad(publication.Load, evidence.Load) {
		t.Fatalf("wrong accepted publication: %#v", effects)
	}
	publication.Load.Loader = "mutated publication"
	if confirmed.load.Loader != "loader" {
		t.Fatal("effect aliases confirmed evidence")
	}
	again, effects := nextLoadScope(confirmed, loadCommitDone{Reservation: reservation, Success: true})
	if !reflect.DeepEqual(again, confirmed) || len(effects) != 0 {
		t.Fatal("consumed reservation confirmed twice")
	}
}

func TestLoadScopeEvidenceIsOwnedAndFactsAreExact(t *testing.T) {
	for _, mutate := range []func(*loadEvidence){
		func(e *loadEvidence) { e.Load = nil },
		func(e *loadEvidence) { e.Load.Root = 0 },
		func(e *loadEvidence) { e.Load.Loader = "" },
		func(e *loadEvidence) { e.Load.Target = "other" },
		func(e *loadEvidence) { e.Selection.Session = "other" },
		func(e *loadEvidence) { e.Selection.Target = "other" },
		func(e *loadEvidence) { e.Selection.Frame = "other" },
		func(e *loadEvidence) { e.Selection.Frame = "" },
	} {
		evidence := scopeTestEvidence()
		mutate(&evidence)
		next, effects := nextLoadScope(scopeTestPending(), loadSnapshotReady{Generation: 1, Evidence: evidence})
		if next.phase != loadUnavailable || !reflect.DeepEqual(effects, []loadEffect{loadPublishUnavailable{Generation: 1}}) {
			t.Fatalf("incomplete/mismatched evidence obtained authority: %#v %#v", next, effects)
		}
	}
	evidence := scopeTestEvidence()
	committing, reservation := scopeTestReserve(t, scopeTestPending(), evidence)
	evidence.Load.Loader = "changed after reservation"
	copy := reservation.evidenceCopy()
	copy.Load.Root = 1
	confirmed, _ := nextLoadScope(committing, loadCommitDone{Reservation: reservation, Success: true})
	if confirmed.load.Loader != "loader" || confirmed.load.Root != math.MaxInt64 {
		t.Fatal("snapshot caller mutated reserved identity")
	}
	initial := scopeTestEvidence()
	initial.Selection.Frame = ""
	pending, _ := nextLoadScope(loadState{}, loadInvalidate{Selection: initial.Selection})
	scopeTestReserve(t, pending, scopeTestEvidence())
	long := scopeTestEvidence()
	long.Selection.Session = strings.Repeat("s", 4096)
	long.Selection.Target = strings.Repeat("t", 4096)
	long.Selection.Frame = strings.Repeat("f", 4096)
	long.Load.Target, long.Load.Loader = long.Selection.Target, strings.Repeat("l", 4096)
	pending, _ = nextLoadScope(loadState{}, loadInvalidate{Selection: long.Selection})
	scopeTestReserve(t, pending, long)
}

func TestLoadScopeSupersededCommitCannotChangeSuccessor(t *testing.T) {
	old, reservation := scopeTestReserve(t, scopeTestPending(), scopeTestEvidence())
	next, effects := nextLoadScope(old, loadInvalidate{Selection: loadSelection{Session: "new-session", Target: "new-target", Frame: "new-frame"}})
	if next.phase != loadPending || next.generation != 2 || !reflect.DeepEqual(effects, []loadEffect{loadCancelSnapshot{Generation: 1}, loadPublishPending{Generation: 2}}) {
		t.Fatalf("navigation did not supersede reserved write: %#v %#v", next, effects)
	}
	for _, event := range []loadEvent{loadCommitDone{Reservation: reservation, Success: true}, loadCommitDone{Reservation: reservation}, loadSnapshotFailed{Generation: 1}, loadSnapshotReady{Generation: 1, Evidence: scopeTestEvidence()}} {
		unchanged, effects := nextLoadScope(next, event)
		if !reflect.DeepEqual(unchanged, next) || len(effects) != 0 {
			t.Fatalf("stale event changed successor: %T %#v", event, effects)
		}
	}
}

func TestLoadScopeFailuresAndCloseAreFailClosed(t *testing.T) {
	pending := scopeTestPending()
	committing, reservation := scopeTestReserve(t, pending, scopeTestEvidence())
	for _, input := range []struct {
		state loadState
		event loadEvent
	}{{pending, loadSnapshotFailed{Generation: 1}}, {committing, loadCommitDone{Reservation: reservation}}} {
		next, effects := nextLoadScope(input.state, input.event)
		if next.phase != loadUnavailable || next.load != nil || !reflect.DeepEqual(effects, []loadEffect{loadPublishUnavailable{Generation: 1}}) {
			t.Fatalf("failure retained load authority: %#v %#v", next, effects)
		}
		retry, _ := nextLoadScope(next, loadInvalidate{Selection: scopeTestEvidence().Selection})
		if retry.phase != loadPending || retry.generation != 2 {
			t.Fatal("explicit retry cannot reestablish pending scope")
		}
	}
	closed, effects := nextLoadScope(committing, loadClose{})
	if closed.phase != loadClosed || !reflect.DeepEqual(effects, []loadEffect{loadCancelSnapshot{Generation: 1}, loadPublishUnavailable{Generation: 1}}) {
		t.Fatalf("close did not cancel and clear scope: %#v %#v", closed, effects)
	}
	for _, event := range []loadEvent{loadClose{}, loadInvalidate{}, loadSnapshotReady{Generation: 1, Evidence: scopeTestEvidence()}, loadSnapshotFailed{Generation: 1}, loadCommitDone{Reservation: reservation, Success: true}} {
		next, effects := nextLoadScope(closed, event)
		if !reflect.DeepEqual(next, closed) || len(effects) != 0 {
			t.Fatal("closed scope resurrected")
		}
	}
	last := loadState{phase: loadPending, generation: math.MaxUint64}
	next, _ := nextLoadScope(last, loadInvalidate{})
	if next.generation < last.generation || next.phase != loadClosed {
		t.Fatal("generation exhaustion wrapped into older authority")
	}
}

func TestLoadScopeGeneratedSequences(t *testing.T) {
	random := rand.New(rand.NewSource(857))
	totalReserved, totalConfirmed, totalClosed := 0, 0, 0
	for round := 0; round < 128; round++ {
		state := loadState{}
		var reservations []*loadReservation
		reserved := map[uint64]int{}
		for step := 0; step < 256; step++ {
			generation := state.generation
			if random.Intn(3) == 0 && generation > 0 {
				generation--
			}
			var event loadEvent
			switch random.Intn(6) {
			case 0:
				event = loadInvalidate{Selection: scopeTestEvidence().Selection}
			case 1:
				event = loadSnapshotReady{Generation: generation, Evidence: scopeTestEvidence()}
			case 2:
				event = loadSnapshotReady{Generation: generation}
			case 3:
				event = loadSnapshotFailed{Generation: generation}
			case 4:
				var reservation *loadReservation
				if len(reservations) > 0 {
					reservation = reservations[random.Intn(len(reservations))]
				}
				event = loadCommitDone{Reservation: reservation, Success: random.Intn(2) == 0}
			case 5:
				if step > 240 {
					event = loadClose{}
				} else {
					event = loadSnapshotFailed{Generation: generation + 1}
				}
			}
			next, effects := nextLoadScope(state, event)
			if next.generation < state.generation {
				t.Fatal("generation regressed")
			}
			if state.phase == loadClosed && (!reflect.DeepEqual(state, next) || len(effects) != 0) {
				t.Fatal("closed is not terminal")
			}
			for _, effect := range effects {
				switch effect := effect.(type) {
				case loadCommitSnapshot:
					totalReserved++
					reserved[effect.Reservation.generation]++
					if reserved[effect.Reservation.generation] != 1 || state.phase != loadPending || next.phase != loadCommitting {
						t.Fatal("reservation not linearized once from pending")
					}
					reservations = append(reservations, effect.Reservation)
				case loadPublishConfirmed:
					totalConfirmed++
					done, ok := event.(loadCommitDone)
					if !ok || !done.Success || done.Reservation == nil || done.Reservation != state.reservation || state.phase != loadCommitting || effect.Generation != state.generation || effect.Load.Validate() != nil {
						t.Fatal("confirmation lacked successful current reserved commit")
					}
				}
			}
			if next.phase != loadConfirmed && next.load != nil {
				t.Fatal("unconfirmed state exposed load authority")
			}
			state = next
		}
		if state.phase == loadClosed {
			totalClosed++
		}
	}
	if totalReserved == 0 || totalConfirmed == 0 || totalClosed == 0 {
		t.Fatalf("generated traces did not exercise authority: reserved=%d confirmed=%d closed=%d", totalReserved, totalConfirmed, totalClosed)
	}
}
