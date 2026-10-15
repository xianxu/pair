package couchcore

import (
	"encoding/base64"
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
	if err != nil || got != want {
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
