package sessionsync

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLedgerRetainsFutureProvenanceThroughAppendTrimReopen(t *testing.T) {
	l := &Ledger{Dir: t.TempDir(), Now: func() time.Time { return t0 }}
	p := filepath.Join(l.Dir, "ops%3A0.jsonl")
	seed := `{"ID":"remote","From":"ops:0","At":"2026-10-10T12:00:00Z","Head":"submitted: /w/tl.md","Origin":"remote","RemoteEdited":true,"RemoteSource":{"Email":"operator@example.com","IP":"192.0.2.1","future_source":{ "n":9007199254740993, "huge":1e10000 }},"future_root":{ "opaque":[1, 2, 3] }}`
	// Old rows force the actual append path to trim, while the remote row stays newest.
	old := `{"ID":"old","At":"2020-01-01T00:00:00Z","Head":"` + strings.Repeat("x", 200) + `"}` + "\n"
	if err := os.WriteFile(p, append([]byte(strings.Repeat(old, 1200)), []byte(seed+"\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.Append("ops:0", Entry{ID: "next", At: t0}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"future_root":{ "opaque":[1, 2, 3] }`, `"future_source":{ "n":9007199254740993, "huge":1e10000 }`} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Fatalf("trim erased retained field %s: %s", want, raw)
		}
	}
	rows, err := (&Ledger{Dir: l.Dir}).Read("ops:0")
	if err != nil {
		t.Fatal(err)
	}
	owed := OwedForDocument(rows, State{}, "/w/tl.md", nil, false, identity)
	if len(owed) != 1 {
		t.Fatalf("owed=%+v", owed)
	}
	projection, err := json.Marshal(owed[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"Origin":"remote"`, `"RemoteEdited":true`, `"Email":"operator@example.com"`, `"IP":"192.0.2.1"`} {
		if !bytes.Contains(projection, []byte(want)) {
			t.Fatalf("owed erased provenance %s: %s", want, projection)
		}
	}
}

func TestLedgerRefusesMalformedRecognizedProvenance(t *testing.T) {
	for name, fields := range map[string]string{
		"origin_type":        `"Origin":false`,
		"edited_type":        `"RemoteEdited":"false"`,
		"edited_null":        `"RemoteEdited":null`,
		"source_missing":     `"Origin":"remote","RemoteEdited":true`,
		"source_wrong_type":  `"Origin":"remote","RemoteSource":"operator"`,
		"source_duplicate":   `"Origin":"remote","RemoteSource":{"Email":"operator@example.com","IP":"192.0.2.1","IP":"192.0.2.2"}`,
		"source_null":        `"RemoteSource":null`,
		"source_on_local":    `"RemoteSource":{"Email":"operator@example.com","IP":"192.0.2.1"}`,
		"alias":              `"remoteedited":true`,
		"document_id_type":   `"Origin":"","RemoteEdited":false,"DocumentID":true`,
		"source_email_alias": `"Origin":"remote","RemoteEdited":true,"RemoteSource":{"email":"operator@example.com","IP":"192.0.2.1"}`,
		"root_duplicate":     `"Origin":"remote","Origin":"","RemoteEdited":false`,
		"source_incomplete":  `"Origin":"remote","RemoteEdited":true,"RemoteSource":{"Email":"operator@example.com"}`,
		"source_ip_invalid":  `"Origin":"remote","RemoteEdited":true,"RemoteSource":{"Email":"operator@example.com","IP":"not an ip"}`,
	} {
		t.Run(name, func(t *testing.T) {
			l := &Ledger{Dir: t.TempDir()}
			raw := `{"ID":"bad","At":"2026-10-10T12:00:00Z",` + fields + "}\n"
			if err := os.WriteFile(filepath.Join(l.Dir, "ops%3A0.jsonl"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if rows, err := l.Read("ops:0"); err == nil {
				t.Fatalf("recognized malformed provenance vanished or became local: %+v", rows)
			}
		})
	}
}

func TestLedgerDocumentIdentityCannotMoveToReusedPath(t *testing.T) {
	const original = "11111111111111111111111111111111"
	const replacement = "22222222222222222222222222222222"
	l := &Ledger{Dir: t.TempDir()}
	var entry Entry
	if err := json.Unmarshal([]byte(`{"ID":"remote","DocumentID":"`+original+`","At":"2026-10-10T12:00:00Z","Head":"submitted: /w/tl.md","Origin":"remote","RemoteEdited":true,"RemoteSource":{"Email":"operator@example.com","IP":"192.0.2.1"}}`), &entry); err != nil {
		t.Fatal(err)
	}
	if err := l.Append("ops:0", entry); err != nil {
		t.Fatal(err)
	}
	rows, err := (&Ledger{Dir: l.Dir}).Read("ops:0")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{replacement, ""} {
		if got := OwedForRegistration(rows, State{}, "/w/tl.md", id, nil, false, identity); len(got) != 0 {
			t.Fatalf("replacement %q inherited old submit: %+v", id, got)
		}
		if SubmittedSinceForRegistration(rows, "/w/tl.md", id, nil, t0.Add(-time.Second), identity) {
			t.Fatalf("replacement %q inherited submitted-since", id)
		}
	}
	if got := OwedForRegistration(rows, State{}, "/w/tl.md", original, nil, false, identity); len(got) != 1 || got[0].DocumentID != original {
		t.Fatalf("original owed=%+v", got)
	}
	if !SubmittedSinceForRegistration(rows, "/w/tl.md", original, nil, t0.Add(-time.Second), identity) {
		t.Fatal("original submit missing")
	}
	// Historical path attribution remains available for legacy rows.
	rows[0].DocumentID = ""
	if got := OwedForRegistration(rows, State{}, "/w/tl.md", replacement, nil, false, identity); len(got) != 1 {
		t.Fatalf("legacy attribution changed: %+v", got)
	}
}

func TestLedgerMalformedProvenanceCannotBeTrimmedAway(t *testing.T) {
	l := &Ledger{Dir: t.TempDir(), Now: func() time.Time { return t0 }}
	p := filepath.Join(l.Dir, "ops%3A0.jsonl")
	bad := `{"ID":"bad","Origin":"remote","RemoteEdited":true,"RemoteSource":{"Email":"operator@example.com","IP":null}}` + "\n"
	old := `{"ID":"old","At":"2020-01-01T00:00:00Z","Head":"` + strings.Repeat("x", 200) + `"}` + "\n"
	raw := []byte(bad + strings.Repeat(old, 1200))
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.Append("ops:0", Entry{ID: "next", At: t0}); err == nil {
		t.Fatal("trim silently removed malformed provenance")
	}
	kept, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(kept, []byte(bad)) {
		t.Fatal("refused trim changed damaged authority")
	}
}

func TestLedgerProvenanceCodecBoundsAndExplicitFalse(t *testing.T) {
	const prefix = `{"ID":"local","Origin":"","RemoteEdited":false,"future":`
	for _, depth := range []int{32, 33} {
		raw := []byte(prefix + strings.Repeat("[", depth-1) + "0" + strings.Repeat("]", depth-1) + "}")
		var entry Entry
		err := entry.UnmarshalJSON(raw)
		if (err != nil) != (depth > 32) {
			t.Fatalf("depth %d: %v", depth, err)
		}
		if err == nil {
			encoded, err := encodeLedgerEntry(entry)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(encoded, []byte(`"Origin":""`)) || !bytes.Contains(encoded, []byte(`"RemoteEdited":false`)) {
				t.Fatalf("explicit false metadata erased: %s", encoded)
			}
		}
	}
	for _, size := range []int{ledgerEntryMaxBytes, ledgerEntryMaxBytes + 1} {
		raw := []byte(prefix + `"` + strings.Repeat("x", size-len(prefix)-3) + `"}`)
		var entry Entry
		err := entry.UnmarshalJSON(raw)
		if (err != nil) != (size > ledgerEntryMaxBytes) {
			t.Fatalf("size %d: %v", size, err)
		}
	}
}

func TestLedgerMalformedFutureValueCannotHideLaterProvenance(t *testing.T) {
	for _, key := range []string{`"Origin"`, `"\u004frigin"`, `"RemoteEdited"`, `"DocumentID"`} {
		path := filepath.Join(t.TempDir(), "rows")
		raw := `{"A_future":broken,` + key + `:"remote"}` + "\n"
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readRows(path); err == nil {
			t.Fatalf("malformed value hid provenance key %s", key)
		}
	}
}
