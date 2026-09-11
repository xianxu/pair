package couchcore

// pair#206 M1: what startup PROVES before the first frame.
//
// StartInteractive takes one inventory, and three readers consume it:
// ResolveLayoutConflicts, SelectResumableRoot, and the one-thread-per-path
// guards inside spawnResolved. Each filters before it reads -- the selectors to
// the cwd, the layout guard to rows whose layout differs -- so proving anything
// about a thread outside both sets is work whose answer nobody looks at.
//
// It is charged twice over: one `list-clients` per detach candidate (about
// 250 ms against a real detached session, pair#228) AND one ResolveEstablished
// per resume-shaped record, which reads that thread's own ledger. Both grow
// with the store, which is what the operator felt as "the initial startup
// attach is still slow" after #228.

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// startupFixture builds a cwd thread plus `others` detached threads at other
// paths, all in couch's layout, and returns the env.
func startupFixture(t *testing.T, others int, otherLayout Layout) (*testEnv, ThreadAddress) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = NormalizeLayout("layout2")
	env.Couch.resumeRegistrationTimeout = 2 * time.Second

	cwd := detachedThreadAt(t, env, "/repo", "cwd")
	for i := 0; i < others; i++ {
		address := detachedThreadAt(t, env, fmt.Sprintf("/other-%02d", i), fmt.Sprintf("o%02d", i))
		if otherLayout != "" {
			bumpLayout(t, env, address, otherLayout)
		}
	}
	return env, cwd
}

func detachedThreadAt(t *testing.T, env *testEnv, path, suffix string) ThreadAddress {
	t.Helper()
	// The scope key is the REAL one for this path. A hand-written key would
	// never match what StartInteractive resolves, and the narrowing predicate
	// compares scope keys -- so the fixture would silently exercise the
	// "nothing at the cwd" branch and pass for the wrong reason.
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record := validThreadRecord(t)
	record.Address.RepoScope = scope.Key
	record.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", threadCounter(suffix)))
	record.StartingPath, record.WorkingPath = path, path
	record.Reservation = false
	record.LatestLaunchProfile = &profile
	record.Layout = NormalizeLayout("layout2")
	created, createErr := env.Couch.Threads.CreateThread(record)
	if createErr != nil {
		t.Fatal(createErr)
	}
	name := "pair-" + string(created.Address.Tag)
	env.Artifacts.SetDetachedSession(created.Address, name)
	env.Artifacts.SetPairSession(created.Address, name, true)
	// The inventory's detached proof needs a native id, so without this every
	// row reads binding-lost and the fixture would test nothing.
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-"+suffix)
	return created.Address
}

