package couchcore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

// sandboxedChecker is the ONLY way these tests build the production checker,
// and it redirects every IO seam the checker holds -- not just the one under
// test (pair#228 plan gate PQ-1). NewScopedThreadArtifactCollisionChecker wires
// a REAL session deleter; a red run here would otherwise reach the post-ack
// quiesce path and delete zellij sessions on the developer's machine.
//
// The enumeration below is backed by a structural guard, because the
// enumeration is what the first draft of this plan got wrong: the stub's
// directory goes first on PATH, so a zellij exec from ANY route -- including one
// nobody listed -- hits the logging stub, and cleanup fails the test if the log
// holds anything but the two read-only queries.
func sandboxedChecker(t *testing.T, dataDir string, sessions map[string]string) (ScopedThreadArtifactCollisionChecker, string) {
	t.Helper()
	path, log := pairlifecycletest.StubZellij(t, sessions)
	t.Setenv("PATH", filepath.Dir(path)+string(os.PathListSeparator)+os.Getenv("PATH"))
	// The guard for the guard: with correct code no tested route execs a bare
	// `zellij`, so dropping the shim would change nothing observable -- until a
	// later route did, and reached the host silently. Assert it is in force.
	if got, err := exec.LookPath("zellij"); err != nil || got != path {
		t.Fatalf("zellij resolves to %q (%v), want the stub %q -- the PATH shim is not in force", got, err, path)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	deleter := &fakeSessionDeleter{}
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)
	checker.Sessions = deleter
	checker.Zellij = launcher.ZellijSource{Path: path}

	t.Cleanup(func() {
		if len(deleter.deleted) > 0 {
			t.Errorf("the checker deleted sessions %v -- a test must never reach session deletion", deleter.deleted)
		}
		for _, call := range pairlifecycletest.LoggedCalls(t, log) {
			if !strings.HasPrefix(call, "list-sessions") && !strings.HasSuffix(call, "action list-clients") {
				t.Errorf("zellij was asked %q -- only list-sessions and list-clients belong in these tests", call)
			}
		}
	})
	return checker, log
}

