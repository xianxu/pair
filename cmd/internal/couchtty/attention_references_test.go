package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestAttentionReferenceCells(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		cols             int
	}{
		{"prose", "See pair:3 ariadne#575 and #841", "pair:0pair:3ariadne#575#841", 80},
		{"punctuation", "(pair:3): [parley.nvim#419]; #841.", "pair:0pair:3parley.nvim#419#841", 80},
		{"joined punctuation", "pair:3,#841!", "pair:0pair:3#841", 80},
		{"paths", "/tmp/pair:3 x/ariadne#575 C:\\x\\#841", "pair:0", 80},
		{"urls", "https://x/pair:3 https://x/#841", "pair:0", 80},
		{"words", "pair:3tail word#841tail #841tail", "pair:0", 80},
		{"unicode", "界 (pair:3) é#841 #841é", "pair:0pair:3", 80},
		{"zero width", "\u200b pair:3", "pair:0pair:3", 80},
		{"controls", "\x1b[31mpair:3\x1b[0m #841", "pair:0pair:3#841", 80},
		{"hard wrap", strings.Repeat("r", 40) + ":3", "pair:0" + strings.Repeat("r", 40) + ":3", 38},
		{"wrapped path", "/" + strings.Repeat("x", 35) + "/pair:3 #841", "pair:0#841", 38},
		{"clip", "#1 " + strings.Repeat("pair:3 ", 50), "", 38},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []couchcore.AttentionKind{couchcore.AttentionNotify, couchcore.AttentionPrompt, couchcore.AttentionQuestion} {
				item := couchcore.AttentionItem{ID: "n", Kind: kind, Label: "pair:0", Title: tc.body, Since: time.Now()}
				if kind == couchcore.AttentionNotify {
					item.Title = ""
					item.Body = tc.body
				}
				c := &Console{}
				c.attn.open = true
				if kind == couchcore.AttentionNotify {
					c.attn.list = c.attn.list.Apply(couchcore.AttentionFact{Kind: kind, Notify: item}, item.Since)
				} else {
					c.attn.list = c.attn.list.Apply(couchcore.AttentionFact{Kind: kind, Live: map[couchcore.ThreadAddress]couchcore.AttentionCondition{{Tag: "t"}: {Label: item.Label, Title: item.Title}}}, item.Since)
				}
				c.layoutAttentionLocked(tc.cols, 24)
				box := c.attn.box
				for _, hover := range []int{0, 2} {
					f := box.Draw(menuFrame(t, tc.cols, 24), hover, broadcast.NoteStyle{})
					var colored strings.Builder
					for _, cell := range f.Cells {
						if cell.Style.Fg == notesMenuLink {
							colored.WriteString(cell.Content)
						}
					}
					if tc.name != "clip" && colored.String() != tc.want {
						t.Errorf("kind %v hover %d colored %q want %q", kind, hover, colored.String(), tc.want)
					}
					if tc.name == "clip" && colored.Len() == 0 {
						t.Error("clipped references lost color")
					}
					for _, line := range box.lines {
						if strings.ContainsRune(line.text, 0x1b) {
							t.Fatal("escape in stored text")
						}
					}
					for y := box.items[1].Top; y < box.items[1].Bottom; y++ {
						if box.ItemAt(box.X+5, y) != 2 {
							t.Fatal("reference changed hit owner")
						}
						wantBG := notesMenuBg
						if hover == 2 {
							wantBG = notesMenuHover
						}
						for x := box.X + 4; x < box.X+box.W; x++ {
							if f.Cells[y*tc.cols+x].Style.Bg != wantBG {
								t.Fatal("accent changed hover background")
							}
						}
					}
				}
				// Shared notes do not opt in, and the source remains unchanged.
				m := RenderNotesMenu([]broadcast.Note{{ID: 1, Text: c.attn.shown[0].text}}, tc.cols, 24)
				for _, cell := range m.Draw(menuFrame(t, tc.cols, 24), 0, broadcast.NoteStyle{}).Cells {
					if cell.Style.Fg == notesMenuLink {
						t.Fatal("notes unexpectedly accented")
					}
				}
				if !strings.Contains(c.attn.shown[0].text, strings.TrimSpace(tc.body)) {
					t.Fatal("source rewritten")
				}
				if !reflect.DeepEqual(box.items, c.attn.box.items) {
					t.Fatal("draw mutated layout")
				}
			}
		})
	}
}

func TestAttentionReferenceCopyStaysPlain(t *testing.T) {
	copied := noticeClipboard(t)
	body := "Review pair:3 ariadne#575 and #841"
	f, threads := pagingFixture(t)
	f.con.ObserveAttention(hubNotify(threads[2], body))
	f.con.mu.Lock()
	f.con.feed.Push(Notice{Kind: "status", Body: body})
	f.con.mu.Unlock()
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	_, _ = f.stdin.Write([]byte(ctrlAltA))
	waitFor(t, "bell", f.attentionOpen)
	_, _ = f.stdin.Write([]byte{0x19})
	waitFor(t, "copy", func() bool { return f.con.menuSnapshot().CopyFeedback.Kind == CopyFeedbackSuccess })
	if got := copied(); got != body {
		t.Fatalf("copied %q want %q", got, body)
	}
}

// The classifier is invoked once per candidate, independent of wrapping or
// the number of references already seen. Rejected URL/path chunks skip it.
func TestAttentionReferenceWorkScalesLinearly(t *testing.T) {
	for _, n := range []int{8, 16} {
		text := strings.Repeat("pair:3,#841 https://x/pair:3 /tmp/#841 ", n)
		calls := 0
		accents := referenceAccents(text, func(token string) bool { calls++; return referenceToken(token) })
		if calls != 2*n || len(accents) != len(text) {
			t.Fatalf("N=%d calls=%d mask=%d bytes=%d", n, calls, len(accents), len(text))
		}
		colored := 0
		for _, on := range accents {
			if on {
				colored++
			}
		}
		if colored != 10*n {
			t.Fatalf("N=%d accented bytes=%d", n, colored)
		}
	}
}
