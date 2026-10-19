package couchtty

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

const ctrlAltA = "\x1b[97;7u"

// memoryAttentionStore is the store's stateful fake: what Save wrote is what
// the next Load reads.
type memoryAttentionStore struct {
	mu    sync.Mutex
	items []couchcore.AttentionItem
	saves int
}

func (s *memoryAttentionStore) Load() ([]couchcore.AttentionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]couchcore.AttentionItem(nil), s.items...), nil
}

func (s *memoryAttentionStore) Save(items []couchcore.AttentionItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items, s.saves = append([]couchcore.AttentionItem(nil), items...), s.saves+1
	return nil
}

func (s *memoryAttentionStore) saved() []couchcore.AttentionItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]couchcore.AttentionItem(nil), s.items...)
}

func (f *consoleFixture) attentionOpen() bool {
	f.con.attn.mu.Lock()
	defer f.con.attn.mu.Unlock()
	return f.con.attn.open
}

func (f *consoleFixture) attentionLen() int { return f.con.attentionCount() }

func hubNotify(hub couchcore.ThreadAddress, body string) couchcore.AttentionFact {
	return couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: couchcore.AttentionItem{
		ID: "notify-" + body, Thread: hub, Label: "pair:0", Title: "Couch", Body: body}}
}

// The Done-when's first clause: a hub notification is in the list at once
// (the badge, then the panel over the top right), and Enter jumps to the hub
// slot through the notification funnel and reads the item.
func TestAttentionNotifyAppearsAndEnterJumpsToTheHub(t *testing.T) {
	f, threads := pagingFixture(t)
	store := &memoryAttentionStore{}
	f.con.SetAttentionStore(store)
	hub := threads[2]

	f.con.ObserveAttention(hubNotify(hub, "review the plan"))
	waitFor(t, "the badge", func() bool { return strings.HasSuffix(strings.TrimRight(f.lastRow(), " "), AttentionGlyph+"1") })
	if saved := store.saved(); len(saved) != 1 || saved[0].Body != "review the plan" {
		t.Fatalf("the notification was not saved: %+v", saved)
	}

	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "the panel", func() bool { return f.attentionOpen() && strings.Contains(f.screenText(), "Waiting on you") })
	top := strings.Split(f.screenText(), "\n")[1]
	if !strings.Contains(top, "pair:0") || !strings.Contains(top, "Couch: review the plan") {
		t.Fatalf("the panel's first item is not at the top right: %q", top)
	}

	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "the jump to the hub", func() bool { return activeOf(f) == "c3" })
	if f.attentionOpen() || f.attentionLen() != 0 || len(store.saved()) != 0 {
		t.Fatalf("a picked notification stays: open=%v len=%d saved=%+v", f.attentionOpen(), f.attentionLen(), store.saved())
	}
	f.con.mu.Lock()
	tracker := f.con.tracker
	f.con.mu.Unlock()
	if previous, ok := tracker.Previous(); !ok || previous != threads[0] {
		t.Fatalf("the jump did not arrive as a notification (Ctrl+Backspace home): %+v", tracker)
	}
}

// The Done-when's second clause: a slot entering the prompt set is listed,
// red, and stays after a visit; leaving the set clears it.
func TestAttentionPromptAppearsAndClearsWhenAnswered(t *testing.T) {
	f, threads := pagingFixture(t)
	f.con.SetAttentionStore(&memoryAttentionStore{})
	prompts := func(live map[couchcore.ThreadAddress]couchcore.AttentionCondition) {
		f.con.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionPrompt, Live: live})
	}
	prompts(map[couchcore.ThreadAddress]couchcore.AttentionCondition{threads[1]: {Label: "pair:2", Title: "waiting on a prompt"}})
	waitFor(t, "a red badge", func() bool {
		row := f.lastRow()
		return strings.Contains(row, AttentionGlyph+"1") && f.statusModel().AttentionPrompt
	})

	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "the panel", func() bool { return strings.Contains(f.screenText(), "pair:2  waiting on a prompt") })
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "the jump to the prompting slot", func() bool { return activeOf(f) == "c2" })
	if f.attentionLen() != 1 {
		t.Fatal("a visit cleared an unanswered prompt")
	}

	prompts(map[couchcore.ThreadAddress]couchcore.AttentionCondition{})
	waitFor(t, "the badge gone", func() bool { return !strings.Contains(f.lastRow(), AttentionGlyph) })
	if f.attentionLen() != 0 {
		t.Fatal("an answered prompt stayed")
	}
}

