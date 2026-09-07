package couchcore

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

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

// The conformance check the whole feature rests on: `pair` must parse the flag
// couch emits. Until now the only evidence was a prose comment in couch.go
// saying it had been measured by hand once; this turns that claim into a test
// that fails if either side's spelling drifts.
//
// It is cheap because Layout aliases launcher.LayoutMode and Flag() lives in
// launcher beside the parser -- but the assertion is still worth making, since
// what matters is the round trip through ParseArgs, not the shared constant.
func TestCouchLayoutFlagsAreWhatPairParses(t *testing.T) {
	for _, layout := range []Layout{Layout2, Layout3} {
		args, err := launcher.ParseArgs([]string{"resume", "couch-0102030405060708", layout.Flag()})
		if err != nil {
			t.Fatalf("pair refused couch's own argv for %s: %v", layout, err)
		}
		if !args.Layout.Explicit || args.Layout.Mode != layout {
			t.Fatalf("pair parsed %q as %+v; want an explicit %s", layout.Flag(), args.Layout, layout)
		}
	}
}

// Flag() must be TOTAL: a Couch built by struct literal (which ~10 test files
// do) has a zero Layout, and a partial Flag() would emit a bare "--" into argv.
func TestFlagIsTotalOverUnsetAndUnknownLayouts(t *testing.T) {
	for _, layout := range []Layout{"", LayoutUnknown, Layout("garbage")} {
		if got := layout.Flag(); got != "--layout2" {
			t.Fatalf("Layout(%q).Flag() = %q; want the --layout2 default", layout, got)
		}
	}
}

func TestKnownLayoutRejectsWhatCouchCannotLaunch(t *testing.T) {
	for _, layout := range []Layout{Layout2, Layout3} {
		if !KnownLayout(layout) {
			t.Fatalf("KnownLayout(%s) = false", layout)
		}
	}
	for _, layout := range []Layout{"", LayoutUnknown, Layout("layout9")} {
		if KnownLayout(layout) {
			t.Fatalf("KnownLayout(%q) = true; it is not a layout couch can launch", layout)
		}
	}
}

// The gap that let `couch --unknown` ship, and then the gap that let a VACUOUS
// fix ship: the first version of this test only asserted the message was
// well-formed, which a total Flag() satisfies while still telling an operator
// blocked by a mixed set to run a couch that would itself refuse.
//
// So these assert the NEGATIVE direction -- what the message must NOT offer --
// which is the half that goes red when the remedy logic is removed.
func TestRefusalOffersNoHostRemedyWhenNoCouchCanHostThemAll(t *testing.T) {
	cases := map[string][]LayoutConflict{
		"blocking set that disagrees with itself": {
			{Address: ThreadAddress{Tag: "brain"}, Layout: Layout2, State: ThreadLive},
			{Address: ThreadAddress{Tag: "ariadne"}, Layout: Layout3, State: ThreadDetached},
		},
		"lone unreadable witness": {
			{Address: ThreadAddress{Tag: "brain"}, Layout: LayoutUnknown, State: ThreadDetached},
		},
		"unreadable mixed in": {
			{Address: ThreadAddress{Tag: "brain"}, Layout: Layout2, State: ThreadLive},
			{Address: ThreadAddress{Tag: "tools"}, Layout: LayoutUnknown, State: ThreadBusy},
		},
	}
	for name, conflicts := range cases {
		t.Run(name, func(t *testing.T) {
			message := layoutConflictRefusal(Layout3, conflicts).Error()
			// The ONLY layout flag allowed here is the one the operator asked
			// for. Naming any other means suggesting a couch that would refuse
			// for the same reason -- which is what a total Flag() made look
			// well-formed.
			for _, field := range strings.Fields(message) {
				if !strings.HasPrefix(field, "--layout") {
					continue
				}
				if field != Layout3.Flag() {
					t.Fatalf("refusal offers %q as a host, but no couch can host these threads:\n%s", field, message)
				}
			}
			if strings.Contains(message, "park them first") {
				t.Fatalf("refusal offers the one-pass park remedy, which does not apply:\n%s", message)
			}
		})
	}
}

// The unreadable case closes every in-tool route -- `park` is a TUI row action,
// so it needs a couch, and no couch will start. The refusal must leave the tool
// rather than dead-end.
func TestRefusalNamesAnOutOfToolRemedyWhenNoCouchCanStart(t *testing.T) {
	conflicts := []LayoutConflict{
		{Address: ThreadAddress{Tag: "brain"}, Layout: LayoutUnknown, State: ThreadDetached},
	}
	message := layoutConflictRefusal(Layout3, conflicts).Error()
	for _, want := range []string{"couch --show brain", "kill-session"} {
		if !strings.Contains(message, want) {
			t.Fatalf("refusal gives no reachable way out (missing %q):\n%s", want, message)
		}
	}
}

// Mixed-but-readable is reachable, just not in one pass: the message must say
// so rather than either offering a single host or giving up.
func TestRefusalExplainsThePerThreadRouteWhenLayoutsDisagree(t *testing.T) {
	conflicts := []LayoutConflict{
		{Address: ThreadAddress{Tag: "brain"}, Layout: Layout2, State: ThreadLive},
		{Address: ThreadAddress{Tag: "ariadne"}, Layout: Layout3, State: ThreadDetached},
	}
	message := layoutConflictRefusal(Layout3, conflicts).Error()
	if !strings.Contains(message, "DIFFERENT layouts") || !strings.Contains(message, "matching its own layout") {
		t.Fatalf("refusal does not explain the per-thread route:\n%s", message)
	}
}

// The ordinary case keeps its concrete two-step remedy -- dropping to generic
// advice everywhere would be a regression, not a fix.
func TestRefusalKeepsTheConcreteRemedyWhenOneHostExists(t *testing.T) {
	conflicts := []LayoutConflict{
		{Address: ThreadAddress{Tag: "brain"}, Layout: Layout2, State: ThreadLive},
		{Address: ThreadAddress{Tag: "pair"}, Layout: Layout2, State: ThreadDetached},
	}
	message := layoutConflictRefusal(Layout3, conflicts).Error()
	if !strings.Contains(message, "couch --layout2") || !strings.Contains(message, "couch --layout3") {
		t.Fatalf("refusal lost its two-step remedy:\n%s", message)
	}
}

// renderError already prefixes "couch: ", so the message must not repeat it.
func TestRefusalDoesNotDoubleTheProgramPrefix(t *testing.T) {
	conflicts := []LayoutConflict{{Address: ThreadAddress{Tag: "brain"}, Layout: Layout2, State: ThreadLive}}
	if message := layoutConflictRefusal(Layout3, conflicts).Error(); strings.HasPrefix(message, "couch:") {
		t.Fatalf("refusal prefixes the program name itself:\n%s", message)
	}
}
