package gcruntime

import (
	"context"
	"encoding/json"
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
