package couchcore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

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
			collision, err := checker.Collides(address)
			if err != nil || !collision {
				t.Fatalf("Collides = %v, %v", collision, err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := os.WriteFile(filepath.Join(scopeDir, "draft-couch-00010203040506070.md"), []byte("neighbor"), 0o600); err != nil {
		t.Fatal(err)
	}
	if collision, err := checker.Collides(address); err != nil || collision {
		t.Fatalf("neighbor tag collision = %v, %v", collision, err)
	}
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
	collision, err := NewScopedThreadArtifactCollisionChecker(dataDir).Collides(address)
	if err != nil || !collision {
		t.Fatalf("session binding collision = %v, %v", collision, err)
	}
}
