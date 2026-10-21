package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

const ctrlAltA = "\x1b[97;7u"

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
	hub := threads[2]

	f.con.ObserveAttention(hubNotify(hub, "review the plan"))
	waitFor(t, "the badge", func() bool { return strings.HasSuffix(strings.TrimRight(f.lastRow(), " "), AttentionGlyph+"1") })

	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "the panel", func() bool { return f.attentionOpen() && strings.Contains(f.screenText(), "Waiting on you") })
	top := strings.Split(f.screenText(), "\n")[2]
	if !strings.Contains(top, "pair:0") || !strings.Contains(top, "Couch: review the plan") {
		t.Fatalf("the panel's first item is not at the top right: %q", top)
	}

	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "the jump to the hub", func() bool { return activeOf(f) == "c3" })
	if f.attentionOpen() || f.attentionLen() != 0 {
		t.Fatalf("a picked notification stays: open=%v len=%d", f.attentionOpen(), f.attentionLen())
	}
	f.con.mu.Lock()
	tracker := f.con.tracker
	f.con.mu.Unlock()
	if previous, ok := tracker.Previous(); !ok || previous != threads[0] {
		t.Fatalf("the jump did not arrive as a notification (Ctrl+Backspace home): %+v", tracker)
	}
}

// The Done-when's second clause: a slot entering the prompt set is listed,
// red, and is removed on a visit; leaving the set also clears it.
func TestAttentionPromptAppearsAndClearsWhenAnswered(t *testing.T) {
	f, threads := pagingFixture(t)
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
	if f.attentionLen() != 0 {
		t.Fatal("a visit kept the prompt nudge")
	}

	prompts(map[couchcore.ThreadAddress]couchcore.AttentionCondition{})
	waitFor(t, "the badge gone", func() bool { return !strings.Contains(f.lastRow(), AttentionGlyph) })
	if f.attentionLen() != 0 {
		t.Fatal("an answered prompt stayed")
	}
}

// A fresh Console never inherits notification events or suppression.
func TestAttentionRestartStartsEmpty(t *testing.T) {
	first, threads := pagingFixture(t)
	first.con.ObserveAttention(hubNotify(threads[2], "old event"))
	if first.attentionLen() != 1 {
		t.Fatal("notification not observed")
	}
	second, again := pagingFixture(t)
	if second.attentionLen() != 0 {
		t.Fatal("restart inherited events")
	}
	second.con.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionPrompt, Live: map[couchcore.ThreadAddress]couchcore.AttentionCondition{again[1]: {Label: "pair:2"}}})
	if second.attentionLen() != 1 {
		t.Fatal("current live condition not re-derived")
	}
}

// A click on the badge opens the panel; a click on an item jumps; d
// dismisses; Escape closes.
func TestAttentionMouseAndDismiss(t *testing.T) {
	f, threads := pagingFixture(t)
	f.con.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionQuestion,
		Live: map[couchcore.ThreadAddress]couchcore.AttentionCondition{threads[1]: {Label: "pair:2", Title: "which branch?"}}})
	f.con.ObserveAttention(hubNotify(threads[2], "later"))
	waitFor(t, "the badge", func() bool { return strings.Contains(f.lastRow(), AttentionGlyph+"2") })

	f.con.mu.Lock()
	span := f.con.statusAttention
	f.con.mu.Unlock()
	f.clickStatus(span.Start + 1)
	waitFor(t, "the panel from a click", f.attentionOpen)
	// A second click on the badge closes it, and the click does not go on to
	// reopen it (BR-5). The decoder hands a mouse event on once later input
	// arrives; a motion report at the top-left corner flushes each click.
	flush := func() { _, _ = f.stdin.Write([]byte("\x1b[<35;1;1M")) }
	f.clickStatus(span.Start + 1)
	flush()
	waitFor(t, "the badge click closes", func() bool { return !f.attentionOpen() })
	time.Sleep(50 * time.Millisecond)
	if f.attentionOpen() {
		t.Fatal("a badge click on the open panel reopened it")
	}
	f.clickStatus(span.Start + 1)
	flush()
	waitFor(t, "the panel from a third click", f.attentionOpen)

	_, _ = f.stdin.Write([]byte("d")) // the newest, the notification
	waitFor(t, "the dismissal", func() bool { return f.attentionLen() == 1 })
	if !f.attentionOpen() {
		t.Fatal("dismissing one of two closed the panel")
	}

	f.con.attn.mu.Lock()
	box := f.con.attn.box
	f.con.attn.mu.Unlock()
	row := box.items[1].Top
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", box.X+6, row+1, box.X+6, row+1)
	waitFor(t, "the click jump", func() bool { return activeOf(f) == "c2" })
	if f.attentionOpen() || f.attentionLen() != 0 {
		t.Fatal("a question stayed after a visit, or the panel stayed open")
	}

	f.con.ObserveAttention(hubNotify(threads[2], "another"))
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

