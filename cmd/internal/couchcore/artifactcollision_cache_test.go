package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"testing"
)

func TestPairSessionNameAndScopedBindingsShareIndexCache(t *testing.T) {
	dir := t.TempDir()
	scope := "aaaaaaaaaaaaaaaa"
	scopeDir := filepath.Join(dir, "repos", scope)
	if err := os.MkdirAll(scopeDir, 0700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: "pair-one", ScopeKey: scope, Tag: "work", RepoRoot: "/tmp/project", RepoName: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scopeDir, "session-names.jsonl"), []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reads, decodes := 0, 0
	c := NewScopedThreadArtifactCollisionChecker(dir)
	if c.SessionIndexes == nil {
		t.Fatal("constructor must own cache")
	}
	c.SessionIndexes = launcher.NewSessionNameIndexCacheWithIO(launcher.SessionNameIndexCacheIO{ReadFile: func(path string) ([]byte, error) { b, e := os.ReadFile(path); reads += len(b); return b, e }, Decode: func(raw string) (launcher.SessionNameIndex, error) {
		decodes += len(raw)
		return launcher.DecodeSessionNameIndex(raw)
	}})
	address := ThreadAddress{RepoScope: scope, Tag: "work"}
	if name, err := c.PairSessionName(context.Background(), address); err != nil || name != "pair-one" {
		t.Fatalf("cold %s %v", name, err)
	}
	if reads == 0 || decodes == 0 {
		t.Fatal("counters disconnected")
	}
	reads, decodes = 0, 0
	copy := c
	for range 3 {
		if _, err := copy.PairSessionName(context.Background(), address); err != nil {
			t.Fatal(err)
		}
		bindings, _, _, err := c.resolveScopedBindings(context.Background(), []ThreadAddress{address}, nil)
		if err != nil || len(bindings) != 1 {
			t.Fatalf("bindings %v %v", bindings, err)
		}
	}
	if reads != 0 || decodes != 0 {
		t.Fatalf("warm read=%d decode=%d", reads, decodes)
	}
}
