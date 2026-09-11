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

var threadCounters = map[string]uint64{}

func threadCounter(suffix string) uint64 {
	threadCounters[suffix]++
	// Stable per suffix so a tag is deterministic within one test.
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
			_ = cwd
		})
	}
}

// Narrowing must not shrink the layout guard's reach: a thread at ANY path
// whose layout differs still refuses startup, so it is still asked about.
func TestStartupStillRefusesAConflictingLayoutAnywhere(t *testing.T) {
	env, _ := startupFixture(t, 1, NormalizeLayout("layout1"))

	_, err := env.Couch.StartInteractive(context.Background(), StartArgs{Worktree: "/repo"})
	if err == nil {
		t.Fatal("startup did not refuse a mixed-layout tree")
	}
	if got := err.Error(); !containsAll(got, "layout") {
		t.Fatalf("error = %v, want the layout-conflict refusal", err)
	}
}

// The equivalence that licenses the narrowing: for every record shape, the
// three readers answer identically whether startup proved every candidate or
// only the ones they consume.
func TestNarrowedStartupAnswersAsAFullProofWould(t *testing.T) {
	for _, tt := range []struct {
		name        string
		others      int
		otherLayout Layout
	}{
		{"same layout elsewhere", 3, ""},
		{"conflicting layout elsewhere", 2, NormalizeLayout("layout1")},
		{"unreadable layout elsewhere", 2, NormalizeLayout("unreadable")},
		{"nothing else", 0, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			full, _ := startupFixture(t, tt.others, tt.otherLayout)
			narrow, _ := startupFixture(t, tt.others, tt.otherLayout)

			fullRows, err := full.Couch.ActionableThreadInventoryContext(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			cwdScope, err := launcher.ResolveRepoScope("/repo")
			if err != nil {
				t.Fatal(err)
			}
			scope := cwdScope.Key
			narrowRows, err := narrow.Couch.startupInventory(context.Background(), scope, "/repo")
			if err != nil {
				t.Fatal(err)
			}
			fullConflicts := ResolveLayoutConflicts(full.Couch.Layout, fullRows)
			narrowConflicts := ResolveLayoutConflicts(narrow.Couch.Layout, narrowRows)
			if len(fullConflicts) != len(narrowConflicts) {
				t.Fatalf("layout conflicts: full %+v, narrowed %+v", fullConflicts, narrowConflicts)
			}

			fullRoot, fullOK := SelectResumableRoot(fullRows, scope, "/repo")
			narrowRoot, narrowOK := SelectResumableRoot(narrowRows, scope, "/repo")
			if fullOK != narrowOK || fullRoot != narrowRoot {
				t.Fatalf("root: full (%+v,%v), narrowed (%+v,%v)", fullRoot, fullOK, narrowRoot, narrowOK)
			}

			fullHeld, fullHeldOK := PathHoldsUsableThread(fullRows, scope, "/repo")
			narrowHeld, narrowHeldOK := PathHoldsUsableThread(narrowRows, scope, "/repo")
			if fullHeldOK != narrowHeldOK || fullHeld != narrowHeld {
				t.Fatalf("path guard: full (%+v,%v), narrowed (%+v,%v)", fullHeld, fullHeldOK, narrowHeld, narrowHeldOK)
			}
		})
	}
}

func containsAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		found := false
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
