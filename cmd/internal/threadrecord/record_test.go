package threadrecord

import (
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
		SchemaVersion:   1,
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
		"schema version":        func(r *Record) { r.SchemaVersion = 2 },
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
