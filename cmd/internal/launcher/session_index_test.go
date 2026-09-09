package launcher

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAssignSessionNameUsesReadableBaseWhenFree(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	name, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "📁pair-work" {
		t.Fatalf("name = %q, want 📁pair-work", name)
	}
	if strings.Contains(name, scope.Key) {
		t.Fatalf("name %q exposed hidden key %q", name, scope.Key)
	}
}

func TestAssignSessionNameDisambiguatesSameRepoNameDifferentScope(t *testing.T) {
	first := mustScope(t, "/Users/a/work/pair")
	second := mustScope(t, "/tmp/other/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁pair-work",
		ScopeKey:    first.Key,
		RepoRoot:    first.Root,
		RepoName:    first.DisplayName,
		Tag:         "work",
	}}}

	name, _, err := AssignSessionName(index, []Session{{Name: "📁pair-work", State: SessionDetached}}, second, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "📁pair-work-2" {
		t.Fatalf("name = %q, want 📁pair-work-2", name)
	}
}

func TestAssignSessionNameReusesSameScopeBinding(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁pair-work-2",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "work",
	}}}

	name, _, err := AssignSessionName(index, []Session{{Name: "📁pair-work-2", State: SessionDetached}}, scope, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "📁pair-work-2" {
		t.Fatalf("name = %q, want prior binding", name)
	}
}

// The transition, and the reason the reuse short-circuit is prefix-gated (#130):
// a legacy binding must NOT be reused, or the new scheme never reaches any tag
// that already has a ledger row.
func TestAssignSessionNameMigratesLegacyBinding(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "pair-📁work-2",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "work",
	}}}

	name, updated, err := AssignSessionName(index, []Session{{Name: "pair-📁work-2", State: SessionExited}}, scope, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "📁pair-work" {
		t.Fatalf("name = %q, want the migrated 📁pair-work", name)
	}
	entry := updated.Entries[len(updated.Entries)-1]
	if entry.Superseded != "pair-📁work-2" {
		t.Fatalf("Superseded = %q, want the legacy name it replaced", entry.Superseded)
	}

	// Second create must NOT append again: the fresh row is already 📁-prefixed,
	// so it short-circuits. Without the prefix gate this grows without bound.
	before := len(updated.Entries)
	name2, updated2, err := AssignSessionName(updated, nil, scope, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("second AssignSessionName: %v", err)
	}
	if name2 != "📁pair-work" {
		t.Fatalf("second name = %q, want the same migrated name", name2)
	}
	if len(updated2.Entries) != before {
		t.Fatalf("ledger grew from %d to %d entries on a repeat create", before, len(updated2.Entries))
	}
}

func TestAssignSessionNameShortensOverlongReadableName(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/repositorywithaverylongname")
	name, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "featurewithaverylongname", func(s string) bool {
		return len(s) <= 24
	})
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if len(name) > 24 {
		t.Fatalf("name = %q len=%d, want <= 24", name, len(name))
	}
	if strings.Contains(name, scope.Key) {
		t.Fatalf("name %q exposed hidden key %q", name, scope.Key)
	}
	if !strings.HasPrefix(name, "📁") {
		t.Fatalf("name = %q, want the 📁 prefix", name)
	}
}

func TestAssignSessionNameErrorsWhenNoCandidateFits(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	if _, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "work", func(string) bool { return false }); err == nil {
		t.Fatal("AssignSessionName returned nil error")
	}
}

func TestSessionsForScopeFiltersAndAnnotatesIndexedSessions(t *testing.T) {
	pair := mustScope(t, "/Users/a/work/pair")
	other := mustScope(t, "/tmp/other/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "📁pair-work", ScopeKey: pair.Key, RepoRoot: pair.Root, RepoName: pair.DisplayName, Tag: "work"},
		{SessionName: "📁pair-work-2", ScopeKey: other.Key, RepoRoot: other.Root, RepoName: other.DisplayName, Tag: "work"},
	}}
	sessions := []Session{
		{Name: "📁pair-work", State: SessionDetached},
		{Name: "📁pair-work-2", State: SessionDetached},
		{Name: "pair-legacy", State: SessionDetached},
	}

	got := SessionsForScope(sessions, index, pair)
	if len(got) != 1 {
		t.Fatalf("SessionsForScope returned %#v, want one current-scope session", got)
	}
	if got[0].Name != "📁pair-work" || got[0].Tag != "work" || got[0].RepoName != "pair" {
		t.Fatalf("session = %#v, want annotated current-scope work", got[0])
	}
}

