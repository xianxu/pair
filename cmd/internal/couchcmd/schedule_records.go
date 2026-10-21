package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func runSummaryForRecord(r operationalrun.Record) couchschedule.RunSummary {
	return couchschedule.RunSummary{ID: r.Key.ID, Revision: r.DeclarationRevision, Source: r.Source, DueAt: r.ScheduledAt, StartedAt: r.StartedAt, EndedAt: r.EndedAt, Exit: r.Exit, Outcome: r.Outcome, Notice: r.Notice}
}

// ensurePassiveRun is called under the existing state lock before replacing a
// retained summary. Its input facts must already be in the committed ledger.
// Native output cannot be reconstructed; only no-execution outcomes are repaired.
func ensurePassiveRun(ctx context.Context, namespace string, state couchschedule.TaskState) error {
	if state.Last == nil || state.Last.Source != "scheduled" || (state.Last.Outcome != "skipped" && state.Last.Outcome != "refused" && state.Last.Outcome != "completed") {
		return nil
	}
	_, err := ensureRetainedRun(ctx, namespace, state, false)
	return err
}
func ensureRetainedRun(ctx context.Context, namespace string, state couchschedule.TaskState, unknown bool) (operationalrun.Record, error) {
	last := state.Last
	if last == nil {
		return operationalrun.Record{}, errors.New("missing retained run facts")
	}
	history := operationalrun.Store{Namespace: namespace}
	key := operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: last.ID}
	record, _, err := history.Read(ctx, key)
	if err == nil {
		record, _, err = history.PublishOccurrence(ctx, key, record, nil)
		return record, err
	}
	if !errors.Is(err, os.ErrNotExist) {
		return record, err
	}
	passive := last.Source == "scheduled" && (last.Outcome == "skipped" || last.Outcome == "refused" || last.Outcome == "completed")
	if !passive && !(unknown && last.Outcome == "unknown" && state.Active != nil && state.Active.Stage == "uncertain" && state.Active.ID == last.ID) {
		return record, fmt.Errorf("retained execution history unavailable; no facts or output reconstructed: %w", err)
	}
	revision := last.Revision
	if revision == "" && state.Active != nil && state.Active.ID == last.ID {
		revision = state.Active.Revision
	}
	if revision == "" {
		return record, errors.New("retained passive occurrence has no declaration revision; inspect history before repair")
	}
	proposed := operationalrun.Record{Version: 1, Key: key, DeclarationRevision: revision, Source: last.Source, ScheduledAt: last.DueAt, StartedAt: last.StartedAt, EndedAt: last.EndedAt, Exit: last.Exit, Outcome: last.Outcome, Notice: last.Notice}
	record, _, err = history.PublishOccurrence(ctx, key, proposed, nil)
	return record, err
}

// Unknown classification is committed before history publication. This second
// transaction uses those durable facts and adopts a same-ID canonical record;
// it never starts work or admits a notice.
func ensureUncertainRuns(ctx context.Context, store couchschedule.StateStore) error {
	ledger, err := store.View(ctx)
	if err != nil {
		return err
	}
	needed := false
	for _, s := range ledger.Tasks {
		if s.Active != nil && s.Active.Stage == "uncertain" && s.Last != nil && s.Last.Outcome == "unknown" {
			needed = true
			break
		}
	}
	if !needed {
		return nil
	}
	return store.Update(ctx, func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			if s.Active == nil || s.Active.Stage != "uncertain" || s.Last == nil || s.Last.Outcome != "unknown" {
				continue
			}
			record, err := ensureRetainedRun(ctx, filepath.Dir(store.Dir), s, true)
			if err == nil && (record.Source != s.Active.Source || record.DeclarationRevision != s.Active.Revision) {
				err = errors.New("result identity differs from retained authority")
			}
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// The state store is available; damage or contention in one task's
				// history must not turn into a refusal of every configured hub.
				s.Allowed = false
				s.Refusal = fmt.Sprintf("%s result recovery: %v", s.Active.Source, err)
				if len(s.Refusal) > 4096 {
					s.Refusal = s.Refusal[:4096]
				}
				l.Tasks[key] = s
				continue
			}
			if record.Notice == "pending" {
				if err := (operationalrun.Store{Namespace: filepath.Dir(store.Dir)}).FinalizeNotice(ctx, record.Key, record.DeclarationRevision, "uncertain"); err != nil {
					return err
				}
				record.Notice = "uncertain"
			}
			summary := runSummaryForRecord(record)
			next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ResultRecovered, At: record.EndedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, Recovered: &summary})
			if next.Last == nil || next.Last.ID != record.Key.ID {
				return errors.New("canonical uncertain history refused by task state")
			}
			l.Tasks[key] = next
		}
		return nil
	})
}
