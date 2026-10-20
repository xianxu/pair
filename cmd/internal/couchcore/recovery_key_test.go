package couchcore

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
)

func TestRecoveryKeyBindsExactSelectionAndEvidence(t *testing.T) {
	want := RecoveryCandidateSelector{Version: 1, Address: ThreadAddress{RepoScope: "abcdef1234567890", Tag: "1-pair-42"}, Agent: "codex", NativeID: "01a1333f-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Digest: strings.Repeat("a", 64)}
	key, err := EncodeRecoveryCandidateKey(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ValidateRecoveryCandidateKey(key, want)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip: %#v %v", got, err)
	}
	for _, change := range []func(*RecoveryCandidateSelector){
		func(s *RecoveryCandidateSelector) { s.Address.Tag = "1-pair-43" },
		func(s *RecoveryCandidateSelector) { s.NativeID = "other" },
		func(s *RecoveryCandidateSelector) { s.Digest = strings.Repeat("b", 64) },
		func(s *RecoveryCandidateSelector) { s.Agent = "claude" },
	} {
		stale := want
		change(&stale)
		if _, err := ValidateRecoveryCandidateKey(key, stale); err == nil {
			t.Fatalf("accepted changed evidence %#v", stale)
		}
	}
	fresh := want
	fresh.NativeID = ""
	freshKey, err := EncodeRecoveryCandidateKey(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if freshKey == key {
		t.Fatal("fresh and native share a selector")
	}
}

func TestRecoveryKeyRejectsUntrustedEncoding(t *testing.T) {
	expected := RecoveryCandidateSelector{}
	for _, raw := range []string{`{}`, `{"version":1,"version":2}`, `{"version":1,"unknown":true}`, `null`} {
		key := base64.RawURLEncoding.EncodeToString([]byte(raw))
		if _, err := ValidateRecoveryCandidateKey(key, expected); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, key := range []string{"", strings.Repeat("a", 4097), "%%%"} {
		if _, err := ValidateRecoveryCandidateKey(key, expected); err == nil {
			t.Fatal("accepted malformed key")
		}
	}
}

func FuzzRecoveryCandidateKey(f *testing.F) {
	f.Add("garbage")
	f.Fuzz(func(t *testing.T, key string) { _, _ = ValidateRecoveryCandidateKey(key, RecoveryCandidateSelector{}) })
}

func TestNativeRecoveryKeyBindsSourceWindow(t *testing.T) {
	raw := `{"version":2,"address":{"repo_scope":"abcdef1234567890","tag":"1-pair-42"},"agent":"codex","native_id":"native","digest":"` + strings.Repeat("a", 64) + `","source":{"kind":"native-only","cutoff":"2026-10-19T12:00:00Z","revision":7},"expires_at":"2026-10-19T12:05:00Z"}`
	key := base64.RawURLEncoding.EncodeToString([]byte(raw))
	got, err := decodeRecoveryCandidateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeRecoveryCandidateKey(got)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateRecoveryCandidateKey(encoded, got); err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ old, new string }{
		{`"revision":7`, `"revision":8`},
		{`12:05:00Z`, `12:06:00Z`},
		{`"native_id":"native"`, `"native_id":""`},
		{`"kind":"native-only"`, `"kind":"capture-backed"`},
	} {
		changed := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(raw, change.old, change.new, 1)))
		if _, err := ValidateRecoveryCandidateKey(changed, got); err == nil {
			t.Fatalf("accepted %s", change.new)
		}
	}
}

func TestRecoveryKeyVersionBoundariesAndFrozenV1Wire(t *testing.T) {
	selector := RecoveryCandidateSelector{Version: 1, Address: ThreadAddress{RepoScope: "abcdef1234567890", Tag: "1-pair-42"}, Agent: "codex", NativeID: "native", Digest: strings.Repeat("a", 64)}
	key, err := EncodeRecoveryCandidateKey(selector)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":1,"address":{"repo_scope":"abcdef1234567890","tag":"1-pair-42"},"agent":"codex","native_id":"native","digest":"` + strings.Repeat("a", 64) + `"}`
	if string(raw) != want {
		t.Fatalf("v1 wire changed: %s", raw)
	}
	for _, addition := range []string{`"source":null,`, `"expires_at":"0001-01-01T00:00:00Z",`} {
		invalid := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(want, "{", "{"+addition, 1)))
		if _, err := decodeRecoveryCandidateKey(invalid); err == nil {
			t.Fatalf("v1 accepted %s", addition)
		}
	}
}
