package storagegc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func storeDirectory(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestRegistryMigrationAndRegistration(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	if _, err := c.ReadRegistry(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing registry: %v", err)
	}
	entries, _ := os.ReadDir(c.Root)
	if len(entries) != 0 {
		t.Fatal("read created metadata")
	}
	first := storeDirectory(t)
	if err := c.RegisterStore(ctx, first); err != nil {
		t.Fatal(err)
	}
	r, err := c.ReadRegistry()
	if err != nil || r.Version != 1 || r.MigrationComplete || !reflect.DeepEqual(r.Stores, []string{first}) {
		t.Fatalf("initial registry: %+v %v", r, err)
	}
	if err := c.CompleteMigration(ctx, nil); err == nil {
		t.Fatal("incomplete inventory accepted")
	}
	if err := c.CompleteMigration(ctx, []string{first, first}); err == nil {
		t.Fatal("duplicate acknowledgment accepted")
	}
	if err := c.CompleteMigration(ctx, []string{first}); err != nil {
		t.Fatal(err)
	}
	second := storeDirectory(t)
	if err := c.RegisterStore(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(ctx, first); err != nil {
		t.Fatal(err)
	}
	r, err = c.ReadRegistry()
	if err != nil || !r.MigrationComplete || len(r.Stores) != 2 {
		t.Fatalf("registered runtime invalidated migration: %+v %v", r, err)
	}
}
func TestRegistryCanonicalizesRegistrationButRejectsPersistedAliases(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	physical := storeDirectory(t)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(physical, alias); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(ctx, alias); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(ctx, physical); err != nil {
		t.Fatal(err)
	}
	r, err := c.ReadRegistry()
	if err != nil || !reflect.DeepEqual(r.Stores, []string{physical}) {
		t.Fatalf("alias not deduplicated: %+v %v", r, err)
	}
	r.Stores = []string{alias}
	raw, _ := json.Marshal(r)
	if err := os.WriteFile(c.registryPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("persisted alias accepted")
	}
}
func TestRegistryUnavailableStoreBlocksAllMutations(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	store := storeDirectory(t)
	if err := c.RegisterStore(ctx, store); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(c.registryPath())
	if err := os.Rename(store, store+"-offline"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(store+"-offline", store)
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("unavailable registered store accepted")
	}
	if err := c.CompleteMigration(ctx, []string{store}); err == nil {
		t.Fatal("unavailable store allowed completion")
	}
	if err := c.RegisterStore(ctx, storeDirectory(t)); err == nil {
		t.Fatal("outage overwritten by registration")
	}
	called := false
	if err := c.UnregisterStore(ctx, store, func(string) (bool, error) { called = true; return true, nil }); err == nil || called {
		t.Fatal("unavailable store unregistered")
	}
	after, _ := os.ReadFile(c.registryPath())
	if string(before) != string(after) {
		t.Fatal("registry changed on unavailable store")
	}
}
func TestRegistryRejectsMalformedMetadata(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"version":2,"stores":[]}`, `{"version":1,"stores":[],"unknown":true}`, `{"version":1,"stores":[]} {}`, `{"version":1,"stores":["relative"]}`, `{"version":1,"stores":null}`} {
		t.Run(raw, func(t *testing.T) {
			c, _ := coordinatorFixture(t)
			ctx := context.Background()
			if err := c.WithLock(ctx, func(*Locked) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(c.registryPath(), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := c.ReadRegistry(); err == nil {
				t.Fatal("invalid registry accepted")
			}
			if err := c.RegisterStore(ctx, storeDirectory(t)); err == nil {
				t.Fatal("invalid registry replaced")
			}
		})
	}
}
func TestUnregisterRequiresProofUnderCoordinator(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	store := storeDirectory(t)
	if err := c.RegisterStore(ctx, store); err != nil {
		t.Fatal(err)
	}
	for _, proof := range []func(string) (bool, error){nil, func(string) (bool, error) { return false, nil }, func(string) (bool, error) { return true, errors.New("unreadable references") }} {
		if err := c.UnregisterStore(ctx, store, proof); err == nil {
			t.Fatal("unregister without proof")
		}
	}
	if err := c.UnregisterStore(ctx, store, func(path string) (bool, error) {
		if path != store {
			t.Fatal("proof got wrong store")
		}
		fd, err := unix.Open(filepath.Join(c.Root, ".retention", "coordinator.lock"), unix.O_RDWR, 0)
		if err != nil {
			return false, err
		}
		defer unix.Close(fd)
		if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); !errors.Is(err, unix.EWOULDBLOCK) {
			t.Fatalf("proof ran outside coordination: %v", err)
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	r, err := c.ReadRegistry()
	if err != nil || len(r.Stores) != 0 {
		t.Fatalf("not removed: %+v %v", r, err)
	}
}
func TestRegistryDurabilityFailurePreservesRegistration(t *testing.T) {
	c, _ := coordinatorFixture(t)
	ctx := context.Background()
	first := storeDirectory(t)
	if err := c.RegisterStore(ctx, first); err != nil {
		t.Fatal(err)
	}
	c.BeforePersist = func() error { return errors.New("disk fault") }
	if err := c.RegisterStore(ctx, storeDirectory(t)); err == nil {
		t.Fatal("persist failure hidden")
	}
	if err := c.CompleteMigration(ctx, []string{first}); err == nil {
		t.Fatal("completion failure hidden")
	}
	if err := c.UnregisterStore(ctx, first, func(string) (bool, error) { return true, nil }); err == nil {
		t.Fatal("unregister failure hidden")
	}
	r, err := c.ReadRegistry()
	if err != nil || len(r.Stores) != 1 || r.MigrationComplete {
		t.Fatalf("failed mutations persisted: %+v %v", r, err)
	}
}
func TestRegistryRejectsFIFOWithoutBlocking(t *testing.T) {
	c, _ := coordinatorFixture(t)
	if err := c.WithLock(context.Background(), func(*Locked) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(c.registryPath(), 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := c.ReadRegistry(); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO read blocked")
	}
}

func TestLockedHoldsOnlyWithinCallback(t *testing.T) {
	c, _ := coordinatorFixture(t)
	var saved *Locked
	if err := c.WithLock(context.Background(), func(l *Locked) error {
		saved = l
		if !l.Holds(c.Root) || l.Holds(c.Root+"/other") {
			t.Fatal("incorrect root ownership")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if saved.Holds(c.Root) {
		t.Fatal("released lock still claims root")
	}
	if (*Locked)(nil).Holds(c.Root) {
		t.Fatal("nil lock claims root")
	}
}

func TestRegistryRejectsDuplicateStoreAndSymlinkFile(t *testing.T) {
	c, _ := coordinatorFixture(t)
	store := storeDirectory(t)
	if err := c.RegisterStore(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(StoreRegistry{Version: 1, Stores: []string{store, store}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.registryPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("duplicate store accepted")
	}
	if err := os.Remove(c.registryPath()); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "foreign.json")
	if err := os.WriteFile(target, []byte(`{"version":1,"stores":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, c.registryPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("followed registry symlink")
	}
	if err := c.RegisterStore(context.Background(), store); err == nil {
		t.Fatal("replaced registry symlink")
	}
}

func TestRegistryConcurrentRegistrationsKeepBothStores(t *testing.T) {
	c, _ := coordinatorFixture(t)
	peer, err := NewCoordinator(c.Root)
	if err != nil {
		t.Fatal(err)
	}
	first, second := storeDirectory(t), storeDirectory(t)
	start := make(chan struct{})
	done := make(chan error, 2)
	for i, coord := range []*Coordinator{c, peer} {
		store := []string{first, second}[i]
		go func() { <-start; done <- coord.RegisterStore(context.Background(), store) }()
	}
	close(start)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	r, err := c.ReadRegistry()
	if err != nil || len(r.Stores) != 2 {
		t.Fatalf("lost registration: %+v %v", r, err)
	}
}

func TestLockedRegisterStoreSharesPublicationLock(t *testing.T) {
	c, _ := coordinatorFixture(t)
	store := storeDirectory(t)
	var token *Locked
	if err := c.WithLock(context.Background(), func(l *Locked) error {
		token = l
		if err := l.RegisterStore(store); err != nil {
			return err
		}
		r, err := c.ReadRegistry()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(r.Stores, []string{store}) {
			t.Fatal("registration not visible before membership publish")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := token.RegisterStore(storeDirectory(t)); err == nil {
		t.Fatal("expired token registered store")
	}
	if err := (*Locked)(nil).RegisterStore(store); err == nil {
		t.Fatal("nil token registered store")
	}
}
