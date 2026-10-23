package couchtty

import (
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestChromeStatusTextUsesFortyDisplayCells(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"short", "ready", "ready"},
		{"exact", strings.Repeat("x", 40), strings.Repeat("x", 40)},
		{"long", strings.Repeat("x", 200), strings.Repeat("x", 39) + "…"},
		{"wide", strings.Repeat("界", 100), strings.Repeat("界", 19) + "…"},
		{"combining", strings.Repeat("e\u0301", 41), strings.Repeat("e\u0301", 39) + "…"},
		{"control", "\x1b[31mready\x1b[0m\n\r\x0e", "ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, capability := range []bool{false, true} {
				m := StatusModel{}
				if capability {
					m.CapabilityNotice = tc.input
				} else {
					m.Notice = tc.input
				}
				legacy := RenderStatusRow(240, m)
				bar := RenderStatusBar(240, 2, m)
				for _, body := range []string{legacy.Body, bar.Body()} {
					got := strings.TrimSpace(rowtext.Sanitize(body))
					if got != tc.want || textwidth.Width(got) > 40 || !utf8.ValidString(got) {
						t.Fatalf("capability=%v got %q (%d cells), want %q", capability, got, textwidth.Width(got), tc.want)
					}
				}
			}
		})
	}
}

func TestChromeLongNoticePreservesLayoutAndFullFeed(t *testing.T) {
	full := strings.Repeat("n", 200)
	feed := NewFeed(8, nil, NoticeLifetime)
	feed.Push(Notice{Kind: "status", Body: full})
	m := StatusModel{Actors: []StatusActor{{Label: "agent", Active: true}}, Notice: feed.Row().Body}
	bar := RenderStatusBar(100, 2, m)
	m.Notice = strings.Repeat("n", 39) + "…"
	want := RenderStatusBar(100, 2, m)
	if bar.Body() != want.Body() || bar.Height() != want.Height() {
		t.Fatalf("long notice changed layout: %q vs %q", bar.Body(), want.Body())
	}
	if feed.Messages()[0].Body != full {
		t.Fatal("display cap destroyed full text")
	}
}

func TestChromeCappedNoticeCopiesFullText(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	full := strings.Repeat("diagnostic 界 ", 30) + "final tail"
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) { c.feed.Push(Notice{Kind: "status", Body: full}) })
	f.con.copyLastNotice()
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(full)) + "\x1b\\"
	waitFor(t, "full notice copy", func() bool { return strings.Contains(f.host.Written(), want) })
}

func TestChromeStatusTextInputWorkIsLinear(t *testing.T) {
	for _, atom := range []string{"x", "界", "e\u0301", "\x1b[31mtext\x1b[0m"} {
		previous := 0
		for _, n := range []int{1024, 2048} {
			input := strings.Repeat(atom, n)
			work := 0
			got := chromeStatusText(input, &work)
			if work > 3*len(input) || (previous > 0 && work > 2*previous) {
				t.Fatalf("%q N=%d work=%d previous=%d", atom, n, work, previous)
			}
			if textwidth.Width(got) > ChromeStatusCells {
				t.Fatalf("unbounded result %q", got)
			}
			previous = work
		}
	}
}
