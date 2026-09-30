package couchcore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSlotOperationsIgnoreNestedIndependentHostedActor(t *testing.T) {
	for _, registry := range []string{"Actors", "ActorRegistry"} {
		for _, operation := range []string{"open", "fresh"} {
			t.Run(registry+"/"+operation, func(t *testing.T) {
				env, local := slotRecoveryOperationFixture(t)
				outer := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "outer")
				outer.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
				if _, err := local.CreateThread(outer); err != nil {
					t.Fatal(err)
				}
				nested := filepath.Join(local.slot.WorktreeRoot, "independent")
				if err := os.MkdirAll(nested, 0700); err != nil {
					t.Fatal(err)
				}
				if output, err := exec.Command("git", "-C", nested, "init").CombinedOutput(); err != nil {
					t.Fatalf("git init: %v %s", err, output)
				}
				inner := recordAtCheckout(t, nested, nested, "inner")
				inner.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
				inner.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "nested-live", State: IncarnationLive, RepoIdentity: filepath.Join(nested, ".git"), LaunchProfile: inner.LatestLaunchProfile}}
				if _, err := env.Couch.Threads.CreateThread(inner); err != nil {
					t.Fatal(err)
				}
				actor := ActorRecord{ID: "nested-owner", Thread: inner.Address, Args: StartArgs{Worktree: Worktree(nested), Cwd: nested, Stack: "claude"}, StartedAt: env.Now, PID: 4242, Identity: "nested-live"}
				env.Proc.Set(actor.PID, actor.Identity)
				if registry == "Actors" {
					env.Couch.reg = env.Couch.reg.Insert(actor)
				} else if err := env.Couch.Store.Save(NewRegistry().Insert(actor), env.Couch.names); err != nil {
					t.Fatal(err)
				}
				if operation == "open" {
					env.Artifacts.SetPairSession(outer.Address, "pair-outer-live", true)
					env.Artifacts.SetDetachedSession(outer.Address, "pair-outer-live")
				}
				var result StartResult
				var err error
				if operation == "open" {
					result, err = env.Couch.OpenSlot(context.Background(), local.slot.WorktreeRoot, "claude")
				} else {
					result, err = env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude")
				}
				if err != nil {
					t.Fatalf("nested actor blocked outer %s: %v", operation, err)
				}
				if result.Handle == nil || result.Record.Thread.RepoScope != outer.Address.RepoScope {
					t.Fatalf("wrong outer operation result: %+v", result)
				}
				if _, err := env.Couch.Threads.GetThread(inner.Address); err != nil {
					t.Fatalf("outer operation lost nested conversation: %v", err)
				}
				if env.Proc.Exists(actor.PID) != Live {
					t.Fatal("outer operation killed nested actor")
				}
			})
		}
	}
}

func TestSlotHostedActorSameScopeRemainsConservative(t *testing.T) {
	for _, location := range []string{"inside", "outside"} {
		t.Run(location, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			owner := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "live-owner")
			path := local.slot.WorktreeRoot
			if location == "outside" {
				path = t.TempDir()
			}
			actor := ActorRecord{ID: "owner", Thread: owner.Address, Args: StartArgs{Worktree: Worktree(path), Cwd: path, Stack: "claude"}, StartedAt: env.Now, PID: 4343, Identity: "live"}
			env.Proc.Set(actor.PID, actor.Identity)
			env.Couch.reg = env.Couch.reg.Insert(actor)
			observation, err := env.Couch.ObserveSlotSessions(context.Background(), *local.slot)
			if err == nil && observation.Absent {
				t.Fatal("same-scope actor authorized absence")
			}
			if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
				t.Fatal("same-scope live actor allowed fresh replacement")
			}
		})
	}
}

func TestSlotHostedActorWithoutAddressCannotProveAbsence(t *testing.T) {
	for _, registry := range []string{"Actors", "ActorRegistry"} {
		t.Run(registry, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			actor := ActorRecord{ID: "legacy-owner", Args: StartArgs{Worktree: Worktree(local.slot.WorktreeRoot), Stack: "claude"}, StartedAt: env.Now, PID: 4545, Identity: "legacy-live"}
			env.Proc.Set(actor.PID, actor.Identity)
			if registry == "Actors" {
				env.Couch.reg = env.Couch.reg.Insert(actor)
			} else if err := env.Couch.Store.Save(NewRegistry().Insert(actor), env.Couch.names); err != nil {
				t.Fatal(err)
			}
			if observation, err := env.Couch.ObserveSlotSessions(context.Background(), *local.slot); err == nil {
				t.Fatalf("unaddressed contained actor granted observation: %+v", observation)
			}
		})
	}
}
