package couchcore

import (
	"context"
	"testing"
	"time"
)

func TestPublicWarmAttachCompletionNeedsListingNotLifecycleOwner(t *testing.T) {
	for _, state := range []string{"attached", "detached"} {
		t.Run(state, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.resumeRegistrationTimeout = 100 * time.Millisecond
			record := validThreadRecord(t)
			record.Reservation = false
			before, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			const name = "📁repo-legacy"
			indexSession(t, root, before.Address, name)
			checker, _ := sandboxedChecker(t, root, map[string]string{name: state})
			checker.OwnerProbe = orphanOwnerProbe{}
			env.Couch.Artifacts = checker
			_, h, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
			if h != nil {
				t.Cleanup(func() { env.Runner.SetExited(h.ID(), 0) })
			}
			if err != nil {
				t.Fatalf("listed terminal rejected unresolved lifecycle ownership: %v", err)
			}
			if h == nil || !h.Alive() {
				t.Fatal("missing live attach helper")
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}

func TestPublicWarmAttachRejectsExitedAcknowledgedHelper(t *testing.T) {
	env, before, _ := readonlyWarmFixture(t, validThreadRecord(t), Dead)
	env.Runner.AfterAcknowledge = func(id string) error { env.Runner.SetExited(id, 0); return nil }
	_, _, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
	if err == nil {
		t.Fatal("exited helper accepted because terminal remains listed")
	}
	assertWarmRecordUnchanged(t, env, before)
}
