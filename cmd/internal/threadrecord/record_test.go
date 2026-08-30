package threadrecord

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

var testValidators = Validators{
	RepoScope: func(value string) error {
		if value != "816fc349d3faebf8" {
			return errText("bad scope")
		}
		return nil
	},
	Tag: func(value string) error {
		if !strings.HasPrefix(value, "couch-") {
			return errText("bad tag")
		}
		return nil
	},
}

type errText string

func (e errText) Error() string { return string(e) }

func validRecord() Record {
	return Record{
		SchemaVersion:   SchemaVersion,
		Address:         Address{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"},
		StartingPath:    "/repo",
		WorkingPath:     "/repo",
		CreatedAt:       time.Unix(1, 0).UTC(),
		Revision:        1,
		ClaimGeneration: 1,
		Incarnations: []Incarnation{{
			PID: 42, Identity: "helper", State: "creating",
			Start: &StartClaim{
				Nonce: "nonce", OwnerPID: 7, OwnerIdentity: "owner",
				LaunchProfile: &LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}},
			},
		}},
	}
}

func TestValidateEnumeratesThreadRecordStructuralInvariants(t *testing.T) {
	tests := map[string]func(*Record){
		"schema version":        func(r *Record) { r.SchemaVersion = 3 },
		"repository scope":      func(r *Record) { r.Address.RepoScope = "other" },
		"tag":                   func(r *Record) { r.Address.Tag = "other" },
		"absolute start path":   func(r *Record) { r.StartingPath = "repo" },
		"absolute working path": func(r *Record) { r.WorkingPath = "repo" },
		"creation time":         func(r *Record) { r.CreatedAt = time.Time{} },
		"positive revision":     func(r *Record) { r.Revision = 0 },
		"incarnation state":     func(r *Record) { r.Incarnations[0].State = "done" },
		"nonnegative pid":       func(r *Record) { r.Incarnations[0].PID = -1 },
		"start only creating":   func(r *Record) { r.Incarnations[0].State = "live" },
		"start nonce":           func(r *Record) { r.Incarnations[0].Start.Nonce = "/" },
		"start owner pid":       func(r *Record) { r.Incarnations[0].Start.OwnerPID = 0 },
		"start owner identity":  func(r *Record) { r.Incarnations[0].Start.OwnerIdentity = "" },
		"pending profile agent": func(r *Record) { r.Incarnations[0].Start.LaunchProfile.Agent = "" },
		"pending profile argv":  func(r *Record) { r.Incarnations[0].Start.LaunchProfile.Argv = nil },
		"helper pid pair":       func(r *Record) { r.Incarnations[0].PID = 0 },
		"helper identity pair":  func(r *Record) { r.Incarnations[0].Identity = "" },
		"one tracked start": func(r *Record) {
			second := r.Incarnations[0]
			claim := *second.Start
			claim.Nonce = "second"
			second.Start = &claim
			r.Incarnations = append(r.Incarnations, second)
		},
	}
	if err := Validate(validRecord(), testValidators); err != nil {
		t.Fatalf("valid record: %v", err)
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			record := validRecord()
			mutate(&record)
			if err := Validate(record, testValidators); err == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
}

func TestValidateCommittedLaunchProfileRequiresRegistrationAndCompleteProfile(t *testing.T) {
	tests := map[string]func(*Record){
		"creating incarnation": func(r *Record) {
			r.Incarnations[0].LaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{}}
		},
		"missing agent": func(r *Record) {
			r.Incarnations[0].State = "live"
			r.Incarnations[0].Start = nil
			r.Incarnations[0].LaunchProfile = &LaunchProfile{Argv: []string{}}
		},
		"missing argv": func(r *Record) {
			r.Incarnations[0].State = "live"
			r.Incarnations[0].Start = nil
			r.Incarnations[0].LaunchProfile = &LaunchProfile{Agent: "codex"}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			record := validRecord()
			mutate(&record)
			if err := Validate(record, testValidators); err == nil {
				t.Fatal("invalid committed launch profile accepted")
			}
		})
	}
}