// #673 BR-4: row text is time, slot, then what waits: a notification shows
// its title and body (the body alone with no title),
// and each kind has its own chip color; the order is the list's.
func TestAttentionRowsFormat(t *testing.T) {
	at := time.Date(2026, 10, 18, 9, 5, 0, 0, time.Local)
	items := []couchcore.AttentionItem{
		{ID: "n1", Kind: couchcore.AttentionNotify, Label: "pair:0", Title: "Couch", Body: "review ready", Since: at},
		{ID: "n2", Kind: couchcore.AttentionNotify, Label: "pair:0", Body: "body only", Since: at},
		{ID: "p", Kind: couchcore.AttentionPrompt, Label: "pair:2", Title: "waiting on a prompt", Since: at},
		{ID: "q", Kind: couchcore.AttentionQuestion, Label: "pair:3", Title: "Should I merge?", Since: at},
	}
	rows := attentionRows(items)
	want := []struct {
		text string
		bg   any
	}{
		{"09:05  pair:0  Couch: review ready", attentionNotifyBg},
		{"09:05  pair:0  body only", attentionNotifyBg},
		{"09:05  pair:2  waiting on a prompt", attentionPromptBg},
		{"09:05  pair:3  Should I merge?", attentionQuestionBg},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows %+v", rows)
	}
	for i, w := range want {
		if rows[i].item.ID != items[i].ID || rows[i].text != w.text || rows[i].bg != w.bg {
			t.Fatalf("row %d = %q %v, want %q %v", i, rows[i].text, rows[i].bg, w.text, w.bg)
		}
	}
}

// #673 BR-4: the panel's box is anchored top-right at any size: flush with
// the right edge and the top row, items inside it, an overflow line when the
// rows run out, and nothing at all on a screen too small to hold one item.
func TestAttentionBoxTopRightGeometry(t *testing.T) {
	var items []menuItem
	for i := 1; i <= 9; i++ {
		items = append(items, menuItem{ID: i, Text: fmt.Sprintf("item %d", i), Chip: true})
	}
	for _, size := range [][2]int{{80, 24}, {41, 7}, {23, 9}, {200, 50}} {
		cols, rows := size[0], size[1]
		m := renderMenuAt("Attention", items, cols, rows, true)
		if m.H == 0 {
			t.Fatalf("%dx%d: no box", cols, rows)
		}
		if m.Y != 0 || m.X+m.W != cols || m.W != min(notesMenuMaxWidth, cols-2) || m.H > rows-3 {
			t.Fatalf("%dx%d: box %+v is not top-right within the screen", cols, rows, m)
		}
		for _, it := range m.items {
			if it.Top < m.Y+1 || it.Bottom > m.Y+m.H {
				t.Fatalf("%dx%d: item %+v outside box %+v", cols, rows, it, m)
			}
		}
		if len(m.items) < len(items) && !strings.Contains(m.lines[len(m.lines)-1].text, "more") {
			t.Fatalf("%dx%d: %d of %d items with no overflow line", cols, rows, len(m.items), len(items))
		}
		if bottom := renderMenuAt("Attention", items, cols, rows, false); bottom.Y+bottom.H != rows-1 {
			t.Fatalf("%dx%d: the bottom anchor moved: %+v", cols, rows, bottom)
		}
	}
	for _, size := range [][2]int{{6, 24}, {80, 4}} {
		if m := renderMenuAt("Attention", items, size[0], size[1], true); m.H != 0 || len(m.items) != 0 {
			t.Fatalf("%dx%d: drew a box on a screen too small: %+v", size[0], size[1], m)
		}
	}
}

func TestAttentionClearAllKeyboardAndMouse(t *testing.T) {
	for _, mouse := range []bool{false, true} {
		t.Run(fmt.Sprint(mouse), func(t *testing.T) {
			f, threads := pagingFixture(t)
			for i := 0; i < 50; i++ {
				f.con.ObserveAttention(hubNotify(threads[2], fmt.Sprint(i)))
			}
			_, _ = f.stdin.Write([]byte(ctrlAltA))
			waitFor(t, "visible clear-all in overflowing panel", func() bool { return f.attentionOpen() && strings.Contains(f.screenText(), "Clear all (c)") })
			if mouse {
				f.con.attn.mu.Lock()
				box := f.con.attn.box
				row := box.items[0].Top
				f.con.attn.mu.Unlock()
				_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", box.X+6, row+1, box.X+6, row+1)
			} else {
				_, _ = f.stdin.Write([]byte("c"))
			}
			waitFor(t, "cleared panel and badge", func() bool {
				return f.attentionLen() == 0 && !f.attentionOpen() && !strings.Contains(f.lastRow(), AttentionGlyph)
			})
		})
	}
}

func TestAttentionActivityTickExpiresWithoutAnotherFact(t *testing.T) {
	clock := &testClock{now: time.Now()}
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) { c.now = clock.Now; c.activityInterval = 5 * time.Millisecond })
	f.con.ObserveAttention(hubNotify(couchcore.ThreadAddress{Tag: "hub"}, "expire me"))
	waitFor(t, "notification badge", func() bool { return strings.Contains(f.lastRow(), AttentionGlyph+"1") })
	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "open panel", f.attentionOpen)
	clock.advance(couchcore.AttentionMaxAge)
	waitFor(t, "tick closes expired panel and badge", func() bool {
		return f.attentionLen() == 0 && !f.attentionOpen() && !strings.Contains(f.lastRow(), AttentionGlyph)
	})
}