func TestSessionNameIndexRoundTripSkipsMalformedRows(t *testing.T) {
	entry := SessionNameEntry{
		SessionName: "📁pair-work",
		ScopeKey:    "scope1",
		RepoRoot:    "/repo",
		RepoName:    "pair",
		Tag:         "work",
	}
	line, err := BuildSessionNameIndexLine(entry)
	if err != nil {
		t.Fatalf("BuildSessionNameIndexLine: %v", err)
	}
	for _, want := range []string{`"session_name"`, `"scope_key"`, `"repo_root"`, `"repo_name"`, `"tag"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("index line = %s, want key %s", line, want)
		}
	}
	index := ParseSessionNameIndex(line + "\nnot-json\n")
	if len(index.Entries) != 1 {
		t.Fatalf("entries = %#v, want one valid entry", index.Entries)
	}
	if index.Entries[0] != entry {
		t.Fatalf("entry = %#v, want %#v", index.Entries[0], entry)
	}
}

func acceptAllSessionNames(string) bool { return true }

func mustScope(t *testing.T, root string) RepoScope {
	t.Helper()
	scope, err := ResolveRepoScope(root)
	if err != nil {
		t.Fatalf("ResolveRepoScope(%q): %v", root, err)
	}
	return scope
}

// Name assignment must cost a BOUNDED number of zellij probes, not one per
// candidate (#215).
//
// Every probe is a subprocess. The candidate count grows with every couch thread
// the repo has ever had -- nothing releases a suffix on archive -- and the walk
// restarts at suffix 1 each time, so a repo owning suffixes 1..25 pays two probes
// per suffix (the over-long 16-hex candidate rejected for length, then the short
// one found owned). Measured at 52 against the real index, inside couch's 5s
// registration deadline, on a machine where one `zellij action` round-trip goes
// from 17.6ms calm to 467ms under load.
//
// The fix is arithmetic: the socket budget is a LOCAL fact, so a candidate's
// length can be judged without asking zellij. The probe stays the oracle for the
// BUDGET, which is measured once.
func TestAssignSessionNameCostsABoundedNumberOfProbes(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	const tag = "couch-7bd0c2986975082c"

	// seedOwned builds the index an aged repo has: suffixes 1..n already taken
	// by other tags. Nothing releases a suffix on archive, so this only grows.
	seedOwned := func(n int) SessionNameIndex {
		var index SessionNameIndex
		for suffix := 1; suffix <= n; suffix++ {
			for _, candidate := range BuildSessionNameCandidates(scope, tag, suffix) {
				if len(candidate) <= 24 { // the macOS budget; the short rung
					index.Entries = append(index.Entries, SessionNameEntry{
						SessionName: candidate,
						ScopeKey:    scope.Key,
						RepoRoot:    scope.Root,
						RepoName:    scope.DisplayName,
						Tag:         "someone-else-" + candidate,
					})
					break
				}
			}
		}
		return index
	}

	assign := func(t *testing.T, owned int) (string, int) {
		t.Helper()
		// The PRODUCTION acceptor, not a hand-rolled one: what must be bounded is
		// zellij SUBPROCESSES, and a test counting `accepts` calls would measure
		// ladder iterations instead -- passing or failing for a reason unrelated
		// to the cost this issue is about.
		rt := &fakeRuntime{maxSessionNameBytes: 24}
		name, _, err := AssignSessionName(seedOwned(owned), nil, scope, tag, singleUseAcceptor(rt))
		if err != nil {
			t.Fatalf("AssignSessionName(owned=%d) returned error: %v", owned, err)
		}
		if name == "" {
			t.Fatalf("AssignSessionName(owned=%d) assigned no name", owned)
		}
		return name, rt.probeCount
	}

	name25, probes25 := assign(t, 25)
	name60, probes60 := assign(t, 60)

	// A small constant. The budget is found by binary search over [13,64];
	// anything near the candidate count means the per-candidate probe is back.
	const budget = 12
	if probes25 > budget {
		t.Errorf("assigning %q cost %d zellij probes; want <= %d. A subprocess per "+
			"candidate makes startup O(threads-this-repo-ever-had), which is what "+
			"blows couch's registration deadline (#215)", name25, probes25, budget)
	}

	// O(1), stated as invariance rather than as a bound: the cost must not move
	// with the index at all. A bound alone would still pass an implementation
	// that grew slowly, and this index only ever grows.
	if probes25 != probes60 {
		t.Errorf("probe count moved with index size: %d at 25 owned suffixes, %d at 60 "+
			"(names %q and %q). Name assignment must not be O(threads)",
			probes25, probes60, name25, name60)
	}
}

// The RESUME path must stay at one probe (#215).
//
// Its companion above bounds the COLD path, and satisfying that one eagerly --
// discovering the byte budget up front -- made this one SEVEN times worse:
// the ledger short-circuit asks about a single name, and it was paying for a
// binary search it had no use for. Measured, 1 -> 7, and shipped, because
// nothing asserted the cheap path stayed cheap.
//
// Two tests, because "bounded" and "cheap" are different claims and a fix for
// either can silently pay for it out of the other.
func TestResumingAKnownThreadCostsOneProbe(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/brain")
	const tag = "couch-e1a31510b7033d08"
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁brain-couch-23",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         tag,
	}}}

	rt := &fakeRuntime{maxSessionNameBytes: 24}
	name, _, err := AssignSessionName(index, nil, scope, tag, singleUseAcceptor(rt))
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "📁brain-couch-23" {
		t.Fatalf("name = %q, want the ledger's own answer", name)
	}
	if rt.probeCount != 1 {
		t.Errorf("resuming a thread whose name is already in the ledger cost %d probes, "+
			"want exactly 1: it asks about ONE name, so anything more is a budget "+
			"search it has no use for", rt.probeCount)
	}
}

// The acceptor must never accept a name zellij would refuse (#215 BR-1).
//
// BR-1 was a guess used as an oracle: discoverSessionNameBudget falls back to
// defaultSessionNameBudget when it cannot measure, and judging candidates against
// that number would accept every rung under 24 bytes on a machine where zellij
// refuses all of them -- so assignment returned the LONGEST remaining rung and
// pair advised "pick a shorter tag", which the ladder is what implements. A
// machine where pair worked could not start a session at all.
//
// The bracket removed the guess rather than marking it: every answer now derives
// from a probe of THIS machine. So the durable assertion is soundness itself --
// the acceptor must agree with the probe on every name -- which subsumes BR-1 and
// also pins the monotonicity the bracket rests on (a session name is a socket
// filename, so if an n-byte name fits, every shorter one does).
//
// An earlier version of this test asserted a PROBE COUNT (">= 20, because with no
// measurable budget every candidate must be probed"). That was a proxy for "does
// not trust a fallback", and the bracket makes it false while making the property
// it stood for structural: 2 probes now, both real observations.
func TestTheAcceptorNeverDisagreesWithTheProbe(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	for _, budget := range []int{1, 11, 24, 40, 200} {
		t.Run(fmt.Sprintf("budget=%d", budget), func(t *testing.T) {
			oracle := &fakeRuntime{maxSessionNameBytes: budget}
			accepts := singleUseAcceptor(&fakeRuntime{maxSessionNameBytes: budget})
			for suffix := 1; suffix <= 30; suffix++ {
				for _, candidate := range BuildSessionNameCandidates(scope, "couch-e1a31510b7033d08", suffix) {
					want := oracle.ProbeSessionName(candidate) == nil
					if got := accepts(candidate); got != want {
						t.Fatalf("acceptor said %v for a %d-byte name, zellij says %v: %q",
							got, len(candidate), want, candidate)
					}
				}
			}
		})
	}
}

// And an assignment that cannot succeed must fail HONESTLY rather than hand back
// a name zellij will refuse.
func TestAnImpossibleBudgetExhaustsInsteadOfGuessing(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	rt := &fakeRuntime{maxSessionNameBytes: 1} // nothing fits at all
	_, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "work", singleUseAcceptor(rt))
	var exhausted SessionNameExhausted
	if !errors.As(err, &exhausted) {
		t.Fatalf("err = %v (%T), want SessionNameExhausted: a machine where zellij "+
			"accepts no name must fail honestly, not return a name it will refuse", err, err)
	}
}

// O(1) probes in BOTH budget regimes (#215 BR-18).
//
// The first fix went arithmetic only AFTER a rejection, which is O(1) only where
// some candidate is actually refused. On a machine whose socket directory is
// short enough that the longest candidate fits -- Linux `~/.cache/zellij` against
// macOS's temp path -- nothing is ever refused and it paid one probe per suffix:
// the O(threads) cost this issue exists to remove, quietly restored on the
// machines with the most headroom. Code, atlas and Done-when all claimed O(1)
// unconditionally.
//
// So both regimes, and invariance rather than a bound in each.
func TestProbeCountIsInvariantInBothBudgetRegimes(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/brain")
	const tag = "couch-e1a31510b7033d08"

	seed := func(owned, budget int) SessionNameIndex {
		var idx SessionNameIndex
		for suffix := 1; suffix <= owned; suffix++ {
			for _, c := range BuildSessionNameCandidates(scope, tag, suffix) {
				if len(c) <= budget {
					idx.Entries = append(idx.Entries, SessionNameEntry{
						SessionName: c, ScopeKey: scope.Key, RepoRoot: scope.Root,
						RepoName: scope.DisplayName, Tag: "other-" + c,
					})
					break
				}
			}
		}
		return idx
	}
	assign := func(owned, budget int) int {
		rt := &fakeRuntime{maxSessionNameBytes: budget}
		if _, _, err := AssignSessionName(seed(owned, budget), nil, scope, tag,
			singleUseAcceptor(rt)); err != nil {
			t.Fatalf("budget=%d owned=%d: %v", budget, owned, err)
		}
		return rt.probeCount
	}

	for _, tc := range []struct {
		name   string
		budget int
	}{
		{"tight budget: the longest candidates are refused", 24},
		{"generous budget: nothing is ever refused", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			few, many := assign(25, tc.budget), assign(60, tc.budget)
			if few != many {
				t.Errorf("probe count moved with index size: %d at 25 owned suffixes, %d at "+
					"60. Assignment must not be O(threads-this-repo-ever-had) in ANY budget "+
					"regime", few, many)
			}
			if few > 12 {
				t.Errorf("%d probes; want a small constant", few)
			}
		})
	}
}
