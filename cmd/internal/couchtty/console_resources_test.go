package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"testing"
)

func TestResourceInventoryNotificationDoesNotCallIOUnderLock(t *testing.T) {
	c := &Console{}
	changes := c.ResourceChanges()
	c.mu.Lock()
	c.resourceRows = []couchcore.ActionableThreadSummary{{Name: "one", Recovery: &couchcore.RecoveryDecision{Diagnosis: "proof"}}}
	c.resourcesChangedLocked()
	c.mu.Unlock()
	select {
	case <-changes:
	default:
		t.Fatal("missing change")
	}
	snapshot := c.ResourceInventory()
	snapshot.Rows[0].Name = "mutated"
	snapshot.Rows[0].Recovery.Diagnosis = "mutated"
	if c.ResourceInventory().Rows[0].Recovery.Diagnosis != "proof" {
		t.Fatal("nested metadata aliases snapshot")
	}
	if c.ResourceInventory().Rows[0].Name != "one" {
		t.Fatal("snapshot aliases inventory")
	}
	c.mu.Lock()
	c.resourceRows = nil
	c.resourcesChangedLocked()
	c.mu.Unlock()
	if len(c.ResourceInventory().Rows) != 0 || c.ResourceInventory().Generation <= snapshot.Generation {
		t.Fatal("removal not published")
	}
}

func TestResourceInventoryRefreshIgnoresStaleResults(t *testing.T) {
	c := &Console{refreshRequests: make(chan struct{}, 1), refreshResults: make(chan menuRefreshResult, 1)}
	c.ResourceChanges()
	// Invoke the snapshot publication section through the real refresh finisher.
	c.refreshSchedule.Running = 2
	c.finishMenuRefresh(menuRefreshResult{generation: 1, inventory: []couchcore.ActionableThreadSummary{{Name: "stale"}}})
	if len(c.ResourceInventory().Rows) != 0 {
		t.Fatal("stale inventory published")
	}
}
