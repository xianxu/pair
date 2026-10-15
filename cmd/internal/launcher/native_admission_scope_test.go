package launcher

import (
	"context"
	"strings"
	"testing"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func admissionCodexScope(t *testing.T) string {
	t.Helper()
	encoded, err := EncodeNativeAdmissionCodexScope(si.CodexSelectionScope{CodexHome: "/private/codex", SQLiteHome: "/private/sqlite", Provider: "openai", CWD: "/private/work", CWDs: []string{"/private/work"}, Mode: si.CodexModeTUI})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
func TestNativeAdmissionScopedDigestAndValidation(t *testing.T) {
	scope := admissionCodexScope(t)
	argv := []string{"codex", "resume", "id"}
	legacy, _ := NativeAdmissionCommandDigest("codex", argv)
	empty, err := NativeAdmissionScopedCommandDigest("codex", argv, "")
	if err != nil || empty != legacy {
		t.Fatal("legacy digest changed")
	}
	scoped, err := NativeAdmissionScopedCommandDigest("codex", argv, scope)
	if err != nil || scoped == legacy {
		t.Fatal("scope not bound")
	}
	for _, invalid := range []string{"{}", scope + "x", strings.Replace(scope, "openai", "", 1)} {
		if _, err := NativeAdmissionScopedCommandDigest("codex", argv, invalid); err == nil {
			t.Fatal("invalid scope accepted")
		}
	}
	if _, err := NativeAdmissionScopedCommandDigest("claude", argv, scope); err == nil {
		t.Fatal("Codex scope accepted for another agent")
	}
}
func TestNativeAdmissionScopedGrantTamperingCannotConsume(t *testing.T) {
	w := newAdmissionWorld()
	s := w.store(t.TempDir())
	r := admissionRequest(t, "scoped")
	r.CodexScope = admissionCodexScope(t)
	r.CommandDigest, _ = NativeAdmissionScopedCommandDigest(r.Agent, []string{"codex", "resume", r.NativeID}, r.CodexScope)
	raw := reserveAdmission(t, s, r)
	got, err := s.InspectCodexScope(raw)
	if err != nil || got != r.CodexScope {
		t.Fatal("inspection lost binding", err)
	}
	grant, err := decodeAdmissionGrant(raw)
	if err != nil {
		t.Fatal(err)
	}
	w.as(12)
	for _, scope := range []string{"", strings.Replace(r.CodexScope, "/private/sqlite", "/private/other", 1)} {
		changed := grant
		changed.Request.CodexScope = scope
		changed.Request.CommandDigest, _ = NativeAdmissionScopedCommandDigest(r.Agent, []string{"codex", "resume", r.NativeID}, scope)
		forged, _ := encodeAdmissionGrant(changed)
		if _, err := s.Claim(context.Background(), forged, r.Agent, changed.Request.CommandDigest); err == nil {
			t.Fatal("forged scope consumed reservation")
		}
	}
	if _, err := s.Claim(context.Background(), raw, r.Agent, r.CommandDigest); err != nil {
		t.Fatal("tampering consumed real reservation", err)
	}
}

func TestNativeAdmissionScopedGrantEnvelope(t *testing.T) {
	scope := si.CodexSelectionScope{CodexHome: "/" + strings.Repeat("h", 3500), SQLiteHome: "/" + strings.Repeat("s", 3500), Provider: "openai", CWD: "/work", Mode: si.CodexModeTUI}
	encoded, err := EncodeNativeAdmissionCodexScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	r := admissionRequest(t, strings.Repeat("i", 4096))
	r.Attempt = strings.Repeat("a", 1024)
	r.CodexScope = encoded
	g := nativeAdmissionGrant{Version: 1, Request: r, Token: strings.Repeat("b", 64), Helper: storagegc.ProcessIdentity{PID: 7, Birth: strings.Repeat("c", 1024)}}
	raw, err := encodeAdmissionGrant(g)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeAdmissionGrant(raw); err != nil {
		t.Fatalf("valid scoped request generated unreadable grant: %v", err)
	}
	w := newAdmissionWorld()
	store := w.store(t.TempDir())
	raw = reserveAdmission(t, store, r)
	if scope, err := store.InspectCodexScope(raw); err != nil || scope != encoded {
		t.Fatal("large reserved scope lost", err)
	}
	w.as(12)
	if _, err := store.Claim(context.Background(), raw, r.Agent, r.CommandDigest); err != nil {
		t.Fatal("large reserved grant cannot claim", err)
	}
}

func FuzzNativeAdmissionCodexScope(f *testing.F) {
	f.Add(`{}`)
	f.Add(`{"CodexHome":"/codex","SQLiteHome":"/db","Provider":"openai","CWD":"/work","Worktrees":false,"CWDs":null,"Mode":"tui"}`)
	f.Fuzz(func(t *testing.T, raw string) {
		scope, err := decodeNativeAdmissionCodexScope(raw)
		if err != nil {
			return
		}
		encoded, err := EncodeNativeAdmissionCodexScope(scope)
		if err != nil || encoded != raw {
			t.Fatal("noncanonical scope accepted")
		}
		digest, err := NativeAdmissionScopedCommandDigest("codex", []string{"codex", "resume", "id"}, raw)
		if err != nil || digest == "" {
			t.Fatal("validated scope cannot bind command")
		}
	})
}

func TestNativeAdmissionScopedGrantOversizeRejected(t *testing.T) {
	if _, err := decodeAdmissionGrant(strings.Repeat("a", (64<<10)+1)); err == nil {
		t.Fatal("oversized grant decoded")
	}
	if _, err := encodeAdmissionGrant(nativeAdmissionGrant{Token: strings.Repeat("a", 64<<10)}); err == nil {
		t.Fatal("oversized grant encoded")
	}
}
