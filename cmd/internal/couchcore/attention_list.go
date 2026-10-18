package couchcore

import (
	"sort"
	"time"
)

// The attention list (pair#673) is what waits on the operator: the hub's
// notifications, slots blocked on a prompt, and slots that asked a question.
// AttentionList is its one owner; sources report facts and never hold state.

type AttentionKind string

const (
	// AttentionNotify is a hub `couch --notify`. Selecting it reads it.
	AttentionNotify AttentionKind = "notify"
	// AttentionPrompt is a slot whose wrapper reports an input-holding picker.
	AttentionPrompt AttentionKind = "prompt"
	// AttentionQuestion is a slot whose turn ended with a question to the
	// operator (the wait watcher's question class).
	AttentionQuestion AttentionKind = "question"
)

// AttentionItem is one row. A condition item (prompt, question) is keyed by
// kind and thread, so a slot has at most one of each; a notify item by its ID.
type AttentionItem struct {
	ID      string        `json:"id"`
	Kind    AttentionKind `json:"kind"`
	Thread  ThreadAddress `json:"thread"`
	Label   string        `json:"label"`
	Title   string        `json:"title,omitempty"`
	Body    string        `json:"body,omitempty"`
	Since   time.Time     `json:"since"`
	Visited bool          `json:"visited,omitempty"`
}

// AttentionCondition is one slot in a condition set: its label and what it
// waits on.
type AttentionCondition struct {
	Label, Title string
}

const (
	// AttentionCap bounds the list; the oldest notify item goes first, and a
	// condition item never goes while its condition holds.
	AttentionCap = 50
	// AttentionRestoreGrace is how long a condition item restored from disk
	// waits for its live condition to re-raise it with its original time.
	AttentionRestoreGrace = 2 * time.Minute
)

// AttentionList is a value: every transition returns a new list.
type AttentionList struct {
	items []AttentionItem
	// restored holds condition items loaded at restart until their live
	// condition re-raises them (keeping Since and Visited) or the grace ends.
	restored map[string]restoredAttention
}

type restoredAttention struct {
	item  AttentionItem
	until time.Time
}

func conditionID(kind AttentionKind, thread ThreadAddress) string {
	return string(kind) + ":" + thread.RepoScope + ":" + string(thread.Tag)
}

// RestoreAttention starts from a saved list. Notify items are kept as they
// were; condition items wait, unseen, for live state to confirm them, since
// a prompt answered while Couch was down no longer waits on anyone.
func RestoreAttention(saved []AttentionItem, now time.Time) AttentionList {
	var l AttentionList
	for _, item := range saved {
		switch item.Kind {
		case AttentionNotify:
			l.items = append(l.items, item)
		case AttentionPrompt, AttentionQuestion:
			if l.restored == nil {
				l.restored = map[string]restoredAttention{}
			}
			l.restored[conditionID(item.Kind, item.Thread)] = restoredAttention{item: item, until: now.Add(AttentionRestoreGrace)}
		}
	}
	return l.capped()
}

// Items is the list newest first.
func (l AttentionList) Items() []AttentionItem {
	out := append([]AttentionItem(nil), l.items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Since.After(out[j].Since) })
	return out
}

// Saved is what persists at now: the shown items and any restored condition
// items still inside their grace, so a second restart inside it loses nothing.
func (l AttentionList) Saved(now time.Time) []AttentionItem {
	out := l.Items()
	for _, r := range l.restored {
		if now.Before(r.until) {
			out = append(out, r.item)
		}
	}
	return out
}

// AttentionFact is what a source reports: one hub notification, or the whole
// current condition set of one kind.
type AttentionFact struct {
	Kind   AttentionKind
	Notify AttentionItem
	Live   map[ThreadAddress]AttentionCondition
}

// Apply folds one fact into the list.
func (l AttentionList) Apply(f AttentionFact, now time.Time) AttentionList {
	if f.Kind == AttentionNotify {
		if f.Notify.Since.IsZero() {
			f.Notify.Since = now
		}
		return l.Notify(f.Notify)
	}
	return l.Conditions(f.Kind, f.Live, now)
}

// Len is the number of items shown.
func (l AttentionList) Len() int { return len(l.items) }

// Has reports whether any shown item is of kind.
func (l AttentionList) Has(kind AttentionKind) bool {
	for _, item := range l.items {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func (l AttentionList) clone() AttentionList {
	out := AttentionList{items: append([]AttentionItem(nil), l.items...)}
	if len(l.restored) > 0 {
		out.restored = make(map[string]restoredAttention, len(l.restored))
		for k, v := range l.restored {
			out.restored[k] = v
		}
	}
	return out
}

// Notify adds a hub notification.
func (l AttentionList) Notify(item AttentionItem) AttentionList {
	out := l.clone()
	item.Kind = AttentionNotify
	out.items = append(out.items, item)
	return out.capped()
}

// Conditions replaces kind's condition set: a slot new to it raises an item,
// a slot already listed keeps its item (and time), and a listed slot missing
// from it clears, whoever answered it.
func (l AttentionList) Conditions(kind AttentionKind, live map[ThreadAddress]AttentionCondition, now time.Time) AttentionList {
	out := l.clone()
	kept := out.items[:0]
	have := map[string]bool{}
	for _, item := range out.items {
		if item.Kind == kind {
			c, ok := live[item.Thread]
			if !ok {
				continue
			}
			item.Label, item.Title = c.Label, c.Title
			have[item.ID] = true
		}
		kept = append(kept, item)
	}
	out.items = kept
	for thread, c := range live {
		id := conditionID(kind, thread)
		if have[id] {
			continue
		}
		item := AttentionItem{ID: id, Kind: kind, Thread: thread, Label: c.Label, Title: c.Title, Since: now}
		if r, ok := out.restored[id]; ok && now.Before(r.until) {
			item.Since, item.Visited = r.item.Since, r.item.Visited
		}
		delete(out.restored, id)
		out.items = append(out.items, item)
	}
	for id, r := range out.restored {
		if !now.Before(r.until) {
			delete(out.restored, id)
		}
	}
	return out.capped()
}

// Select is the operator picking an item: it names the item to jump to. A
// notify item is read by being picked and goes; a condition item stays,
// visited, until its condition clears.
func (l AttentionList) Select(id string) (AttentionList, AttentionItem, bool) {
	out := l.clone()
	for i, item := range out.items {
		if item.ID != id {
			continue
		}
		if item.Kind == AttentionNotify {
			out.items = append(out.items[:i], out.items[i+1:]...)
		} else {
			out.items[i].Visited = true
		}
		return out, item, true
	}
	return l, AttentionItem{}, false
}

// Dismiss removes an item of any kind.
func (l AttentionList) Dismiss(id string) AttentionList {
	out := l.clone()
	for i, item := range out.items {
		if item.ID == id {
			out.items = append(out.items[:i], out.items[i+1:]...)
			break
		}
	}
	return out
}

// capped drops the oldest notify items past AttentionCap.
func (l AttentionList) capped() AttentionList {
	for len(l.items) > AttentionCap {
		oldest := -1
		for i, item := range l.items {
			if item.Kind == AttentionNotify && (oldest < 0 || item.Since.Before(l.items[oldest].Since)) {
				oldest = i
			}
		}
		if oldest < 0 {
			break
		}
		l.items = append(l.items[:oldest], l.items[oldest+1:]...)
	}
	return l
}
