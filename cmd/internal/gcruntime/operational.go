package gcruntime

import (
	"context"
	"errors"
	"os"
	"sort"

	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// operationalCursor spans only registered Couch namespaces; run discovery and
// bundle authorization belong to operationalrun, independently of session clocks.
type operationalCursor struct {
	Namespace string                `json:"namespace,omitempty"`
	Run       operationalrun.Cursor `json:"run,omitempty"`
	After     bool                  `json:"after,omitempty"`
}

func (s *Service) operationalPage(ctx context.Context, cursor operationalCursor, apply bool, limit int) ([]operationalrun.Row, operationalCursor, bool, error) {
	registry, e := s.Collector.Coordinator.ReadRegistry()
	if errors.Is(e, os.ErrNotExist) {
		return nil, cursor, true, nil
	}
	if e != nil {
		return nil, cursor, false, e
	}
	stores := append([]string(nil), registry.Stores...)
	sort.Strings(stores)
	i := sort.SearchStrings(stores, cursor.Namespace)
	if cursor.After && i < len(stores) && stores[i] == cursor.Namespace {
		i++
	}
	if i == len(stores) {
		return nil, cursor, true, nil
	}
	store := stores[i]
	runCursor := operationalrun.Cursor{}
	if store == cursor.Namespace && !cursor.After {
		runCursor = cursor.Run
	}
	rows, next, complete, e := (operationalrun.Store{Namespace: store}).SweepPage(ctx, s.Collector.Coordinator.Now(), runCursor, apply, min(max(limit, 1), 100))
	if errors.Is(e, operationalrun.ErrBusy) {
		return rows, cursor, false, storagegc.ErrCoordinatorBusy
	}
	if ctx.Err() != nil {
		return rows, cursor, false, ctx.Err()
	}
	if e != nil {
		rows = append(rows, operationalrun.Row{Path: store, Reason: e.Error()})
		complete = true
	}
	out := operationalCursor{Namespace: store, Run: next, After: complete}
	return rows, out, complete && i+1 == len(stores), nil
}
func (s *Service) operationalRows(ctx context.Context, apply bool, limit int) ([]operationalrun.Row, error) {
	var result []operationalrun.Row
	cursor := operationalCursor{}
	if limit <= 0 {
		limit = 100
	}
	for page := 0; page < 10000; page++ {
		remaining := limit - len(result)
		if remaining <= 0 && apply {
			return result, nil
		}
		batchLimit := 100
		if apply {
			batchLimit = min(batchLimit, remaining)
		}
		rows, next, done, e := s.operationalPage(ctx, cursor, apply, batchLimit)
		result = append(result, rows...)
		if e != nil || done {
			return result, e
		}
		if next == cursor {
			return result, errors.New("operational cursor did not advance")
		}
		cursor = next
	}
	return result, errors.New("operational traversal budget exceeded")
}
func (s *Service) addOperational(ctx context.Context, r *Report, apply bool, limit int) error {
	rows, e := s.operationalRows(ctx, apply, limit)
	r.Operational = append(r.Operational, rows...)
	if apply {
		for _, row := range rows {
			if row.Eligible {
				r.OperationalCollectedBytes += row.Bytes
			}
		}
	}
	return e
}
