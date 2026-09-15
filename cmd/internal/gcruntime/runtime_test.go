package gcruntime

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyDiagnosticsExpireWithoutSessionGrace(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	c := s.Collector.Coordinator
	s.Collector.LegacyRoot = func(context.Context, []storagegc.ProcessIdentity) (storagegc.Liveness, error) {
		return storagegc.ProcessDead, nil
	}
	s.DiagnosticOptions.Proof = func(context.Context, string, []diagnosticlog.Registration) error { return nil }
	path := filepath.Join(c.Root, "wrap-events-tag.jsonl")
	os.WriteFile(path, []byte("old diagnostic"), 0600)
	at := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(path, at, at)
	preview, err := s.Preview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Diagnostics) != 1 || !preview.Diagnostics[0].Eligible {
		t.Fatalf("%+v", preview)
	}
	if _, err := os.Stat(filepath.Join(c.Root, ".retention")); !os.IsNotExist(err) {
		t.Fatal("preview wrote metadata")
	}
	if err := c.CompleteMigration(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	applied, err := s.Apply(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if applied.DiagnosticCollectedBytes != 14 {
		t.Fatalf("%+v", applied)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("old diagnostic retained", err)
	}
}

// Explicit apply scans past retained owners even when its deletion limit is one.
// Automatic pages use a separate visited-owner cursor.
func TestExplicitApplyDoesNotStarveBehindRetainedOwner(t *testing.T) {
	s, oldOwner := scheduleFixture(t)
	ctx := context.Background()
	c := s.Collector.Coordinator
	if err := c.Initialize(ctx, oldOwner); err != nil {
		t.Fatal(err)
	}
	old := c.Now()
	c.Now = func() time.Time { return old.Add(61 * 24 * time.Hour) }
	young, err := artifactpath.NewStorageOwner(c.Root, "", "aaa-young")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(c.Root, "draft-aaa-young.md")
	if err := os.WriteFile(path, []byte("young"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.Initialize(ctx, young); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteMigration(ctx, nil); err != nil {
		t.Fatal(err)
	}
	report, err := s.Apply(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if report.Storage.Collected != 1 {
		t.Fatalf("old owner starved: %+v", report.Storage)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("young payload removed", err)
	}
	if _, err := os.Stat(filepath.Join(c.Root, "draft-tag.md")); !os.IsNotExist(err) {
		t.Fatal("old payload retained", err)
	}
}
