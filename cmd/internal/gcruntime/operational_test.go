package gcruntime

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func operationalFixture(t *testing.T) (*Service, operationalrun.Store, operationalrun.Record) {
	t.Helper()
	s, e := NewWithInspection(t.TempDir(), &heldDiagnostic{})
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	ns := t.TempDir()
	if e = s.Collector.Coordinator.RegisterStore(ctx, ns); e != nil {
		t.Fatal(e)
	}
	store := operationalrun.Store{Namespace: ns}
	at := time.Now().Add(-31 * 24 * time.Hour)
	r := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: "hub", Task: "task", ID: "old"}, Outcome: "completed", EndedAt: at}
	if e = store.Publish(ctx, r.Key, r, []byte("output")); e != nil {
		t.Fatal(e)
	}
	return s, store, r
}
func TestOperationalRetentionBeforeSessionMigration(t *testing.T) {
	s, store, r := operationalFixture(t)
	ctx := context.Background()
	report, e := s.Preview(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(report.Operational) != 1 || !report.Operational[0].Eligible {
		t.Fatal(report.Operational)
	}
	if _, _, e = store.Read(ctx, r.Key); e != nil {
		t.Fatal("preview changed run", e)
	}
	report, e = s.Apply(ctx, 100)
	if e != nil {
		t.Fatal(e)
	}
	if report.OperationalCollectedBytes == 0 || len(report.Operational) != 1 {
		t.Fatal(report)
	}
	if _, _, e = store.Read(ctx, r.Key); !os.IsNotExist(e) {
		t.Fatal("expired run survived", e)
	}
}
func TestOperationalBatchAfterEmptyDiagnostics(t *testing.T) {
	s, store, r := operationalFixture(t)
	ctx := context.Background()
	cursor, _ := json.Marshal(maintenanceCursor{Phase: "diagnostics"})
	next, done, e := s.Batch(ctx, string(cursor), 1)
	if e != nil || done || next == "" {
		t.Fatal(next, done, e)
	}
	for i := 0; i < 10 && !done; i++ {
		next, done, e = s.Batch(ctx, next, 1)
		if e != nil {
			t.Fatal(e)
		}
	}
	if !done {
		t.Fatal("operational sweep did not finish")
	}
	if _, e = os.Stat(filepath.Join(store.Namespace, "schedule", r.Key.Hub, r.Key.Task, "runs", r.Key.ID)); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
