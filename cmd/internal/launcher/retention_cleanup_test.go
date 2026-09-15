package launcher

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"os"
	"path/filepath"
	"testing"
)

func TestRetentionBindingsRemoveOnlyExactOwner(t *testing.T) {
	root := t.TempDir()
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(c.Root, "repos", "scope")
	os.MkdirAll(selected, 0700)
	rt := NewScopedOSRuntime(c.Root, selected, "")
	for _, e := range []SessionNameEntry{{SessionName: "📁repo-tag", ScopeKey: "scope", Tag: "tag", RepoName: "repo", RepoRoot: "/repo"}, {SessionName: "📁other-tag", ScopeKey: "other", Tag: "tag", RepoName: "repo", RepoRoot: "/repo"}, {SessionName: "📁repo-kept", ScopeKey: "scope", Tag: "kept", RepoName: "repo", RepoRoot: "/repo"}} {
		if err := rt.AppendSessionNameIndex(e); err != nil {
			t.Fatal(err)
		}
	}
	owner, _ := artifactpath.NewStorageOwner(c.Root, "scope", "tag")
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return RemoveOwnerSessionBindings(l, owner) }); err != nil {
		t.Fatal(err)
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 2 {
		t.Fatalf("%+v", index)
	}
	for _, e := range index.Entries {
		if e.ScopeKey == "scope" && e.Tag == "tag" {
			t.Fatal("expired binding retained")
		}
	}
}
