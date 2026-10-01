package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func aliasTestStore(t *testing.T, roots ...string) *ThreadStore {
	t.Helper()
	store, _ := newTestThreadStore(t)
	if err := store.withLock(func() error {
		raw, err := json.Marshal(threadManifest{SchemaVersion: 2, Threads: []ThreadAddress{}, SlotRepositories: roots})
		if err != nil {
			return err
		}
		return os.WriteFile(store.manifestPath(), raw, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	return store
}

func aliasOf(t *testing.T, store *ThreadStore, root string) string {
	t.Helper()
	names, err := store.RepositoryNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if n.Key == root {
			return n.Alias
		}
	}
	t.Fatalf("%s not enrolled in %+v", root, names)
	return ""
}

func TestRepositoryAliasStoreLifecycle(t *testing.T) {
	store := aliasTestStore(t, "/w/xianxu.dev", "/w/pair")
	if got := aliasOf(t, store, "/w/xianxu.dev"); got != "" {
		t.Fatalf("absent file: %q", got)
	}
	if err := store.SetRepositoryAlias("/w/xianxu.dev", "blog"); err != nil {
		t.Fatal(err)
	}
	if got := aliasOf(t, store, "/w/xianxu.dev"); got != "blog" {
		t.Fatalf("set: %q", got)
	}
	if err := store.SetRepositoryAlias("/w/pair", "p"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetRepositoryAlias("/w/xianxu.dev", ""); err != nil {
		t.Fatal(err)
	}
	if got, other := aliasOf(t, store, "/w/xianxu.dev"), aliasOf(t, store, "/w/pair"); got != "" || other != "p" {
		t.Fatalf("clear: %q, other %q", got, other)
	}
}

func TestRepositoryAliasStoreRefusalLeavesFileUntouched(t *testing.T) {
	store := aliasTestStore(t, "/w/xianxu.dev", "/w/pair")
	if err := store.SetRepositoryAlias("/w/xianxu.dev", "blog"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.repositoryAliasPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"blog", "xianxu.dev", "a:b"} {
		if err := store.SetRepositoryAlias("/w/pair", alias); err == nil {
			t.Fatalf("%q admitted", alias)
		}
	}
	if err := store.SetRepositoryAlias("/w/missing", "m"); !errors.Is(err, ErrRepositoryNotFound) {
		t.Fatalf("unenrolled: %v", err)
	}
	after, err := os.ReadFile(store.repositoryAliasPath())
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("refusal rewrote the file: %v\n%s\n%s", err, before, after)
	}
}

func TestRepositoryAliasStoreDropsUnenrolledAndWithholdsShadowing(t *testing.T) {
	store := aliasTestStore(t, "/w/xianxu.dev", "/w/pair")
	raw := `{"schema_version":1,"aliases":{"/w/gone":"old","/w/xianxu.dev":"pair"}}`
	if err := os.WriteFile(store.repositoryAliasPath(), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	// "pair" would shadow the pair directory, so it is withheld on read.
	if got := aliasOf(t, store, "/w/xianxu.dev"); got != "" {
		t.Fatalf("shadowing alias surfaced: %q", got)
	}
	if err := store.SetRepositoryAlias("/w/pair", "p"); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(store.repositoryAliasPath())
	if err != nil || strings.Contains(string(written), "/w/gone") {
		t.Fatalf("unenrolled entry kept: %v %s", err, written)
	}
}

func TestRepositoryAliasStoreRefusesMalformedFile(t *testing.T) {
	store := aliasTestStore(t, "/w/pair")
	for _, raw := range []string{`{"schema_version":1,"aliases":{},"extra":1}`, `{"schema_version":2,"aliases":{}}`, `{"schema_version":1}`, `not json`} {
		if err := os.WriteFile(store.repositoryAliasPath(), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RepositoryNames(); err == nil || !strings.Contains(err.Error(), "remove the file") {
			t.Fatalf("%s: %v", raw, err)
		}
	}
}

func TestRepositoryAliasStoreSlotLocalWritesRoot(t *testing.T) {
	global, local, _, _ := slotRetentionFixture(t)
	if err := local.SetRepositoryAlias(local.slot.PrimaryRoot, "short"); err != nil {
		t.Fatal(err)
	}
	if got := aliasOf(t, global, local.slot.PrimaryRoot); got != "short" {
		t.Fatalf("root store alias: %q", got)
	}
	if _, err := os.Stat(local.repositoryAliasPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("slot-local alias file written: %v", err)
	}
}

func TestRepositoryNamesWaitsOutABusyStore(t *testing.T) {
	store := aliasTestStore(t, "/w/xianxu.dev")
	if err := store.SetRepositoryAlias("/w/xianxu.dev", "blog"); err != nil {
		t.Fatal(err)
	}
	hold := func(d time.Duration) chan struct{} {
		locked, done := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			_ = store.withLock(func() error { close(locked); time.Sleep(d); return nil })
		}()
		<-locked
		return done
	}
	done := hold(100 * time.Millisecond)
	names, err := store.RepositoryNamesContext(context.Background())
	<-done
	if err != nil || len(names) != 1 || names[0].Alias != "blog" {
		t.Fatalf("brief writer: %+v %v", names, err)
	}
	done = hold(300 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = store.RepositoryNamesContext(ctx)
	<-done
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("caller deadline not honored: %v", err)
	}
}
