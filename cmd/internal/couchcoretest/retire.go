// Package couchcoretest holds couchcore fixtures for other packages' tests.
package couchcoretest

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// RetireProof is a teardown proof for the record as it stands, for a test
// that arranges archived state through the store directly: the store asks
// every retirement for its reason (pair#695), and in such a test the tree
// was never up.
func RetireProof(t testing.TB, store *couchcore.ThreadStore, address couchcore.ThreadAddress) couchcore.TeardownProof {
	t.Helper()
	record, err := store.GetThread(address)
	if err != nil {
		t.Fatalf("retire proof: %v", err)
	}
	lifecycle, err := couchcore.NewSlotAddress(couchcore.SlotIdentity{}, record.StartingPath)
	if err != nil {
		t.Fatalf("retire proof: %v", err)
	}
	return couchcore.TeardownProof{Address: lifecycle, Thread: address, Revision: record.Revision, ProvenAt: time.Now()}
}
