package gcruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestScheduledDiagnosticsAdvancePastMissingDeclaration(t *testing.T) {
	s, err := NewWithInspection(t.TempDir(), &heldDiagnostic{})
	if err != nil {
		t.Fatal(err)
	}
	root := s.Collector.Coordinator.Root
	missingDir := filepath.Join(root, "a-gone")
	if err := os.Mkdir(missingDir, 0700); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(missingDir, "wrap-events-missing.jsonl")
	writer, err := diagnosticlog.Open(missing, s.DiagnosticOptions)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(missingDir); err != nil {
		t.Fatal(err)
	}
	registry := diagnosticlog.RegistryDirectory(root)
	before, err := os.ReadDir(registry)
	if err != nil {
		t.Fatalf("registry %v: %v", before, err)
	}
	declaration := ""
	for _, entry := range before {
		if strings.HasSuffix(entry.Name(), ".json") {
			declaration = filepath.Join(registry, entry.Name())
		}
	}
	if declaration == "" {
		t.Fatal("missing registered declaration")
	}
	declarationBytes, err := os.ReadFile(declaration)
	if err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(root, "wrap-events-z-good.jsonl")
	if err := os.WriteFile(good, []byte("expired diagnostic"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(good, old, old); err != nil {
		t.Fatal(err)
	}
	scheduler := storagegc.Scheduler{Coordinator: s.Collector.Coordinator, Batch: s.Batch, Limit: 1}
	// Session phase, missing declaration, eligible path: each is a bounded pass.
	for attempt := 0; attempt < 8; attempt++ {
		ran, err := scheduler.RunOnce(context.Background())
		if err != nil {
			t.Fatalf("batch %d stopped at stale declaration: %v", attempt, err)
		}
		if !ran {
			break
		}
	}
	if _, err := os.Stat(good); !os.IsNotExist(err) {
		t.Fatalf("later eligible diagnostic remains: %v", err)
	}
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Fatalf("stale parent recreated: %v", err)
	}
	afterBytes, err := os.ReadFile(declaration)
	if err != nil || string(afterBytes) != string(declarationBytes) {
		t.Fatalf("stale declaration mutated: %v", err)
	}
	preview, err := s.PreviewDiagnostics(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range preview.Diagnostics {
		if row.Path == missing && !row.Eligible && row.Reason != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("stale blocker disappeared: %+v", preview.Diagnostics)
	}
	var state struct {
		Cursor    string    `json:"cursor"`
		Completed time.Time `json:"completed"`
	}
	raw, err := os.ReadFile(filepath.Join(root, ".retention", "schedule.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	if state.Cursor != "" || state.Completed.IsZero() {
		t.Fatalf("scheduler did not finish: %+v", state)
	}
}

func TestScheduledDiagnosticsMalformedMissingDeclarationFailsClosed(t *testing.T) {
	s, err := NewWithInspection(t.TempDir(), &heldDiagnostic{})
	if err != nil {
		t.Fatal(err)
	}
	root := s.Collector.Coordinator.Root
	dir := filepath.Join(root, "a-gone")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	writer, err := diagnosticlog.Open(filepath.Join(dir, "wrap-events-missing.jsonl"), s.DiagnosticOptions)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	registry := diagnosticlog.RegistryDirectory(root)
	entries, err := os.ReadDir(registry)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(registry, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var declaration map[string]any
		if err := json.Unmarshal(raw, &declaration); err != nil {
			t.Fatal(err)
		}
		declaration["version"] = 99
		raw, err = json.Marshal(declaration)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	good := filepath.Join(root, "wrap-events-z-good.jsonl")
	if err := os.WriteFile(good, []byte("keep on malformed inventory"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(good, old, old); err != nil {
		t.Fatal(err)
	}
	cursor, done, err := s.Batch(context.Background(), `{"phase":"diagnostics"}`, 1)
	if err == nil || done || cursor != "" {
		t.Fatalf("malformed declaration accepted: %q %v %v", cursor, done, err)
	}
	if raw, err := os.ReadFile(good); err != nil || string(raw) != "keep on malformed inventory" {
		t.Fatalf("inventory failure mutated later path: %q %v", raw, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("malformed stale parent recreated: %v", err)
	}
}

// Each failure belongs to one log, not to the sweep's ownership inventory.
// Exercise the durable scheduler, since preserving its old cursor would strand
// the later eligible path on every subsequent runtime restart.
func TestScheduledDiagnosticsIsolateLocalBlockers(t *testing.T) {
	for _, scenario := range []string{"changed-certificate", "malformed-state", "held-recovery", "storage-fault"} {
		t.Run(scenario, func(t *testing.T) {
			probe := &scheduledPathInspection{holders: map[string][]diagnosticlog.Registration{}}
			s, err := NewWithInspection(t.TempDir(), probe)
			if err != nil {
				t.Fatal(err)
			}
			root := s.Collector.Coordinator.Root
			blocked := filepath.Join(root, "wrap-events-a-blocked.jsonl")
			opts := s.DiagnosticOptions
			opts.Now = func() time.Time { return time.Now().Add(-8 * 24 * time.Hour) }
			writer, err := diagnosticlog.Open(blocked, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write([]byte("blocked payload")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			stopped := errors.New("injected local storage fault")
			statePath := blocked + ".pair-diagnostics/state.json"
			switch scenario {
			case "changed-certificate":
				fd, err := os.OpenFile(blocked, os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := fd.WriteString("external write"); err != nil {
					t.Fatal(err)
				}
				if err := fd.Close(); err != nil {
					t.Fatal(err)
				}
			case "malformed-state":
				if err := os.WriteFile(statePath, []byte("{invalid"), 0600); err != nil {
					t.Fatal(err)
				}
			case "held-recovery":
				interrupted := s.DiagnosticOptions
				interrupted.Fault = func(step string) error {
					if step == "delete-intent" {
						return stopped
					}
					return nil
				}
				if _, err := diagnosticlog.Collect(blocked, interrupted, 1); !errors.Is(err, stopped) {
					t.Fatalf("seed deletion intent: %v", err)
				}
				probe.holders[blocked] = []diagnosticlog.Registration{{PID: 123, Birth: "held-target"}}
			case "storage-fault":
				first := true
				s.DiagnosticOptions.Fault = func(step string) error {
					if step == "delete-intent" && first {
						first = false
						return stopped
					}
					return nil
				}
			}
			payload, err := os.ReadFile(blocked)
			if err != nil {
				t.Fatal(err)
			}
			state, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			good := filepath.Join(root, "wrap-events-z-good.jsonl")
			if err := os.WriteFile(good, []byte("old diagnostic"), 0600); err != nil {
				t.Fatal(err)
			}
			old := time.Now().Add(-8 * 24 * time.Hour)
			if err := os.Chtimes(good, old, old); err != nil {
				t.Fatal(err)
			}
			scheduler := storagegc.Scheduler{Coordinator: s.Collector.Coordinator, Batch: s.Batch, Limit: 1}
			for n := 0; n < 8; n++ {
				ran, err := scheduler.RunOnce(context.Background())
				if err != nil {
					t.Fatalf("local blocker terminated scheduler: %v", err)
				}
				if !ran {
					break
				}
			}
			if _, err := os.Stat(good); !os.IsNotExist(err) {
				t.Fatalf("later eligible path stranded: %v", err)
			}
			if after, err := os.ReadFile(blocked); err != nil || string(after) != string(payload) {
				t.Fatalf("blocked payload changed: %q %v", after, err)
			}
			// Storage-fault intentionally publishes a durable recovery intent before
			// failing. All other blocked certificates must stay byte-for-byte intact.
			if scenario != "storage-fault" {
				if after, err := os.ReadFile(statePath); err != nil || string(after) != string(state) {
					t.Fatalf("blocked certificate changed: %v", err)
				}
			}
			preview, err := s.PreviewDiagnostics(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			visible := false
			for _, row := range preview.Diagnostics {
				if row.Path == blocked && !row.Eligible && row.Reason != "" {
					visible = true
				}
			}
			if !visible {
				t.Fatalf("blocker disappeared from preview: %+v", preview.Diagnostics)
			}
		})
	}
}

type scheduledPathInspection struct {
	holders             map[string][]diagnosticlog.Registration
	fileErr, runtimeErr error
}

func (p *scheduledPathInspection) OpenFiles(_ context.Context, path string) ([]diagnosticlog.Registration, error) {
	return p.holders[path], p.fileErr
}
func (p *scheduledPathInspection) Runtimes(context.Context) ([]diagnosticlog.Runtime, error) {
	return nil, p.runtimeErr
}

func TestScheduledDiagnosticsIncompleteInspectionRetainsCursor(t *testing.T) {
	for _, stage := range []string{"open-files", "runtimes", "canceled", "deadline"} {
		t.Run(stage, func(t *testing.T) {
			failed := errors.New("native inventory incomplete")
			if stage == "canceled" {
				failed = context.Canceled
			}
			if stage == "deadline" {
				failed = context.DeadlineExceeded
			}
			probe := &scheduledPathInspection{holders: map[string][]diagnosticlog.Registration{}}
			if stage != "runtimes" {
				probe.fileErr = failed
			} else {
				probe.runtimeErr = failed
			}
			s, err := NewWithInspection(t.TempDir(), probe)
			if err != nil {
				t.Fatal(err)
			}
			root := s.Collector.Coordinator.Root
			path := filepath.Join(root, "wrap-events-old.jsonl")
			if err := os.WriteFile(path, []byte("retain without complete evidence"), 0600); err != nil {
				t.Fatal(err)
			}
			old := time.Now().Add(-8 * 24 * time.Hour)
			if err := os.Chtimes(path, old, old); err != nil {
				t.Fatal(err)
			}
			scheduler := storagegc.Scheduler{Coordinator: s.Collector.Coordinator, Batch: s.Batch, Limit: 1}
			if ran, err := scheduler.RunOnce(context.Background()); err != nil || !ran {
				t.Fatal(ran, err)
			}
			statePath := filepath.Join(root, ".retention", "schedule.json")
			before, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			ran, err := scheduler.RunOnce(context.Background())
			if stage == "deadline" {
				if !ran || err != nil {
					t.Fatalf("deadline must yield: ran=%v err=%v", ran, err)
				}
			} else if !ran || !errors.Is(err, diagnosticlog.ErrInspectionUnavailable) || !errors.Is(err, failed) {
				t.Fatalf("incomplete inspection treated as local skip: ran=%v err=%v", ran, err)
			}
			after, err := os.ReadFile(statePath)
			if err != nil || string(after) != string(before) {
				t.Fatalf("global failure advanced cursor: %v", err)
			}
			if raw, err := os.ReadFile(path); err != nil || string(raw) != "retain without complete evidence" {
				t.Fatalf("incomplete inspection mutated payload: %q %v", raw, err)
			}
		})
	}
}
