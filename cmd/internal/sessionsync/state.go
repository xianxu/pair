package sessionsync

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// State is the agent's per-file record, kept beside the file: where the
// owed horizon starts, when the agent last wrote, and what it has folded.
// It is bounded by the ledger: Folded is pruned to IDs still in the ledger
// on every save.
type State struct {
	// SubmitPaths preserves renamed paths when a registration hands authority
	// back to a legacy session-sync file. Registered documents use the registry.
	SubmitPaths []PathInterval `json:",omitempty"`
	Since       time.Time
	LastWrite   *time.Time `json:",omitempty"`
	Folded      []string
	// GapAccepted is the latest ledger gap the agent acknowledged for this
	// file (--accept-gap); a later gap refuses again.
	GapAccepted *time.Time `json:",omitempty"`
}

// LoadState reads the sidecar. Without one, the horizon starts at the
// shadow's mtime (the last write by tl-status.py, the prototype this
// replaces), else at the zero time: everything in the ledger is owed.
func LoadState(p Paths) (State, error) {
	raw, err := os.ReadFile(p.State)
	if err == nil {
		var st State
		if err := json.Unmarshal(raw, &st); err != nil {
			return State{}, err
		}
		return st, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return State{}, err
	}
	if info, err := os.Stat(p.Shadow); err == nil {
		return State{Since: info.ModTime()}, nil
	}
	return State{}, nil
}

// Fold adds ids and prunes folds whose rows have left the ledger.
func (st *State) Fold(ids []string, rows []Entry) {
	live := make(map[string]bool, len(rows))
	for _, r := range rows {
		live[r.ID] = true
	}
	seen := map[string]bool{}
	var kept []string
	for _, id := range append(append([]string{}, st.Folded...), ids...) {
		if live[id] && !seen[id] {
			seen[id] = true
			kept = append(kept, id)
		}
	}
	st.Folded = kept
}

func saveState(p Paths, st State) error {
	return saveStateUsing(p, st, writeAtomic)
}

func saveStateUsing(p Paths, st State, publish func(string, []byte) error) error {
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return publish(p.State, append(raw, '\n'))
}
