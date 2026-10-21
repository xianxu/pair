package couchschedule

import (
	"testing"
)

func TestStatusSnapshotIsSortedDetachedAndBounded(t *testing.T) {
	s, due := taskFixture(t, true)
	s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "first"})
	other := s
	other.Key.Hub = "/another"
	other.Active = nil
	other.Allowed = false
	other.Refusal = "owner missing"
	snapshot := BuildStatus([]TaskState{s, other}, "UTC")
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tasks) != 2 || snapshot.Tasks[0].Hub != "/another" || snapshot.Tasks[0].State != "refused" || snapshot.Tasks[1].State != "reserved" {
		t.Fatalf("unexpected projection: %+v", snapshot)
	}
	snapshot.Tasks[1].Active.Stage = "changed"
	if s.Active.Stage != "reserved" {
		t.Fatal("snapshot aliases authoritative attempt")
	}
	code := 10
	s.Last = &RunSummary{Exit: &code}
	snapshot = BuildStatus([]TaskState{s}, "UTC")
	*snapshot.Tasks[0].Last.Exit = 7
	if *s.Last.Exit != 10 {
		t.Fatal("snapshot aliases the last run exit")
	}
	snapshot.Version = 2
	if snapshot.Validate() == nil {
		t.Fatal("accepted unknown snapshot version")
	}
}
