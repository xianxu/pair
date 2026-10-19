package couchcore

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	attentionT0  = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	attentionHub = ThreadAddress{RepoScope: "/scope", Tag: "hub"}
	attentionW1  = ThreadAddress{RepoScope: "/scope", Tag: "w1"}
	attentionW2  = ThreadAddress{RepoScope: "/scope", Tag: "w2"}
)

func attentionIDs(l AttentionList) []string {
	var out []string
	for _, item := range l.Items() {
		out = append(out, item.ID)
	}
	return out
}

func notifyItem(id string, at time.Time) AttentionItem {
	return AttentionItem{ID: id, Thread: attentionHub, Label: "pair:0", Title: "Couch", Body: "needs you", Since: at}
}

// The state table's notify row: raised by a hub notification, read by being
// selected, and never cleared by condition sets (viewing the hub is not one).
func TestAttentionNotifyRow(t *testing.T) {
	l := AttentionList{}.Notify(notifyItem("n1", attentionT0)).Notify(notifyItem("n2", attentionT0.Add(time.Second)))
	if got := attentionIDs(l); fmt.Sprint(got) != "[n2 n1]" {
		t.Fatalf("newest first: %v", got)
	}
	l = l.Conditions(AttentionPrompt, nil, attentionT0).Conditions(AttentionQuestion, nil, attentionT0)
	if l.Len() != 2 {
		t.Fatal("a condition set cleared a notification")
	}
	l, item, ok := l.Select("n1")
	if !ok || item.Thread != attentionHub || fmt.Sprint(attentionIDs(l)) != "[n2]" {
		t.Fatalf("select reads a notification: %v %+v %v", ok, item, attentionIDs(l))
	}
	if l = l.Dismiss("n2"); l.Len() != 0 {
		t.Fatal("dismiss")
	}
	if _, _, ok := l.Select("n2"); ok {
		t.Fatal("selected a gone item")
	}
}

// The prompt and question rows: entering the set raises one item per slot
// and kind, a repeat keeps its time, selecting marks it visited without
// clearing it, and leaving the set clears it whoever answered.
func TestAttentionConditionRows(t *testing.T) {
	for _, kind := range []AttentionKind{AttentionPrompt, AttentionQuestion} {
		t.Run(string(kind), func(t *testing.T) {
			live := map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1", Title: "waiting"}}
			l := AttentionList{}.Conditions(kind, live, attentionT0)
			l = l.Conditions(kind, live, attentionT0.Add(time.Minute))
			items := l.Items()
			if len(items) != 1 || items[0].Kind != kind || !items[0].Since.Equal(attentionT0) || items[0].Label != "pair:1" {
				t.Fatalf("raise once, keep time: %+v", items)
			}
			l, item, ok := l.Select(items[0].ID)
			if !ok || item.Thread != attentionW1 || l.Len() != 1 || !l.Items()[0].Visited {
				t.Fatalf("select keeps a condition item, visited: %+v", l.Items())
			}
			// The other kind's set never touches this one.
			other := AttentionPrompt
			if kind == AttentionPrompt {
				other = AttentionQuestion
			}
			if l = l.Conditions(other, nil, attentionT0); l.Len() != 1 {
				t.Fatal("another kind's set cleared the item")
			}
			if l = l.Conditions(kind, map[ThreadAddress]AttentionCondition{attentionW2: {Label: "pair:2"}}, attentionT0); fmt.Sprint(attentionIDs(l)) != "["+conditionID(kind, attentionW2)+"]" {
				t.Fatalf("a slot leaving the set clears, a new one raises: %v", attentionIDs(l))
			}
			if l = l.Dismiss(conditionID(kind, attentionW2)); l.Len() != 0 {
				t.Fatal("dismiss a condition item")
			}
		})
	}
}

