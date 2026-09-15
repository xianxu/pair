package storagegc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectedOwnerExplicitNamespaces(t *testing.T) {
	root := storeDirectory(t)
	scoped := filepath.Join(root, "repos", "scope")
	if err := os.MkdirAll(scoped, 0700); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "scope"} {
		o, err := SelectedOwner(scoped, key, "tag")
		if err != nil || o.DataDir != root || o.RepoScope != "scope" || o.Tag != "tag" {
			t.Fatalf("scoped = %+v %v", o, err)
		}
	}
	o, err := SelectedOwner(root, "", "tag")
	if err != nil || o.DataDir != root || o.RepoScope != "" {
		t.Fatalf("legacy = %+v %v", o, err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	o, err = SelectedOwner(alias, "", "tag")
	if err != nil || o.DataDir != root {
		t.Fatalf("physical root = %+v %v", o, err)
	}
	for _, args := range [][3]string{{"", "", "tag"}, {"relative", "", "tag"}, {scoped, "other", "tag"}, {root, "", ""}, {root, "", "../tag"}, {root + "/absent", "", "tag"}} {
		if _, err := SelectedOwner(args[0], args[1], args[2]); err == nil {
			t.Fatalf("invalid env accepted: %v", args)
		}
	}
}

func TestCancelUnchangedUseRetiresOnlyIntent(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.BeginUse(ctx, o, p, "draft")
	if err != nil {
		t.Fatal(err)
	}
	before, err := c.ReadOwner(o)
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return before.Activity.LastUse.Add(time.Hour) }
	if err := c.CancelUnchangedUse(ctx, o, id); err != nil {
		t.Fatal(err)
	}
	after, err := c.ReadOwner(o)
	if err != nil || len(after.Intents) != 0 || !after.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatalf("unchanged affected use: %+v %v", after, err)
	}
	if err := c.CancelUnchangedUse(ctx, o, id); err == nil {
		t.Fatal("unknown operation accepted")
	}
}

func TestFlatOverrideIgnoresLogicalRepositoryScope(t *testing.T) {
	root := storeDirectory(t)
	owner, err := SelectedOwner(root, "logical-scope", "tag")
	if err != nil || owner.DataDir != root || owner.RepoScope != "" {
		t.Fatalf("flat override changed namespace: %+v %v", owner, err)
	}
}