func bumpLayout(t *testing.T, env *testEnv, address ThreadAddress, layout Layout) {
	t.Helper()
	current, err := env.Couch.Threads.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Threads.UpdateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
		next.Layout = layout
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// threadCounter derives a stable tag suffix from a label, so a fixture's tags
// are deterministic within one test.
func threadCounter(suffix string) uint64 {
	var h uint64 = 1469598103934665603
	for _, b := range []byte(suffix) {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

// Startup's proof work must not grow with the threads it cannot act on.
func TestStartupProvesOnlyTheThreadsItsReadersConsume(t *testing.T) {
	for _, others := range []int{2, 12} {
		t.Run(fmt.Sprintf("others=%d", others), func(t *testing.T) {
			env, cwd := startupFixture(t, others, "")

			if _, err := env.Couch.StartInteractive(context.Background(), StartArgs{Worktree: "/repo"}); err != nil {
				t.Fatalf("startup: %v", err)
			}

			// Counted in CANDIDATES, not calls: each candidate is one
			// list-clients. Startup proves the cwd thread once, then
			// ResumeContext and confirmStillDetached prove it again -- three,
			// whatever else the store holds.
			if got := env.Artifacts.DetachedCandidatesAsked(); got != 3 {
				t.Fatalf("detached candidates = %d, want 3 regardless of the other %d threads", got, others)
			}
			// And the ledger reads: the cwd thread is the only resume-shaped
			// record startup resolves a binding for.
			if got := env.Artifacts.BindingResolutions(); got != 1 {
				t.Fatalf("binding resolutions = %d, want 1: each reads a thread's own ledger", got)
			}
			// And the startup actually resumed the cwd thread, rather than the
			// counts coming out right because nothing at the cwd was found.
			current, err := env.Couch.Threads.GetThread(cwd)
			if err != nil {
				t.Fatal(err)
			}
			if len(current.Incarnations) != 1 {
				t.Fatalf("cwd thread = %+v, want it resumed: a narrowing that matched nothing would still produce low counts", current)
			}
		})
	}
}

// Narrowing must not shrink the layout guard's reach: a thread at ANY path in
// another layout still refuses startup, so it is still asked about. Both kinds
// of conflict: a valid different layout, and one this binary cannot read
// (NormalizeLayout's unknown, which conflicts with everything).
func TestStartupStillRefusesAConflictingLayoutAnywhere(t *testing.T) {
	for _, other := range []Layout{NormalizeLayout("layout3"), NormalizeLayout("layout1")} {
		t.Run(string(other), func(t *testing.T) {
			env, _ := startupFixture(t, 1, other)
			_, err := env.Couch.StartInteractive(context.Background(), StartArgs{Worktree: "/repo"})
			if err == nil || !strings.Contains(err.Error(), "layout") {
				t.Fatalf("startup error = %v, want the layout-conflict refusal", err)
			}
		})
	}
}

// The equivalence that licenses the narrowing: for every record shape, all
// FOUR readers of startup's rows answer identically whether startup proved
// every candidate or only the ones they consume.
//
// Each row is a shape where the narrowing could plausibly change an answer.
// The fixtures build them through the fake, which answers the detached question
// through production's own ProjectDetachedSessions and claimsFromBindings
// (pair#206 PQ-8), so the shared-name row composes the narrowing with the
// duplicate-name rule end to end. The index-file merge that feeds those claims
// in production is pinned separately, at the real checker, by
// TestDetachedSessionsCountsALegacyThreadOnceAcrossScopes.
func TestNarrowedStartupAnswersAsAFullProofWould(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build func(t *testing.T, env *testEnv)
	}{
		{"detached at the cwd, others in the same layout", func(t *testing.T, env *testEnv) {
			addOthers(t, env, 3, "")
		}},
		{"detached at the cwd, another in a valid different layout", func(t *testing.T, env *testEnv) {
			addOthers(t, env, 2, NormalizeLayout("layout3"))
		}},
		{"detached at the cwd, another in an unreadable layout", func(t *testing.T, env *testEnv) {
			addOthers(t, env, 2, NormalizeLayout("layout1"))
		}},
		{"nothing else in the store", func(*testing.T, *testEnv) {}},
		{"the cwd thread's session is gone", func(t *testing.T, env *testEnv) {
			env.Artifacts.SetDetachedSession(env.cwd, "")
			addOthers(t, env, 2, "")
		}},
		{"the cwd thread is parked, not detached", func(t *testing.T, env *testEnv) {
			parkThread(t, env, env.cwd)
			addOthers(t, env, 2, "")
		}},
		{"the cwd thread shares its session name with an unasked thread", func(t *testing.T, env *testEnv) {
			other := addOthers(t, env, 1, "")[0]
			env.Artifacts.SetDetachedSession(other, "pair-"+string(env.cwd.Tag))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			full := equivalenceEnv(t)
			tt.build(t, full)
			narrow := equivalenceEnv(t)
			tt.build(t, narrow)

			cwdScope, err := launcher.ResolveRepoScope("/repo")
			if err != nil {
				t.Fatal(err)
			}
			scope := cwdScope.Key

			fullRows, err := full.Couch.ActionableThreadInventoryContext(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			narrowRows, err := narrow.Couch.startupInventory(context.Background(), scope, "/repo")
			if err != nil {
				t.Fatal(err)
			}

			if got, want := conflictAddresses(narrow.Couch.Layout, narrowRows), conflictAddresses(full.Couch.Layout, fullRows); !slices.Equal(got, want) {
				t.Fatalf("layout conflicts: narrowed %v, full %v", got, want)
			}
			fullRoot, fullOK := SelectResumableRoot(fullRows, scope, "/repo")
			narrowRoot, narrowOK := SelectResumableRoot(narrowRows, scope, "/repo")
			if fullOK != narrowOK || fullRoot != narrowRoot {
				t.Fatalf("root: narrowed (%+v,%v), full (%+v,%v)", narrowRoot, narrowOK, fullRoot, fullOK)
			}
			fullHeld, fullHeldOK := PathHoldsUsableThread(fullRows, scope, "/repo")
			narrowHeld, narrowHeldOK := PathHoldsUsableThread(narrowRows, scope, "/repo")
			if fullHeldOK != narrowHeldOK || fullHeld != narrowHeld {
				t.Fatalf("usable-path guard: narrowed (%+v,%v), full (%+v,%v)", narrowHeld, narrowHeldOK, fullHeld, fullHeldOK)
			}
			fullUnreadable, fullUnreadableOK := PathHoldsUnreadableThread(fullRows, scope)
			narrowUnreadable, narrowUnreadableOK := PathHoldsUnreadableThread(narrowRows, scope)
			if fullUnreadableOK != narrowUnreadableOK || fullUnreadable != narrowUnreadable {
				t.Fatalf("unreadable-path guard: narrowed (%+v,%v), full (%+v,%v)", narrowUnreadable, narrowUnreadableOK, fullUnreadable, fullUnreadableOK)
			}
		})
	}
}

// equivalenceEnv is a store holding only the cwd thread; each row adds the rest.
func equivalenceEnv(t *testing.T) *testEnv {
	t.Helper()
	env, cwd := startupFixture(t, 0, "")
	env.cwd = cwd
	return env
}

func addOthers(t *testing.T, env *testEnv, n int, layout Layout) []ThreadAddress {
	t.Helper()
	var added []ThreadAddress
	for i := 0; i < n; i++ {
		address := detachedThreadAt(t, env, fmt.Sprintf("/other-%02d", i), fmt.Sprintf("o%02d", i))
		if layout != "" {
			bumpLayout(t, env, address, layout)
		}
		added = append(added, address)
	}
	return added
}

// parkThread turns a detached thread into a verified park: its session is gone
// and a verified park is the resume authority instead.
func parkThread(t *testing.T, env *testEnv, address ThreadAddress) {
	t.Helper()
	current, err := env.Couch.Threads.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	withIncarnation, err := env.Couch.Threads.UpdateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
		profile := *next.LatestLaunchProfile
		next.Incarnations = []ThreadIncarnation{{
			PID: 42, Identity: "pair-helper", State: IncarnationLive,
			RepoIdentity: "/repo/.git", LaunchProfile: &profile,
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "park-equivalence", Address: address, PID: 42, ProcessIdentity: "pair-helper"}
	begun, err := env.Couch.Threads.BeginPark(address, withIncarnation.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Threads.FinalizePark(address, begun.Revision, identity, 1, env.Now); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(address, "")
}

// conflictAddresses compares conflicts by WHICH threads conflict, not by how
// many -- two different conflicting sets of the same size are not equivalent.
func conflictAddresses(layout Layout, rows []ActionableThreadSummary) []ThreadAddress {
	var out []ThreadAddress
	for _, conflict := range ResolveLayoutConflicts(layout, rows) {
		out = append(out, conflict.Address)
	}
	slices.SortFunc(out, func(a, b ThreadAddress) int {
		return strings.Compare(a.RepoScope+string(a.Tag), b.RepoScope+string(b.Tag))
	})
	return out
}

// The cwd arm of the predicate matches on scope AND path, and only a COUNT can
// show why: an equivalence test cannot catch a predicate that asks too much,
// because over-asking never changes an answer -- it only costs.
//
// The case the scope half pays for is a stale record at the cwd's path under a
// scope that is no longer this repo's (a repository that moved, a record from
// before a scope derivation changed). The selectors skip it either way, so
// proving it is a list-clients and a ledger read that nothing consults.
func TestStartupDoesNotProveAForeignScopeRecordAtTheCwdPath(t *testing.T) {
	env, _ := startupFixture(t, 0, "")
	detachedThreadAt(t, env, "/repo", "second")
	// Same working path, a scope that is not this repository's.
	stale := validThreadRecord(t)
	stale.Address = ThreadAddress{RepoScope: "fedcba9876543210", Tag: "couch-00000000000000ff"}
	stale.StartingPath, stale.WorkingPath = "/repo", "/repo"
	stale.Reservation = false
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	stale.LatestLaunchProfile = &profile
	stale.Layout = NormalizeLayout("layout2")
	if _, err := env.Couch.Threads.CreateThread(stale); err != nil {
		t.Fatal(err)
	}

	cwdScope, err := launcher.ResolveRepoScope("/repo")
	if err != nil {
		t.Fatal(err)
	}
	before := env.Artifacts.BindingResolutions()
	if _, err := env.Couch.startupInventory(context.Background(), cwdScope.Key, "/repo"); err != nil {
		t.Fatal(err)
	}
	// The two in-scope threads at /repo are proved; the foreign-scope one is not.
	if got := env.Artifacts.BindingResolutions() - before; got != 2 {
		t.Fatalf("binding resolutions = %d, want 2: the foreign-scope record at the same path is consulted by nobody", got)
	}
}