// The restart column: notify items come back as they were; condition items
// come back only when live state re-raises them inside the grace, with their
// original time and visit, and a slot that is no longer waiting stays gone.
func TestAttentionRestore(t *testing.T) {
	before := AttentionList{}.Notify(notifyItem("n1", attentionT0)).
		Conditions(AttentionPrompt, map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}, attentionW2: {Label: "pair:2"}}, attentionT0)
	before, _, _ = before.Select(conditionID(AttentionPrompt, attentionW1))
	saved := before.Saved(attentionT0)

	now := attentionT0.Add(time.Hour)
	l := RestoreAttention(saved, now)
	if fmt.Sprint(attentionIDs(l)) != "[n1]" {
		t.Fatalf("only notify items show before live state: %v", attentionIDs(l))
	}
	if len(l.Saved(now)) != 3 {
		t.Fatal("a second restart inside the grace would lose the waiting items")
	}
	// The startup set is empty (no picker reported yet): nothing is lost.
	l = l.Conditions(AttentionPrompt, nil, now)
	l = l.Conditions(AttentionPrompt, map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}}, now.Add(time.Minute))
	var w1 AttentionItem
	for _, item := range l.Items() {
		if item.Thread == attentionW1 {
			w1 = item
		}
	}
	if !w1.Since.Equal(attentionT0) || !w1.Visited {
		t.Fatalf("re-raised inside the grace keeps time and visit: %+v", w1)
	}
	// After the grace, w2 coming back is a new episode.
	late := now.Add(AttentionRestoreGrace + time.Second)
	l = l.Conditions(AttentionPrompt, map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}, attentionW2: {Label: "pair:2"}}, late)
	for _, item := range l.Items() {
		if item.Thread == attentionW2 && !item.Since.Equal(late) {
			t.Fatalf("a re-raise after the grace is new: %+v", item)
		}
	}
	if len(l.Saved(late)) != l.Len() || len(RestoreAttention(saved, now).Saved(late)) != 1 {
		t.Fatal("expired restored items still persist")
	}
}

// Apply routes a fact by kind and stamps an untimed notification.
func TestAttentionApply(t *testing.T) {
	l := AttentionList{}.Apply(AttentionFact{Kind: AttentionNotify, Notify: AttentionItem{ID: "n", Thread: attentionHub}}, attentionT0)
	l = l.Apply(AttentionFact{Kind: AttentionQuestion, Live: map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}}}, attentionT0)
	if l.Len() != 2 || !l.Items()[0].Since.Equal(attentionT0) || l.Items()[0].Kind != AttentionNotify && l.Items()[1].Kind != AttentionNotify {
		t.Fatalf("apply: %+v", l.Items())
	}
}

// The cap drops the oldest notification and never a held condition.
func TestAttentionCap(t *testing.T) {
	live := map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}}
	l := AttentionList{}.Conditions(AttentionPrompt, live, attentionT0.Add(-time.Hour))
	for i := 0; i < AttentionCap+5; i++ {
		l = l.Notify(notifyItem(fmt.Sprintf("n%02d", i), attentionT0.Add(time.Duration(i)*time.Second)))
	}
	if l.Len() != AttentionCap || !l.Has(AttentionPrompt) {
		t.Fatalf("cap: %d items, prompt kept %v", l.Len(), l.Has(AttentionPrompt))
	}
	for _, id := range attentionIDs(l) {
		if id == "n00" || id == "n05" {
			t.Fatalf("kept an oldest notification %s", id)
		}
	}
}

// #673 BR-2: the writer is bounded by the store's own reader cap. A run of
// maximum-size notifications (4096-byte bodies of characters JSON escapes)
// drops the oldest until the list encodes within AttentionStoreLimit, so
// every Save still succeeds and the newest notification is kept.
func TestAttentionListStaysWithinTheStoreLimit(t *testing.T) {
	body := strings.Repeat("<\x01", 2048)
	var l AttentionList
	for i := 0; i < AttentionCap; i++ {
		item := notifyItem(fmt.Sprintf("n%02d", i), attentionT0.Add(time.Duration(i)*time.Second))
		item.Body = body
		l = l.Notify(item)
		raw, err := encodeAttention(l.Saved(attentionT0))
		if err != nil || len(raw) > AttentionStoreLimit {
			t.Fatalf("after %d notifications the list encodes to %d bytes (limit %d): %v", i+1, len(raw), AttentionStoreLimit, err)
		}
	}
	if l.Len() == AttentionCap {
		t.Fatalf("no notification was dropped for size")
	}
	if ids := attentionIDs(l); ids[0] != fmt.Sprintf("n%02d", AttentionCap-1) {
		t.Fatalf("the newest notification was dropped: %v", ids)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := (AttentionStore{Dir: dir}).Save(l.Saved(attentionT0)); err != nil {
		t.Fatalf("save at the size bound: %v", err)
	}
}
