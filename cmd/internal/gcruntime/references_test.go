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

func TestCouchReferencesOnboardOnlySelectedLegacyArchives(t *testing.T) {
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }
	ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	store, err := couchcore.NewCoordinatedThreadStore(ns, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(); err != nil {
		t.Fatal(err)
	}
	scope := "816fc349d3faebf8"
	tags := []string{"couch-0000000000000001", "couch-0000000000000002"}
	dir := filepath.Join(ns.Dir(), "threadstore", "archive", scope)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, tag := range tags {
		if err := os.WriteFile(filepath.Join(dir, tag+".json"), []byte("legacy archive"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	adapter := CouchReferences{c}
	owner, _ := artifactpath.NewStorageOwner(c.Root, scope, tags[0])
	standalone, err := artifactpath.NewStorageOwner(c.Root, "", "standalone")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		if err := adapter.Onboard(context.Background(), l, []string{ns.Dir()}, nil); err != nil {
			return err
		}
		if err := adapter.Onboard(context.Background(), l, []string{ns.Dir()}, []artifactpath.StorageOwner{standalone}); err != nil {
			return err
		}
		refs, err := adapter.Snapshot(context.Background(), l, []string{ns.Dir()})
		if err != nil {
			return err
		}
		for _, a := range refs.Archives {
			if !a.ArchivedAt.IsZero() {
				t.Fatal("nil selection onboarded")
			}
		}
		if err := adapter.Onboard(context.Background(), l, []string{ns.Dir()}, []artifactpath.StorageOwner{standalone, owner}); err != nil {
			return err
		}
		refs, err = adapter.Snapshot(context.Background(), l, []string{ns.Dir()})
		if err != nil {
			return err
		}
		if len(refs.Archives) != 2 || !refs.Archives[0].ArchivedAt.Equal(now) || !refs.Archives[1].ArchivedAt.IsZero() {
			t.Fatalf("selection %+v", refs)
		}
		activity := &storagegc.ActivityRecord{Version: 1, Owner: owner, Incarnation: "test", InitializedAt: now.Add(-365 * 24 * time.Hour)}
		evidence := storagegc.Evidence{Owner: owner, Activity: activity, Live: storagegc.ProcessDead, Complete: true, ArchiveTimes: []time.Time{refs.Archives[0].ArchivedAt}}
		if d := storagegc.Decide(now.Add(storagegc.RetentionPeriod-time.Nanosecond), evidence); d.State != storagegc.Grace {
			t.Fatalf("short grace %+v", d)
		}
		if d := storagegc.Decide(now.Add(storagegc.RetentionPeriod), evidence); d.State != storagegc.Eligible {
			t.Fatalf("never eligible %+v", d)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
