package draftroute

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type cancelAfterRouteEffect struct {
	*fakeRuntime
	cancel context.CancelFunc
	after  int
}

func (r cancelAfterRouteEffect) RunZellijAction(args ...string) error {
	err := r.fakeRuntime.RunZellijAction(args...)
	if len(r.ops) == r.after {
		r.cancel()
	}
	return err
}

func TestDraftEffectSequenceStopsAfterCancellation(t *testing.T) {
	// Focus plus every addressed input action is its own dispatch boundary.
	for _, after := range []int{0, 1, 2, 3, 4, 5, -1} {
		t.Run(fmt.Sprint(after), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt := cancelAfterRouteEffect{fakeRuntime: &fakeRuntime{}, cancel: cancel, after: after}
			if after == 0 {
				cancel()
			}
			err := RouteLuaToPaneContext(ctx, rt, "2", "PairConfirmQuit", true)
			want := after
			if want < 0 {
				want = 5
			}
			if len(rt.ops) != want {
				t.Fatalf("committed=%v, want %d", rt.ops, want)
			}
			if after >= 0 && after < 5 {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation=%v", err)
				}
			} else if err != nil {
				t.Fatalf("issued final result changed: %v", err)
			}
		})
	}
}
