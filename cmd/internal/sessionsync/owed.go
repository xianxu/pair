package sessionsync

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SubmitPrefix opens the message parley.nvim sends on the operator's submit
// (parley.nvim#313): `submitted: <absolute path>`.
const SubmitPrefix = "submitted: "

// Kind is why an event is owed.
type Kind string

const (
	KindSubmit   Kind = "submit"
	KindDelivery Kind = "delivery"
)

// OwedEvent is a delivery to the file's owner that no write or release has
// folded yet. The JSON shape is agreed with parley.nvim#314.
type OwedEvent struct {
	ID        string
	Kind      Kind
	From      string
	At        time.Time
	Uncertain bool `json:",omitempty"`
}

// SubmitPath returns the path a submit's head names, or ok=false when the
// head is not a submit.
func SubmitPath(head string) (string, bool) {
	p, ok := strings.CutPrefix(head, SubmitPrefix)
	p = strings.TrimSpace(p)
	return p, ok && p != ""
}

// Owed lists the events the owner still owes file: ledger rows from Since on,
// not folded, deduplicated by ID, minus submits of other files. Every plain
// delivery is owed by each session-sync file of that owner, and each folds it
// for itself. canonical maps a path to its comparable form (symlinks
// resolved); the caller passes it so this stays pure.
func Owed(rows []Entry, st State, file string, canonical func(string) string) []OwedEvent {
	folded := make(map[string]bool, len(st.Folded))
	for _, id := range st.Folded {
		folded[id] = true
	}
	self := canonical(file)
	seen := map[string]bool{}
	out := []OwedEvent{}
	for _, r := range rows {
		if r.At.Before(st.Since) || folded[r.ID] || seen[r.ID] {
			continue
		}
		kind := KindDelivery
		// A submit names an absolute path; a malformed one stays owed as a
		// plain delivery rather than vanishing.
		if p, ok := SubmitPath(r.Head); ok && filepath.IsAbs(p) {
			if canonical(p) != self {
				continue
			}
			kind = KindSubmit
		}
		seen[r.ID] = true
		out = append(out, OwedEvent{ID: r.ID, Kind: kind, From: r.From, At: r.At, Uncertain: r.Uncertain})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}
