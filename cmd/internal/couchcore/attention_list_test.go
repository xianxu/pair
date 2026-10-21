package couchcore

import (
	"fmt"
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
// and kind, a repeat keeps its time, and selecting suppresses that occurrence.
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
			if !ok || item.Thread != attentionW1 || l.Len() != 0 {
				t.Fatalf("select removes a condition item: %+v", l.Items())
			}
			l = l.Conditions(kind, live, attentionT0.Add(time.Minute))
			if l.Len() != 0 {
				t.Fatal("selected condition reappeared while still live")
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

// #673 BR-2: retain the in-memory payload byte bound. A run of
// maximum-size notifications (4096-byte bodies of characters JSON escapes)
// drops the oldest until the list encodes within AttentionByteLimit, so
// the newest notification is kept without unbounded event payloads.
func TestAttentionListStaysWithinTheStoreLimit(t *testing.T) {
	body := strings.Repeat("<\x01", 2048)
	var l AttentionList
	for i := 0; i < AttentionCap; i++ {
		item := notifyItem(fmt.Sprintf("n%02d", i), attentionT0.Add(time.Duration(i)*time.Second))
		item.Body = body
		l = l.Notify(item)
		raw, err := encodeAttention(l.Items())
		if err != nil || len(raw) > AttentionByteLimit {
			t.Fatalf("after %d notifications the list encodes to %d bytes (limit %d): %v", i+1, len(raw), AttentionByteLimit, err)
		}
	}
	if l.Len() == AttentionCap {
		t.Fatalf("no notification was dropped for size")
	}
	if ids := attentionIDs(l); ids[0] != fmt.Sprintf("n%02d", AttentionCap-1) {
		t.Fatalf("the newest notification was dropped: %v", ids)
	}

}

func TestAttentionDismissSuppressesUntilConditionClears(t *testing.T) {
	for _, kind := range []AttentionKind{AttentionPrompt, AttentionQuestion} {
		live := map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}}
		l := AttentionList{}.Conditions(kind, live, attentionT0)
		l = l.Dismiss(l.Items()[0].ID).Conditions(kind, live, attentionT0.Add(time.Minute))
		if l.Len() != 0 {
			t.Fatal("dismissed live condition reappeared")
		}
		l = l.Conditions(kind, nil, attentionT0.Add(time.Hour)).Conditions(kind, live, attentionT0.Add(2*time.Hour))
		if l.Len() != 1 || !l.Items()[0].Since.Equal(attentionT0.Add(2*time.Hour)) {
			t.Fatal("fresh occurrence missing")
		}
	}
}

func TestAttentionExpiryOnlyDropsNotificationsAtTwelveHours(t *testing.T) {
	live := map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}}
	before := AttentionList{}.Conditions(AttentionPrompt, live, attentionT0.Add(-24*time.Hour)).Conditions(AttentionQuestion, live, attentionT0.Add(-24*time.Hour)).Notify(notifyItem("old", attentionT0)).Notify(notifyItem("fresh", attentionT0.Add(time.Nanosecond)))
	after := before.Expire(attentionT0.Add(AttentionMaxAge))
	if after.Len() != 3 || before.Len() != 4 {
		t.Fatalf("expiry: before=%v after=%v", attentionIDs(before), attentionIDs(after))
	}
	for _, item := range after.Items() {
		if item.ID == "old" {
			t.Fatal("expired event kept")
		}
	}
	if !after.Has(AttentionPrompt) || !after.Has(AttentionQuestion) {
		t.Fatal("old live condition expired")
	}
	if before.Expire(attentionT0.Add(AttentionMaxAge-time.Nanosecond)).Len() != 4 {
		t.Fatal("expired too early")
	}
	if before.Apply(AttentionFact{Kind: AttentionNotify, Notify: notifyItem("new", attentionT0.Add(AttentionMaxAge))}, attentionT0.Add(AttentionMaxAge)).Len() != 4 {
		t.Fatal("new fact did not expire old notification")
	}
}

func TestAttentionClearSuppressesAllCurrentConditions(t *testing.T) {
	live := map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1"}}
	before := AttentionList{}.Conditions(AttentionPrompt, live, attentionT0).Conditions(AttentionQuestion, live, attentionT0).Notify(notifyItem("event", attentionT0))
	cleared := before.Clear()
	for _, kind := range []AttentionKind{AttentionPrompt, AttentionQuestion} {
		cleared = cleared.Conditions(kind, live, attentionT0)
	}
	if cleared.Len() != 0 || before.Len() != 3 {
		t.Fatal("clear mutated input or re-raised a current condition")
	}
	fresh := cleared.Conditions(AttentionPrompt, nil, attentionT0).Conditions(AttentionPrompt, live, attentionT0.Add(time.Hour))
	if fresh.Len() != 1 || !fresh.Has(AttentionPrompt) || cleared.Len() != 0 {
		t.Fatal("clear suppression did not end at the condition boundary")
	}
	if cleared.Notify(notifyItem("new", attentionT0)).Len() != 1 {
		t.Fatal("clear hid a new notification")
	}
}
