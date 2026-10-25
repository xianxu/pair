package editorial

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestSourceObserverMatchesDefaultGoldmark(t *testing.T) {
	sources := []string{"- item\nlazy continuation\n\nOutside paragraph.\n", "- item\n# Outside heading\n", "- item\n---\n", "> - Ask\n>\n>   ```\n>   ```\n>\n> outside continuation\n\nOutside.\n", "-\n\n- #\n\n- ```\n  ```\n", "\t- code\n\n  - list\n    - nested\n\n      ---\n", "- text\n  ```\n  data\n  ```\n\nOutside\n"}
	paths, _ := filepath.Glob("testdata/contract/examples/editorial-*/before/*.md")
	for _, path := range paths {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		sources = append(sources, string(raw))
	}
	for i, source := range sources {
		raw := []byte(source)
		observed, _, _ := parseSourceMarkdown(raw, raw, sourceLines(raw))
		ordinary := goldmark.DefaultParser().Parse(text.NewReader(raw))
		var got, want bytes.Buffer
		if e := goldmark.DefaultRenderer().Render(&got, raw, observed); e != nil {
			t.Fatal(e)
		}
		if e := goldmark.DefaultRenderer().Render(&want, raw, ordinary); e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(got.Bytes(), want.Bytes()) {
			t.Fatalf("case %d changed Markdown semantics\ngot %s\nwant %s", i, got.String(), want.String())
		}
	}
}

func TestListSourceObserverTrailingBlankAndOutsideBoundary(t *testing.T) {
	for _, tc := range []struct{ name, item, after string }{{"lazy rollback", "- Ask\nlazy continuation\n", "\nOutside paragraph.\n"}, {"interrupting heading", "- Ask\n", "# Outside heading\n"}, {"empty item", "-\n", "\nOutside.\n"}, {"quoted blank", " > - Ask\n", " >\n\nOutside.\n"}, {"unicode content", "- Ask\n  \u00a0\n", "\nOutside.\n"}} {
		t.Run(tc.name, func(t *testing.T) {
			prefix := "---\ncollaboration-mode: chat\n---\n# Chat\n"
			src := prefix + tc.item + tc.after
			p, e := Compile([]byte(src))
			if e != nil {
				t.Fatal(e)
			}
			if len(p.ListItems) != 1 || p.ListItems[0] != (Range{len(prefix), len(prefix) + len(tc.item)}) {
				t.Fatalf("item source ranges %+v, expected %d:%d", p.ListItems, len(prefix), len(prefix)+len(tc.item))
			}
		})
	}
}

func TestListSourceObserverGrowth(t *testing.T) {
	previous := 0
	for _, n := range []int{64, 128} {
		var source strings.Builder
		source.WriteString("---\ncollaboration-mode: chat\n---\n# Chat\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&source, "- Ask%d\n\n  ```\n  ```\n\n  Last continuation.\n\n", i)
		}
		p, e := Compile([]byte(source.String()))
		if e != nil {
			t.Fatal(e)
		}
		if len(p.ListItems) != n || p.Stats.ListSourceEvents > 12*n {
			t.Fatalf("n=%d items=%d source events=%d", n, len(p.ListItems), p.Stats.ListSourceEvents)
		}
		if previous > 0 && p.Stats.ListSourceEvents > 2*previous+4 {
			t.Fatalf("superlinear source events %d -> %d", previous, p.Stats.ListSourceEvents)
		}
		previous = p.Stats.ListSourceEvents
	}
}
