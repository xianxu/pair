package couchtty

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// The operator saw `detached · 106751d ago`. 106751.99 days is math.MaxInt64
// nanoseconds: now.Sub(zeroTime) did not compute a large age, it OVERFLOWED and
// saturated. The row stated an age it did not have, to the day.
//
// The adversarial pair is the whole point: ONLY the zero value means absent.
// time.Unix(0,0) is 1970 — genuinely ancient, and it must still render an age.
// A guard written as "very old implies absent" passes the reported bug and
// fails this.
func TestHasRecordedActivityDistinguishesAbsenceFromAncient(t *testing.T) {
	for _, tc := range []struct {
		name string
		at   time.Time
		want bool
	}{
		{"the zero value is absence", time.Time{}, false},
		{"the unix epoch is a real, ancient timestamp", time.Unix(0, 0), true},
		{"a recent time is present", time.Unix(1_700_000_000, 0), true},
	} {
		if got := hasRecordedActivity(tc.at); got != tc.want {
			t.Errorf("%s: hasRecordedActivity(%v) = %v, want %v", tc.name, tc.at, got, tc.want)
		}
	}
}

// Tested at the ROW, not only at the helper: the helper can be correct while
// the row still lies, which is exactly how this shipped.
func TestRootStateTextOmitsTheAgeItDoesNotHave(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	row := func(state couchcore.ActionableThreadState, at time.Time) couchcore.ActionableThreadSummary {
		return couchcore.ActionableThreadSummary{
			Address: menuAddress("brain"), Name: "brain", State: state, LastActiveAt: at,
		}
	}
	for _, tc := range []struct {
		name  string
		state couchcore.ActionableThreadState
		at    time.Time
		want  string
	}{
		{"detached with no recorded activity", couchcore.ThreadDetached, time.Time{}, "detached"},
		{"parked with no recorded activity", couchcore.ThreadParked, time.Time{}, "parked"},
		{"detached two hours ago", couchcore.ThreadDetached, now.Add(-2 * time.Hour), "detached · 2h ago"},
		// 1970 is not absence. It is an age, and a big one.
		{"detached at the epoch", couchcore.ThreadDetached, time.Unix(0, 0), "detached · 19675d ago"},
		// Clock skew: a future timestamp clamps rather than rendering negative.
		{"detached in the future", couchcore.ThreadDetached, now.Add(time.Hour), "detached · <1h ago"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := rootStateText(row(tc.state, tc.at), now); got != tc.want {
				t.Errorf("rootStateText = %q, want %q", got, tc.want)
			}
		})
	}
}

// The colouring carries the same claim more quietly: now.Sub(zero) saturates to
// AgeOld, so a thread with no recorded activity is painted as though it were
// ancient. Asserted through ageColor rather than on the band alone -- a band
// that renders identically to another band is a distinction the operator cannot
// see, and a test of it would only assert the enum against itself.
func TestAgeColourSaysNothingWhenThereIsNoAge(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	unknown := ageColor(AgeBandFor(now, time.Time{}))
	if unknown != "" {
		t.Errorf("a row with no recorded activity was coloured %q; absence has no age to paint", unknown)
	}
	ancient := ageColor(AgeBandFor(now, time.Unix(0, 0)))
	if ancient == "" || ancient == unknown {
		t.Errorf("the epoch is a real age and must be coloured (got %q, unknown %q)", ancient, unknown)
	}
	recent := ageColor(AgeBandFor(now, now.Add(-time.Hour)))
	if recent == ancient {
		t.Errorf("recent and ancient share a colour: %q", recent)
	}
	// Band boundaries, pinned so the guard cannot quietly move them.
	if got := AgeBandFor(now, now.Add(-25*time.Hour)); got != AgeDays {
		t.Errorf("25h ago = %v, want AgeDays", got)
	}
	if got := AgeBandFor(now, now.Add(-8*24*time.Hour)); got != AgeOld {
		t.Errorf("8d ago = %v, want AgeOld", got)
	}
}
