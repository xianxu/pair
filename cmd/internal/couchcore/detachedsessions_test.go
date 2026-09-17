package couchcore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestProjectDetachedSessions(t *testing.T) {
	one := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	two := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000002"}

	tests := []struct {
		name     string
		bindings []SessionNameBinding
		sessions []launcher.Session
		want     []DetachedSessionObservation
	}{
		{
			name:     "a live session with no client is detached",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one", Agent: "claude"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
			want:     []DetachedSessionObservation{{Address: one, SessionName: "pair-one", Agent: "claude"}},
		},
		{
			name:     "an attached session is not detached",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionAttached}},
		},
		{
			name:     "an exited session is not detached",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionExited}},
		},
		{
			name:     "a bound name with no session at all yields nothing",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-other", State: launcher.SessionDetached}},
		},
		{
			name:     "an unbound session is not attributed to any thread",
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
		},
		{
			name: "each bound address is judged independently",
			bindings: []SessionNameBinding{
				{Address: one, SessionName: "pair-one", Agent: "claude"},
				{Address: two, SessionName: "pair-two", Agent: "claude"},
			},
			sessions: []launcher.Session{
				{Name: "pair-one", State: launcher.SessionAttached},
				{Name: "pair-two", State: launcher.SessionDetached},
			},
			want: []DetachedSessionObservation{{Address: two, SessionName: "pair-two", Agent: "claude"}},
		},
		{
			name:     "an empty session name is never a binding",
			bindings: []SessionNameBinding{{Address: one, SessionName: ""}},
			sessions: []launcher.Session{{Name: "", State: launcher.SessionDetached}},
		},
		{
			// Fail closed: two rows claiming one name cannot both be that
			// session, and couch cannot tell which is right.
			name: "an ambiguous session name yields nothing for either address",
			bindings: []SessionNameBinding{
				{Address: one, SessionName: "pair-shared"},
				{Address: two, SessionName: "pair-shared"},
			},
			sessions: []launcher.Session{{Name: "pair-shared", State: launcher.SessionDetached}},
		},
		{
			// Two zellij rows with one name is a state couch cannot resolve.
			name:     "a duplicated session row yields nothing",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{
				{Name: "pair-one", State: launcher.SessionDetached},
				{Name: "pair-one", State: launcher.SessionAttached},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ProjectDetachedSessions(test.bindings, test.sessions, claimsOf(test.bindings))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("ProjectDetachedSessions() = %+v, want %+v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("ProjectDetachedSessions() = %+v, want %+v", got, test.want)
				}
			}
		})
	}
}

// The refresh's detached observation costs 2 + C zellij subprocesses
// (list-sessions twice, plus one list-clients per CANDIDATE session since
// pair#228), so what keeps it proportional is asking ONLY about records that
// could be detached. This
// pins that bound directly, because a benchmark of the pure reducer cannot see
// it -- the query lives on the refresh worker, not the keystroke path.
func TestActionableInventoryAsksOnlyAboutDetachCandidates(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	profile := &LaunchProfile{Agent: "claude", Argv: []string{}}

	newRecord := func(tag string, mutate func(*ThreadRecord)) ThreadAddress {
		t.Helper()
		seed := validThreadRecord(t)
		seed.Address.Tag = ThreadTag(tag)
		seed.StartingPath, seed.WorkingPath = ns.Dir(), ns.Dir()
		created, err := store.CreateThread(seed)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
			next.Reservation = false
			next.LatestLaunchProfile = profile
			mutate(next)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return created.Address
	}

	candidate := newRecord("couch-0000000000000001", func(*ThreadRecord) {})
	// Carries an incarnation. It USED to be excluded from the session question
	// on the grounds that it "cannot be detached" -- which is exactly the
	// assumption #272 disproves: the launcher dies with couch while the session
	// does not, so this is the record that most needs asking about.
	occupied := newRecord("couch-0000000000000002", func(r *ThreadRecord) {
		r.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 5, Identity: "id-5", StartedAt: time.Unix(2, 0).UTC()}}
	})
	// No profile: still asked about, because presence is one host-wide call and
	// bounding it per-record buys nothing.
	noProfile := newRecord("couch-0000000000000003", func(r *ThreadRecord) { r.LatestLaunchProfile = nil })

	artifacts := NewFakeThreadArtifactCollisionChecker()
	var asked [][]ThreadAddress
	artifacts.SessionPresenceHook = func(addresses []ThreadAddress) error {
		asked = append(asked, addresses)
		return nil
	}
	// The refresh must no longer reach the CLIENT-counting query at all: that
	// is the optimistic-inventory trade, and a `list-clients` costs ~250 ms per
	// live session (#228).
	var detachedQueries int
	artifacts.DetachedSessionsHook = func([]ThreadAddress) error {
		detachedQueries++
		return nil
	}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	if _, err := couch.ActionableThreadInventoryContext(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Fatalf("SessionPresence called %d times, want exactly one host-wide query", len(asked))
	}
	if detachedQueries != 0 {
		t.Fatalf("the refresh ran %d client-counting queries; it must ask none", detachedQueries)
	}
	got := map[ThreadAddress]bool{}
	for _, address := range asked[0] {
		got[address] = true
	}
	for _, want := range []ThreadAddress{candidate, occupied, noProfile} {
		if !got[want] {
			t.Errorf("presence was not asked about %+v; it must cover EVERY record", want)
		}
	}
}

