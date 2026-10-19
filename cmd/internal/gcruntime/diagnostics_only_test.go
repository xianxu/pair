package gcruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type heldDiagnostic struct{ holders []diagnosticlog.Registration }

func (p *heldDiagnostic) OpenFiles(context.Context, string) ([]diagnosticlog.Registration, error) {
	return p.holders, nil
}
func (*heldDiagnostic) Runtimes(context.Context) ([]diagnosticlog.Runtime, error) { return nil, nil }

func TestDiagnosticsOnlyRevalidatesHoldersAndNeverTouchesSessionClocks(t *testing.T) {
	for _, owners := range []int{8, 16} {
		t.Run(fmt.Sprint(owners), func(t *testing.T) {
			root := t.TempDir()
			probe := &heldDiagnostic{}
			s, err := NewWithInspection(root, probe)
			if err != nil {
				t.Fatal(err)
			}
			c := s.Collector.Coordinator
			for n := range owners {
				if err := os.WriteFile(filepath.Join(c.Root, fmt.Sprintf("draft-owner%d.md", n)), []byte("authored"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(c.Root, "wrap-events-owner0.jsonl")
			if err := os.WriteFile(path, []byte("expired diagnostic"), 0600); err != nil {
				t.Fatal(err)
			}
			at := time.Now().Add(-8 * 24 * time.Hour)
			if err := os.Chtimes(path, at, at); err != nil {
				t.Fatal(err)
			}
			probes, persists := 0, 0
			s.Collector.Legacy = func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
				probes++
				return storagegc.ProcessDead, nil
			}
			c.BeforePersist = func() error { persists++; return nil }
			before, err := os.ReadDir(c.Root)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := s.PreviewDiagnostics(context.Background())
			if err != nil || len(preview.Diagnostics) != 1 || !preview.Diagnostics[0].Eligible {
				t.Fatal(preview, err)
			}
			after, err := os.ReadDir(c.Root)
			if err != nil || len(after) != len(before) {
				t.Fatal("preview mutated root", err)
			}
			// An open holder appearing after preview defeats that advisory decision.
			probe.holders = []diagnosticlog.Registration{{PID: 123, Birth: "live"}}
			blocked, err := s.ApplyDiagnostics(context.Background(), 100)
			if err != nil || blocked.DiagnosticCollectedBytes != 0 || len(blocked.Diagnostics) != 1 || blocked.Diagnostics[0].Eligible {
				t.Fatal(blocked, err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatal("held payload changed", err)
			}
			probe.holders = nil
			collected, err := s.ApplyDiagnostics(context.Background(), 100)
			if err != nil || collected.DiagnosticCollectedBytes != int64(len("expired diagnostic")) {
				t.Fatal(collected, err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("expired payload remains", err)
			}
			if probes != 0 || persists != 0 {
				t.Fatalf("diagnostic scope evaluated sessions: probes=%d persists=%d", probes, persists)
			}
			for n := range owners {
				if b, err := os.ReadFile(filepath.Join(c.Root, fmt.Sprintf("draft-owner%d.md", n))); err != nil || string(b) != "authored" {
					t.Fatal("authored data changed", err)
				}
			}
			if _, err := os.Stat(filepath.Join(c.Root, ".retention", "stores.json")); !os.IsNotExist(err) {
				t.Fatal("migration state changed", err)
			}
		})
	}
}
