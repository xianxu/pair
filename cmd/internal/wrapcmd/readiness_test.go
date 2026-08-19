package wrapcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestPublishAgentReadyWritesRecordFromPairEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("PAIR_SESSION_NAME", "pair-work")
	t.Setenv("PAIR_LAUNCH_NONCE", "nonce-1")

	p := &proxy{agentBasename: "codex"}
	p.resolvePaths()
	if p.agentReadyPath == "" {
		t.Fatal("agentReadyPath is empty")
	}
	if err := p.publishAgentReady(321); err != nil {
		t.Fatalf("publishAgentReady returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "agent-ready-work-codex.json"))
	if err != nil {
		t.Fatalf("read ready record: %v", err)
	}
	got, err := readiness.Decode(string(raw))
	if err != nil {
		t.Fatalf("Decode ready record: %v", err)
	}
	if got.Tag != "work" || got.Agent != "codex" || got.Session != "pair-work" || got.Nonce != "nonce-1" || got.PID != 321 {
		t.Fatalf("ready record = %+v", got)
	}
}

func TestPublishAgentReadySkipsWhenPairEnvIncomplete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("PAIR_SESSION_NAME", "")
	t.Setenv("PAIR_LAUNCH_NONCE", "")

	p := &proxy{agentBasename: "codex"}
	p.resolvePaths()
	if err := p.publishAgentReady(321); err != nil {
		t.Fatalf("publishAgentReady returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent-ready-work-codex.json")); !os.IsNotExist(err) {
		t.Fatalf("ready file stat err = %v, want not exist", err)
	}
}