// With no candidates at all, the refresh must not spawn the query.
func TestActionableInventorySkipsTheQueryWithNoCandidates(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	seed := validThreadRecord(t)
	seed.StartingPath, seed.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(seed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 5, Identity: "id-5", StartedAt: time.Unix(2, 0).UTC()}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	artifacts := NewFakeThreadArtifactCollisionChecker()
	called := 0
	artifacts.DetachedSessionsHook = func([]ThreadAddress) error {
		called++
		return nil
	}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}
	if _, err := couch.ActionableThreadInventoryContext(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if called != 0 {
		t.Fatalf("DetachedSessions was called %d times with no candidates", called)
	}
}

// The pure function must emit observations the pure PROJECTOR accepts.
//
// It used to emit {Address, SessionName} only, which
// ProjectActionableThreads -- once it started enforcing the resume proof --
// always rejected. Production worked anyway because the IO shell patched Agent
// onto the answer afterwards, which meant this function's own
// tests asserted a shape nothing downstream would take. Composing the two pure
// functions is the guard.
func TestProjectDetachedSessionsEmitsObservationsTheProjectorAccepts(t *testing.T) {
	address := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	bindings := []SessionNameBinding{{Address: address, SessionName: "pair-one", Agent: "claude"}}
	observed, err := ProjectDetachedSessions(
		bindings,
		[]launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
		claimsOf(bindings),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 {
		t.Fatalf("ProjectDetachedSessions() = %+v, want one observation", observed)
	}

	record := ThreadRecord{
		SchemaVersion: ThreadSchemaVersion, Address: address,
		StartingPath: "/repo", WorkingPath: "/repo",
		CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
		LatestLaunchProfile: &LaunchProfile{Agent: "claude", Argv: []string{}},
	}
	rows := actionableRows([]ThreadRecord{record}, nil, nil, observed)
	if len(rows) != 1 || rows[0].State != ThreadDetached {
		t.Fatalf("rows = %+v, want the projector to accept its own upstream's output", rows)
	}
}

// The couch member of RequireAttachState's class (pair#228 close review): a
// snapshot that never asked for clients is refused, not read as "no thread is
// detached". Launcher's members are TestDecideLaunchRefusesAttachStateItWasNotGiven
// and TestThePickerRefusesAttachStateItWasNotGiven.
func TestProjectDetachedSessionsRefusesAttachStateItWasNotGiven(t *testing.T) {
	address := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	bindings := []SessionNameBinding{{Address: address, SessionName: "pair-one", Agent: "claude"}}
	observed, err := ProjectDetachedSessions(
		bindings,
		[]launcher.Session{{Name: "pair-one", State: launcher.SessionLive}},
		claimsOf(bindings),
	)
	if err == nil || !strings.Contains(err.Error(), "attach state") {
		t.Fatalf("observed = %+v, err = %v; want a refusal naming the missing attach state", observed, err)
	}
}

// claimsOf is the identity case for these pure tests: every claimant is among
// the bindings passed. It counts through claimsFromBindings -- the one counting
// rule -- rather than restating it, so a change to the rule cannot leave these
// tests asserting the old one. The case that needs the WIDER index is
// TestProjectDetachedSessionsRefusesAContestedName and, at the IO seam,
// TestDetachedSessionsRefusesANameTwoThreadsClaim.
func claimsOf(bindings []SessionNameBinding) map[string]int {
	byThread := make(map[ThreadAddress]string, len(bindings))
	for _, binding := range bindings {
		byThread[binding.Address] = binding.SessionName
	}
	return claimsFromBindings(byThread)
}

// The rule the narrowed ask depends on: a name some OTHER thread also binds
// proves nothing, even when the caller passed only one claimant.
func TestProjectDetachedSessionsRefusesAContestedName(t *testing.T) {
	address := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	bindings := []SessionNameBinding{{Address: address, SessionName: "pair-one", Agent: "claude"}}
	sessions := []launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}}

	if observed, err := ProjectDetachedSessions(bindings, sessions, map[string]int{"pair-one": 1}); err != nil || len(observed) != 1 {
		t.Fatalf("uncontested: observed %+v, err %v; want the one observation", observed, err)
	}
	observed, err := ProjectDetachedSessions(bindings, sessions, map[string]int{"pair-one": 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 0 {
		t.Fatalf("contested: observed %+v; a name two threads bind proves nothing", observed)
	}
}

// effectiveBindings is each thread's current binding over the UNION of the
// index files a call reads -- and the union is where summing went wrong.
//
// Every scoped read replays the shared legacy file before its own, so a thread
// bound only by a legacy row appears in every read. Summing per-read counts
// charged it once per scope asked: two scopes made its own session look
// contested, it got no detached observation, and it read session-gone
// (pair#206 PQ-1, measured on the operator's host at 56 legacy-only bindings).
func TestEffectiveBindingsCountsEachThreadOnceAcrossReads(t *testing.T) {
	entry := func(scope, tag, name string) launcher.SessionNameEntry {
		return launcher.SessionNameEntry{ScopeKey: scope, Tag: tag, SessionName: name}
	}
	legacyOnly := entry("scope-x", "couch-legacy", "pair-legacy")
	legacyThenScoped := entry("scope-a", "couch-moved", "pair-old")
	scopedNewer := entry("scope-a", "couch-moved", "pair-new")

	// Two reads, as DetachedSessions makes for two scopes: each is the legacy
	// rows followed by that scope's own.
	reads := []scopedIndexRead{
		{scope: "scope-a", index: launcher.SessionNameIndex{Entries: []launcher.SessionNameEntry{legacyOnly, legacyThenScoped, scopedNewer}}},
		{scope: "scope-b", index: launcher.SessionNameIndex{Entries: []launcher.SessionNameEntry{legacyOnly, legacyThenScoped}}},
	}

	for _, order := range [][]scopedIndexRead{reads, {reads[1], reads[0]}} {
		current := effectiveBindings(order)
		claims := claimsFromBindings(current)

		if claims["pair-legacy"] != 1 {
			t.Fatalf("pair-legacy claimed %d times; a legacy-only thread is ONE claimant however many scopes replay it", claims["pair-legacy"])
		}
		// The thread's own scope read is authoritative, and it holds the
		// newer scope row -- whatever order the reads arrive in.
		moved := ThreadAddress{RepoScope: "scope-a", Tag: "couch-moved"}
		if current[moved] != "pair-new" {
			t.Fatalf("couch-moved binds %q, want pair-new: its own scope's newer row wins over the legacy row replayed elsewhere", current[moved])
		}
		if claims["pair-old"] != 0 {
			t.Fatalf("pair-old claimed %d times; a name the thread moved off claims nothing", claims["pair-old"])
		}
	}
}
