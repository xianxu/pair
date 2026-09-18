package couchcmd

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// seedLiveIncarnation puts a record in the state a running helper leaves behind,
// by driving the REAL start transaction rather than assigning the field.
//
// #256 M3 closed the arbitrary-mutation door, and these two acceptance tests are
// the reason the plan budgeted for its test callers rather than meeting them at
// compile time: a fixture that assigns `Incarnations` writes a shape no start
// sequence produces, so an acceptance test could pass against a record
// production can never create. The claim → helper-recorded → registered sequence
// is what couch itself runs, so the fixture and production agree by
// construction.
func seedLiveIncarnation(t *testing.T, store *couchcore.ThreadStore, record couchcore.ThreadRecord, helper couchcore.ProcessIdentity) couchcore.ThreadRecord {
	t.Helper()
	const nonce = "seed-live-incarnation"
	claimed, err := store.CommitStartClaim(record.Address, record.Revision, "/repo/.git", time.Unix(1, 0).UTC(),
		couchcore.StartEvent{
			Kind: couchcore.StartClaimed, Nonce: nonce, Shape: couchcore.StartFreshExisting,
			Owner:   couchcore.SupervisorOwner{PID: 900, Identity: "fake-supervisor-token"},
			Profile: record.LatestLaunchProfile,
		})
	if err != nil {
		t.Fatalf("seed start claim: %v", err)
	}
	recorded, err := store.AdvanceStart(record.Address, claimed.Revision, couchcore.StartEvent{
		Kind: couchcore.StartHelperRecorded, Nonce: nonce, Helper: helper,
	})
	if err != nil {
		t.Fatalf("seed helper record: %v", err)
	}
	live, err := store.AdvanceStart(record.Address, recorded.Revision, couchcore.StartEvent{
		Kind: couchcore.StartRegistered, Nonce: nonce,
	})
	if err != nil {
		t.Fatalf("seed registration: %v", err)
	}
	return live
}
