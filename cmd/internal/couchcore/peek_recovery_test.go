package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"testing"
)

func TestPreviousPeekUsesRecoveryScreenWithoutManufacturedPark(t *testing.T) {
	r, o, now := recoveryPreviewFixture()
	o.Capture.Token = "20261013T080000"
	p, err := DecideSlotRecovery(r, o, now)
	if err != nil {
		t.Fatal(err)
	}
	r.RecoveryContext = threadrecord.CloneRecoveryContext(p.Snapshot)
	got, err := SelectPeekSource(r, true)
	if err != nil || got.Token != o.Capture.Token || got.Agent != "codex" {
		t.Fatalf("previous screen %+v %v", got, err)
	}
	if r.VerifiedPark != nil || r.PreviousSession != nil {
		t.Fatal("invented park proof")
	}
	r.RecoveryContext.Capture.Token = ""
	if _, err := SelectPeekSource(r, true); err == nil {
		t.Fatal("unarchived source treated as previous")
	}
}