// The Done-when's restart clause: what was saved comes back in a new
// Console. A notification shows at once; a prompt waits for the live set.
func TestAttentionSurvivesARestart(t *testing.T) {
	store := &memoryAttentionStore{}
	first, threads := pagingFixture(t)
	first.con.SetAttentionStore(store)
	first.con.ObserveAttention(hubNotify(threads[2], "still here"))
	first.con.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionPrompt,
		Live: map[couchcore.ThreadAddress]couchcore.AttentionCondition{threads[1]: {Label: "pair:2", Title: "waiting on a prompt"}}})
	if len(store.saved()) != 2 {
		t.Fatalf("saved %+v", store.saved())
	}

	second, again := pagingFixture(t)
	second.con.SetAttentionStore(store)
	waitFor(t, "the restored notification", func() bool { return strings.Contains(second.lastRow(), AttentionGlyph+"1") })
	second.con.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionPrompt,
		Live: map[couchcore.ThreadAddress]couchcore.AttentionCondition{again[1]: {Label: "pair:2", Title: "waiting on a prompt"}}})
	waitFor(t, "the confirmed prompt", func() bool { return strings.Contains(second.lastRow(), AttentionGlyph+"2") })
}

// A click on the badge opens the panel; a click on an item jumps; d
// dismisses; Escape closes.
func TestAttentionMouseAndDismiss(t *testing.T) {
	f, threads := pagingFixture(t)
	f.con.SetAttentionStore(&memoryAttentionStore{})
	f.con.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionQuestion,
		Live: map[couchcore.ThreadAddress]couchcore.AttentionCondition{threads[1]: {Label: "pair:2", Title: "which branch?"}}})
	f.con.ObserveAttention(hubNotify(threads[2], "later"))
	waitFor(t, "the badge", func() bool { return strings.Contains(f.lastRow(), AttentionGlyph+"2") })

	f.con.mu.Lock()
	span := f.con.statusAttention
	f.con.mu.Unlock()
	f.clickStatus(span.Start + 1)
	waitFor(t, "the panel from a click", f.attentionOpen)

	_, _ = f.stdin.Write([]byte("d")) // the newest, the notification
	waitFor(t, "the dismissal", func() bool { return f.attentionLen() == 1 })
	if !f.attentionOpen() {
		t.Fatal("dismissing one of two closed the panel")
	}

	f.con.attn.mu.Lock()
	box := f.con.attn.box
	f.con.attn.mu.Unlock()
	row := box.items[0].Top
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", box.X+6, row+1, box.X+6, row+1)
	waitFor(t, "the click jump", func() bool { return activeOf(f) == "c2" })
	if f.attentionOpen() || f.attentionLen() != 1 {
		t.Fatal("a question cleared on a visit, or the panel stayed open")
	}

	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "the panel again", f.attentionOpen)
	_, _ = f.stdin.Write([]byte("\x1b"))
	waitFor(t, "Escape closes", func() bool { return !f.attentionOpen() })
}

// An empty list opens nothing and says so.
func TestAttentionEmptySaysSo(t *testing.T) {
	f, _ := pagingFixture(t)
	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "the notice", func() bool { return strings.Contains(noticeOf(f), "Nothing is waiting on you") })
	if f.attentionOpen() {
		t.Fatal("opened an empty list")
	}
}

func (f *consoleFixture) statusModel() StatusModel {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.statusModelLocked()
}

// #673 TL decision 5: the badge sits at the right end of the status row, so
// it never covers slot content; the rest of the row clips short of it; it
// takes the blocked color while a prompt item is open; and it is absent with
// an empty list or a row too narrow to hold it.
func TestAttentionBadgeRendersAtTheStatusRowRightEnd(t *testing.T) {
	notice := strings.Repeat("n", 100)
	row := RenderStatusRow(40, StatusModel{Notice: notice, Attention: 3})
	plain := rowPlain(row)
	if textwidth.Width(plain) != 40 || !strings.HasSuffix(plain, " "+AttentionGlyph+"3") {
		t.Fatalf("badge not at the right end of a full row: %q", plain)
	}
	if row.Attention.End != 40 || row.Attention.Start != 40-textwidth.Width(AttentionGlyph+"3") {
		t.Fatalf("badge span = %+v", row.Attention)
	}
	if !strings.Contains(row.Body, attentionSGR+AttentionGlyph+"3") {
		t.Fatalf("badge without a prompt item is not the attention color: %q", row.Body)
	}
	if blocked := RenderStatusRow(40, StatusModel{Attention: 3, AttentionPrompt: true}); !strings.Contains(blocked.Body, blockedSGR+AttentionGlyph+"3") {
		t.Fatalf("badge with a prompt item is not the blocked color: %q", blocked.Body)
	}
	if empty := RenderStatusRow(40, StatusModel{Notice: "x"}); strings.Contains(empty.Body, AttentionGlyph) || empty.Attention != (ColumnSpan{}) {
		t.Fatalf("an empty list drew a badge: %q", empty.Body)
	}
	if narrow := RenderStatusRow(3, StatusModel{Attention: 12}); strings.Contains(narrow.Body, AttentionGlyph) {
		t.Fatalf("a row too narrow for the badge drew it: %q", narrow.Body)
	}
}
