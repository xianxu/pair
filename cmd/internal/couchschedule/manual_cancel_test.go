package couchschedule

import (
	"testing"
	"time"
)

func TestManualReservedCancellationRetainsSettlementHistory(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	prior := TaskState{Key: TaskKey{Hub: "/hub", Task: "task"}, Revision: "r", Allowed: true, Active: &Attempt{ID: "manual-1", Revision: "r", Source: "manual", Stage: "reserved", StartedAt: now.Add(-time.Second), Deadline: now.Add(time.Minute), ManualIntentDigest: "digest"}}
	result := PlanSettlement(prior, nil, Event{Kind: CancelRequested, At: now, Attempt: "manual-1", Revision: "r"})
	if result.State.Active != nil || result.State.Last == nil || result.State.Last.Outcome != "cancelled" || result.State.Last.ManualIntentDigest != "digest" || result.Next != SettlementPublishHistory {
		t.Fatalf("cancellation lost occurrence: %+v", result)
	}
}
