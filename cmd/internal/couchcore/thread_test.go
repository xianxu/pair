package couchcore

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestThreadRecordLifecycleConvertersAndClonesAreDefensive(t *testing.T) {
	record := validThreadRecord(t)
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "helper", State: IncarnationLive,
		LaunchProfile: &LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}},
	}}
	record.LatestLaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	record.LastActiveAt = time.Unix(2, 0).UTC()
	record.Park = &ParkTransaction{
		Identity: ParkIdentity{
			Nonce: "park-0123456789abcdef", Address: record.Address,
			PID: 42, ProcessIdentity: "helper",
		},
		BaseRevision: 1, RecordRevision: 2, Phase: ParkRequested,
		Attempts: []ParkAttempt{{Number: 1, Failure: &ParkFailure{Code: "request_publish_failed", Diagnostic: "disk full"}}},
	}
	record.ParkHistory = []ParkTransaction{{
		Identity: ParkIdentity{
			Nonce: "park-fedcba9876543210", Address: record.Address,
			PID: 41, ProcessIdentity: "old-helper",
		},
		BaseRevision: 1, RecordRevision: 2, Phase: ParkUnknown,
		Attempts: []ParkAttempt{{Number: 1}}, Closed: true, Tombstoned: true,
	}}

	persisted := toPersistedThreadRecord(record)
	roundTrip := fromPersistedThreadRecord(persisted)
	if !reflect.DeepEqual(roundTrip, record) {
		t.Fatalf("round trip = %#v, want %#v", roundTrip, record)
	}

	cloned := cloneThreadRecord(record)
	cloned.LatestLaunchProfile.Argv[0] = "changed"
	cloned.Park.Identity.Nonce = "changed"
	cloned.Park.Attempts[0].Failure.Diagnostic = "changed"
	cloned.ParkHistory[0].Attempts[0].Number = 99
	if record.LatestLaunchProfile.Argv[0] != "--sandbox" ||
		record.Park.Identity.Nonce != "park-0123456789abcdef" ||
		record.Park.Attempts[0].Failure.Diagnostic != "disk full" ||
		record.ParkHistory[0].Attempts[0].Number != 1 {
		t.Fatalf("clone aliased lifecycle input: %+v", record)
	}
}

func validThreadRecord(t *testing.T) ThreadRecord {
	t.Helper()
	return ThreadRecord{
		SchemaVersion: ThreadSchemaVersion,
		Address:       ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"},
		StartingPath:  "/repo",
		WorkingPath:   "/repo",
		CreatedAt:     time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		Revision:      1,
	}
}

func TestThreadStoreValidateThreadRecordRejectsInvalidBoundaries(t *testing.T) {
	base := validThreadRecord(t)
	tests := []struct {
		name   string
		mutate func(*ThreadRecord)
	}{
		{name: "zero schema", mutate: func(r *ThreadRecord) { r.SchemaVersion = 0 }},
		{name: "unknown schema", mutate: func(r *ThreadRecord) { r.SchemaVersion++ }},
		{name: "scope traversal", mutate: func(r *ThreadRecord) { r.Address.RepoScope = "../other" }},
		{name: "tag traversal", mutate: func(r *ThreadRecord) { r.Address.Tag = "../other" }},
		{name: "relative starting path", mutate: func(r *ThreadRecord) { r.StartingPath = "relative" }},
		{name: "relative working path", mutate: func(r *ThreadRecord) { r.WorkingPath = "relative" }},
		{name: "zero creation", mutate: func(r *ThreadRecord) { r.CreatedAt = time.Time{} }},
		{name: "zero revision", mutate: func(r *ThreadRecord) { r.Revision = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := cloneThreadRecord(base)
			tt.mutate(&record)
			if err := ValidateThreadRecord(record); err == nil {
				t.Fatalf("accepted invalid record: %+v", record)
			}
		})
	}
}

func TestThreadStoreValidateThreadRecordAcceptsLegacyTag(t *testing.T) {
	record := validThreadRecord(t)
	record.Address.Tag = "pair-work_2"
	if err := ValidateThreadRecord(record); err != nil {
		t.Fatalf("legacy tag rejected: %v", err)
	}
	if strings.Contains(string(record.Address.Tag), "/") {
		t.Fatal("test setup contains traversal")
	}
}

