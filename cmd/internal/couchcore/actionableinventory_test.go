package couchcore

import (
	"reflect"
	"testing"
	"time"
)

func TestProjectActionableThreadsRequiresExactLifecycleProof(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	live := actionableTestThread("couch-0000000000000002", now)
	live.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "live-process", State: IncarnationLive,
	}}
	parked := actionableTestThread("couch-0000000000000001", now.Add(-time.Hour))
	parked.VerifiedPark = &VerifiedPark{ParkedAt: now.Add(-time.Hour)}

	rows := ProjectActionableThreads(
		[]ThreadRecord{live, parked},
		[]LiveTTYObservation{{
			Address: live.Address,
			Process: ProcessIdentity{PID: 42, Identity: "live-process"},
		}},
	)

	want := []ActionableThreadSummary{
		{Address: parked.Address, StartingPath: "/repo", WorkingPath: "/repo", State: ThreadParked, LastActiveAt: parked.LastActiveAt},
		{Address: live.Address, StartingPath: "/repo", WorkingPath: "/repo", State: ThreadLive, LastActiveAt: live.LastActiveAt},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("actionable rows = %+v, want %+v", rows, want)
	}
}

func TestProjectActionableThreadsFailsClosedOnContradictoryEvidence(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	tests := []struct {
		name        string
		mutate      func(*ThreadRecord)
		observation *LiveTTYObservation
	}{
		{
			name: "persisted live without owner observation",
			mutate: func(record *ThreadRecord) {
				record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "live-process", State: IncarnationLive}}
			},
		},
		{
			name: "owner observation mismatches process identity",
			mutate: func(record *ThreadRecord) {
				record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "live-process", State: IncarnationLive}}
			},
			observation: &LiveTTYObservation{Process: ProcessIdentity{PID: 42, Identity: "replacement"}},
		},
		{
			name: "verified park with occupied incarnation",
			mutate: func(record *ThreadRecord) {
				record.VerifiedPark = &VerifiedPark{ParkedAt: now}
				record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "unknown", State: IncarnationUnknown}}
			},
		},
		{
			name: "verified park with active transaction",
			mutate: func(record *ThreadRecord) {
				record.VerifiedPark = &VerifiedPark{ParkedAt: now}
				record.Park = &ParkTransaction{Phase: ParkUnknown}
			},
		},
		{
			name: "simultaneously live and verified parked",
			mutate: func(record *ThreadRecord) {
				record.VerifiedPark = &VerifiedPark{ParkedAt: now}
				record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "live-process", State: IncarnationLive}}
			},
			observation: &LiveTTYObservation{Process: ProcessIdentity{PID: 42, Identity: "live-process"}},
		},
		{
			name: "reservation",
			mutate: func(record *ThreadRecord) {
				record.Reservation = true
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := actionableTestThread("couch-0000000000000001", now)
			tc.mutate(&record)
			var observations []LiveTTYObservation
			if tc.observation != nil {
				observation := *tc.observation
				observation.Address = record.Address
				observations = append(observations, observation)
			}
			if rows := ProjectActionableThreads([]ThreadRecord{record}, observations); len(rows) != 0 {
				t.Fatalf("contradictory row projected as actionable: %+v", rows)
			}
		})
	}
}

func TestActionableThreadSummaryOwnsDisplayMetadata(t *testing.T) {
	row := ActionableThreadSummary{
		Address:          ThreadAddress{Tag: "couch-0000000000000001"},
		Name:             "compiler",
		Description:      "operator",
		PublishedSummary: "agent",
		State:            ThreadLive,
	}
	if !row.Live() || row.Label() != "compiler" || row.DisplaySummary() != "agent" {
		t.Fatalf("display projection = %+v", row)
	}
}

func TestActionableThreadInventorySnapshotsAndOwnsRows(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.Name = "compiler"
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "live-process", State: IncarnationLive,
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	couch := &Couch{Threads: store}
	observations := []LiveTTYObservation{{
		Address: created.Address,
		Process: ProcessIdentity{PID: 42, Identity: "live-process"},
	}}

	rows, err := couch.ActionableThreadInventory(observations)
	if err != nil || len(rows) != 1 || rows[0].Name != "compiler" || rows[0].State != ThreadLive {
		t.Fatalf("ActionableThreadInventory = %+v, %v", rows, err)
	}
	rows[0].Name = "mutated"
	again, err := couch.ActionableThreadInventory(observations)
	if err != nil || len(again) != 1 || again[0].Name != "compiler" {
		t.Fatalf("inventory row aliases durable state: %+v, %v", again, err)
	}
}

func TestActionableThreadInventoryDistinguishesSnapshotFailureFromEmpty(t *testing.T) {
	rows, err := (&Couch{}).ActionableThreadInventory(nil)
	if err == nil || rows != nil {
		t.Fatalf("snapshot failure = %+v, %v; want nil rows and error", rows, err)
	}

	store, _ := newTestThreadStore(t)
	rows, err = (&Couch{Threads: store}).ActionableThreadInventory(nil)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty inventory = %#v, %v; want owned empty slice", rows, err)
	}
}

func actionableTestThread(tag ThreadTag, active time.Time) ThreadRecord {
	return ThreadRecord{
		Address:      ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: tag},
		StartingPath: "/repo",
		WorkingPath:  "/repo",
		LastActiveAt: active,
	}
}
