package couchcore

import (
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

// Every writer of a persisted thread record goes through encodeThreadRecord,
// so none can drop the preserved members.
func TestThreadRecordWritersUseThePreservingEncoder(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, "(toPersistedThreadRecord(") && strings.Contains(line, "json.Marshal") {
				t.Errorf("%s:%d marshals a thread record directly; use encodeThreadRecord", name, i+1)
			}
		}
	}
}
