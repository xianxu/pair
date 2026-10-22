package couchcore

import (
	"sort"
	"time"
)

// DrainObservation contains current observational facts, never readiness or
// authority to hold work. Nil booleans and a zero clock mean unknown.
type DrainObservation struct {
	Connected                   bool
	LastOutputAt                time.Time
	Settled, Picker, Background *bool
}

type DrainRow struct {
	Notice, NoticeID, OperationID, NoticeDetail string
	Workspace                                   string
	Connected                                   bool
	IdleMillis                                  *int64
	Settled, Picker, Background                 *bool
}

type DrainBoard struct{ Rows []DrainRow }

// ProjectDrainBoard returns a detached, stable-order snapshot. Unknown,
// disconnected, active, and future output clocks never become a fabricated
// idle age. All activity facts must positively establish quiescence.
func ProjectDrainBoard(flags DrainFlags, observations map[string]DrainObservation, now time.Time) DrainBoard {
	keys := make([]string, 0, len(flags))
	for workspace, flagged := range flags {
		if flagged {
			keys = append(keys, workspace)
		}
	}
	sort.Strings(keys)
	board := DrainBoard{Rows: make([]DrainRow, 0, len(keys))}
	for _, workspace := range keys {
		o := observations[workspace]
		row := DrainRow{Workspace: workspace, Connected: o.Connected, Settled: copyDrainBool(o.Settled), Picker: copyDrainBool(o.Picker), Background: copyDrainBool(o.Background)}
		if o.Connected && o.Settled != nil && *o.Settled && o.Picker != nil && !*o.Picker && o.Background != nil && !*o.Background && !now.IsZero() && !o.LastOutputAt.IsZero() && !o.LastOutputAt.After(now) {
			millis := now.Sub(o.LastOutputAt).Milliseconds()
			row.IdleMillis = &millis
		}
		board.Rows = append(board.Rows, row)
	}
	return board
}

func copyDrainBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
