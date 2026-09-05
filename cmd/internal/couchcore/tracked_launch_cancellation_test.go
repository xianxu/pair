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

	_, handle, err := env.Couch.SpawnPrepared(ctx, StartArgs{Worktree: "/repo"}, prepared.Resolution.Fingerprint)
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

// After the helper has ACKNOWLEDGED, the target has already executed and Pair
// has registered. Cancellation then reaps the helper but must NOT delete the
// record: registration is established, so the thread cannot be proven free, and
// occupied-or-proven-free (starttransaction.go) keeps it as an `unknown`
// incarnation for the crash reconciler.
//
// This test previously claimed the opposite -- "ReapsAndRollsBack" -- and
// asserted ErrThreadNotFound at a HARDCODED address. It passed only because
// that address was never the one allocated (the real record sat at a different
// tag), so the assertion proved nothing. pair#170 M4 shifted the entropy the
// tag derives from, the address collided with the real record, and the vacuum
// showed. Asserting over the whole snapshot is what keeps it honest.
func TestTrackedLaunchCancellationAfterAcknowledgementReapsAndKeepsRecordOccupied(t *testing.T) {
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

	_, handle, err := env.Couch.SpawnPrepared(ctx, StartArgs{Worktree: "/repo"}, prepared.Resolution.Fingerprint)
	if !errors.Is(err, context.Canceled) || handle == nil {
		t.Fatalf("SpawnPrepared handle/error = %T, %v", handle, err)
	}
	child := env.Runner.Child(handle.ID())
	if handle.Alive() || child.ExecCount != 1 {
		t.Fatalf("post-ack canceled child = %+v", child)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 {
		t.Fatalf("post-ack cancel left %d records, want the one it could not prove free", len(snapshot.Records))
	}
	record := snapshot.Records[0]
	if len(record.Incarnations) != 1 {
		t.Fatalf("retained record incarnations = %+v, want exactly one", record.Incarnations)
	}
	if incarnation := record.Incarnations[0]; incarnation.State != IncarnationUnknown || incarnation.Start != nil {
		t.Fatalf("retained incarnation = %+v, want unknown state with the start claim resolved", incarnation)
	}
}
