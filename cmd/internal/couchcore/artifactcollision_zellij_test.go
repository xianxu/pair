package couchcore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
