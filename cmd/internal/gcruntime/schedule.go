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
	Phase       string            `json:"phase"`
	Operational operationalCursor `json:"operational,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Path        string            `json:"path,omitempty"`
	Segment     string            `json:"segment,omitempty"`
	After       bool              `json:"after,omitempty"`
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
	if state.Phase != "sessions" && state.Phase != "diagnostics" && state.Phase != "operational" {
		return "", false, errors.New("invalid maintenance cursor phase")
	}
	if state.Phase == "operational" {
		_, next, done, err := s.operationalPage(ctx, state.Operational, true, limit)
		if err != nil {
			return "", false, err
		}
		if done {
			return "", true, nil
		}
		return nextMaintenance(maintenanceCursor{Phase: "operational", Operational: next})
	}
	if err := diagnosticlog.RecoverRegistry(ctx, s.Collector.Coordinator.Root); err != nil {
		if errors.Is(err, diagnosticlog.ErrBusy) {
			err = storagegc.ErrCoordinatorBusy
		}
		return "", false, err
	}
	entries, err := s.prepareContext(ctx)
	if err != nil {
		return "", false, err
	}
	if state.Phase == "sessions" {
		report, err := s.Collector.ApplyPage(ctx, state.Owner, limit)
		if err != nil {
			return "", false, err
		}
		if !report.BatchComplete {
			state.Owner = report.NextOwner
			return nextMaintenance(state)
		}
		return nextMaintenance(maintenanceCursor{Phase: "diagnostics"})
	}
	report, err := s.Collector.DiagnosticInventory(ctx)
	if err != nil {
		return "", false, err
	}
	paths := diagnosticPaths(&report, entries)
	index := sort.SearchStrings(paths, state.Path)
	if state.After && index < len(paths) && paths[index] == state.Path {
		index++
	}
	if index == len(paths) {
		return nextMaintenance(maintenanceCursor{Phase: "operational"})
	}
	path := paths[index]
	ctx = diagnosticlog.SweepContext(ctx, []string{path})
	segment := ""
	if path == state.Path && !state.After {
		segment = state.Segment
	}
	var next string
	var complete bool
	err = s.Collector.Coordinator.TryWithLock(ctx, func(held *storagegc.Locked) error {
		options := s.DiagnosticOptions
		options.Context = ctx
		// Advisory row classification may report proof errors as blocked rows
		// rather than return them. Preserve sweep-wide failures across that
		// conversion so an incomplete observation cannot advance the scheduler.
		var proofFailure error
		observeProof := func(err error) error {
			if errors.Is(err, diagnosticlog.ErrInspectionUnavailable) ||
				errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				proofFailure = err
			}
			return err
		}
		if proof := options.Proof; proof != nil {
			options.Proof = func(ctx context.Context, path string, writers []diagnosticlog.Registration) error {
				return observeProof(proof(ctx, path, writers))
			}
		}
		if proof := options.ReaderProof; proof != nil {
			options.ReaderProof = func(ctx context.Context, path string) error {
				return observeProof(proof(ctx, path))
			}
		}
		_, next, complete, err = diagnosticlog.CollectLegacyPage(path, options, segment, min(limit, 100))
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if proofFailure != nil && !errors.Is(err, diagnosticlog.ErrBusy) &&
			!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			return proofFailure
		}
		// Collection errors at this boundary belong to this one log. Keep its
		// declaration and certificates visible to PreviewDiagnostics, but do not
		// let a damaged certificate, held recovery or local filesystem failure
		// strand unrelated logs on every restart. Cancellation, contention and
		// incomplete shared inspection cannot establish a safe sweep and retain
		// the durable cursor for the scheduler's existing retry/error handling.
		if err != nil && ctx.Err() == nil &&
			!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) &&
			!errors.Is(err, diagnosticlog.ErrBusy) &&
			!errors.Is(err, diagnosticlog.ErrInspectionUnavailable) {
			next, complete = "", true
			return nil
		}
		if errors.Is(err, diagnosticlog.ErrBusy) {
			return storagegc.ErrCoordinatorBusy
		}
		return err
	})
	if err != nil {
		return "", false, err
	}
	if complete && index+1 == len(paths) {
		return nextMaintenance(maintenanceCursor{Phase: "operational"})
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
