package couchtty

import (
	"strings"
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

// The colouring carries the same claim more quietly: now.Sub(zero) saturated to
// AgeOld, so a thread with no recorded activity was painted as though it were
// ancient. Asserted through ageColor rather than on the band alone -- a band
// rendering identically to another is a distinction the operator cannot see.
//
// The rule the two findings converge on: unknown must be DISTINCT from every
// other band (or its test asserts the enum against itself) and must not be
// brighter than recent (or absence reads as recency). Nothing but its own dim
// satisfies both.
func TestAgeColourForAnUnknownAgeIsItsOwnAndNeverReadsAsRecent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	unknown := ageColor(AgeBandFor(now, time.Time{}))
	if unknown == "" {
		t.Fatal("an unknown age is unpainted, so it renders in the terminal default -- brighter than AgeRecent")
	}
	seen := map[string]AgeBand{unknown: AgeUnknown}
	for band, at := range map[AgeBand]time.Time{
		AgeRecent: now.Add(-time.Hour),
		AgeDays:   now.Add(-3 * 24 * time.Hour),
		AgeOld:    now.Add(-30 * 24 * time.Hour),
	} {
		colour := ageColor(AgeBandFor(now, at))
		if previous, clash := seen[colour]; clash {
			t.Errorf("band %v shares its colour %q with %v", band, colour, previous)
		}
		seen[colour] = band
	}
}

// The zero value is not the only timestamp that overflows time.Duration, which
// saturates at ~292 years. Guarding only the zero value would fix the reachable
// case and leave the arithmetic able to tell the same lie from a corrupt or
// hand-edited record.
func TestAnUnrenderablyOldTimeDoesNotResurrectTheSaturatedAge(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	// An absolute date, because 300 years cannot be written as a Duration at all
	// -- the compiler rejects the constant, which is the same overflow the render
	// path hits at runtime.
	ancient := time.Date(1500, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got := relativeMenuAge(now, ancient); strings.Contains(got, "106751") {
		t.Errorf("relativeMenuAge = %q -- the saturated value is back", got)
	}
	if got := relativeMenuAge(now, ancient); got != "long ago" {
		t.Errorf("relativeMenuAge = %q, want %q", got, "long ago")
	}
	// It is still an AGE, not an absence: the row keeps its clause.
	row := couchcore.ActionableThreadSummary{
		Address: menuAddress("brain"), Name: "brain",
		State: couchcore.ThreadDetached, LastActiveAt: ancient,
	}
	if got, want := rootStateText(row, now), "detached · long ago"; got != want {
		t.Errorf("rootStateText = %q, want %q", got, want)
	}
	if got := AgeBandFor(now, ancient); got != AgeOld {
		t.Errorf("AgeBandFor = %v, want AgeOld -- unrenderable is old, not unknown", got)
	}
}

// The colour is applied at the ROW, so the rule has to hold there too.
//
// ageColor returning "" is not enough: the render site wrapped unconditionally,
// so an unknown-age row became `"" + text + reset` — which paints in the
// terminal's DEFAULT and therefore brighter than AgeRecent's grey. "We do not
// know how old this is" then reads as "this is the most recent one", which is
// the original bug's shape reintroduced by its own fix.
func TestAnUnknownAgeRowIsNotPaintedLikeARecentOne(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	// TWO rows, and the assertion is on the UNSELECTED one: a selected row takes
	// selectedMenuLine and never reaches the age colouring at all, so a
	// single-row state would test the wrong branch. The header carries its own
	// escape too, which is why this reads one line rather than the whole render.
	selected := menuAddress("selected")
	subject := menuAddress("subject")
	rowFor := func(at time.Time) string {
		state := NewMenuState([]couchcore.ActionableThreadSummary{
			{Address: selected, Name: "selected", WorkingPath: "/w/a", State: couchcore.ThreadParked, LastActiveAt: now},
			{Address: subject, Name: "subject", WorkingPath: "/w/b", State: couchcore.ThreadParked, LastActiveAt: at},
		}, selected)
		for _, line := range strings.Split(RenderMenu(state, 60, 12, now, true), "\r\n") {
			if strings.Contains(line, "subject") {
				return line
			}
		}
		t.Fatal("the subject row was not rendered")
		return ""
	}

	unknown := rowFor(time.Time{})
	recent := rowFor(now.Add(-time.Hour))
	if !strings.Contains(unknown, "\x1b[38;5;") {
		t.Errorf("an unknown-age row is unpainted, so it renders brighter than a recent one: %q", unknown)
	}
	if unknownColour, recentColour := ansiColourOf(unknown), ansiColourOf(recent); unknownColour == recentColour {
		t.Errorf("an unknown-age row is painted like a recent one (%q); absence must not read as recency", unknownColour)
	}
}

// ansiColourOf returns the first 256-colour escape in a rendered line.
func ansiColourOf(line string) string {
	const prefix = "\x1b[38;5;"
	i := strings.Index(line, prefix)
	if i < 0 {
		return ""
	}
	if end := strings.Index(line[i:], "m"); end >= 0 {
		return line[i : i+end+1]
	}
	return ""
}
