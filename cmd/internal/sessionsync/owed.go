package sessionsync

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// SubmitPrefix opens the message parley.nvim sends on the operator's submit
// (parley.nvim#313): `submitted: <absolute path>`.
const SubmitPrefix = "submitted: "

// PingPrefix opens the reminder parley.nvim sends the owner slot when a file
// has been owed too long (parley.nvim#314): `session-sync ping: <path> …`.
const PingPrefix = "session-sync ping: "

// Kind is why an event is owed.
type Kind string

const (
	KindSubmit   Kind = "submit"
	KindDelivery Kind = "delivery"
)

// OwedEvent is a delivery to the file's owner that no write has
// folded yet. The JSON shape is agreed with parley.nvim#314.
type OwedEvent struct {
	ID           string
	Kind         Kind
	From         string
	At           time.Time
	Uncertain    bool                      `json:",omitempty"`
	Origin       couchmessage.Origin       `json:",omitempty"`
	RemoteEdited bool                      `json:",omitempty"`
	RemoteSource couchmessage.RemoteSource `json:",omitzero"`
	DocumentID   string                    `json:",omitempty"`
}

// SubmitPath returns the path a submit's head names, or ok=false when the
// head is not a submit.
func SubmitPath(head string) (string, bool) {
	p, ok := strings.CutPrefix(head, SubmitPrefix)
	p = strings.TrimSpace(p)
	return p, ok && p != ""
}

// Owed lists the events the owner still owes a session-sync file: ledger rows
// from Since on, not folded, deduplicated by ID, minus submits of other files.
// canonical maps a path to its comparable form (symlinks resolved); the caller
// passes it so this stays pure.
func Owed(rows []Entry, st State, file string, canonical func(string) string) []OwedEvent {
	return OwedForDocument(rows, st, file, st.SubmitPaths, true, canonical)
}

// OwedForDocument uses the historical binding at the event's timestamp; a
// reused old pathname does not inherit the former document's submissions.
// plain says whether this document owes plain deliveries (no document named):
// only the owner's session file does, so a delivery is owed once, not once per
// agentic document of that owner (pair#599). Submits are owed only by the
// document they name.
func OwedForDocument(rows []Entry, st State, file string, paths []PathInterval, plain bool, canonical func(string) string) []OwedEvent {
	return OwedForRegistration(rows, st, file, "", paths, plain, canonical)
}

// OwedForRegistration pins new submit rows to the immutable registration ID.
// Legacy rows retain their historical pathname/time attribution.
func OwedForRegistration(rows []Entry, st State, file, documentID string, paths []PathInterval, plain bool, canonical func(string) string) []OwedEvent {
	folded := make(map[string]bool, len(st.Folded))
	for _, id := range st.Folded {
		folded[id] = true
	}
	self := canonical(file)
	seen := map[string]bool{}
	out := []OwedEvent{}
	for _, r := range rows {
		if r.At.Before(st.Since) || folded[r.ID] || seen[r.ID] || (r.DocumentID != "" && r.DocumentID != documentID) {
			continue
		}
		kind := KindDelivery
		// A submit names an absolute path; a malformed one stays owed as a
		// plain delivery rather than vanishing.
		p, ok := SubmitPath(r.Head)
		if !ok || !filepath.IsAbs(p) {
			if !plain {
				continue
			}
		} else {
			// New rows carry the registration captured at admission. A rename
			// before the broker timestamps the row cannot change that identity.
			matches := r.DocumentID != "" || submitNames(r, p, self, paths, canonical)
			if !matches {
				continue
			}
			kind = KindSubmit
		}
		seen[r.ID] = true
		out = append(out, OwedEvent{ID: r.ID, Kind: kind, From: r.From, At: r.At, Uncertain: r.Uncertain, Origin: r.Origin, RemoteEdited: r.RemoteEdited, RemoteSource: r.RemoteSource, DocumentID: r.DocumentID})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

// submitNames says whether a submit of path p, in row r, names the document
// whose canonical path is self: by its historical bindings at the time the
// submit was addressed when it has any, else by path.
func submitNames(r Entry, p, self string, paths []PathInterval, canonical func(string) string) bool {
	if len(paths) == 0 {
		return canonical(p) == self
	}
	addressedAt := r.AddressedAt
	if addressedAt.IsZero() {
		addressedAt = r.At
	}
	for _, binding := range paths {
		if addressedAt.Before(binding.From) || (binding.Until != nil && !addressedAt.Before(*binding.Until)) {
			continue
		}
		spellingMatches := p == binding.Spelling
		for _, alias := range binding.Spellings {
			if p == alias {
				spellingMatches = true
				break
			}
		}
		if canonical(p) == binding.File || spellingMatches {
			return true
		}
	}
	return false
}

// SubmittedSince says whether any row records a submit of the document after
// at, folded or not (pair#661: a round clears on the next submit).
func SubmittedSince(rows []Entry, file string, paths []PathInterval, at time.Time, canonical func(string) string) bool {
	return SubmittedSinceForRegistration(rows, file, "", paths, at, canonical)
}

// SubmittedSinceForRegistration uses the same immutable attribution as owed.
func SubmittedSinceForRegistration(rows []Entry, file, documentID string, paths []PathInterval, at time.Time, canonical func(string) string) bool {
	self := canonical(file)
	for _, r := range rows {
		if r.DocumentID != "" && r.DocumentID != documentID {
			continue
		}
		p, ok := SubmitPath(r.Head)
		if ok && filepath.IsAbs(p) && r.At.After(at) && (r.DocumentID != "" || submitNames(r, p, self, paths, canonical)) {
			return true
		}
	}
	return false
}
