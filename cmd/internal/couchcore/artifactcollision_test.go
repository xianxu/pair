package couchcore

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestAllocateThreadTagAtomicallyClaimsAgainstArtifactProducers(t *testing.T) {
	dataDir := t.TempDir()
	scope := launcher.RepoScope{Key: "0123456789abcdef"}
	producer, err := launcher.ClaimNewThreadAddress(dataDir, scope, "couch-0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = producer.Release() })
	store, ns := newTestThreadStore(t)
	entropy := append(make([]byte, 8), []byte{1, 2, 3, 4, 5, 6, 7, 8}...)
	got, err := store.AllocateThreadTag(scope.Key, ns.Dir(), time.Now(), bytes.NewReader(entropy), NewScopedThreadArtifactCollisionChecker(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got.Address.Tag != "couch-0102030405060708" {
		t.Fatalf("allocation reused producer-owned address: %+v", got.Address)
	}
}

func TestScopedArtifactCheckerReportsPairRegistrationEvidence(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)
	claim, err := checker.Claim(address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = claim.Release() })
	if got, err := checker.Registration(address); err != nil || got != RegistrationAbsent {
		t.Fatalf("reserved registration = %q, %v", got, err)
	}
	if err := launcher.EnsureThreadAddressForPair(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag), true); err != nil {
		t.Fatal(err)
	}
	if got, err := checker.Registration(address); err != nil || got != RegistrationEstablished {
		t.Fatalf("established registration = %q, %v", got, err)
	}
}

func TestScopedArtifactCollisionCheckerFindsEveryTagNameShape(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)
	scopeDir := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag)).ScopeDir()
	if err := os.MkdirAll(scopeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	for _, path := range []string{
		paths.Ledger(), paths.Draft(), paths.Log(), paths.QueueDir(), paths.Agent(),
		paths.AgentPID(), paths.AgentOutput(), paths.AgentPicks(), paths.AdaptLog(),
		paths.OuterTTY(), paths.NvimDraftPID(), paths.NvimScrollbackPID(),
		paths.Config("codex"), paths.LegacyCodexConfig(), paths.AgentReady("claude"),
		paths.Pane("codex"), paths.ScrollbackRaw("codex"), paths.ScrollbackANSI("codex"),
		paths.ScrollbackEvents("codex"), paths.ScrollbackViewport("codex"),
		paths.Changelog("codex"), paths.AgentDraft("codex"),
	} {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte("owned"), 0o600); err != nil {
				t.Fatal(err)
			}
			claim, err := checker.Claim(address)
			if !errors.Is(err, launcher.ErrThreadAddressClaimed) || claim != nil {
				t.Fatalf("Claim = %T, %v", claim, err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := os.WriteFile(filepath.Join(scopeDir, "draft-couch-00010203040506070.md"), []byte("neighbor"), 0o600); err != nil {
		t.Fatal(err)
	}
	claim, err := checker.Claim(address)
	if err != nil {
		t.Fatalf("neighbor tag claim: %v", err)
	}
	_ = claim.Release()
}

func TestScopedArtifactCollisionCheckerFindsDetachedSessionBinding(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: "session", ScopeKey: address.RepoScope, Tag: string(address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "session-names.jsonl"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	claim, err := NewScopedThreadArtifactCollisionChecker(dataDir).Claim(address)
	if !errors.Is(err, launcher.ErrThreadAddressClaimed) || claim != nil {
		t.Fatalf("session binding claim = %T, %v", claim, err)
	}
}

func TestScopedArtifactClaimerRejectsNonScopedPathsAndFutureFamilies(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"draft-pane-couch-0001020304050607.json",
		"image-capture-couch-0001020304050607.done",
		"parked-scrollback-couch-0001020304050607-20260826.raw",
		"last-terminal-pane-couch-0001020304050607",
		"review-definition-request-couch-0001020304050607.json",
		"future-family-couch-0001020304050607-variant.bin",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(paths.ScopeDir(), name)
			if err := os.WriteFile(path, []byte("owned"), 0o600); err != nil {
				t.Fatal(err)
			}
			claim, err := NewScopedThreadArtifactCollisionChecker(dataDir).Claim(address)
			if !errors.Is(err, launcher.ErrThreadAddressClaimed) || claim != nil {
				t.Fatalf("Claim = %T, %v", claim, err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
