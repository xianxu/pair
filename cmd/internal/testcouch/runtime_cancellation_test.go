package testcouch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type constructionContextGate struct {
	context.Context
	check func()
}

func (g constructionContextGate) Err() error { g.check(); return g.Context.Err() }

func TestInstallRuntimeAdmissionCanceledDuringValidationHasNoEffects(t *testing.T) {
	b, p, lookup := runtimeFilesFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	err := installRuntimeWithToolsContext(ctx, b, p, func(name string) (string, error) { calls++; cancel(); return lookup(name) })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled validation installed runtime: %v", err)
	}
	if calls != 1 {
		t.Fatalf("lookups continued after cancellation: %d", calls)
	}
	for _, dir := range []string{p.Mutable["bin"], p.Mutable["config"]} {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("late install in %s: %v %v", dir, entries, err)
		}
	}
}

func TestRuntimeCopyCancellationStopsBeforeNextChunk(t *testing.T) {
	b, p, _ := runtimeFilesFixture(t)
	content := make([]byte, 96<<10)
	if err := os.WriteFile(b.Pair, content, 0700); err != nil {
		t.Fatal(err)
	}
	expected, err := os.Stat(b.Pair)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(p.Mutable["bin"], "pair")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	gate := constructionContextGate{Context: ctx, check: func() {
		if info, err := os.Stat(destination); err == nil && info.Size() > 0 {
			cancel()
		}
	}}
	err = copyRuntimeExecutableContext(gate, b.Pair, destination, expected)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copy ignored chunk cancellation: %v", err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 0 || info.Size() > 32<<10 {
		t.Fatalf("late chunks written after cancellation: %d", info.Size())
	}
	if _, err := os.Stat(filepath.Join(p.Mutable["bin"], "couch")); !os.IsNotExist(err) {
		t.Fatalf("later runtime unexpectedly installed: %v", err)
	}
}
