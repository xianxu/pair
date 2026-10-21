package couchcore

import (
	"maps"
	"slices"
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
	ID     string        `json:"id"`
	Kind   AttentionKind `json:"kind"`
	Thread ThreadAddress `json:"thread"`
	Label  string        `json:"label"`
	Title  string        `json:"title,omitempty"`
	Body   string        `json:"body,omitempty"`
	Since  time.Time     `json:"since"`
}

// AttentionCondition is one slot in a condition set: its label and what it
// waits on.
type AttentionCondition struct {
	Label, Title string
}

const (
	// AttentionCap bounds the list; the oldest notify item goes first, and a
	// condition item never goes while its condition holds.
	AttentionCap    = 50
	AttentionMaxAge = 12 * time.Hour
)

// AttentionList is a value: every transition returns a new list.
type AttentionList struct {
	items      []AttentionItem
	suppressed map[string]AttentionItem // live conditions hidden until their source clears
}

func conditionID(kind AttentionKind, thread ThreadAddress) string {
	return string(kind) + ":" + thread.RepoScope + ":" + string(thread.Tag)
}

// Items is the list newest first.
func (l AttentionList) Items() []AttentionItem {
	out := append([]AttentionItem(nil), l.items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Since.After(out[j].Since) })
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
	l = l.Expire(now)
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
	return AttentionList{items: append([]AttentionItem(nil), l.items...), suppressed: maps.Clone(l.suppressed)}
}

// Expire removes old notification events; a live condition never ages out.
func (l AttentionList) Expire(now time.Time) AttentionList {
	out := l.clone()
	out.items = slices.DeleteFunc(out.items, func(item AttentionItem) bool {
		return item.Kind == AttentionNotify && now.Sub(item.Since) >= AttentionMaxAge
	})
	return out
}

func (l *AttentionList) suppress(item AttentionItem) {
	if item.Kind == AttentionNotify {
		return
	}
	if l.suppressed == nil {
		l.suppressed = map[string]AttentionItem{}
	}
	l.suppressed[item.ID] = item
}

// Clear removes the current list without re-raising its live conditions.
func (l AttentionList) Clear() AttentionList {
	out := l.clone()
	for _, item := range out.items {
		out.suppress(item)
	}
	out.items = nil
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
	for id, item := range out.suppressed {
		_, present := live[item.Thread]
		if item.Kind == kind && !present {
			delete(out.suppressed, id)
		}
	}
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
		if _, hidden := out.suppressed[id]; have[id] || hidden {
			continue
		}
		item := AttentionItem{ID: id, Kind: kind, Thread: thread, Label: c.Label, Title: c.Title, Since: now}
		out.items = append(out.items, item)
	}
	return out.capped()
}

// Select is the operator picking an item: it names the item to jump to. A
// picked item is removed; live conditions stay suppressed until they clear.
func (l AttentionList) Select(id string) (AttentionList, AttentionItem, bool) {
	for _, item := range l.items {
		if item.ID == id {
			return l.Dismiss(id), item, true
		}
	}
	return l, AttentionItem{}, false
}

// Dismiss removes an item of any kind.
func (l AttentionList) Dismiss(id string) AttentionList {
	out := l.clone()
	for i, item := range out.items {
		if item.ID == id {
			out.suppress(item)
			out.items = append(out.items[:i], out.items[i+1:]...)
			break
		}
	}
	return out
}

// capped drops the oldest notifications at the count and payload byte bounds.
func (l AttentionList) capped() AttentionList {
	for len(l.items) > AttentionCap || !l.fits() {
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

// fits keeps the in-memory event payload within the existing byte bound.
func (l AttentionList) fits() bool {
	raw, err := encodeAttention(l.items)
	return err == nil && len(raw) <= AttentionByteLimit
}
