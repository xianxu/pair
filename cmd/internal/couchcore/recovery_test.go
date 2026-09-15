package couchcore

import (
	"reflect"
	"testing"
)

func TestDecideRecoveryNeverInfersDeathFromMissingAttachment(t *testing.T) {
	for _, helper := range []Liveness{Live, Unknown, Dead} {
		for _, presence := range []SessionPresence{PresenceUnobserved, PresenceAbsent, PresencePresent} {
			for _, detached := range []bool{false, true} {
				r := ThreadRecord{Incarnations: []ThreadIncarnation{{PID: 42, Identity: "original", State: IncarnationLive}}}
				in := RecoveryEvidence{Thread: r, Helper: helper, Presence: presence, Detached: detached, Checkpoint: true}
				before := cloneThreadRecord(r)
				got := DecideRecovery(in)
				allowed := helper == Dead && (presence == PresenceAbsent || presence == PresencePresent && detached)
				if got.Recover != allowed || got.Archive != allowed {
					t.Fatalf("helper=%v presence=%v detached=%v: %+v", helper, presence, detached, got)
				}
				if !reflect.DeepEqual(before, r) {
					t.Fatal("decision mutated its input")
				}
			}
		}
	}
}

func TestDecideRecoveryMissingCheckpointLeavesArchive(t *testing.T) {
	got := DecideRecovery(RecoveryEvidence{Presence: PresenceAbsent, Helper: Dead})
	if got.Recover || !got.FromCheckpoint || !got.Archive || got.Diagnosis == "" {
		t.Fatalf("decision %+v", got)
	}
}
