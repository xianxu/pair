package remotesession

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestAuditCodecRejectsInvalidMetadata(t *testing.T) {
	r := auditFixture(time.Now().UTC())
	r.kind = "intent"
	raw, err := encodeAuditRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		key   string
		value string
	}{{"version", "2"}, {"kind", `"body"`}, {"metadata", "null"}} {
		var root map[string]json.RawMessage
		json.Unmarshal(raw, &root)
		root[tc.key] = json.RawMessage(tc.value)
		if _, err := decodeAuditRecord(marshalRemoteFields(root)); err == nil {
			t.Fatalf("accepted %s", tc.key)
		}
	}
	for _, modify := range []func(*AuditRecord){func(r *AuditRecord) { r.Action = "shell" }, func(r *AuditRecord) { r.Principal = "secret" }, func(r *AuditRecord) { r.Before = "/private/doc" }, func(r *AuditRecord) { r.Outcome = "success" }, func(r *AuditRecord) { r.ReceiptID = "document body" }} {
		bad := r
		modify(&bad)
		if _, err := encodeAuditRecord(bad); err == nil {
			t.Fatal("accepted invalid metadata")
		}
	}
	duplicate := bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1)
	if _, err := decodeAuditRecord(duplicate); err == nil {
		t.Fatal("duplicate accepted")
	}
}
func FuzzAuditCodec(f *testing.F) {
	r := auditFixture(time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC))
	r.kind = "intent"
	b, _ := encodeAuditRecord(r)
	f.Add(b)
	f.Fuzz(func(t *testing.T, b []byte) {
		r, err := decodeAuditRecord(b)
		if err != nil {
			return
		}
		out, err := encodeAuditRecord(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeAuditRecord(out); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAuditRejectsZeroEpochSequence(t *testing.T) {
	for _, field := range []string{"epoch", "sequence"} {
		t.Run(field, func(t *testing.T) {
			r := auditFixture(time.Now().UTC())
			r.kind = "intent"
			if field == "epoch" {
				r.Epoch = 0
			} else {
				r.Sequence = 0
			}
			if _, err := encodeAuditRecord(r); err == nil {
				t.Fatal("accepted zero " + field)
			}
		})
	}
}
