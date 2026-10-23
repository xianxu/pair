package couchtty

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func FuzzNoticeAttention(f *testing.F) {
	for _, seed := range []string{"", "failure", "\x1b[2Jbad\r\n\x9b", strings.Repeat("界́", 2000), string([]byte{255, 254, 128})} {
		f.Add(seed, seed)
	}
	f.Fuzz(func(t *testing.T, body, label string) {
		thread := couchcore.ThreadAddress{RepoScope: "exact", Tag: "origin"}
		n := Notice{Body: body, Label: label, Thread: thread}
		item, ok := noticeAttention(n, 42)
		if !ok {
			return
		}
		if item.Thread != thread || item.ID != "console-notice-42" || strings.TrimSpace(item.Body) == "" {
			t.Fatalf("lost explicit identity: %+v", item)
		}
		for _, text := range []struct {
			s     string
			bound int
		}{{item.Body, 4096}, {item.Label, 128}} {
			if !utf8.ValidString(text.s) || len(text.s) > text.bound {
				t.Fatalf("unsafe bound: valid=%t bytes=%d limit=%d", utf8.ValidString(text.s), len(text.s), text.bound)
			}
			for _, r := range text.s {
				if r < 32 || r == 127 || r >= 128 && r <= 159 {
					t.Fatalf("control rune %U", r)
				}
			}
		}
		global, _ := noticeAttention(Notice{Actor: "focused-is-not-the-source", Body: body}, 43)
		if global.Thread != (couchcore.ThreadAddress{}) {
			t.Fatal("global warning inferred a source")
		}
	})
}
