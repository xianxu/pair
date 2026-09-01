package couchcore

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestProjectActionableThreadsRequiresExactLifecycleProof(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	live := actionableTestThread("couch-0000000000000002", now)
	live.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "live-process", State: IncarnationLive,
	}}
	parked := actionableTestThread("couch-0000000000000001", now.Add(-time.Hour))
	parked.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&parked, now.Add(-time.Hour))

	rows := ProjectActionableThreads(
		[]ThreadRecord{live, parked},
		[]LiveTTYObservation{{
			Address: live.Address,
			Process: ProcessIdentity{PID: 42, Identity: "live-process"},
		}}, []ParkedResumeObservation{{Address: parked.Address, Agent: "claude", NativeID: "native-root-1"}},
	)

	want := []ActionableThreadSummary{
		{Address: parked.Address, StartingPath: "/repo", WorkingPath: "/repo", State: ThreadParked, LastActiveAt: parked.LastActiveAt},
		{Address: live.Address, StartingPath: "/repo", WorkingPath: "/repo", State: ThreadLive, LastActiveAt: live.LastActiveAt},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("actionable rows = %+v, want %+v", rows, want)
	}
}

func TestProjectActionableThreadsOmitsVerifiedParkWithoutResumeAuthority(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	parked := actionableTestThread("couch-0000000000000001", now)
	parked.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&parked, now)

	if rows := ProjectActionableThreads([]ThreadRecord{parked}, nil, nil); len(rows) != 0 {
		t.Fatalf("unbound verified park projected as actionable: %+v", rows)
	}
}

func TestProjectActionableThreadsRequiresOneMatchingSupportedResumeProof(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	base := actionableTestThread("couch-0000000000000001", now)
	base.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&base, now)
	proof := ParkedResumeObservation{Address: base.Address, Agent: "claude", NativeID: "native-root-1"}
	tests := []struct {
		name   string
		mutate func(*ThreadRecord)
		proofs []ParkedResumeObservation
		want   int
	}{
		{name: "exact", proofs: []ParkedResumeObservation{proof}, want: 1},
		{name: "missing"},
		{name: "duplicate", proofs: []ParkedResumeObservation{proof, proof}},
		{name: "wrong address", proofs: []ParkedResumeObservation{{Address: ThreadAddress{RepoScope: "other", Tag: base.Address.Tag}, Agent: "claude", NativeID: "native-root-1"}}},
		{name: "wrong agent", proofs: []ParkedResumeObservation{{Address: base.Address, Agent: "codex", NativeID: "native-root-1"}}},
		{name: "empty native id", proofs: []ParkedResumeObservation{{Address: base.Address, Agent: "claude"}}},
		{name: "unsupported saved agent", mutate: func(record *ThreadRecord) {
			record.LatestLaunchProfile.Agent = "unknown"
		}, proofs: []ParkedResumeObservation{{Address: base.Address, Agent: "unknown", NativeID: "native-root-1"}}},
		{name: "missing argv", mutate: func(record *ThreadRecord) {
			record.LatestLaunchProfile.Argv = nil
		}, proofs: []ParkedResumeObservation{proof}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := cloneThreadRecord(base)
			if tc.mutate != nil {
				tc.mutate(&record)
			}
			if rows := ProjectActionableThreads([]ThreadRecord{record}, nil, tc.proofs); len(rows) != tc.want {
				t.Fatalf("rows = %+v, want %d", rows, tc.want)
			}
		})
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
			name: "structurally invalid live record",
			mutate: func(record *ThreadRecord) {
				record.SchemaVersion = 0
				record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "live-process", State: IncarnationLive}}
			},
			observation: &LiveTTYObservation{Process: ProcessIdentity{PID: 42, Identity: "live-process"}},
		},
		{
			name: "structurally invalid verified park",
			mutate: func(record *ThreadRecord) {
				record.VerifiedPark = &VerifiedPark{ParkedAt: now}
			},
		},
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
			if rows := ProjectActionableThreads([]ThreadRecord{record}, observations, nil); len(rows) != 0 {
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

func TestActionableThreadInventoryIncludesOnlyEstablishedParkedBinding(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	rows, err := couch.ActionableThreadInventory(nil)
	if err != nil || len(rows) != 1 || rows[0].Address != created.Address || rows[0].State != ThreadParked {
		t.Fatalf("established parked inventory = %+v, %v", rows, err)
	}
}

func TestActionableThreadInventoryOmitsParkedThreadWithUnavailableWorkingPath(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	paths := NewFakePathOps(nil)
	paths.Fail(record.WorkingPath)
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: paths}

	rows, err := couch.ActionableThreadInventory(nil)
	if err != nil || len(rows) != 0 {
		t.Fatalf("missing-path parked inventory = %+v, %v", rows, err)
	}
}

func TestActionableThreadInventoryExposesContextBoundQuery(t *testing.T) {
	type contextInventory interface {
		ActionableThreadInventoryContext(context.Context, []LiveTTYObservation) ([]ActionableThreadSummary, error)
	}
	if _, ok := any(&Couch{}).(contextInventory); !ok {
		t.Fatal("Couch actionable inventory has no context-bound query")
	}
}

type cancelingNativeBindingArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	entered chan struct{}
}

func (a *cancelingNativeBindingArtifacts) ResolveEstablished(ctx context.Context, _, _, _ string) (NativeBindingResolution, error) {
	close(a.entered)
	<-ctx.Done()
	return NativeBindingResolution{}, ctx.Err()
}

func TestActionableThreadInventoryCancelsBlockedBindingResolution(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	artifacts := &cancelingNativeBindingArtifacts{
		FakeThreadArtifactCollisionChecker: NewFakeThreadArtifactCollisionChecker(),
		entered:                            make(chan struct{}),
	}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := couch.ActionableThreadInventoryContext(ctx, nil)
		done <- err
	}()

	select {
	case <-artifacts.entered:
		cancel()
	case err := <-done:
		t.Fatalf("inventory returned before entering context resolver: %v", err)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("inventory did not enter binding resolver")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("inventory error = %v, want context canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("canceled inventory did not return")
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
		SchemaVersion:   ThreadSchemaVersion,
		Address:         ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: tag},
		StartingPath:    "/repo",
		WorkingPath:     "/repo",
		CreatedAt:       time.Unix(1, 0).UTC(),
		Revision:        1,
		ClaimGeneration: 1,
		LastActiveAt:    active,
	}
}

func markActionableParked(record *ThreadRecord, parkedAt time.Time) {
	identity := ParkIdentity{
		Nonce: "park-0123456789abcdef", Address: record.Address,
		PID: 42, ProcessIdentity: "parked-process",
	}
	record.Revision = 3
	record.LastActiveAt = parkedAt
	record.ParkHistory = []ParkTransaction{{
		Identity: identity, BaseRevision: 1, RecordRevision: 2,
		Phase: ParkAwaitingCompletion, Attempts: []ParkAttempt{{Number: 1, Closed: true}},
		Closed: true, SuccessfulAttempt: 1,
	}}
	record.VerifiedPark = &VerifiedPark{Identity: identity, Attempt: 1, ParkedAt: parkedAt}
}
