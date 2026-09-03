package couchcore

import (
	"context"
	"errors"
	"io"
	"testing"
)

// AllocateThreadTag claims the Pair artifact AND persists a pristine
// `Reservation: true` record. Every failure after it must undo BOTH, or the
// reservation is permanent: ProjectActionableThreads hides reserved records, so
// it never appears in the switcher, and reconcileInterruptedStarts skips
// records with no start transaction, so nothing reclaims it.
//
// The three post-allocation sites used to call releaseClaimIfThreadAbsent,
// which returns nil the moment GetThread SUCCEEDS -- and it always succeeds
// there, because the record was just written. Protection that can never fire.
// Admission owned this rollback before pair#170 M4 deleted it, and the two
// tests that pinned it went with it.
func TestStartFailuresAfterTagAllocationRollBackTheReservation(t *testing.T) {
	for _, test := range []struct {
		name   string
		break_ func(*testEnv)
	}{
		{
			name:   "supervisor identity unavailable",
			break_: func(env *testEnv) { env.Proc.CurrentErr = errors.New("no supervisor identity") },
		},
		{
			name: "start nonce entropy exhausted",
			// AllocateThreadTag draws from the SAME reader, so a reader that
			// fails immediately never reaches the post-allocation path -- it
			// fails before anything is claimed and the test passes vacuously.
			// The budget is what puts the failure after allocation.
			break_: func(env *testEnv) { env.Couch.Entropy = &budgetedReader{remaining: 8, source: env.Couch.Entropy} },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			test.break_(env)

			if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
				t.Fatal("start succeeded despite an injected post-allocation failure")
			}
			assertPreparedStartHadNoEffects(t, env)
			if got := env.Artifacts.Releases(); len(got) != 1 {
				t.Fatalf("artifact claim releases = %+v, want exactly the one that was claimed", got)
			}
		})
	}
}

// budgetedReader passes through a fixed number of bytes and then fails, so a
// test can place an entropy failure at a chosen point in a sequence of draws.
type budgetedReader struct {
	remaining int
	source    io.Reader
}

func (r *budgetedReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, errors.New("entropy unavailable")
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.source.Read(p)
	r.remaining -= n
	return n, err
}

// Every canned FakeGit reply in the tree answers ".git", the repo-root shape.
// Real git answers relative to the QUERY directory, so a subdirectory gets
// "../.git" and a linked worktree gets an absolute path -- measured in
// TestGitConformance_LinkedWorktree. This pins all three without needing
// PAIR_LIVE_COUCH, so a future simplification of the join reddens here first.
func TestRepoIdentityJoinsGitsAnswerAgainstTheQueryDirectory(t *testing.T) {
	for _, test := range []struct {
		name    string
		dir     string
		gitSays string
		want    string
	}{
		{name: "repository root", dir: "/repo", gitSays: ".git", want: "/repo/.git"},
		{name: "one level down", dir: "/repo/sub", gitSays: "../.git", want: "/repo/.git"},
		{name: "two levels down", dir: "/repo/a/b", gitSays: "../../.git", want: "/repo/.git"},
		{name: "linked worktree answers absolute", dir: "/wt", gitSays: "/repo/.git", want: "/repo/.git"},
	} {
		t.Run(test.name, func(t *testing.T) {
			couch := &Couch{Git: NewFakeGit(map[GitCall]string{
				{Dir: test.dir, Args: "rev-parse --git-common-dir"}: test.gitSays,
			})}
			got, err := couch.resolveRepoIdentity(context.Background(), test.dir)
			if err != nil {
				t.Fatalf("resolveRepoIdentity(%s): %v", test.dir, err)
			}
			if got != test.want {
				t.Fatalf("identity = %q, want %q -- a shifted identity orphans every saved launch preference", got, test.want)
			}
		})
	}
}

// The identity resolution runs on the preview worker. Before pair#170 M4 the
// IO on that path was a fleet-policy subprocess with a 5s bound; the git call
// that replaced it must carry the caller's context or a hung git hangs the
// start form with no way out.
func TestRepoIdentityResolutionHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	couch := &Couch{Git: NewFakeGit(map[GitCall]string{
		{Dir: "/repo", Args: "rev-parse --git-common-dir"}: ".git",
	})}
	if _, err := couch.resolveRepoIdentity(ctx, "/repo"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled identity resolution = %v, want context.Canceled", err)
	}
}
