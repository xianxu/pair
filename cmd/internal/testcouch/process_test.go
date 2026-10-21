package testcouch

import (
	"context"
	"errors"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

type processWorld struct {
	rows      []procutil.Process
	err       error
	signals   []int
	snapshots int
}

func (w *processWorld) Table(context.Context) ([]procutil.Process, error) {
	w.snapshots++
	return append([]procutil.Process(nil), w.rows...), w.err
}
func (w *processWorld) Signal(pid int, _ string, _ syscall.Signal) error {
	w.signals = append(w.signals, pid)
	return nil
}

func TestStopOwnedFencesGenerationAndUnknownTable(t *testing.T) {
	for _, kind := range []string{"same", "reused", "absent", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			w := &processWorld{}
			if kind == "same" {
				w.rows = []procutil.Process{{PID: 42, Identity: "old"}}
			}
			if kind == "reused" {
				w.rows = []procutil.Process{{PID: 42, Identity: "new"}}
			}
			if kind == "unknown" {
				w.err = errors.New("table unavailable")
			}
			s := ReduceRun(OwnedRun{}, RunEvent{Kind: "own", Resource: Resource{ID: "42", Generation: "old"}})
			next, err := stopOwned(context.Background(), w, s, syscall.SIGTERM)
			if kind == "same" && len(w.signals) != 1 {
				t.Fatal("owned process not stopped")
			}
			if kind != "same" && len(w.signals) != 0 {
				t.Fatal("signalled foreign or unknown process")
			}
			if kind == "unknown" && (err == nil || len(next.Resources) != 1) {
				t.Fatal("unknown observation released authority")
			}
			if (kind == "reused" || kind == "absent") && len(next.Resources) != 0 {
				t.Fatal("old generation still retained")
			}
			if w.snapshots != 1 {
				t.Fatalf("snapshots=%d", w.snapshots)
			}
		})
	}
}

func TestStopOwnedUsesOneSnapshotForWholeCohort(t *testing.T) {
	for _, n := range []int{8, 16} {
		w := &processWorld{}
		s := OwnedRun{}
		for i := 1; i <= n; i++ {
			p := procutil.Process{PID: i, Identity: "one"}
			w.rows = append(w.rows, p)
			s = ReduceRun(s, RunEvent{Kind: "own", Resource: resourceFor(p)})
		}
		if _, err := stopOwned(context.Background(), w, s, syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		if w.snapshots != 1 || len(w.signals) != n {
			t.Fatalf("n=%d snapshots=%d signals=%d", n, w.snapshots, len(w.signals))
		}
	}
}
