package gcruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/crashreport"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// Couch crash files sit inside a registered store, which the inventory walk
// excludes, so pair gc reaches them through its own sweep (#397).
func TestCrashFilesInRegisteredStoresAwaitBackup(t *testing.T) {
	ctx := context.Background()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := s.Collector.Coordinator
	s.Collector.LegacyRoot = func(context.Context, []storagegc.ProcessIdentity) (storagegc.Liveness, error) {
		return storagegc.ProcessDead, nil
	}
	store, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(ctx, store); err != nil {
		t.Fatal(err)
	}
	dir := crashreport.Dir(store)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dir, "20251001T000000Z-1.crash")
	young := filepath.Join(dir, "20261006T000000Z-2.crash")
	for path, at := range map[string]time.Time{old: time.Now().Add(-(diagnosticlog.RetentionPeriod + 24*time.Hour)), young: time.Now()} {
		if err := os.WriteFile(path, []byte("panic: x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}

	preview, err := s.Preview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	eligible := map[string]bool{}
	for _, row := range preview.Diagnostics {
		eligible[row.Path] = row.Eligible
		if row.Reason != diagnosticlog.BackupFenceReason {
			t.Fatalf("crash backup fence not reported: %+v", row)
		}
	}
	if len(eligible) != 2 || eligible[old] || eligible[young] {
		t.Fatalf("preview diagnostics %+v", preview.Diagnostics)
	}
	if err := c.CompleteMigration(ctx, []string{store}); err != nil {
		t.Fatal(err)
	}
	applied, err := s.Apply(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if applied.DiagnosticCollectedBytes != 0 {
		t.Fatalf("applied %+v", applied)
	}
	if body, err := os.ReadFile(old); err != nil || string(body) != "panic: x\n" {
		t.Fatalf("backup-fenced crash changed: %q %v", body, err)
	}
	if _, err := os.Stat(young); err != nil {
		t.Fatalf("young crash file removed: %v", err)
	}
}
