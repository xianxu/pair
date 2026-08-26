package launcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestStandalonePairReadsCouchThreadStoreAndPreservesScopedArtifacts(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	storeDir := filepath.Join(root, "couch")
	namespace, err := couchcore.ResolveCouchNamespace(storeDir, root)
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewThreadStore(namespace)
	record, err := store.CreateThread(couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address: couchcore.ThreadAddress{
			RepoScope: scope.Key,
			Tag:       "couch-0102030405060708",
		},
		StartingPath: repo,
		WorkingPath:  repo,
		CreatedAt:    time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		Revision:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	name := "compiler"
	if _, err := store.ApplyThreadMetadata(record.Address, record.Revision, couchcore.ThreadMetadataPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}

	globalData := filepath.Join(root, "pair-data")
	paths := launcher.NewScopedPaths(globalData, scope, string(record.Address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Draft(), []byte("durable draft"), 0o600); err != nil {
		t.Fatal(err)
	}

	index, err := launcher.LoadThreadIndex(storeDir, func(path string) (string, error) {
		raw, err := os.ReadFile(path)
		return string(raw), err
	})
	if err != nil {
		t.Fatal(err)
	}
	matches, err := launcher.ResolveThreadIndexReference(index.Entries, scope.Key, "compiler")
	if err != nil || len(matches) != 1 || matches[0].Address.Tag != string(record.Address.Tag) {
		t.Fatalf("standalone resolution = %+v, %v", matches, err)
	}
	if raw, err := os.ReadFile(paths.Draft()); err != nil || string(raw) != "durable draft" {
		t.Fatalf("scoped artifact changed during lookup: %q, %v", raw, err)
	}
}
