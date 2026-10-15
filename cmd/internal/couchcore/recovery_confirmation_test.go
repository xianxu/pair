package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRecoveryConfirmationBindsActorSelectionAndSource(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	ctx := WithRecoveryConfirmer(context.Background(), "hub-generation-A")
	request := RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh, Reference: "pair:99"}
	witness := recoveryConfirmationWitness{Address: f.record.Address, Revision: f.record.Revision, Selection: request, Digest: "owner-generation-A"}
	offer, err := f.Couch.issueRecoveryConfirmation(ctx, witness, []RecoveryStopProcess{{PID: 42, Identity: "old-helper", Role: "client helper"}})
	if err != nil || offer.Token == "" || !strings.Contains(offer.Command, "couch --recovery-confirmation ") {
		t.Fatalf("offer=%+v err=%v", offer, err)
	}
	for _, change := range []string{"actor", "source", "selection"} {
		t.Run(change, func(t *testing.T) {
			check, caller := witness, ctx
			switch change {
			case "actor":
				caller = WithRecoveryConfirmer(context.Background(), "another-hub")
			case "source":
				check.Digest = "reused-pid"
			case "selection":
				check.Selection.Kind = RecoveryProven
			}
			if err := f.Couch.consumeRecoveryConfirmation(caller, offer.Token, check); err == nil {
				t.Fatal("changed authority accepted")
			}
		})
	}
	if ref, err := f.Couch.RecoveryConfirmationReference(ctx, offer.Token); err != nil || ref != "pair:99" {
		t.Fatalf("reference=%q err=%v", ref, err)
	}
	if err := f.Couch.consumeRecoveryConfirmation(ctx, offer.Token, witness); err != nil {
		t.Fatal(err)
	}
	if err := f.Couch.consumeRecoveryConfirmation(ctx, offer.Token, witness); err == nil {
		t.Fatal("acceptance replayed")
	}
}

func TestRecoveryConfirmationExpiryCancellationAndRestart(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	ctx := WithRecoveryConfirmer(context.Background(), "hub")
	witness := recoveryConfirmationWitness{Address: f.record.Address, Revision: f.record.Revision, Selection: RecoverySelection{Kind: RecoveryFresh, Reference: "pair:99"}, Digest: "source"}
	offer, err := f.Couch.issueRecoveryConfirmation(ctx, witness, nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := f.Couch.consumeRecoveryConfirmation(cancelled, offer.Token, witness); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled acceptance: %v", err)
	}
	f.Couch.Clock = FixedClock{T: f.Now.Add(2*time.Minute + time.Nanosecond)}
	if _, err := f.Couch.RecoveryConfirmationReference(ctx, offer.Token); err == nil {
		t.Fatal("expired acceptance accepted")
	}
	other := newRecoveryExecutionFixture(t)
	if _, err := other.Couch.RecoveryConfirmationReference(ctx, offer.Token); err == nil {
		t.Fatal("acceptance survived Couch replacement")
	}
}
