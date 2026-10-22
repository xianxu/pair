package couchcmd

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestDeliveryRecorderCarriesProvenanceThroughTrimReopenAndOwed(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ledger := &sessionsync.Ledger{Dir: t.TempDir(), Now: func() time.Time { return now }}
	recorder := &deliveryRecorder{ledger: ledger, now: func() time.Time { return now }}
	const documentID = "11111111111111111111111111111111"
	owner := couchmessage.Binding{Slot: "ops:0"}
	source := couchmessage.RemoteSource{Email: "operator@example.com", IP: "192.0.2.1"}
	message := couchmessage.Message{ID: "remote", From: owner, To: owner, Body: "submitted: /w/tl.md", Origin: couchmessage.OriginRemote, RemoteEdited: true, RemoteSource: source, DocumentID: documentID}
	recorder.record(message, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	rows, err := ledger.Read(owner.Slot)
	if err != nil || len(rows) != 1 {
		t.Fatalf("first record: %+v %v", rows, err)
	}
	if row := rows[0]; row.Origin != message.Origin || !row.RemoteEdited || row.RemoteSource != source || row.DocumentID != documentID {
		t.Fatalf("recorder erased provenance: %+v", row)
	}
	p := filepath.Join(ledger.Dir, url.QueryEscape(owner.Slot)+".jsonl")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(raw), &fields); err != nil {
		t.Fatal(err)
	}
	fields["future_root"] = json.RawMessage(`{"n":9007199254740993}`)
	fields["RemoteSource"] = json.RawMessage(`{"Email":"operator@example.com","IP":"192.0.2.1","future_source":{"opaque":true}}`)
	seed, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	old := `{"ID":"old","At":"2020-01-01T00:00:00Z","Head":"` + strings.Repeat("x", 200) + `"}` + "\n"
	if err := os.WriteFile(p, append([]byte(strings.Repeat(old, 1200)), append(seed, '\n')...), 0600); err != nil {
		t.Fatal(err)
	}
	// A real delivery invokes Append's threshold-triggered rewrite, not a codec-only helper.
	recorder.record(couchmessage.Message{ID: "after", From: owner, To: owner, Body: "not a submit"}, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	raw, err = os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"future_root":{"n":9007199254740993}`, `"future_source":{"opaque":true}`} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Fatalf("actual recorder/trim lost %s: %s", want, raw)
		}
	}
	rows, err = (&sessionsync.Ledger{Dir: ledger.Dir}).Read(owner.Slot)
	if err != nil {
		t.Fatal(err)
	}
	owed := sessionsync.OwedForRegistration(rows, sessionsync.State{}, "/w/tl.md", documentID, nil, false, func(s string) string { return s })
	if len(owed) != 1 || owed[0].Origin != message.Origin || !owed[0].RemoteEdited || owed[0].RemoteSource != source || owed[0].DocumentID != documentID {
		t.Fatalf("owed erased provenance: %+v", owed)
	}
}

func TestDeliveryRecorderMalformedProvenanceRecordsGap(t *testing.T) {
	now := time.Now()
	ledger := &sessionsync.Ledger{Dir: t.TempDir()}
	recorder := &deliveryRecorder{ledger: ledger, now: func() time.Time { return now }}
	m := couchmessage.Message{ID: "bad", To: couchmessage.Binding{Slot: "ops:0"}, Origin: couchmessage.OriginRemote}
	recorder.record(m, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	if gap, err := ledger.Gap(m.To.Slot); err != nil || gap == nil {
		t.Fatalf("malformed provenance has no recording gap: %+v %v", gap, err)
	}
	if rows, err := ledger.Read(m.To.Slot); err != nil || len(rows) != 0 {
		t.Fatalf("malformed row became local: %+v %v", rows, err)
	}
}

func TestDeliveryRecorderLocalSubmitKeepsRemoteEditedWithoutRemoteSource(t *testing.T) {
	ledger := &sessionsync.Ledger{Dir: t.TempDir()}
	recorder := &deliveryRecorder{ledger: ledger, now: time.Now}
	m := couchmessage.Message{ID: "local", To: couchmessage.Binding{Slot: "ops:0"}, Body: "submitted: /w/tl.md", RemoteEdited: true, DocumentID: "11111111111111111111111111111111"}
	recorder.record(m, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	rows, err := ledger.Read(m.To.Slot)
	if err != nil || len(rows) != 1 || !rows[0].RemoteEdited || rows[0].Origin != "" || rows[0].RemoteSource != (couchmessage.RemoteSource{}) {
		t.Fatalf("local marked row: %+v %v", rows, err)
	}
}
