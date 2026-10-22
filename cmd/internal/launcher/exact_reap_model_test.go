package launcher

import (
	"math/rand"
	"reflect"
	"syscall"
	"testing"
)

// Feed generated valid and adversarial event streams through the production
// reducer. The independent ledger tracks the prerequisites of every effect.
func checkExactTrace(t *testing.T, events []int) {
	t.Helper()
	keys := []ReapStep{{PID: 7, Identity: "a"}, {PID: 8, Identity: "b"}}
	state, _ := AdvanceExactReap(ExactReapState{}, exactReapEvent{kind: exactBegin, steps: keys})
	term, kill, acked, survivor := map[int]int{}, map[int]int{}, map[int]bool{}, map[int]bool{}
	var pending *exactReapEffect
	canceled := false
	for _, choice := range events {
		before := state
		event := exactReapEvent{kind: exactNext}
		switch choice {
		case 1, 2, 3, 4:
			if pending != nil && pending.kind == exactSignalEffect {
				event = exactReapEvent{kind: exactSignalReply, token: pending.token, index: pending.index, status: []string{"sent", "refused", "skipped", "failed"}[choice-1]}
				if choice == 1 && pending.signal == syscall.SIGTERM {
					acked[pending.step.PID] = true
				}
				pending = nil
			}
		case 5, 6, 7, 8:
			if pending != nil && pending.kind == exactProbeEffect {
				presence := exactAlive
				if choice == 6 {
					presence = exactAbsent
				}
				if choice == 7 {
					presence = exactUnknown
				}
				event = exactReapEvent{kind: exactObservation, token: pending.token, observations: []exactPresence{presence, presence}, deadline: choice == 8}
				if choice == 8 && pending.phase == exactTermWait {
					for _, row := range state.rows {
						if row.state == exactWaitingTerm {
							survivor[row.step.PID] = true
						}
					}
				}
				pending = nil
			}
		case 9:
			event = exactReapEvent{kind: exactCancel, detail: "canceled"}
			canceled = true
		case 10:
			event = exactReapEvent{kind: exactSignalReply, token: 999999, index: 0, status: "sent"}
		case 11:
			event = exactReapEvent{kind: exactObservation, token: 999999, observations: []exactPresence{exactAbsent, exactAbsent}, deadline: true}
		case 12:
			event = exactReapEvent{kind: exactBegin, steps: []ReapStep{{PID: 99, Identity: "foreign"}}}
		}
		next, effects := AdvanceExactReap(state, event)
		if !reflect.DeepEqual(before, state) {
			t.Fatal("reducer mutated input")
		}
		state = next
		if len(state.rows) != len(keys) {
			t.Fatal("key set changed")
		}
		for i, row := range state.rows {
			if row.step != keys[i] {
				t.Fatal("key changed")
			}
			if before.rows[i].state == exactRowDone && row.state != exactRowDone {
				t.Fatal("finished row reopened")
			}
		}
		for _, effect := range effects {
			if pending != nil {
				t.Fatal("overlapping effects")
			}
			copy := effect
			pending = &copy
			if effect.kind != exactSignalEffect {
				continue
			}
			if canceled {
				t.Fatal("post-cancel signal")
			}
			if effect.signal == syscall.SIGTERM {
				term[effect.step.PID]++
				if term[effect.step.PID] > 1 {
					t.Fatal("duplicate TERM")
				}
			} else {
				kill[effect.step.PID]++
				if kill[effect.step.PID] > 1 || !acked[effect.step.PID] || !survivor[effect.step.PID] {
					t.Fatal("KILL without acknowledged TERM, deadline and positive survivor")
				}
			}
		}
	}
}
func TestExactReapModelGeneratedAndExhaustive(t *testing.T) {
	for seed := int64(0); seed < 100; seed++ {
		rng := rand.New(rand.NewSource(seed))
		events := make([]int, 200)
		for i := range events {
			events[i] = rng.Intn(13)
		}
		checkExactTrace(t, events)
	}
	var walk func([]int, int)
	walk = func(events []int, left int) {
		if left == 0 {
			checkExactTrace(t, events)
			return
		}
		for e := 0; e < 13; e++ {
			walk(append(events, e), left-1)
		}
	}
	walk(nil, 4)
	// Reach escalation and replay duplicate events beyond the shallow exhaustive
	// prefix, including cancellation at every effect boundary.
	happy := []int{0, 1, 0, 1, 0, 8, 0, 1, 0, 1, 0, 6, 0}
	checkExactTrace(t, happy)
	for i := range happy {
		events := append([]int(nil), happy[:i]...)
		events = append(events, 9)
		events = append(events, happy[i:]...)
		checkExactTrace(t, events)
	}
}
