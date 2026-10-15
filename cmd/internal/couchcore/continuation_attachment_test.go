package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestContinuationPublicationKeepsBorrowedAttachmentUntilExecution(t *testing.T) {
	env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Dead)
	actor, h := assertReadonlyWarmAttach(t, env, before, session)
	env.Proc.Set(actor.PID, actor.Identity)
	proof := ContinuationSource{Agent: "claude", Session: session, LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "checkpoint.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nVerify attachment ownership.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := env.Couch.RequestContinuation(context.Background(), before.Address, proof, path)
	if err != nil {
		t.Fatal(err)
	}
	published, err := env.Couch.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(published.Incarnations, before.Incarnations) {
		t.Fatal("publication adopted helper")
	}
	if published.Continuation.Source.Helper.PID != actor.PID || published.Continuation.ID != status.RequestID {
		t.Fatal("publication lost exact local helper")
	}
	ctx, release, err := env.Couch.hold(context.Background(), before.Address, "continue-thread")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	local := env.Couch.warmClients
	env.Couch.warmClients = nil
	if _, _, other, err := env.Couch.ensureContinuationAttached(ctx, published, false); err == nil || other != nil {
		t.Fatal("restored registry granted lifecycle execution authority")
	}
	env.Couch.warmClients = local
	adopted, _, newHandle, err := env.Couch.ensureContinuationAttached(ctx, published, false)
	if err != nil {
		t.Fatal(err)
	}
	if newHandle != nil || !h.Alive() {
		t.Fatal("execution replaced existing helper")
	}
	if len(adopted.Incarnations) != 1 || adopted.Incarnations[0].PID != actor.PID {
		t.Fatal("execution did not adopt exact helper")
	}
}

func TestContinuationPublicationFromRegistryRequiresExactLiveListedHelper(t *testing.T) {
	for _, defect := range []string{"unknown-helper", "recycled-helper", "missing-session", "ambiguous-helpers"} {
		t.Run(defect, func(t *testing.T) {
			env, before, session := readonlyWarmFixture(t, warmDetachedRecord(t), Dead)
			actor, _ := assertReadonlyWarmAttach(t, env, before, session)
			env.Proc.Set(actor.PID, actor.Identity)
			env.Couch.warmClients = nil // publication process has only persisted registry evidence
			proof := ContinuationSource{Agent: "claude", Session: session, LaunchOrdinal: 2}
			env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
			switch defect {
			case "unknown-helper":
				env.Proc.SetUnknown(actor.PID)
			case "recycled-helper":
				env.Proc.Set(actor.PID, "reused-generation")
			case "missing-session":
				env.Artifacts.SetDetachedSession(before.Address, "")
			case "ambiguous-helpers":
				other := actor
				other.ID = env.Couch.IDs.NewID()
				other.PID++
				other.Identity = "other-helper"
				env.Proc.Set(other.PID, other.Identity)
				if err := env.Couch.mutateRegistry(func(r Registry) (Registry, error) { return r.Insert(other), nil }); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(t.TempDir(), "checkpoint.md")
			if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nVerify source.\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := env.Couch.RequestContinuation(context.Background(), before.Address, proof, path); err == nil {
				t.Fatal("unproved publication source accepted")
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}
