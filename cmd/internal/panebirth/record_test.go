package panebirth

import (
	"encoding/json"
	"strings"
	"testing"
)

func testLaunchIdentity() LaunchIdentity {
	return LaunchIdentity{Namespace: "/tmp/zellij", Scope: "repo_1", Tag: "tag-1", Agent: "codex", Session: "pair-work", Nonce: "nonce-1"}
}

func TestRecordLegacyAndExactIdentity(t *testing.T) {
	legacy, err := DecodeRecord([]byte(`{"pane_id":"0","cwd":"/work"}`))
	if err != nil || legacy.PaneID != "0" || legacy.Cwd != "/work" {
		t.Fatalf("legacy = %+v, %v", legacy, err)
	}
	identity := testLaunchIdentity()
	if legacy.Identity.Equal(identity) || (LaunchIdentity{}).Equal(LaunchIdentity{}) {
		t.Fatal("absent identity authorized readiness")
	}
	encoded, err := EncodeRecord(Record{PaneID: "42", Cwd: "/work", Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRecord(encoded)
	if err != nil || !decoded.Identity.Equal(identity) {
		t.Fatalf("round trip = %+v, %v", decoded, err)
	}
	for _, change := range []func(*LaunchIdentity){
		func(i *LaunchIdentity) { i.Namespace += "2" }, func(i *LaunchIdentity) { i.Scope += "2" },
		func(i *LaunchIdentity) { i.Tag += "2" }, func(i *LaunchIdentity) { i.Agent += "2" },
		func(i *LaunchIdentity) { i.Session += "2" }, func(i *LaunchIdentity) { i.Nonce += "2" },
	} {
		other := identity
		change(&other)
		if identity.Equal(other) {
			t.Fatalf("mismatch accepted: %+v", other)
		}
	}
}

func TestRecordRejectsMalformedKnownFields(t *testing.T) {
	validIdentity := `{"namespace":"/tmp/zellij","scope":"repo_1","tag":"tag-1","agent":"codex","session":"pair-work","nonce":"nonce-1"}`
	cases := []string{
		`null`, `[]`, `{}`, `{"pane_id":0}`, `{"pane_id":null}`, `{"pane_id":""}`, `{"pane_id":"-1"}`, `{"pane_id":"+1"}`, `{"pane_id":"1.2"}`, `{"pane_id":"1e3"}`,
		`{"pane_id":"1","cwd":null}`, `{"pane_id":"1","cwd":7}`, `{"pane_id":"1","pane_id":"2"}`,
		`{"pane_id":"1","identity":null}`, `{"pane_id":"1","identity":[]}`, `{"pane_id":"1","identity":{}}`,
		`{"pane_id":"1"} {}`, `{"pane_id":"1","x":{"a":1,"a":2}}`,
	}
	for _, field := range []string{"namespace", "scope", "tag", "agent", "session", "nonce"} {
		for _, replacement := range []string{`null`, `4`, `""`} {
			var fields map[string]json.RawMessage
			_ = json.Unmarshal([]byte(validIdentity), &fields)
			fields[field] = json.RawMessage(replacement)
			b, _ := json.Marshal(fields)
			cases = append(cases, `{"pane_id":"1","identity":`+string(b)+`}`)
		}
	}
	cases = append(cases, `{"pane_id":"1","identity":`+strings.Replace(validIdentity, `"nonce":"nonce-1"`, `"nonce":"a","nonce":"b"`, 1)+`}`)
	for _, raw := range cases {
		if _, err := DecodeRecord([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestLaunchIdentityValidation(t *testing.T) {
	for _, change := range []func(*LaunchIdentity){
		func(i *LaunchIdentity) { i.Namespace = "relative" }, func(i *LaunchIdentity) { i.Namespace = "/tmp/\n" },
		func(i *LaunchIdentity) { i.Scope = "../repo" }, func(i *LaunchIdentity) { i.Tag = "a/b" },
		func(i *LaunchIdentity) { i.Agent = "a b" }, func(i *LaunchIdentity) { i.Session = ".." },
		func(i *LaunchIdentity) { i.Session = "a/b" }, func(i *LaunchIdentity) { i.Session = "a\n" },
		func(i *LaunchIdentity) { i.Nonce = "\x00" },
	} {
		identity := testLaunchIdentity()
		change(&identity)
		if identity.Validate() == nil {
			t.Errorf("accepted %+v", identity)
		}
	}
}

func TestRecordPreservesAdditiveFieldsAndSizeBound(t *testing.T) {
	raw := `{"pane_id":"7","cwd":"/work","future":{"number":1e1000},"identity":{"namespace":"/tmp/zellij","scope":"repo_1","tag":"tag-1","agent":"codex","session":"pair-work","nonce":"nonce-1","future":{"x":[1,true]}}}`
	record, err := DecodeRecord([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	next, err := DecodeRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(next.Unknown["future"]) != `{"number":1e1000}` || string(next.IdentityUnknown["future"]) != `{"x":[1,true]}` {
		t.Fatalf("lost unknowns: %s", encoded)
	}
	tooLarge := `{"pane_id":"7","future":"` + strings.Repeat("x", 8192) + `"}`
	if _, err := DecodeRecord([]byte(tooLarge)); err == nil {
		t.Fatal("oversize decode accepted")
	}
	record.Unknown["future"] = json.RawMessage(`"` + strings.Repeat("x", 8192) + `"`)
	if _, err := EncodeRecord(record); err == nil {
		t.Fatal("oversize encode accepted")
	}
	if _, err := EncodeRecord(Record{PaneID: "-1"}); err == nil {
		t.Fatal("invalid encode accepted")
	}
}

func TestRecordEncodeRejectsAmbiguousRetainedMembers(t *testing.T) {
	cases := []Record{
		{PaneID: "1", Unknown: map[string]json.RawMessage{"pane_id": json.RawMessage(`"2"`)}},
		{PaneID: "1", Unknown: map[string]json.RawMessage{"future": json.RawMessage(`{"x":1,"x":2}`)}},
		{PaneID: "1", Identity: testLaunchIdentity(), IdentityUnknown: map[string]json.RawMessage{"nonce": json.RawMessage(`"other"`)}},
		{PaneID: "1", Identity: LaunchIdentity{Namespace: "/tmp"}},
	}
	for _, record := range cases {
		if _, err := EncodeRecord(record); err == nil {
			t.Errorf("accepted %+v", record)
		}
	}
}

func TestRecordMinimalLegacyAndSizeBoundary(t *testing.T) {
	record, err := DecodeRecord([]byte(`{"pane_id":"7"}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := DecodeRecord(encoded)
	if err != nil || roundTrip.Identity != (LaunchIdentity{}) {
		t.Fatalf("legacy round trip: %+v %v", roundTrip, err)
	}
	prefix, suffix := `{"pane_id":"7","future":"`, `"}`
	raw := prefix + strings.Repeat("x", MaxRecordBytes-len(prefix)-len(suffix)) + suffix
	if len(raw) != MaxRecordBytes {
		t.Fatal("test size mismatch")
	}
	if _, err := DecodeRecord([]byte(raw)); err != nil {
		t.Fatalf("exact bound: %v", err)
	}
	if _, err := DecodeRecord([]byte(raw + " ")); err == nil {
		t.Fatal("one byte over bound accepted")
	}
}

func TestIdentityCodecPreservesUnknownAndRejectsInvalidRecords(t *testing.T) {
	identity := testLaunchIdentity()
	raw, err := EncodeIdentity(identity, map[string]json.RawMessage{"future": json.RawMessage(`{"n":1e1000}`)})
	if err != nil {
		t.Fatal(err)
	}
	decoded, unknown, err := DecodeIdentity(raw)
	if err != nil || !decoded.Equal(identity) || string(unknown["future"]) != `{"n":1e1000}` {
		t.Fatalf("identity round trip: %+v %v %v", decoded, unknown, err)
	}
	for _, bad := range []string{`null`, `{}`, string(raw) + ` {}`, strings.Replace(string(raw), `"nonce":"nonce-1"`, `"nonce":"nonce-1","nonce":"other"`, 1)} {
		if _, _, err := DecodeIdentity([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	if _, err := EncodeIdentity(LaunchIdentity{}, nil); err == nil {
		t.Fatal("empty identity encoded")
	}
	if _, err := EncodeIdentity(identity, map[string]json.RawMessage{"future": json.RawMessage(`"` + strings.Repeat("x", MaxRecordBytes) + `"`)}); err == nil {
		t.Fatal("oversize identity encoded")
	}
}
