package couchcore

import (
	"context"
	"errors"
	"testing"
)

// The start-grant token carried two properties. This pins the one that is a
// DOMAIN guarantee: a start refuses a resolution that drifted between preview
// and submit.
//
// The token proved it by keeping the accepted resolution owner-side and handing
// back an opaque handle. The fingerprint proves the same thing without the
// capability table -- it re-resolves and compares -- which is what makes the
// 256-bit token, its TTL, its capacity of 16 and its collision retries
// deletable: they were defending a prepared start against ANOTHER OWNER, and
// couch-lite has one.
//
// The token's other property, at-most-one-consumption, is not a domain
// guarantee at all; it is the start form's armed submit, pinned in couchtty.
func TestSpawnPreparedRefusesDriftByFingerprint(t *testing.T) {
	t.Run("an unchanged resolution starts", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Worktree: "/repo"})
		if err != nil {
			t.Fatalf("PrepareStart: %v", err)
		}
		record, handle, err := env.Couch.SpawnPrepared(context.Background(),
			StartArgs{Worktree: "/repo"}, prepared.Resolution.Fingerprint)
		if err != nil {
			t.Fatalf("SpawnPrepared: %v", err)
		}
		defer env.Runner.SetExited(handle.ID(), 0)
		if record.Thread == (ThreadAddress{}) {
			t.Fatal("started thread has no address")
		}
	})

	t.Run("a drifted preference refuses before any effect", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Worktree: "/repo"})
		if err != nil {
			t.Fatalf("PrepareStart: %v", err)
		}
		// Something else remembered a different agent at this path between
		// preview and submit.
		preference := PathLaunchPreference{
			SchemaVersion: PathLaunchPreferenceSchemaVersion,
			RepoIdentity:  "/repo/.git", PhysicalPath: "/repo", LastAgent: "claude",
			ArgvByAgent: map[string][]string{"claude": {"--model", "opus"}}, Revision: 1,
		}
		if err := writePathLaunchPreferenceForTest(env.Couch.Threads, preference); err != nil {
			t.Fatal(err)
		}

		_, _, err = env.Couch.SpawnPrepared(context.Background(),
			StartArgs{Worktree: "/repo"}, prepared.Resolution.Fingerprint)
		if !errors.Is(err, ErrStartResolutionChanged) {
			t.Fatalf("drifted start err = %v, want ErrStartResolutionChanged", err)
		}
		assertPreparedStartHadNoEffects(t, env)
	})

	t.Run("a fingerprint from another path refuses", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		_, _, err := env.Couch.SpawnPrepared(context.Background(),
			StartArgs{Worktree: "/repo"}, "not-the-fingerprint")
		if !errors.Is(err, ErrStartResolutionChanged) {
			t.Fatalf("foreign fingerprint err = %v, want ErrStartResolutionChanged", err)
		}
		assertPreparedStartHadNoEffects(t, env)
	})
}
