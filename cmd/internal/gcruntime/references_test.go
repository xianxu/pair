package gcruntime

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCouchReferencesCarryVisibilityAndArchiveGrace(t *testing.T) {
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	storePath, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ns, err := couchcore.ExistingCouchNamespace(storePath)
	if err != nil {
		t.Fatal(err)
	}
	store, err := couchcore.NewCoordinatedThreadStore(ns, c)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c.Now = func() time.Time { return now }
	address := couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"}
	record := couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: address, StartingPath: "/repo", WorkingPath: "/repo", CreatedAt: now, Revision: 1, LastActiveAt: now, LatestLaunchProfile: &couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}}
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	owner, _ := artifactpath.NewStorageOwner(c.Root, address.RepoScope, string(address.Tag))
	adapter := CouchReferences{c}
	if err := c.WithReadLock(context.Background(), func(l *storagegc.Locked) error {
		refs, err := adapter.Snapshot(context.Background(), l, []string{ns.Dir()})
		if err != nil {
			return err
		}
		if len(refs.Visible) != 1 || refs.Visible[0] != owner {
			t.Fatalf("%+v", refs)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ArchiveThread(address); err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		refs, err := adapter.Snapshot(context.Background(), l, []string{ns.Dir()})
		if err != nil {
			return err
		}
		if len(refs.Visible) != 0 || len(refs.Archives) != 1 || !refs.Archives[0].ArchivedAt.Equal(now) {
			t.Fatalf("%+v", refs)
		}
		if err := adapter.Detach(l, "test-operation", refs.Archives[0]); err != nil {
			return err
		}
		if err := adapter.Detach(l, "test-operation", refs.Archives[0]); err != nil {
			return err
		}
		return adapter.Forget(l, "test-operation", refs.Archives[0])
	}); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if err := c.WithReadLock(context.Background(), func(l *storagegc.Locked) error {
		_, err := adapter.Snapshot(context.Background(), l, []string{missing})
		if err == nil {
			t.Fatal("missing store accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("preview created missing store")
	}
}
