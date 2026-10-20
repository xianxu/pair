package couchcmd

import (
	"strings"
	"testing"
)

const drainQueueFixture = `{"schema_version":1,"state":"paused","complete":true,"pause_owners":[{"owner":"couch-drain:test","reason":"Couch drain test: reason","cancel_running":false}],"active":[{"id":"op1","workspace":"pair:5","kind":"suite","state":"running"}],"detail":""}`

func TestDrainQueueDecodeStrictKnownSchema(t *testing.T) {
	got, err := decodeQueueSnapshot([]byte(drainQueueFixture))
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "paused" || !got.Complete || len(got.Active) != 1 || got.Active[0].ID != "op1" {
		t.Fatalf("bad snapshot %+v", got)
	}
	additive := strings.Replace(drainQueueFixture, `"detail":""`, `"detail":"","new":{"x":[true,1]}`, 1)
	if _, err := decodeQueueSnapshot([]byte(additive)); err != nil {
		t.Fatalf("additive: %v", err)
	}
	for name, raw := range map[string]string{
		"duplicate":         strings.Replace(drainQueueFixture, `"complete":true`, `"complete":true,"complete":false`, 1),
		"nested duplicate":  strings.Replace(drainQueueFixture, `"id":"op1"`, `"id":"op1","id":"op2"`, 1),
		"missing":           strings.Replace(drainQueueFixture, `"complete":true,`, "", 1),
		"null":              strings.Replace(drainQueueFixture, `"active":[{"id":"op1","workspace":"pair:5","kind":"suite","state":"running"}]`, `"active":null`, 1),
		"wrong version":     strings.Replace(drainQueueFixture, `"schema_version":1`, `"schema_version":2`, 1),
		"contradiction":     strings.Replace(drainQueueFixture, `"state":"paused"`, `"state":"running"`, 1),
		"unknown kind":      strings.Replace(drainQueueFixture, `"kind":"suite"`, `"kind":"test"`, 1),
		"wrong bool":        strings.Replace(drainQueueFixture, `"cancel_running":false`, `"cancel_running":"false"`, 1),
		"missing owner key": strings.Replace(drainQueueFixture, `"reason":"Couch drain test: reason",`, "", 1),
		"trailing":          drainQueueFixture + `{}`,
		"oversized":         drainQueueFixture + strings.Repeat(" ", 1<<20),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeQueueSnapshot([]byte(raw)); err == nil {
				t.Fatal("accepted invalid snapshot")
			}
		})
	}
	unknown := strings.Replace(drainQueueFixture, `"state":"paused"`, `"state":"unknown"`, 1)
	unknown = strings.Replace(unknown, `"detail":""`, `"detail":"registry unreadable"`, 1)
	got, err = decodeQueueSnapshot([]byte(unknown))
	if err != nil {
		t.Fatal(err)
	}
	if got.Quiescent() {
		t.Fatal("unknown proves absence")
	}
}

func TestDrainQueueQuiescenceRequiresCompleteKnownEvidence(t *testing.T) {
	s := QueueSnapshot{SchemaVersion: 1, State: "paused", Complete: true, PauseOwners: []QueuePauseOwner{{Owner: "a"}}}
	if !s.Quiescent() {
		t.Fatal("complete paused empty queue not quiet")
	}
	for _, alter := range []func(*QueueSnapshot){func(q *QueueSnapshot) { q.Complete = false }, func(q *QueueSnapshot) { q.State = "unknown" }, func(q *QueueSnapshot) {
		q.Active = []QueueActivity{{ID: "x", Workspace: "", Kind: "suite", State: "unknown"}}
	}} {
		q := s
		alter(&q)
		if q.Quiescent() {
			t.Fatalf("unsafe quiet: %+v", q)
		}
	}
}

func FuzzDrainQueueDecode(f *testing.F) {
	f.Add([]byte(drainQueueFixture))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		q, err := decodeQueueSnapshot(b)
		if err == nil && q.Quiescent() && (!q.Complete || q.State != "paused" || len(q.Active) != 0) {
			t.Fatal("unsafe quiescence")
		}
	})
}

func TestDrainQueueRejectsBoundsAndRequiredNestedTypes(t *testing.T) {
	for name, raw := range map[string]string{
		"owner duplicate":           strings.Replace(drainQueueFixture, `"pause_owners":[`, `"pause_owners":[{"owner":"couch-drain:test","reason":"x","cancel_running":false},`, 1),
		"null owner":                strings.Replace(drainQueueFixture, `"owner":"couch-drain:test"`, `"owner":null`, 1),
		"null optional head":        strings.Replace(drainQueueFixture, `"id":"op1"`, `"head":null,"id":"op1"`, 1),
		"incomplete without detail": strings.Replace(drainQueueFixture, `"complete":true`, `"complete":false`, 1),
		"reason exceeds bound":      strings.Replace(drainQueueFixture, `Couch drain test: reason`, strings.Repeat("a", 4097), 1),
		"nesting bound":             strings.Replace(drainQueueFixture, `"detail":""`, `"detail":"","unknown":`+strings.Repeat("[", 65)+`0`+strings.Repeat("]", 65), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeQueueSnapshot([]byte(raw)); err == nil {
				t.Fatal("accepted unsafe schema")
			}
		})
	}
	owners := strings.Repeat(`{"owner":"x","reason":"","cancel_running":false},`, 128) + `{"owner":"last","reason":"","cancel_running":false}`
	raw := `{"schema_version":1,"state":"paused","complete":true,"pause_owners":[` + owners + `],"active":[],"detail":""}`
	if _, err := decodeQueueSnapshot([]byte(raw)); err == nil {
		t.Fatal("accepted owner overflow")
	}
}

func TestDrainQueueMalformedDiagnosticBounded(t *testing.T) {
	raw := strings.Replace(drainQueueFixture, `"state":"paused"`, `"state":"`+strings.Repeat("x", 5000)+`"`, 1)
	q, err := decodeQueueSnapshot([]byte(raw))
	if err == nil || len(q.Detail) > 4096 {
		t.Fatalf("malformed diagnostic bound: %d %v", len(q.Detail), err == nil)
	}
}
