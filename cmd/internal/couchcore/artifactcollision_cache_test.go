package couchcore

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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

type collisionLedgerCounter struct{ reads, decodes int64 }

func (c *collisionLedgerCounter) ReadWork(a sessioninventory.Artifact, n int64)   { c.reads += n }
func (c *collisionLedgerCounter) DecodeWork(a sessioninventory.Artifact, n int64) { c.decodes += n }
func TestCollisionNativeBindingReusesLedgerAcrossRuntimeCopies(t *testing.T) {
	dir := t.TempDir()
	scope := "aaaaaaaaaaaaaaaa"
	scopeDir := filepath.Join(dir, "repos", scope)
	if e := os.MkdirAll(scopeDir, 0700); e != nil {
		t.Fatal(e)
	}
	rows := fmt.Sprintf("{\"v\":3,\"kind\":\"launch\",\"scope_key\":%q,\"tag\":\"work\",\"agent\":\"codex\",\"pair_log_offset\":0,\"artifact_boundaries\":[],\"requested_native_id\":\"native\",\"request_origin\":\"resume\",\"baseline_complete\":false}\n", scope)
	if e := os.WriteFile(filepath.Join(scopeDir, "ledger-work.jsonl"), []byte(rows), 0600); e != nil {
		t.Fatal(e)
	}
	c := NewScopedThreadArtifactCollisionChecker(dir)
	counts := &collisionLedgerCounter{}
	c.InventoryWork = counts
	if c.OwnerLaunches == nil {
		t.Fatal("collision constructor lost shared owner cache")
	}
	q, e := c.ResolveEstablished(context.Background(), scope, "work", "codex")
	if e != nil || q.NativeID != "native" {
		t.Fatalf("cold=%+v err=%v", q, e)
	}
	if counts.reads == 0 || counts.decodes == 0 {
		t.Fatal("cold counters disconnected")
	}
	counts.reads, counts.decodes = 0, 0
	copy := c
	for range 3 {
		q, e = copy.ResolveEstablished(context.Background(), scope, "work", "codex")
		if e != nil || q.NativeID != "native" {
			t.Fatalf("warm=%+v err=%v", q, e)
		}
	}
	if counts.reads != 0 || counts.decodes != 0 {
		t.Fatalf("recreated scoped runtime repeated ledger IO/decode %d/%d", counts.reads, counts.decodes)
	}
}

func TestScopedBindingsCacheUnselectedScopeAndInvalidateChangedClaims(t *testing.T) {
	dir := t.TempDir()
	selected, other := "aaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbb"
	write := func(scope, name string) string {
		t.Helper()
		path := filepath.Join(dir, "repos", scope, "session-names.jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: name, ScopeKey: scope, Tag: "work", RepoRoot: "/tmp/project", RepoName: "project"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(line+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	write(selected, "pair-one")
	otherPath := write(other, "pair-other")
	stats := map[string]int{}
	reads, decodes := 0, 0
	c := NewScopedThreadArtifactCollisionChecker(dir)
	c.SessionIndexes = launcher.NewSessionNameIndexCacheWithIO(launcher.SessionNameIndexCacheIO{
		Stat:     func(path string) (os.FileInfo, error) { stats[path]++; return os.Lstat(path) },
		ReadFile: func(path string) ([]byte, error) { raw, err := os.ReadFile(path); reads += len(raw); return raw, err },
		Decode: func(raw string) (launcher.SessionNameIndex, error) {
			decodes += len(raw)
			return launcher.DecodeSessionNameIndex(raw)
		},
	})
	address := ThreadAddress{RepoScope: selected, Tag: "work"}
	otherAddress := ThreadAddress{RepoScope: other, Tag: "work"}
	observe := func(wantClaims int) {
		t.Helper()
		bindings, current, _, err := c.resolveScopedBindings(context.Background(), []ThreadAddress{address}, nil)
		if err != nil || len(bindings) != 1 || bindings[0].Address != address {
			t.Fatalf("selected bindings=%v err=%v", bindings, err)
		}
		if current[otherAddress] == "" || claimsFromBindings(current)["pair-one"] != wantClaims {
			t.Fatalf("unselected scope missing or stale: current=%v want claims=%d", current, wantClaims)
		}
	}
	observe(1)
	if reads == 0 || decodes == 0 || stats[otherPath] == 0 {
		t.Fatalf("census bypassed shared cache: reads=%d decodes=%d stats=%v", reads, decodes, stats)
	}
	reads, decodes, stats = 0, 0, map[string]int{}
	for range 3 {
		observe(1)
	}
	if reads != 0 || decodes != 0 || stats[otherPath] == 0 {
		t.Fatalf("warm census lost cache admission: reads=%d decodes=%d stats=%v", reads, decodes, stats)
	}
	write(other, "pair-one")
	observe(2)
	if reads == 0 || decodes == 0 {
		t.Fatal("changed unselected scope was not decoded")
	}
	reads, decodes, stats = 0, 0, map[string]int{}
	observe(2)
	if reads != 0 || decodes != 0 || stats[otherPath] == 0 {
		t.Fatalf("changed claims did not settle in cache: reads=%d decodes=%d stats=%v", reads, decodes, stats)
	}
}
