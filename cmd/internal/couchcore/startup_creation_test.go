package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type startupAppearedHistory struct {
	*FakeThreadArtifactCollisionChecker
	write func(ThreadAddress) error
}

func (a startupAppearedHistory) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	claim, err := a.FakeThreadArtifactCollisionChecker.Claim(address)
	if err != nil {
		return nil, err
	}
	if err := a.write(address); err != nil {
		return nil, errors.Join(err, claim.Release())
	}
	return claim, nil
}

func TestFreshHubCommitRefusesHistoryAppearingAfterResolution(t *testing.T) {
	for _, number := range []int{0, 1} {
		for _, family := range []string{"current", "archive", "legacy"} {
			t.Run(fmt.Sprintf("target%d/%s", number, family), func(t *testing.T) {
				f := newProvisionFixture(t)
				f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
				env := newTestEnv(t, f.Primary, f.host(1))
				env.Couch.Slots = NewOSSlotCatalog(f)
				env.Couch.Workspaces = NewWorkspaceProvisioner(f)
				env.Git.replies[GitCall{Dir: f.host(1), Args: "rev-parse --git-common-dir"}] = filepath.Join(f.Primary, ".git")
				path, target := f.Primary, env.Couch.Threads
				if number == 1 {
					path = f.host(1)
					repo, err := env.Couch.Slots.Discover(context.Background(), f.Primary)
					if err != nil {
						t.Fatal(err)
					}
					target = newSlotThreadStore(env.Couch.Namespace, repo.Slots[0].Identity)
				}
				scope, err := launcher.ResolveRepoScope(path)
				if err != nil {
					t.Fatal(err)
				}
				old := validThreadRecord(t)
				old.Address.RepoScope = scope.Key
				old.StartingPath, old.WorkingPath = path, path
				raw, err := encodeThreadRecord(old)
				if err != nil {
					t.Fatal(err)
				}
				retainedPath := target.recordPath(old.Address)
				if family == "archive" {
					retainedPath = target.archivePath(old.Address)
				}
				if family == "legacy" {
					retainedPath = env.Couch.Threads.recordPath(old.Address)
				}
				arrivals := 0
				env.Couch.Artifacts = startupAppearedHistory{FakeThreadArtifactCollisionChecker: env.Artifacts, write: func(ThreadAddress) error {
					arrivals++
					if err := os.MkdirAll(filepath.Dir(retainedPath), 0700); err != nil {
						return err
					}
					return os.WriteFile(retainedPath, raw, 0600)
				}}
				start, err := env.Couch.StartFreshHub(context.Background(), path)
				if err == nil || start.Handle != nil || len(env.Runner.Ops) != 0 || arrivals != 1 {
					t.Fatalf("appearance was not refused at publication: %+v err=%v arrivals=%d effects=%v", start, err, arrivals, env.Runner.Ops)
				}
				if got, err := os.ReadFile(retainedPath); err != nil || string(got) != string(raw) {
					t.Fatalf("retained history changed: %v", err)
				}
			})
		}
	}
}

// The prior observation is intentionally outside the commit. A different writer
// may publish retained evidence before startup obtains mutation authority.
func TestEmptyStartupCommitRefusesAppearedHistory(t *testing.T) {
	for _, localTarget := range []bool{false, true} {
		for _, family := range []string{"current", "archive", "legacy", "journal"} {
			t.Run(family+map[bool]string{false: "-primary", true: "-slot"}[localTarget], func(t *testing.T) {
				env, local := slotRecoveryOperationFixture(t)
				global := env.Couch.Threads
				target := global
				if localTarget {
					target = local
				}
				const scope = "retained-scope"
				path := target.recordPath(ThreadAddress{RepoScope: scope, Tag: "existing"})
				switch family {
				case "archive":
					path = target.archivePath(ThreadAddress{RepoScope: scope, Tag: "existing"})
				case "legacy":
					path = global.recordPath(ThreadAddress{RepoScope: scope, Tag: "existing"})
				case "journal":
					path = target.journalPath()
				}
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				const retained = "existing bytes must survive, even when unreadable"
				if err := os.WriteFile(path, []byte(retained), 0600); err != nil {
					t.Fatal(err)
				}
				commits := 0
				err := global.withEmptyStartupTarget(context.Background(), target, scope, func() error { commits++; return nil })
				if err == nil || commits != 0 {
					t.Fatalf("history admitted: err=%v commits=%d", err, commits)
				}
				if raw, err := os.ReadFile(path); err != nil || string(raw) != retained {
					t.Fatalf("retained history changed: %q %v", raw, err)
				}
			})
		}
	}
}

func TestEmptyStartupCommitAllowsOnlyItsEmptyTarget(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	global := env.Couch.Threads
	other := global.recordPath(ThreadAddress{RepoScope: "other", Tag: "retained"})
	if err := os.MkdirAll(filepath.Dir(other), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("another target may have started first"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []*ThreadStore{global, local} {
		commits := 0
		if err := global.withEmptyStartupTarget(context.Background(), target, "empty-target", func() error { commits++; return nil }); err != nil || commits != 1 {
			t.Fatalf("empty target refused: %v commits=%d", err, commits)
		}
	}
}
