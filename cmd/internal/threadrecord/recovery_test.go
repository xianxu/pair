package threadrecord

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

func absentSourceRecord(t *testing.T) Record {
	t.Helper()
	record := validRecord()
	record.Incarnations = nil
	record.Revision = 5
	cp, err := checkpoint.New("/other-worktree/checkpoint.md", "---\ntype: continuation\nagent: codex\n---\n## NEXT ACTION\nresume exact task\n")
	if err != nil {
		t.Fatal(err)
	}
	record.Continuation = &checkpoint.Request{Version: checkpoint.Version, ID: checkpoint.RequestID(record.Address.RepoScope, record.Address.Tag, 3, cp.Digest), Checkpoint: cp, Source: checkpoint.Source{Agent: "codex", Session: "pair-source", LaunchOrdinal: 3}, CreatedAt: time.Unix(2, 0).UTC(), Phase: checkpoint.Pending, SourceAbsence: &checkpoint.SourceAbsence{Session: "pair-source", LaunchOrdinal: 3, ObservedAt: time.Unix(3, 0).UTC(), RecordRevision: 4}}
	return record
}

func TestRecoveryAbsenceRecordRoundTripAndAddressBinding(t *testing.T) {
	record := absentSourceRecord(t)
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePersisted(raw, record.Address, testValidators)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Continuation, record.Continuation) {
		t.Fatalf("lost source absence: %+v", decoded.Continuation)
	}
	for name, mutate := range map[string]func(*Record){
		"future record revision": func(r *Record) { r.Continuation.SourceAbsence.RecordRevision = r.Revision + 1 },
		"another address": func(r *Record) {
			r.Continuation.ID = checkpoint.RequestID("foreign", r.Address.Tag, 3, r.Continuation.Checkpoint.Digest)
		},
		"foreign source session": func(r *Record) { r.Continuation.SourceAbsence.Session = "foreign" },
	} {
		t.Run(name, func(t *testing.T) {
			r := absentSourceRecord(t)
			mutate(&r)
			if err := Validate(r, testValidators); err == nil {
				t.Fatal("invalid source absence accepted")
			}
		})
	}
}
