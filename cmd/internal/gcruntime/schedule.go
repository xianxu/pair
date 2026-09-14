package gcruntime

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// maintenanceCursor resumes one bounded collection phase. A diagnostic path
// cursor is lexical, so deleting a completed path cannot shift the next page.
type maintenanceCursor struct {
	Phase   string `json:"phase"`
	Round   uint64 `json:"round,omitempty"`
	Path    string `json:"path,omitempty"`
	Segment string `json:"segment,omitempty"`
	After   bool   `json:"after,omitempty"`
}

func nextMaintenance(state maintenanceCursor) (string, bool, error) {
	raw, err := json.Marshal(state)
	return string(raw), false, err
}

// Batch is the scheduler's bounded work unit. Session detachment and diagnostic
// pages acquire their own root lock; the scheduler never nests that lock.
func (s *Service) Batch(ctx context.Context, cursor string, limit int) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if limit < 1 || limit > 10000 {
		return "", false, errors.New("invalid maintenance batch budget")
	}
	state := maintenanceCursor{Phase: "sessions"}
	if cursor != "" {
		if err := json.Unmarshal([]byte(cursor), &state); err != nil {
			return "", false, err
		}
	}
	if state.Phase != "sessions" && state.Phase != "diagnostics" {
		return "", false, errors.New("invalid maintenance cursor phase")
	}
	entries, err := s.prepare()
	if err != nil {
		return "", false, err
	}
	if state.Phase == "sessions" {
		report, err := s.Collector.Apply(ctx, limit)
		if err != nil {
			return "", false, err
		}
		if !report.MigrationComplete {
			return "", true, nil
		}
		if report.Collected >= limit {
			state.Round++
			return nextMaintenance(state)
		}
		return nextMaintenance(maintenanceCursor{Phase: "diagnostics"})
	}
	report, err := s.Collector.Preview(ctx)
	if err != nil {
		return "", false, err
	}
	if !report.MigrationComplete {
		return "", true, nil
	}
	paths := diagnosticPaths(&report, entries)
	index := sort.SearchStrings(paths, state.Path)
	if state.After && index < len(paths) && paths[index] == state.Path {
		index++
	}
	if index == len(paths) {
		return "", true, nil
	}
	path := paths[index]
	segment := ""
	if path == state.Path && !state.After {
		segment = state.Segment
	}
	var next string
	var complete bool
	err = s.Collector.Coordinator.WithLock(ctx, func(held *storagegc.Locked) error {
		registry, err := s.Collector.Coordinator.ReadRegistry()
		if err != nil {
			return err
		}
		if !registry.MigrationComplete {
			return errors.New("migration acknowledgement unavailable")
		}
		_, next, complete, err = diagnosticlog.CollectLegacyPage(path, s.DiagnosticOptions, segment, min(limit, 100))
		return err
	})
	if err != nil {
		return "", false, err
	}
	if complete && index+1 == len(paths) {
		return "", true, nil
	}
	return nextMaintenance(maintenanceCursor{Phase: "diagnostics", Path: path, Segment: next, After: complete})
}

// Worker includes initialization in the background runtime. Stop and Wait join
// all scheduler effects before its owning console or wrapper releases its lease.
type Worker struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func Start(ctx context.Context, selectedRoot string) *Worker {
	if selectedRoot == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	w := &Worker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(w.done)
		if ctx.Err() != nil {
			return
		}
		root, err := RootFromSelected(selectedRoot)
		if err != nil {
			w.err = err
			return
		}
		service, err := New(root)
		if err != nil {
			w.err = err
			return
		}
		scheduler := storagegc.Scheduler{Coordinator: service.Collector.Coordinator, Batch: service.Batch}
		worker := scheduler.Start(ctx)
		w.err = worker.Wait()
	}()
	return w
}
func (w *Worker) Stop() {
	if w != nil {
		w.cancel()
	}
}
func (w *Worker) Wait() error {
	if w == nil {
		return nil
	}
	<-w.done
	return w.err
}

// StartAfterReady is shared by both real readiness publishers: failed startup
// performs no maintenance, and successful startup only starts a goroutine.
func StartAfterReady(ctx context.Context, root string, ready func() error) (*Worker, error) {
	if err := ready(); err != nil {
		return nil, err
	}
	return Start(ctx, root), nil
}