func TestValidateThreadRecordAcceptsRecoverableStartBoundaries(t *testing.T) {
	base := validThreadRecord(t)
	base.Reservation = false
	base.Incarnations = []ThreadIncarnation{{
		State:     IncarnationCreating,
		StartedAt: base.CreatedAt,
		Start: &ThreadStartClaim{
			Nonce:         "start-0123456789abcdef",
			OwnerPID:      41,
			OwnerIdentity: "owner-start-token",
		},
	}}

	for _, tt := range []struct {
		name   string
		mutate func(*ThreadRecord)
	}{
		{name: "before helper fork", mutate: func(*ThreadRecord) {}},
		{name: "blocked helper recorded", mutate: func(r *ThreadRecord) {
			r.Incarnations[0].PID = 42
			r.Incarnations[0].Identity = "helper-start-token"
		}},
		{name: "m1 creating record remains readable", mutate: func(r *ThreadRecord) {
			r.Incarnations[0].Start = nil
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			record := cloneThreadRecord(base)
			tt.mutate(&record)
			if err := ValidateThreadRecord(record); err != nil {
				t.Fatalf("valid start boundary rejected: %v", err)
			}
		})
	}
}

func TestValidateThreadRecordRejectsUnrecoverableStartBoundaries(t *testing.T) {
	base := validThreadRecord(t)
	base.Reservation = false
	base.Incarnations = []ThreadIncarnation{{
		State:     IncarnationCreating,
		StartedAt: base.CreatedAt,
		Start: &ThreadStartClaim{
			Nonce:         "start-0123456789abcdef",
			OwnerPID:      41,
			OwnerIdentity: "owner-start-token",
		},
	}}

	for _, tt := range []struct {
		name   string
		mutate func(*ThreadRecord)
	}{
		{name: "nonce traversal", mutate: func(r *ThreadRecord) { r.Incarnations[0].Start.Nonce = "../start" }},
		{name: "owner pid absent", mutate: func(r *ThreadRecord) { r.Incarnations[0].Start.OwnerPID = 0 }},
		{name: "owner identity absent", mutate: func(r *ThreadRecord) { r.Incarnations[0].Start.OwnerIdentity = "" }},
		{name: "start claim on live incarnation", mutate: func(r *ThreadRecord) { r.Incarnations[0].State = IncarnationLive }},
		{name: "helper pid without identity", mutate: func(r *ThreadRecord) { r.Incarnations[0].PID = 42 }},
		{name: "helper identity without pid", mutate: func(r *ThreadRecord) { r.Incarnations[0].Identity = "helper-start-token" }},
		{name: "two tracked starts", mutate: func(r *ThreadRecord) {
			other := r.Incarnations[0]
			other.Start = &ThreadStartClaim{Nonce: "start-fedcba9876543210", OwnerPID: 41, OwnerIdentity: "owner-start-token"}
			r.Incarnations = append(r.Incarnations, other)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			record := cloneThreadRecord(base)
			tt.mutate(&record)
			if err := ValidateThreadRecord(record); err == nil {
				t.Fatalf("accepted unrecoverable start boundary: %+v", record)
			}
		})
	}
}

func FuzzThreadStoreValidateThreadRecordNeverPanics(f *testing.F) {
	f.Add("0123456789abcdef", "couch-0123456789abcdef", "/repo", 1, uint64(1), "start-0123456789abcdef", 41, "owner", 42, "helper")
	f.Add("../scope", "bad/tag", "relative", 0, uint64(0), "../nonce", -1, "", -1, "")
	f.Fuzz(func(t *testing.T, scope, tag, path string, schema int, revision uint64, nonce string, ownerPID int, ownerIdentity string, helperPID int, helperIdentity string) {
		record := ThreadRecord{
			SchemaVersion: schema,
			Address:       ThreadAddress{RepoScope: scope, Tag: ThreadTag(tag)},
			StartingPath:  path,
			WorkingPath:   path,
			CreatedAt:     time.Unix(1, 0),
			Revision:      revision,
			Incarnations: []ThreadIncarnation{{
				PID:      helperPID,
				Identity: helperIdentity,
				State:    IncarnationCreating,
				Start:    &ThreadStartClaim{Nonce: nonce, OwnerPID: ownerPID, OwnerIdentity: ownerIdentity},
			}},
		}
		_ = ValidateThreadRecord(record)
	})
}
