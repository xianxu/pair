package couchcore

import (
	"strings"
	"testing"
)

func TestRecoverySelectionRequiresCurrentOpaqueKey(t *testing.T) {
	record, observation, now := recoveryPreviewFixture()
	p, err := DecideSlotRecovery(record, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	choice, err := selectRecoveryChoice(p, p.Candidates[0].Key)
	if err != nil || choice.NativeID != "native" {
		t.Fatalf("selection %+v %v", choice, err)
	}
	if _, err := selectRecoveryChoice(p, "1"); err == nil {
		t.Fatal("rank became authority")
	}
	if _, err := selectRecoveryChoice(p, strings.Repeat("x", 4096)); err == nil {
		t.Fatal("unbounded key")
	}
	observation.Capture.RawIdentity.Size++
	changed, err := DecideSlotRecovery(record, observation, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selectRecoveryChoice(changed, p.Candidates[0].Key); err == nil {
		t.Fatal("stale source selected")
	}
	p.Incomplete = true
	if _, err := selectRecoveryChoice(p, p.Candidates[0].Key); err == nil {
		t.Fatal("incomplete selection admitted")
	}
}
