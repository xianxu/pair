package couchmessage

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestDrainRequestClosedShape(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	r := Request{Op: "drain", ID: id, Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Drain: &DrainRequest{Action: "start", TimeoutMillis: 600000}}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Request{
		{Op: "drain", Drain: &DrainRequest{Action: "start", TimeoutMillis: 600000}},
		{Op: "drain", Drain: &DrainRequest{Action: "status", Fast: true}},
		{Op: "drain", Drain: &DrainRequest{Action: "ack", ID: id}},
		{Op: "actors", Drain: &DrainRequest{Action: "status"}},
	} {
		if ValidateRequest(bad) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	r.Body = "payload"
	if ValidateRequest(r) == nil {
		t.Fatal("accepted extraneous body")
	}
	if err := ValidateRequest(Request{Op: "drain", Drain: &DrainRequest{Action: "status"}}); err != nil {
		t.Fatal(err)
	}
}

// FuzzValidateDrainRequest uses the public closed intent contract as an
// independent oracle. Adding a Request field cannot silently give drain another
// operation's authority: every field outside this explicit set must stay zero.
func FuzzValidateDrainRequest(f *testing.F) {
	for _, raw := range []string{
		`{"Op":"drain","ID":"00000000-0000-4000-8000-000000000001","Scope":"scope","Tag":"tag","Session":"session","Nonce":"nonce","Drain":{"action":"start","timeout_millis":600000}}`,
		`{"Op":"drain","Drain":{"action":"status"}}`,
		`{"Op":"drain","Scope":"scope","Tag":"tag","Session":"session","Nonce":"nonce","Drain":{"action":"ack","id":"00000000-0000-4000-8000-000000000001"}}`,
		`{"Op":"drain","Scope":"scope","Tag":"tag","Session":"session","Nonce":"nonce","Drain":{"action":"cancel"}}`,
		`{"Op":"drain","Confirmed":true,"Drain":{"action":"status"}}`,
		`{"Op":"drain","Drain":{"action":"status","checkpoint":true}}`,
		`{"Op":"drain","Drain":{"action":"status"},"operator_authorized":true}`,
		`{"Op":"drain","Drain":{"action":"status","action":"start"}}`,
		`{"Op":"drain","Drain":`,
	} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxFrameBytes {
			return
		}
		var r Request
		if strictjson.Decode(raw, &r) != nil {
			return
		}
		if ValidateDrainRequest(r) != nil {
			return
		}
		if r.Op != "drain" || r.Drain == nil {
			t.Fatal("accepted non-drain intent")
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			t.Fatal("malformed JSON admitted")
		}
		fields := reflect.TypeOf(r)
		names := map[string]bool{}
		for i := 0; i < fields.NumField(); i++ {
			field := fields.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" {
				name = field.Name
			}
			if name != "-" {
				names[strings.ToLower(name)] = true
			}
		}
		for name, value := range object {
			if !names[strings.ToLower(name)] {
				t.Fatalf("unknown top-level field %q admitted", name)
			}
			if strings.EqualFold(name, "Drain") {
				var nested map[string]json.RawMessage
				if json.Unmarshal(value, &nested) != nil {
					t.Fatal("malformed drain payload admitted")
				}
				for key := range nested {
					switch strings.ToLower(key) {
					case "action", "id", "reason", "fast", "timeout_millis":
					default:
						t.Fatalf("unknown drain authority %q admitted", key)
					}
				}
			}
		}

		allowed := map[string]bool{"Op": true, "Scope": true, "Tag": true, "Session": true, "Nonce": true, "ID": true, "Drain": true}
		v := reflect.ValueOf(r)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !allowed[typ.Field(i).Name] && !v.Field(i).IsZero() {
				t.Fatalf("accepted unrelated authority field %s", typ.Field(i).Name)
			}
		}
		d := r.Drain
		if d.Action != "status" && (r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "") {
			t.Fatal("mutation accepted absent caller identity")
		}
		switch d.Action {
		case "start":
			if d.ID != "" || d.TimeoutMillis < 1000 || d.TimeoutMillis > 86400000 {
				t.Fatal("invalid start bounds accepted")
			}
			if !drainFuzzUUID(r.ID) || !utf8.ValidString(d.Reason) || len(d.Reason) > 1024 || strings.IndexFunc(d.Reason, unicode.IsControl) >= 0 {
				t.Fatal("unsafe notice data accepted")
			}
		case "ack":
			if !drainFuzzUUID(d.ID) || r.ID != "" || d.Reason != "" || d.Fast || d.TimeoutMillis != 0 {
				t.Fatal("ack accepted another action's authority")
			}
		case "status", "cancel":
			if r.ID != "" || d.ID != "" || d.Reason != "" || d.Fast || d.TimeoutMillis != 0 {
				t.Fatal("read/cancel accepted start or ack payload")
			}
		default:
			t.Fatal("unknown drain action accepted")
		}
		// An otherwise valid request must never turn a destructive-operation flag
		// into drain authority, even when all its ordinary fields are admissible.
		forged := r
		forged.Confirmed = true
		if ValidateDrainRequest(forged) == nil {
			t.Fatal("accepted unrelated confirmation authority")
		}
	})
}
func drainFuzzUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