func TestValidatePersistedAddsGenerationAndAddressInvariants(t *testing.T) {
	record := validRecord()
	expected := record.Address
	if err := ValidatePersisted(record, expected, testValidators); err != nil {
		t.Fatal(err)
	}
	record.ClaimGeneration = 0
	if err := ValidatePersisted(record, expected, testValidators); err == nil {
		t.Fatal("zero claim generation accepted")
	}
	record = validRecord()
	expected.Tag = "couch-1112131415161718"
	if err := ValidatePersisted(record, expected, testValidators); err == nil {
		t.Fatal("path/address mismatch accepted")
	}
}

func TestDecodeV1MigratesToV2(t *testing.T) {
	raw := []byte(`{
  "schema_version": 1,
  "address": {"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"},
  "starting_path": "/repo",
  "working_path": "/repo",
  "created_at": "1970-01-01T00:00:01Z",
  "revision": 1,
  "claim_generation": 1,
  "incarnations": [{"pid":42,"identity":"helper","state":"live","launch_profile":{"agent":"codex","argv":["--sandbox","workspace-write"]}}]
}`)
	record, err := DecodePersisted(raw, Address{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}, testValidators)
	if err != nil {
		t.Fatalf("DecodePersisted(v1): %v", err)
	}
	if record.SchemaVersion != SchemaVersion || SchemaVersion != 2 {
		t.Fatalf("schema = %d, current = %d", record.SchemaVersion, SchemaVersion)
	}
	if record.LatestLaunchProfile != nil || record.Park != nil || record.VerifiedPark != nil || record.ParkHistory != nil || !record.LastActiveAt.IsZero() {
		t.Fatalf("migration invented lifecycle state: %+v", record)
	}
}

func validLifecycleRecord() Record {
	record := validRecord()
	record.SchemaVersion = 2
	record.Incarnations[0] = Incarnation{
		PID: 42, Identity: "helper", State: "live",
		LaunchProfile: &LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}},
	}
	record.LatestLaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	record.Park = &ParkTransaction{
		Identity: ParkIdentity{
			Nonce: "park-0102030405060708", Address: record.Address,
			PID: 42, ProcessIdentity: "helper",
		},
		BaseRevision: 1, RecordRevision: 2, Phase: "requested",
		Attempts: []ParkAttempt{{Number: 1}},
	}
	return record
}

