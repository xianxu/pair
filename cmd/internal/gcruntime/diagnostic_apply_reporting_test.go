package gcruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

type reportingInspection struct {
	holders map[string][]diagnosticlog.Registration
}

func (p *reportingInspection) OpenFiles(_ context.Context, path string) ([]diagnosticlog.Registration, error) {
	return p.holders[path], nil
}
func (*reportingInspection) Runtimes(context.Context) ([]diagnosticlog.Runtime, error) {
	return nil, nil
}

func reportingFixture(t *testing.T) (*Service, string, *reportingInspection, []diagnosticlog.Segment) {
	t.Helper()
	probe := &reportingInspection{holders: map[string][]diagnosticlog.Registration{}}
	s, err := NewWithInspection(t.TempDir(), probe)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Collector.Coordinator.Root, "wrap-events-report.jsonl")
	now := time.Now().Add(-10 * 24 * time.Hour)
	seed := s.DiagnosticOptions
	seed.Now = func() time.Time { return now }
	seed.SynchronousMaintenance = true
	w, err := diagnosticlog.Open(path, seed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	for i, payload := range []string{"first-old", "second-old", "recent-current"} {
		if i == 1 {
			now = now.Add(2 * 24 * time.Hour)
		}
		if i == 2 {
			now = now.Add(7 * 24 * time.Hour)
		}
		if _, err := w.Write([]byte(payload)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewDiagnostics(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Diagnostics) != 3 {
		t.Fatalf("fixture rows: %+v", preview)
	}
	return s, path, probe, preview.Diagnostics
}

func TestManagedDiagnosticApplyReportsRetainedBlockedAndRemoved(t *testing.T) {
	s, current, probe, before := reportingFixture(t)
	var blocked string
	var removedBytes int64
	for _, row := range before {
		if row.Path != current && blocked == "" {
			blocked = row.Path
			continue
		}
		if row.Eligible {
			removedBytes += row.Bytes
		}
	}
	probe.holders[blocked] = []diagnosticlog.Registration{{PID: 123, Birth: "reader"}}
	report, err := s.ApplyDiagnostics(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 3 || report.DiagnosticCollectedBytes != removedBytes {
		t.Fatalf("incomplete managed report: %+v", report)
	}
	seen := map[string]bool{}
	for _, row := range report.Diagnostics {
		if seen[row.Path] {
			t.Fatalf("duplicate row: %+v", row)
		}
		seen[row.Path] = true
		if row.Path == blocked || row.Path == current {
			if row.Eligible || row.Reason == "" {
				t.Fatalf("retained/blocked row claimed removed: %+v", row)
			}
			if _, err := os.Stat(row.Path); err != nil {
				t.Fatal(err)
			}
		} else {
			if !row.Eligible {
				t.Fatalf("removed row lacks success: %+v", row)
			}
			if _, err := os.Stat(row.Path); !os.IsNotExist(err) {
				t.Fatalf("reported removal survives: %v", err)
			}
		}
	}
}

func TestManagedDiagnosticApplyPreservesCompletedRowsOnLaterFault(t *testing.T) {
	s, current, _, _ := reportingFixture(t)
	injected := errors.New("second deletion admission interrupted")
	intents := 0
	s.DiagnosticOptions.Fault = func(step string) error {
		if step == "delete-intent" {
			intents++
			if intents == 2 {
				return injected
			}
		}
		return nil
	}
	report, err := s.ApplyDiagnostics(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	var removedBytes int64
	removed, retained, failures := 0, 0, 0
	for _, row := range report.Diagnostics {
		if row.Eligible {
			removed++
			removedBytes += row.Bytes
			if _, err := os.Stat(row.Path); !os.IsNotExist(err) {
				t.Fatalf("uncompleted row claimed removed: %+v, %v", row, err)
			}
		} else if row.Path == current && row.Bytes > 0 {
			retained++
		} else if row.Reason == injected.Error() {
			failures++
		}
	}
	if intents != 2 || removed != 1 || retained != 1 || failures != 1 || removedBytes == 0 || report.DiagnosticCollectedBytes != removedBytes {
		t.Fatalf("partial progress lost or overstated: %+v; intents=%d removed=%d retained=%d failures=%d bytes=%d", report, intents, removed, retained, failures, removedBytes)
	}
}
