package couchcore

import "testing"

func TestParseLayoutAcceptsKnownValues(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Layout
	}{{"layout2", Layout2}, {"layout3", Layout3}} {
		got, err := ParseLayout(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("ParseLayout(%q) = %v, %v; want %v, nil", tc.in, got, err, tc.want)
		}
	}
}

// An absent field is every record written before #198, and those are layout2
// with certainty -- couch pinned layout2 from 2026-08-22 until then.
func TestParseLayoutTreatsEmptyAsLayout2(t *testing.T) {
	got, err := ParseLayout("")
	if err != nil || got != Layout2 {
		t.Fatalf("ParseLayout(%q) = %v, %v; want Layout2, nil", "", got, err)
	}
}

// ARCH-SECURE: a hand-edited or newer-version value must not coerce to a
// plausible default the guard would then trust.
func TestParseLayoutRejectsUnknown(t *testing.T) {
	if _, err := ParseLayout("layout9"); err == nil {
		t.Fatal("ParseLayout(\"layout9\") = nil error; want a refusal")
	}
}

// NormalizeLayout is the total variant for the row projection, which has no
// error return.
func TestNormalizeLayoutIsTotal(t *testing.T) {
	if got := NormalizeLayout(""); got != Layout2 {
		t.Fatalf("NormalizeLayout(\"\") = %v; want Layout2", got)
	}
	if got := NormalizeLayout("layout3"); got != Layout3 {
		t.Fatalf("NormalizeLayout(\"layout3\") = %v; want Layout3", got)
	}
	if got := NormalizeLayout("layout9"); got != LayoutUnknown {
		t.Fatalf("NormalizeLayout(\"layout9\") = %v; want LayoutUnknown", got)
	}
}

func TestLayoutFlagIsTheOnlyFormatter(t *testing.T) {
	if Layout2.Flag() != "--layout2" || Layout3.Flag() != "--layout3" {
		t.Fatalf("flags = %q, %q; want --layout2, --layout3", Layout2.Flag(), Layout3.Flag())
	}
}

func layoutRow(tag string, state ActionableThreadState, layout Layout) ActionableThreadSummary {
	return ActionableThreadSummary{
		Address: ThreadAddress{RepoScope: "scope", Tag: ThreadTag(tag)},
		State:   state, Layout: layout,
	}
}

// The blocking set is the states that hold a zellij session, not Resumable().
// Every one of the six ActionableThreadState values is pinned here, so adding a
// seventh without deciding its disposition fails.
func TestBlockingSetIsExactlyTheSessionHoldingStates(t *testing.T) {
	for _, tc := range []struct {
		state ActionableThreadState
		block bool
	}{
		{ThreadLive, true}, {ThreadDetached, true}, {ThreadBusy, true},
		{ThreadParked, false}, {ThreadUnusable, false}, {ThreadArchived, false},
	} {
		rows := []ActionableThreadSummary{layoutRow("a", tc.state, Layout2)}
		got := ResolveLayoutConflicts(Layout3, rows)
		if (len(got) == 1) != tc.block {
			t.Fatalf("state %s: conflicts %+v; want block=%v", tc.state, got, tc.block)
		}
	}
}

func TestMatchingLayoutDoesNotConflict(t *testing.T) {
	rows := []ActionableThreadSummary{layoutRow("a", ThreadDetached, Layout3)}
	if got := ResolveLayoutConflicts(Layout3, rows); len(got) != 0 {
		t.Fatalf("same layout blocked startup: %+v", got)
	}
}

// LayoutUnknown cannot be proved to agree, so it conflicts with both requests.
func TestUnknownLayoutConflictsWithEveryRequest(t *testing.T) {
	for _, requested := range []Layout{Layout2, Layout3} {
		rows := []ActionableThreadSummary{layoutRow("a", ThreadDetached, LayoutUnknown)}
		if got := ResolveLayoutConflicts(requested, rows); len(got) != 1 {
			t.Fatalf("requested %v: got %+v; want one conflict", requested, got)
		}
	}
}

// Pins the predicate against the wrong-but-tempting Resumable()
// (Parked||Detached): swapping to it makes the parked row block and this fails.
func TestBlockingSetIsNotResumable(t *testing.T) {
	rows := []ActionableThreadSummary{
		layoutRow("parked", ThreadParked, Layout2),
		layoutRow("detached", ThreadDetached, Layout2),
	}
	got := ResolveLayoutConflicts(Layout3, rows)
	if len(got) != 1 || got[0].Address.Tag != ThreadTag("detached") {
		t.Fatalf("got %+v; want exactly the detached thread", got)
	}
}

// The conflict carries what the refusal message has to print.
func TestConflictCarriesLayoutAndState(t *testing.T) {
	rows := []ActionableThreadSummary{layoutRow("brain", ThreadDetached, Layout2)}
	got := ResolveLayoutConflicts(Layout3, rows)
	if len(got) != 1 || got[0].Layout != Layout2 || got[0].State != ThreadDetached {
		t.Fatalf("got %+v; want one {layout2, detached} conflict", got)
	}
}