// indexSession binds address to a zellij session name in the scope's
// session-name index, the way the launcher records it.
func indexSession(t *testing.T, dataDir string, address ThreadAddress, name string) {
	t.Helper()
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: name, ScopeKey: address.RepoScope, RepoRoot: "/repo", RepoName: "repo", Tag: string(address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(paths.SessionBindings(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}

// withOthers adds n unrelated live pair sessions -- the host population a
// question about one thread must not pay for.
func withOthers(sessions map[string]string, n int) map[string]string {
	for i := 0; i < n; i++ {
		sessions[fmt.Sprintf("📁other-%02d", i)] = "detached"
	}
	return sessions
}

var (
	addressA = ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	addressB = ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050608"}
)

// Sites 1-2: proving a thread detached asks only that thread's own session for
// its clients, however many sessions the host runs (pair#228).
func TestDetachedSessionsAsksOnlyTheCandidatesSessions(t *testing.T) {
	for _, c := range []struct {
		candidates []DetachedCandidate
		want       int
	}{
		{[]DetachedCandidate{{Address: addressA, Agent: "claude"}}, 1},
		{[]DetachedCandidate{{Address: addressA, Agent: "claude"}, {Address: addressB, Agent: "claude"}}, 2},
	} {
		t.Run(fmt.Sprintf("%d candidates", len(c.candidates)), func(t *testing.T) {
			dataDir := t.TempDir()
			indexSession(t, dataDir, addressA, "📁repo-a")
			indexSession(t, dataDir, addressB, "📁repo-b")
			checker, log := sandboxedChecker(t, dataDir, withOthers(map[string]string{"📁repo-a": "detached", "📁repo-b": "detached"}, 20))

			observed, err := checker.DetachedSessions(context.Background(), c.candidates)
			if err != nil {
				t.Fatal(err)
			}
			if len(observed) != len(c.candidates) {
				t.Fatalf("observed %+v, want every candidate detached", observed)
			}
			if n := pairlifecycletest.CountCalls(t, log, "list-clients"); n != c.want {
				t.Fatalf("list-clients = %d, want %d regardless of the 20 other sessions", n, c.want)
			}
		})
	}
}

// Narrowing must not change the proof's meaning: the named session is observed
// detached exactly when it is live with no client.
func TestDetachedSessionsKeepsItsMeaningForTheNamedSession(t *testing.T) {
	for _, c := range []struct {
		state    string // "" = absent from zellij
		observed bool
	}{
		{"detached", true}, {"attached", false}, {"exited", false}, {"", false},
	} {
		t.Run(fmt.Sprintf("state=%q", c.state), func(t *testing.T) {
			dataDir := t.TempDir()
			indexSession(t, dataDir, addressA, "📁repo-a")
			sessions := withOthers(map[string]string{}, 3)
			if c.state != "" {
				sessions["📁repo-a"] = c.state
			}
			checker, _ := sandboxedChecker(t, dataDir, sessions)
			observed, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{{Address: addressA, Agent: "claude"}})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(observed) == 1; got != c.observed {
				t.Fatalf("observed = %+v, want observed=%v", observed, c.observed)
			}
		})
	}
}

// Site 3: PairSession reads only "not exited", so it asks no session for
// clients -- and still reports presence correctly.
func TestPairSessionReadsLivenessOnly(t *testing.T) {
	for _, c := range []struct {
		state   string
		present bool
	}{
		{"detached", true}, {"attached", true}, {"exited", false}, {"", false},
	} {
		t.Run(fmt.Sprintf("state=%q", c.state), func(t *testing.T) {
			dataDir := t.TempDir()
			indexSession(t, dataDir, addressA, "📁repo-a")
			sessions := withOthers(map[string]string{}, 20)
			if c.state != "" {
				sessions["📁repo-a"] = c.state
			}
			checker, log := sandboxedChecker(t, dataDir, sessions)
			binding, err := checker.PairSession(addressA)
			if err != nil {
				t.Fatal(err)
			}
			if binding.Name != "📁repo-a" || binding.Present != c.present {
				t.Fatalf("binding = %+v, want present=%v", binding, c.present)
			}
			if n := pairlifecycletest.CountCalls(t, log, "list-clients"); n != 0 {
				t.Fatalf("list-clients = %d, want 0", n)
			}
		})
	}
}

// The whole couchcore side of a warm reattach, counted at the seam: the path,
// not the functions. This is the test that would have caught the first draft's
// miss -- it counted DetachedSessions, and awaitResumeRegistration's PairSession
// poll was a sixth full snapshot nobody had listed.
func TestWarmResumeAsksTwoSessionsForClientsWhateverTheHostHas(t *testing.T) {
	for _, others := range []int{2, 21} {
		t.Run(fmt.Sprintf("S=%d", others+1), func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.resumeRegistrationTimeout = 2 * time.Second // a bypass fails fast, not after 15 s
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			record := validThreadRecord(t)
			record.Reservation = false
			record.LatestLaunchProfile = &profile
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			dataDir := t.TempDir()
			const name = "📁repo-couch"
			indexSession(t, dataDir, created.Address, name)
			checker, log := sandboxedChecker(t, dataDir, withOthers(map[string]string{name: "detached"}, others))
			env.Couch.Artifacts = checker

			if _, _, err := env.Couch.ResumeContext(context.Background(), created.Address); err != nil {
				t.Fatalf("warm resume: %v", err)
			}
			lc := pairlifecycletest.CountCalls(t, log, "list-clients")
			ls := pairlifecycletest.CountCalls(t, log, "list-sessions")
			if lc != 2 || ls != 6 {
				t.Fatalf("list-clients = %d, list-sessions = %d; want 2 and 6 at every S (the proof twice, then registration's liveness)", lc, ls)
			}
		})
	}
}

// The duplicate-name rule must survive being asked about a SUBSET.
//
// Two addresses bound to one session name is the case ProjectDetachedSessions
// fails closed on: couch cannot tell whose session that is, so neither thread
// gets a row. But the claim count was taken over the bindings the caller
// passed, so asking about only one of the two made its name look unique and
// the ambiguous thread was reported detached -- and startup would resume it.
//
// Narrowing the candidate set is exactly what pair#228 introduced and what
// pair#206's startup narrowing leans on, so the count has to come from the
// scope's whole index, which is already read.
func TestDetachedSessionsRefusesANameTwoThreadsClaim(t *testing.T) {
	dataDir := t.TempDir()
	const shared = "📁repo-shared"
	indexSession(t, dataDir, addressA, shared)
	indexSession(t, dataDir, addressB, shared)
	checker, _ := sandboxedChecker(t, dataDir, map[string]string{shared: "detached"})

	// Asked about ONE of the two claimants, which is all a narrowed caller has.
	observed, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{{Address: addressA, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 0 {
		t.Fatalf("observed %+v; a session two threads claim proves nothing about either", observed)
	}
}

// A name a thread has MOVED OFF claims nothing.
//
// The index is append-only, so a thread's binding is its last entry. Counting
// every entry a name ever had would let a retired binding contest the thread
// that holds the name now -- refusing a reattach that is perfectly valid. That
// is the inverse of the bug the count exists to prevent, and the commoner
// state, since an ordinary relaunch appends.
func TestDetachedSessionsIgnoresANameItsThreadHasLeft(t *testing.T) {
	dataDir := t.TempDir()
	const contested = "📁repo-contested"
	// addressA held the name once, then moved to its own; addressB holds it now.
	indexSession(t, dataDir, addressA, contested)
	indexSession(t, dataDir, addressA, "📁repo-a-current")
	indexSession(t, dataDir, addressB, contested)
	checker, _ := sandboxedChecker(t, dataDir, map[string]string{contested: "detached", "📁repo-a-current": "detached"})

	observed, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{{Address: addressB, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 || observed[0].Address != addressB {
		t.Fatalf("observed %+v; addressB holds that name now, and addressA left it", observed)
	}
}

// And the same thread registering repeatedly is still one claim.
func TestDetachedSessionsCountsAThreadOnceHoweverOftenItRegistered(t *testing.T) {
	dataDir := t.TempDir()
	const name = "📁repo-a"
	indexSession(t, dataDir, addressA, name)
	indexSession(t, dataDir, addressA, name)
	indexSession(t, dataDir, addressA, name)
	checker, _ := sandboxedChecker(t, dataDir, map[string]string{name: "detached"})

	observed, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{{Address: addressA, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 {
		t.Fatalf("observed %+v; three registrations by one thread are still one claim", observed)
	}
}

// A thread bound ONLY by a legacy row is one claimant, however many scopes a
// single call reads -- asserted where the count is consumed, not on the merge
// helper alone.
//
// Every scoped read replays the shared legacy file before its own rows, so the
// legacy thread appears once per scope asked. Summing per-read counts made its
// own session look contested whenever a call spanned two scopes: no detached
// observation, session-gone, and the pass would never seed it (pair#206 PQ-1;
// 56 legacy-only bindings on the operator's host). A test of the merge helper
// alone survived reintroducing exactly that bug, because every other seam test
// here reads one scope, where summing and not summing agree.
func TestDetachedSessionsCountsALegacyThreadOnceAcrossScopes(t *testing.T) {
	dataDir := t.TempDir()
	const legacyName = "📁repo-legacy"
	legacyThread := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-00000000000000aa"}
	otherScope := ThreadAddress{RepoScope: "fedcba9876543210", Tag: "couch-00000000000000bb"}

	indexLegacySession(t, dataDir, legacyThread, legacyName)
	indexSession(t, dataDir, otherScope, "📁repo-other")
	checker, _ := sandboxedChecker(t, dataDir, map[string]string{legacyName: "detached", "📁repo-other": "detached"})

	// Two scopes in ONE call, which is what makes the legacy file replay twice.
	observed, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{
		{Address: legacyThread, Agent: "claude"},
		{Address: otherScope, Agent: "claude"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, observation := range observed {
		if observation.Address == legacyThread {
			found = true
		}
	}
	if !found {
		t.Fatalf("observed %+v; the legacy-bound thread's session is its own and uncontested", observed)
	}
}

// indexLegacySession writes a binding into the shared legacy-global index,
// which every scoped read replays before its own rows.
func indexLegacySession(t *testing.T, dataDir string, address ThreadAddress, name string) {
	t.Helper()
	legacy, err := artifactpath.ResolveLegacyRoot(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(legacy.SessionBindings()), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: name, ScopeKey: address.RepoScope, RepoRoot: "/repo", RepoName: "repo", Tag: string(address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(legacy.SessionBindings(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}

// The reach rule in DetachedSessions' doc, pinned: a scope whose own index file
// will not decode binds NONE of its threads -- not even from the legacy rows
// another scope's successful read replayed.
//
// The unreadable file may hold a newer row that supersedes the legacy one, so
// using the legacy name would judge a thread by a session it has left. The
// union-of-reads refactor briefly did exactly that: it iterated scopes rather
// than reads, and took a failed scope's threads' names from elsewhere.
func TestDetachedSessionsBindsNothingForAnUnreadableScope(t *testing.T) {
	dataDir := t.TempDir()
	const legacyName = "📁repo-legacy"
	// The thread's only readable binding is a legacy row.
	broken := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-00000000000000aa"}
	healthy := ThreadAddress{RepoScope: "fedcba9876543210", Tag: "couch-00000000000000bb"}
	indexLegacySession(t, dataDir, broken, legacyName)
	indexSession(t, dataDir, healthy, "📁repo-healthy")

	// broken's OWN scope file exists but will not decode.
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: broken.RepoScope}, string(broken.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SessionBindings(), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	checker, _ := sandboxedChecker(t, dataDir, map[string]string{legacyName: "detached", "📁repo-healthy": "detached"})

	observed, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{
		{Address: broken, Agent: "claude"},
		{Address: healthy, Agent: "claude"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, observation := range observed {
		if observation.Address == broken {
			t.Fatalf("observed %+v for a thread whose own index could not be read; its current binding is unknown", observation)
		}
	}
	// And the readable scope is unaffected by its neighbour's failure.
	if len(observed) != 1 || observed[0].Address != healthy {
		t.Fatalf("observed %+v, want only the healthy scope's thread", observed)
	}
}

// TestSessionPresenceAnswersThroughTheProductionChecker closes I3 from the M1
// boundary review: every other presence test ran against the fake, which
// unconditionally answers SessionAbsent for anything unset -- so it could not
// catch the one branch that decides archive-eligibility inverting.
//
// That branch (artifactcollision.go) draws the line the whole three-valued type
// exists for: an address in a READABLE scope with no index row was asked about
// and has no session (absent, and therefore archivable), while an address whose
// scope could not be read was never asked (unresolved, and must never be
// retired). A fake with no conformance check against production is one modelled
// world, not two that agree.
func TestSessionPresenceAnswersThroughTheProductionChecker(t *testing.T) {
	dataDir := t.TempDir()
	live := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-00000000000000c1"}
	exited := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-00000000000000c2"}
	unbound := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-00000000000000c3"}
	unreadable := ThreadAddress{RepoScope: "fedcba9876543210", Tag: "couch-00000000000000c4"}

	indexSession(t, dataDir, live, "📁repo-live")
	indexSession(t, dataDir, exited, "📁repo-exited")
	// `unbound` gets no index row, but its scope IS readable -- the other two
	// threads' rows live in the same file.

	// `unreadable`'s own scope file exists and will not decode.
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: unreadable.RepoScope}, string(unreadable.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SessionBindings(), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	checker, _ := sandboxedChecker(t, dataDir, map[string]string{
		"📁repo-live": "detached", "📁repo-exited": "exited",
	})

	got, err := checker.SessionPresence(context.Background(),
		[]ThreadAddress{live, exited, unbound, unreadable})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		address ThreadAddress
		want    SessionState
	}{
		{"bound to a live session", live, SessionPresent},
		{"bound to an EXITED resurrect record", exited, SessionAbsent},
		{"readable scope, no index row — asked, and there is none", unbound, SessionAbsent},
		{"scope could not be read — never asked", unreadable, SessionUnresolved},
	} {
		if state := got[tc.address].State; state != tc.want {
			t.Errorf("%s: state = %v, want %v", tc.name, state, tc.want)
		}
	}
}

// TestSessionPresenceCountsNoClients pins the optimistic-inventory trade at the
// production seam: presence must reach `list-sessions` only. A `list-clients`
// costs ~250 ms per live session (#228), and the refresh now runs this for EVERY
// record rather than only the resume-shaped ones -- so a client query slipping
// back in would scale that cost with the size of the store.
func TestSessionPresenceCountsNoClients(t *testing.T) {
	dataDir := t.TempDir()
	indexSession(t, dataDir, addressA, "📁repo-a")
	indexSession(t, dataDir, addressB, "📁repo-b")
	checker, log := sandboxedChecker(t, dataDir, withOthers(map[string]string{
		"📁repo-a": "detached", "📁repo-b": "detached",
	}, 8))

	if _, err := checker.SessionPresence(context.Background(), []ThreadAddress{addressA, addressB}); err != nil {
		t.Fatal(err)
	}
	for _, call := range pairlifecycletest.LoggedCalls(t, log) {
		if strings.HasSuffix(call, "action list-clients") {
			t.Fatalf("presence asked for clients (%q); it must reach list-sessions only", call)
		}
	}
}
