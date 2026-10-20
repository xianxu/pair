package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDrainStoreAtomicOpaqueRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := newDrainStore(dir)
	ctx := context.Background()
	if b, err := s.Read(ctx); err != nil || b != nil {
		t.Fatalf("missing %q %v", b, err)
	}
	raw := json.RawMessage(`{"schema_version":1,"owner":"couch-drain:one","future":{"nested":[1,{"unknown":true}]},"pending":{"effect":"pause","future":9}}`)
	if err := s.Write(ctx, raw); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(ctx)
	if err != nil || string(got) != string(raw) {
		t.Fatalf("round trip %s %v", got, err)
	}
	info, err := os.Stat(filepath.Join(dir, "drain.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	for _, bad := range []json.RawMessage{[]byte(`{"schema_version":2}`), []byte(`{"schema_version":1,"x":{"a":1,"a":2}}`), []byte(`null`), []byte(strings.Repeat(" ", 1<<20) + `{}`)} {
		if err := s.Write(ctx, bad); err == nil {
			t.Fatalf("accepted %s", bad[:min(len(bad), 100)])
		}
	}
	got, _ = s.Read(ctx)
	if string(got) != string(raw) {
		t.Fatal("invalid rewrite destroyed intent")
	}
}

func TestDrainStoreFailureRetainsIntent(t *testing.T) {
	dir := t.TempDir()
	s := newDrainStore(dir)
	ctx := context.Background()
	old := json.RawMessage(`{"schema_version":1,"pending":"pause"}`)
	if err := s.Write(ctx, old); err != nil {
		t.Fatal(err)
	}
	s.beforeRename = func() error { return errors.New("crash before rename") }
	if err := s.Write(ctx, json.RawMessage(`{"schema_version":1,"pending":"resume"}`)); err == nil {
		t.Fatal("missing injected failure")
	}
	got, _ := s.Read(ctx)
	if string(got) != string(old) {
		t.Fatal("failed replace erased intent")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary residue: %v", entries)
	}
	s.beforeRename = nil
	if err := os.WriteFile(filepath.Join(dir, "drain.json"), []byte(`{"schema_version":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(ctx); err == nil {
		t.Fatal("unsafe newer record treated inactive")
	}
}

func TestDrainStoreContextBoundsSerialization(t *testing.T) {
	s := newDrainStore(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Write(ctx, json.RawMessage(`{"schema_version":1}`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v", err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	s.beforeRename = func() error { close(entered); <-release; return nil }
	done := make(chan error, 1)
	go func() { done <- s.Write(context.Background(), json.RawMessage(`{"schema_version":1}`)) }()
	<-entered
	c, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	_, err := newDrainStore(s.namespace).Read(c)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked read: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDrainStoreUncertainPublicationRetainsNewIntent(t *testing.T) {
	s := newDrainStore(t.TempDir())
	ctx := context.Background()
	if err := s.Write(ctx, json.RawMessage(`{"schema_version":1,"pending":"pause"}`)); err != nil {
		t.Fatal(err)
	}
	s.afterRename = func() error { return errors.New("crash after publication") }
	next := json.RawMessage(`{"schema_version":1,"pending":"resume"}`)
	if err := s.Write(ctx, next); err == nil {
		t.Fatal("missing uncertain publication failure")
	}
	got, err := newDrainStore(s.namespace).Read(ctx)
	if err != nil || string(got) != string(next) {
		t.Fatalf("uncertain publication erased sole intent: %s %v", got, err)
	}
}

func TestDrainStoreRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "private")
	old := []byte(`{"schema_version":1}`)
	if err := os.WriteFile(target, old, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "drain.json")); err != nil {
		t.Fatal(err)
	}
	s := newDrainStore(dir)
	if _, err := s.Read(context.Background()); err == nil {
		t.Fatal("followed symlink")
	}
	if err := s.Write(context.Background(), json.RawMessage(`{"schema_version":1,"changed":true}`)); err == nil {
		t.Fatal("replaced unowned path")
	}
	got, _ := os.ReadFile(target)
	if string(got) != string(old) {
		t.Fatal("changed symlink target")
	}
}

func TestDrainStoreReleasesNamespaceGateResidue(t *testing.T) {
	// Namespace creation must not allocate a permanent per-namespace registry.
	count := func() int { n := 0; drainStoreGates.Range(func(_, _ any) bool { n++; return true }); return n }
	before := count()
	for i := 0; i < 12; i++ {
		s := newDrainStore(filepath.Join(t.TempDir(), "namespace"))
		if _, err := s.Read(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if got := count(); got != before {
		t.Fatalf("namespace gate residue grew %d -> %d", before, got)
	}
}
