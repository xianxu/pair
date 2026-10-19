package gcruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type leasedSupervisor struct {
	heldDiagnostic
	process diagnosticlog.Registration
}

func (p *leasedSupervisor) Runtimes(ctx context.Context) ([]diagnosticlog.Runtime, error) {
	return []diagnosticlog.Runtime{{Process: p.process}}, ctx.Err()
}
func (p *leasedSupervisor) Snapshot(ctx context.Context, paths []string) (diagnosticlog.Inspection, error) {
	p.snapshots++
	return &leasedSupervisor{heldDiagnostic: heldDiagnostic{holders: append([]diagnosticlog.Registration(nil), p.holders...)}, process: p.process}, ctx.Err()
}

func TestDiagnosticSupervisorLeaseRecognizesOnlyPublishedCapability(t *testing.T) {
	ctx := context.Background()
	identity, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	probe := &leasedSupervisor{process: diagnosticlog.Registration{PID: identity.PID, Birth: identity.Birth}}
	s, err := NewWithInspection(t.TempDir(), probe)
	if err != nil {
		t.Fatal(err)
	}
	c := s.Collector.Coordinator
	if err := c.RegisterRuntime(ctx, identity, "couch-runtime"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(c.Root, "wrap-events-supervisor.jsonl")
	if err := os.WriteFile(path, []byte("expired"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	assertEligible := func(want bool) {
		t.Helper()
		report, err := s.PreviewDiagnostics(ctx)
		if err != nil || len(report.Diagnostics) != 1 || report.Diagnostics[0].Eligible != want {
			t.Fatalf("eligible=%t report=%+v err=%v", want, report, err)
		}
	}
	assertEligible(false)
	registry := filepath.Join(c.Root, ".retention", "runtimes.json")
	raw, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	wire["processes"].([]any)[0].(map[string]any)["diagnostic_protocol"] = 1
	upgraded, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registry, upgraded, 0600); err != nil {
		t.Fatal(err)
	}
	assertEligible(true)
	// Preview is advisory: a newly opened payload still blocks the apply.
	probe.holders = []diagnosticlog.Registration{probe.process}
	held, err := s.ApplyDiagnostics(ctx, 10)
	if err != nil || held.DiagnosticCollectedBytes != 0 {
		t.Fatal(held, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("held diagnostic changed", err)
	}
	probe.holders = nil
	if err := os.WriteFile(registry, raw, 0600); err != nil {
		t.Fatal(err)
	}
	assertEligible(false)
	if err := os.WriteFile(registry, upgraded, 0600); err != nil {
		t.Fatal(err)
	}
	applied, err := s.ApplyDiagnostics(ctx, 10)
	if err != nil || applied.DiagnosticCollectedBytes != 7 {
		t.Fatal(applied, err)
	}
}

func TestScheduledDiagnosticsUseSupervisorLeaseAndRetainUncertainty(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprint(corrupt), func(t *testing.T) {
			identity, err := storagegc.CurrentProcessIdentity(os.Getpid())
			if err != nil {
				t.Fatal(err)
			}
			probe := &leasedSupervisor{process: diagnosticlog.Registration{PID: identity.PID, Birth: identity.Birth}}
			s, err := NewWithInspection(t.TempDir(), probe)
			if err != nil {
				t.Fatal(err)
			}
			c := s.Collector.Coordinator
			if err := c.RegisterRuntime(context.Background(), identity, storagegc.CouchRuntimeRole, storagegc.RuntimeRegistrationOptions{DiagnosticProtocol: storagegc.DiagnosticProtocolVersion}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(c.Root, "wrap-events-expired.jsonl")
			if err := os.WriteFile(path, []byte("expired"), 0600); err != nil {
				t.Fatal(err)
			}
			old := time.Now().Add(-8 * 24 * time.Hour)
			if err := os.Chtimes(path, old, old); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"draft-authored.md", "history-authored.jsonl", "wrap-events-recent.jsonl"} {
				if err := os.WriteFile(filepath.Join(c.Root, name), []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if corrupt {
				if err := os.WriteFile(filepath.Join(c.Root, ".retention", "runtimes.json"), []byte(`{`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			scheduler := storagegc.Scheduler{Coordinator: c, Batch: s.Batch, Limit: 1}
			observedUnavailable := false
			for range 12 {
				ran, err := scheduler.RunOnce(context.Background())
				if err != nil {
					if !corrupt || !errors.Is(err, diagnosticlog.ErrInspectionUnavailable) {
						t.Fatal(err)
					}
					observedUnavailable = true
					break
				}
				if !ran {
					break
				}
			}
			if corrupt {
				if !observedUnavailable {
					t.Fatal("scheduler advanced past incomplete lease inventory")
				}
				if _, err := os.Stat(path); err != nil {
					t.Fatal("uncertain payload changed", err)
				}
			} else if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("eligible diagnostic remains", err)
			}
			for _, name := range []string{"draft-authored.md", "history-authored.jsonl", "wrap-events-recent.jsonl"} {
				if raw, err := os.ReadFile(filepath.Join(c.Root, name)); err != nil || string(raw) != "retained" {
					t.Fatal(name, err)
				}
			}
		})
	}
}
