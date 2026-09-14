package gcruntime

import (
	"context"
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
	s.DiagnosticOptions.Proof = func(string, []diagnosticlog.Registration) error { return nil }
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
