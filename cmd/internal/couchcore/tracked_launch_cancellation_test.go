package couchcore

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestTrackedLaunchCancellationBeforeAcknowledgementReapsAndRollsBack(t *testing.T) {
	env := newTestEnv(t, "/repo")
	prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("PrepareStart: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	env.Runner.AfterBlockedStart = func(string) { cancel() }

	_, handle, err := env.Couch.SpawnPrepared(ctx, prepared.Token)
	if !errors.Is(err, context.Canceled) || handle == nil {
		t.Fatalf("SpawnPrepared handle/error = %T, %v", handle, err)
	}
	child := env.Runner.Child(handle.ID())
	if handle.Alive() || child.ExecCount != 0 {
		t.Fatalf("pre-ack canceled child = %+v", child)
	}
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	if _, err := env.Couch.Threads.GetThread(address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("pre-ack canceled thread remained: %v", err)
	}
}

func TestTrackedLaunchCancellationRestoresVerifiedParkOnResume(t *testing.T) {
	for _, phase := range []string{"before-ack", "after-ack"} {
		t.Run(phase, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
			env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), false)
			ctx, cancel := context.WithCancel(context.Background())
			if phase == "before-ack" {
				env.Runner.AfterBlockedStart = func(string) { cancel() }
			} else {
				env.Runner.AfterAcknowledge = func(string) error {
					cancel()
					return nil
				}
			}

			_, handle, err := env.Couch.ResumeContext(ctx, parked.Address)
			if !errors.Is(err, context.Canceled) || handle == nil {
				t.Fatalf("ResumeContext handle/error = %T, %v", handle, err)
			}
			if handle.Alive() {
				t.Fatal("canceled resume helper remained alive")
			}
			assertVerifiedParkRestored(t, env.Couch.Threads, parked)
		})
	}
}

func TestTrackedLaunchCancellationAfterAcknowledgementReapsAndRollsBack(t *testing.T) {
	env := newTestEnv(t, "/repo")
	prepared, err := env.Couch.PrepareStart(context.Background(), StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("PrepareStart: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	env.Runner.AfterAcknowledge = func(string) error {
		cancel()
		return nil
	}

	_, handle, err := env.Couch.SpawnPrepared(ctx, prepared.Token)
	if !errors.Is(err, context.Canceled) || handle == nil {
		t.Fatalf("SpawnPrepared handle/error = %T, %v", handle, err)
	}
	child := env.Runner.Child(handle.ID())
	if handle.Alive() || child.ExecCount != 1 {
		t.Fatalf("post-ack canceled child = %+v", child)
	}
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	if _, err := env.Couch.Threads.GetThread(address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("post-ack canceled thread remained: %v", err)
	}
}
