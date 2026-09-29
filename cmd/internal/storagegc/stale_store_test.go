package storagegc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRegistrationSurvivesUnavailableAuxiliaryStore(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	active, auxiliary := storeDirectory(t), storeDirectory(t)
	for _, path := range []string{active, auxiliary} {
		if err := c.RegisterStore(ctx, path); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.CompleteMigration(ctx, []string{active, auxiliary}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(auxiliary); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(ctx, active); err != nil {
		t.Fatalf("intact store blocked by auxiliary outage: %v", err)
	}
	another := storeDirectory(t)
	if err := c.RegisterStore(ctx, another); err != nil {
		t.Fatal(err)
	}
	registry, err := c.readRegistryFile()
	if err != nil || !slices.Contains(registry.Stores, auxiliary) || !registry.MigrationComplete {
		t.Fatalf("registration discarded inventory: %+v %v", registry, err)
	}
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("GC inventory accepted missing store")
	}
	if err := c.CompleteMigration(ctx, []string{active, another}); err == nil {
		t.Fatal("silently forgot unavailable store")
	}
}

func TestUnavailableStorePreventsCollectionAfterRegistration(t *testing.T) {
	gc, owner := collectorFixture(t)
	ctx := context.Background()
	gone := storeDirectory(t)
	if err := gc.Coordinator.RegisterStore(ctx, gone); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	if err := gc.Coordinator.RegisterStore(ctx, storeDirectory(t)); err != nil {
		t.Fatal(err)
	}
	for _, apply := range []bool{false, true} {
		var report CollectionReport
		var err error
		if apply {
			report, err = gc.Apply(ctx, 100)
		} else {
			report, err = gc.Preview(ctx)
		}
		if err == nil && (report.MigrationComplete || report.BlockReason == "" || report.Collected != 0) {
			t.Fatalf("unavailable inventory accepted: %+v", report)
		}
		for _, item := range report.Items {
			if item.Decision.State == Eligible {
				t.Fatal("preview offered collection with unavailable references")
			}
		}
		if raw, err := os.ReadFile(filepath.Join(owner.Directory(), "draft-tag.md")); err != nil || string(raw) != "draft" {
			t.Fatalf("unavailable references lost payload: %q %v", raw, err)
		}
	}
}

func TestForgetMissingStoreRefusesPresentAndUnsafePaths(t *testing.T) {
	for _, kind := range []string{"present", "symlink", "parent-symlink", "unknown", "unclean"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := coordinatorFixture(t)
			ctx := context.Background()
			parent := storeDirectory(t)
			path := filepath.Join(parent, "store")
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := c.RegisterStore(ctx, path); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(c.registryPath())
			switch kind {
			case "symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(parent, "absent"), path); err != nil {
					t.Fatal(err)
				}
			case "parent-symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(parent); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(storeDirectory(t), parent); err != nil {
					t.Fatal(err)
				}
			case "unknown":
				path = filepath.Join(parent, "absent")
			case "unclean":
				path += "/../store"
			}
			if err := c.ForgetMissingStore(ctx, path); err == nil {
				t.Fatal("unsafe abandonment accepted")
			}
			after, _ := os.ReadFile(c.registryPath())
			if string(before) != string(after) {
				t.Fatal("refused abandonment changed registry")
			}
		})
	}
}

func TestForgetMissingStorePreservesOtherReferences(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	active, gone := storeDirectory(t), storeDirectory(t)
	for _, p := range []string{active, gone} {
		if err := c.RegisterStore(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.CompleteMigration(ctx, []string{active, gone}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	if err := c.ForgetMissingStore(ctx, gone); err != nil {
		t.Fatal(err)
	}
	registry, err := c.ReadRegistry()
	if err != nil || registry.MigrationComplete || !slices.Equal(registry.Stores, []string{active}) {
		t.Fatalf("lost references or migration reset: %+v %v", registry, err)
	}
	if err := c.CompleteMigration(ctx, []string{active}); err != nil {
		t.Fatal(err)
	}
}

func TestUnavailablePermissionsAreNotMissingStoreEvidence(t *testing.T) {
	c, _ := coordinatorFixture(t)
	parent := storeDirectory(t)
	store := filepath.Join(parent, "store")
	if err := os.Mkdir(store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(c.registryPath())
	if err := os.Chmod(parent, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(parent, 0700)
	if _, err := os.Lstat(store); err == nil {
		t.Skip("process bypasses directory permissions")
	}
	if err := c.ForgetMissingStore(context.Background(), store); err == nil {
		t.Fatal("permission failure treated as missing")
	}
	if err := c.RegisterStore(context.Background(), store); err == nil {
		t.Fatal("unreadable selected store accepted")
	}
	if err := c.RegisterStore(context.Background(), storeDirectory(t)); err != nil {
		t.Fatalf("unrelated unreadable store blocked intact registration: %v", err)
	}
	after, _ := os.ReadFile(c.registryPath())
	if string(before) == string(after) {
		t.Fatal("intact registration did not publish")
	}
	registry, err := c.readRegistryFile()
	if err != nil || !slices.Contains(registry.Stores, store) {
		t.Fatal("unavailable registration lost", err)
	}
}
