package couchcore

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// pair#695 §2a: a build that does not know a record member must carry it
// through its own rewrite, byte for byte, or an older couch CLI running a
// direct-store operation would erase a newer build's field.
func TestThreadRecordPreservesUnknownFieldsThroughRewrite(t *testing.T) {
	t.Parallel()
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	path := store.recordPath(created.Address)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	unknown := map[string]string{
		"future_binding": `{ "agent" : "codex", "id": 123456789012345678901234567890 }`,
		"future_note":    `"café \"q\""`,
	}
	seeded := strings.TrimRight(string(raw), "\n")
	seeded = strings.TrimSuffix(seeded, "}")
	for k, v := range unknown {
		seeded += `, "` + k + `": ` + v
	}
	seeded += "}\n"
	if err := os.WriteFile(path, []byte(seeded), 0o600); err != nil {
		t.Fatal(err)
	}

	summary := "rewritten by a build that does not know future_binding"
	updated, err := store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{PublishedSummary: &summary})
	if err != nil {
		t.Fatalf("rewrite of a record with unknown members: %v", err)
	}
	if updated.PublishedSummary != summary {
		t.Fatalf("PublishedSummary = %q", updated.PublishedSummary)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := strictjson.SplitObject(after, strictjson.KnownFields(reflect.TypeOf(threadrecord.Record{})), nil)
	if err != nil {
		t.Fatalf("rewritten record: %v\n%s", err, after)
	}
	for k, v := range unknown {
		if string(got[k]) != v {
			t.Errorf("unknown member %q after rewrite = %s, want %s", k, got[k], v)
		}
	}
	reread, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatalf("GetThread after rewrite: %v", err)
	}
	if reread.PublishedSummary != summary {
		t.Fatalf("reread PublishedSummary = %q", reread.PublishedSummary)
	}

	// Tolerance covers unknown members only: a known member of the wrong type
	// and a duplicate key still refuse.
	for name, corrupt := range map[string]string{
		"wrong type":    strings.Replace(string(after), `"revision": `, `"revision": "x", "ignored": `, 1),
		"duplicate key": strings.Replace(string(after), `"future_note": `, `"future_note": 1, "future_note": `, 1),
	} {
		if err := os.WriteFile(path, []byte(corrupt), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.GetThread(created.Address); err == nil {
			t.Errorf("%s: GetThread accepted %s", name, corrupt)
		}
	}
}

// Every JSON encoding of a record is the persisted one (MarshalJSON), so a
// writer that marshals the record directly keeps the preserved members too.
func TestThreadRecordJSONKeepsUnknownMembers(t *testing.T) {
	t.Parallel()
	record := validThreadRecord(t)
	record.Unknown = map[string]json.RawMessage{"future_binding": json.RawMessage(`{ "id" : 1 }`)}
	for name, marshal := range map[string]func(any) ([]byte, error){
		"Marshal":       json.Marshal,
		"MarshalIndent": func(v any) ([]byte, error) { return json.MarshalIndent(v, "", "  ") },
	} {
		raw, err := marshal(record)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		_, unknown, err := strictjson.SplitObject(raw, strictjson.KnownFields(reflect.TypeOf(threadrecord.Record{})), nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var got, want any
		if err := json.Unmarshal(unknown["future_binding"], &got); err != nil {
			t.Fatalf("%s lost the unknown member: %s", name, raw)
		}
		_ = json.Unmarshal([]byte(`{"id":1}`), &want)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: unknown member = %s", name, unknown["future_binding"])
		}
	}
}

// Slot recovery rewrites the survivor it decoded; the members it does not own
// survive that rewrite (pair#695 M1 review BR-3).
func TestSlotRecoveryRewritePreservesUnknownMembers(t *testing.T) {
	t.Parallel()
	s := testLocalThreadStore(t)
	created, err := s.CreateThread(recordAtCheckout(t, s.slot.WorktreeRoot, s.slot.WorktreeRoot, "couch-0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	path := s.recordPath(created.Address)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	seeded := strings.TrimSuffix(strings.TrimRight(string(raw), "\n"), "}") + `, "future_binding": { "id" : 7 }}` + "\n"
	if err := os.WriteFile(path, []byte(seeded), 0o600); err != nil {
		t.Fatal(err)
	}
	observed, err := s.observeSlotCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if observed.Record == nil {
		t.Fatal("no current record observed")
	}
	next := cloneThreadRecord(*observed.Record)
	next.Name = "recovered"
	next.Revision++
	if err := s.replaceSlotCurrent(observed, next); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, unknown, err := strictjson.SplitObject(after, strictjson.KnownFields(reflect.TypeOf(threadrecord.Record{})), nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(unknown["future_binding"]) != `{ "id" : 7 }` {
		t.Fatalf("recovery rewrite dropped the unknown member:\n%s", after)
	}
}