func TestV2RoundTrip(t *testing.T) {
	want := validLifecycleRecord()
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePersisted(raw, want.Address, testValidators)
	if err != nil {
		t.Fatalf("DecodePersisted(v2): %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestUnsupportedThreadRecordVersionRefuses(t *testing.T) {
	for _, raw := range []string{
		`{"schema_version":0}`,
		`{"schema_version":3}`,
		`{"schema_version":2,"schema_version":1}`,
		`{"schema_version":2} {"schema_version":2}`,
	} {
		if _, err := DecodePersisted([]byte(raw), Address{}, testValidators); err == nil {
			t.Fatalf("accepted unsupported or ambiguous payload %s", raw)
		}
	}
}

func TestValidateLifecycleStructuralInvariants(t *testing.T) {
	tests := map[string]func(*Record){
		"incomplete identity": func(r *Record) { r.Park.Identity.ProcessIdentity = "" },
		"identity address":    func(r *Record) { r.Park.Identity.Address.Tag = "couch-other" },
		"identity incarnation": func(r *Record) {
			r.Park.Identity.PID++
		},
		"zero base revision":       func(r *Record) { r.Park.BaseRevision = 0 },
		"stale record revision":    func(r *Record) { r.Park.RecordRevision = r.Park.BaseRevision },
		"invalid phase":            func(r *Record) { r.Park.Phase = "parked" },
		"attempt starts above one": func(r *Record) { r.Park.Attempts[0].Number = 2 },
		"attempt not increasing": func(r *Record) {
			r.Park.Attempts = append(r.Park.Attempts, ParkAttempt{Number: 1})
		},
		"unknown failure": func(r *Record) {
			r.Park.Attempts[0].Failure = &ParkFailure{Code: "other", Diagnostic: "bad"}
		},
		"failure without diagnostic": func(r *Record) {
			r.Park.Attempts[0].Failure = &ParkFailure{Code: "timeout"}
		},
		"active closed":     func(r *Record) { r.Park.Closed = true },
		"active tombstoned": func(r *Record) { r.Park.Tombstoned = true },
		"active successful": func(r *Record) { r.Park.SuccessfulAttempt = 1 },
		"duplicate nonce": func(r *Record) {
			history := *r.Park
			history.Closed = true
			history.Tombstoned = true
			r.ParkHistory = []ParkTransaction{history}
		},
		"history not closed": func(r *Record) { r.ParkHistory = []ParkTransaction{*r.Park} },
		"verified with incarnation": func(r *Record) {
			park := *r.Park
			park.Closed = true
			park.SuccessfulAttempt = 1
			park.Attempts[0].Closed = true
			r.Park = nil
			r.ParkHistory = []ParkTransaction{park}
			r.VerifiedPark = &VerifiedPark{Identity: park.Identity, Attempt: 1, ParkedAt: time.Unix(2, 0).UTC()}
			r.LastActiveAt = time.Unix(2, 0).UTC()
		},
	}
	if err := Validate(validLifecycleRecord(), testValidators); err != nil {
		t.Fatalf("valid lifecycle: %v", err)
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			record := validLifecycleRecord()
			mutate(&record)
			if err := Validate(record, testValidators); err == nil {
				t.Fatal("invalid lifecycle record accepted")
			}
		})
	}
}

func TestValidateVerifiedParkRequiresClosedSuccessHistoryAndNoIncarnation(t *testing.T) {
	record := validLifecycleRecord()
	closed := *record.Park
	closed.Closed = true
	closed.Attempts = append([]ParkAttempt(nil), record.Park.Attempts...)
	closed.Attempts[0].Closed = true
	closed.SuccessfulAttempt = 1
	record.Park = nil
	record.ParkHistory = []ParkTransaction{closed}
	record.Incarnations = nil
	record.VerifiedPark = &VerifiedPark{Identity: closed.Identity, Attempt: 1, ParkedAt: time.Unix(2, 0).UTC()}
	record.LastActiveAt = time.Unix(2, 0).UTC()
	if err := Validate(record, testValidators); err != nil {
		t.Fatalf("valid verified park: %v", err)
	}

	for name, mutate := range map[string]func(*Record){
		"missing history": func(r *Record) { r.ParkHistory = nil },
		"wrong attempt":   func(r *Record) { r.VerifiedPark.Attempt = 2 },
		"zero parked time": func(r *Record) {
			r.VerifiedPark.ParkedAt = time.Time{}
		},
		"missing last active": func(r *Record) { r.LastActiveAt = time.Time{} },
		"tombstoned success":  func(r *Record) { r.ParkHistory[0].Tombstoned = true },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := record
			candidate.ParkHistory = append([]ParkTransaction(nil), record.ParkHistory...)
			verified := *record.VerifiedPark
			candidate.VerifiedPark = &verified
			mutate(&candidate)
			if err := Validate(candidate, testValidators); err == nil {
				t.Fatal("invalid verified park accepted")
			}
		})
	}
}

func TestValidateLifecycleAllowsReplacementIncarnationUnknown(t *testing.T) {
	record := validLifecycleRecord()
	record.Incarnations[0].PID = 99
	record.Incarnations[0].Identity = "replacement-helper"
	record.Park.Phase = "unknown"
	record.Park.Attempts[0].Failure = &ParkFailure{
		Code: "replacement_incarnation", Diagnostic: "original identity was replaced",
	}
	if err := Validate(record, testValidators); err != nil {
		t.Fatalf("replacement-incarnation recovery state rejected: %v", err)
	}
}
