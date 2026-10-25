package couchcmd

import (
	"errors"
	"testing"
)

func TestProfileLifecycleOwnership(t *testing.T) {
	first := errors.New("first write")
	later := errors.New("later close")
	for _, tc := range []struct {
		name      string
		events    []profileEvent
		want      []profileEffect
		cancelled bool
		failure   error
	}{
		{"complete", []profileEvent{{kind: profileWarmupElapsed}, {kind: profileAcquired}, {kind: profileCaptureElapsed}, {kind: profileReleased}}, []profileEffect{profileAcquire, profileNoEffect, profileReleaseOwned, profileComplete}, false, nil},
		{"warmup cancellation", []profileEvent{{kind: profileCancel}, {kind: profileReleased}}, []profileEffect{profileReleaseUnowned, profileComplete}, true, nil},
		{"cancel during successful acquisition", []profileEvent{{kind: profileWarmupElapsed}, {kind: profileAcquiredCancelled}, {kind: profileReleased}}, []profileEffect{profileAcquire, profileReleaseOwned, profileComplete}, true, nil},
		{"cancel before cpu acquisition", []profileEvent{{kind: profileWarmupElapsed}, {kind: profileCancel}, {kind: profileReleased}}, []profileEffect{profileAcquire, profileReleaseUnowned, profileComplete}, true, nil},
		{"busy owner", []profileEvent{{kind: profileWarmupElapsed}, {kind: profileAcquireFailed, err: first}, {kind: profileReleased, err: later}}, []profileEffect{profileAcquire, profileReleaseUnowned, profileComplete}, false, first},
		{"release failure", []profileEvent{{kind: profileWarmupElapsed}, {kind: profileAcquired}, {kind: profileCancel}, {kind: profileReleased, err: first}}, []profileEffect{profileAcquire, profileNoEffect, profileReleaseOwned, profileComplete}, true, first},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := ProfileLifecycle{}
			for i, event := range tc.events {
				next, effect, err := StepProfile(state, event)
				if err != nil || effect != tc.want[i] {
					t.Fatalf("step%d effect=%v err=%v", i, effect, err)
				}
				state = next
			}
			if state.phase != profileFinished || state.cancelled != tc.cancelled || !errors.Is(state.failure, tc.failure) {
				t.Fatalf("final=%+v", state)
			}
			for _, event := range []profileEvent{{kind: profileCancel}, {kind: profileCaptureElapsed}, {kind: profileReleased, err: later}} {
				next, effect, err := StepProfile(state, event)
				if err != nil || effect != profileNoEffect || next != state {
					t.Fatalf("late event changed completion: %+v %v %v", next, effect, err)
				}
			}
		})
	}
}

func TestProfileLifecycleRejectsUnownedRelease(t *testing.T) {
	// Exhaust all event sequences up to six steps. Track ownership independently
	// from the reducer's representation, including invalid event orderings.
	var visit func(ProfileLifecycle, bool, int, int, int)
	visit = func(state ProfileLifecycle, owned bool, starts, stops, depth int) {
		if depth == 0 {
			return
		}
		for kind := profileWarmupElapsed; kind <= profileReleased; kind++ {
			event := profileEvent{kind: kind}
			if kind == profileAcquireFailed {
				event.err = errors.New("busy")
			}
			next, effect, err := StepProfile(state, event)
			if err != nil {
				continue
			}
			o, a, r := owned, starts, stops
			if state.phase != profileFinished && (kind == profileAcquired || kind == profileAcquiredCancelled) {
				o = true
				a++
			}
			if effect == profileReleaseOwned {
				if !o {
					t.Fatal("released an unowned profiler")
				}
				o = false
				r++
			}
			if a > 1 || r > 1 || r > a {
				t.Fatalf("ownership starts=%d stops=%d", a, r)
			}
			if next.phase == profileFinished && o {
				t.Fatal("finished with acquired ownership")
			}
			visit(next, o, a, r, depth-1)
		}
	}
	visit(ProfileLifecycle{}, false, 0, 0, 6)
}
