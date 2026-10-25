package editorial

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileAdversarial(t *testing.T) {
	cases := []struct {
		name, src string
		bad       bool
		headings  int
	}{
		{"fences", "```md\n# Fake\n<!-- presentation-mode: typo -->\n```\n# Real\ntext\n", false, 1},
		{"detached", "# Real\n\n<!-- presentation-mode: typo -->\n", false, 1},
		{"ordinary", "# Real\n<!-- ordinary comment -->\n", false, 1},
		{"duplicate", "# Real\n<!-- presentation-mode: direct; presentation-mode: propose -->\n", false, 1},
		{"malformed", "# Real\n<!-- presentation-mode : direct -->\n", false, 1},
		{"multiple", "# Real\n<!-- presentation-mode: direct -->\n<!-- processing-mode: free-form -->\n", false, 1},
		{"empty", "# Real\n<!-- presentation-mode:  -->\n", false, 1},
		{"legacy selector only", "---\nprocess-instruction: [copy-edit]\n---\n# Real\n", false, 1},
		{"legacy invalid whole yaml", "---\nprocess-instruction: [copy-edit] Keep this: exactly.\nopaque: [invalid\n---\n# Real\n", false, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, e := Compile([]byte(tc.src))
			if (e != nil) != tc.bad {
				t.Fatalf("err=%v", e)
			}
			if !tc.bad && len(p.Headings) != tc.headings {
				t.Fatalf("headings %+v", p.Headings)
			}
		})
	}
}

func FuzzCompileOwnedSource(f *testing.F) {
	for _, s := range []string{"", "# A\n<!-- presentation-mode: direct -->\n", "---\nprocess-instruction: [copy-edit] text: words\n---\n", "```\n# Fake\n```\n"} {
		f.Add(s)
	}
	paths, _ := filepath.Glob("testdata/contract/examples/editorial-*/before/*.md")
	for _, path := range paths {
		raw, e := os.ReadFile(path)
		if e == nil {
			f.Add(string(raw))
		}
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 65536 {
			t.Skip()
		}
		p, e := Compile([]byte(s))
		if e != nil {
			return
		}
		for _, o := range p.Owned {
			if o.Range.Start < 0 || o.Range.End > len(s) || o.Range.Start > o.Range.End || !bytes.Equal(o.Raw, []byte(s[o.Range.Start:o.Range.End])) {
				t.Fatal("owned range drift")
			}
		}

	})
}

func TestATXNamePreservesLiteralHash(t *testing.T) {
	p, e := Compile([]byte("# C#\n# C ##\n"))
	if e != nil {
		t.Fatal(e)
	}
	if p.Headings[0].Name != "C#" || p.Headings[1].Name != "C" {
		t.Fatalf("%+v", p.Headings)
	}
}

func TestCompileCodeMasks(t *testing.T) {
	src := []byte("# A `literal`\n\n    indented 🤖[ask]\n\n```lang\nfenced 🤖[ask]\n```\n")
	p, e := Compile(src)
	if e != nil {
		t.Fatal(e)
	}
	for _, needle := range []string{"literal", "indented 🤖[ask]", "fenced 🤖[ask]", "lang"} {
		pos := bytes.Index(src, []byte(needle))
		found := false
		for _, r := range p.CodeRanges {
			if r.Start <= pos && r.End >= pos+len(needle) {
				found = true
			}
		}
		if !found {
			t.Fatalf("unmasked %q: %+v", needle, p.CodeRanges)
		}
	}
}

func TestCompileBlockAndListFacts(t *testing.T) {
	src := []byte("# Chat\n- Ask\n  - Nested\n\n- Second\n\n```\n- code\n```\n")
	p, e := Compile(src)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.ListItems) != 2 || len(p.Blocks) < 3 {
		t.Fatalf("list facts %+v blocks %+v", p.ListItems, p.Blocks)
	}
	if !bytes.Contains(src[p.ListItems[0].Start:p.ListItems[0].End], []byte("Nested")) {
		t.Fatal("nested item not included in shallow ask")
	}
}

func TestOwnedBlockScalarIndentAndChomp(t *testing.T) {
	for _, tc := range []struct{ header, want string }{{"  processing-instruction: |2-\n    line one\n    line two\n", "line one\nline two"}, {"  processing-instruction: |2\n    line one\n    line two\n", "line one\nline two\n"}, {"  processing-instruction: >2-\n    line one\n    line two\n", "line one line two"}, {"processing-instruction: |2-\n  line one\n    indented\n", "line one\n  indented"}} {
		p, e := Compile([]byte("---\n" + tc.header + "---\n"))
		if e != nil || p.parsed.fields["processing-instruction"].value.Value != tc.want {
			t.Fatalf("%q instruction=%q want=%q err=%v", tc.header, p.parsed.fields["processing-instruction"].value.Value, tc.want, e)
		}
	}
}

func TestOwnedBlockScalarKeepChomp(t *testing.T) {
	p, e := Compile([]byte("---\n  processing-instruction: |2+\n    line\n\n\n---\n"))
	if e != nil {
		t.Fatal(e)
	}
	if p.parsed.fields["processing-instruction"].value.Value != "line\n\n\n" {
		t.Fatalf("instruction %q", p.parsed.fields["processing-instruction"].value.Value)
	}
}

func TestCompileListItemCompleteSourceMembership(t *testing.T) {
	prefix := "---\ncollaboration-mode: chat\n---\n# Chat\n"
	cases := []struct{ name, item, protected string }{{"nonempty fence", "- Why?\n\n  ```\n  example\n  ```\n", "```"}, {"empty fence", "- Why?\n\n  ```\n  ```\n", "```"}, {"fence only", "- ```\n  ```\n", "```"}, {"thematic break", "- Why?\n\n  ---\n", "---"}, {"empty heading", "- Why?\n\n  #\n", "#"}, {"empty item", "-\n", "-"}, {"nested empty fence", "- Why?\n  - Nested\n\n    ```\n    ```\n", "```"}, {"lazy continuation", "- Why?\nlazy continuation\n\n  ```\n  ```\n", "```"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := prefix + tc.item + "\n# Next\nordinary\n"
			p, e := Compile([]byte(source))
			if e != nil {
				t.Fatal(e)
			}
			if len(p.ListItems) != 1 {
				t.Fatalf("items %+v", p.ListItems)
			}
			want := Range{len(prefix), len(prefix) + len(tc.item)}
			if p.ListItems[0] != want {
				t.Fatalf("item range %+v want %+v source=%q", p.ListItems[0], want, source[p.ListItems[0].Start:p.ListItems[0].End])
			}

			if tc.name == "empty item" {
				if items := selectChatItems([]byte(source), p); len(items) != 0 {
					t.Fatalf("empty Markdown bullet became a chat ask: %+v", items)
				}
				return
			}
			at := len(prefix) + strings.LastIndex(tc.item, tc.protected)
			_, e = ValidateEdits([]byte(source), p, []Change{{Start: at, End: at + len(tc.protected), New: "~~~"}})
			var pe *PolicyError
			if e == nil || !errors.As(e, &pe) || pe.Code != "editorial-chat" {
				t.Fatalf("list syntax modification admitted: %v", e)
			}
		})
	}
}

func TestCompileCompleteListSourceFinalAnswer(t *testing.T) {
	source := "# Chat\n<!-- collaboration-mode: chat -->\n- Why?\n\n  ```\n  example\n  ```\n\n  Last continuation.\n\n# Next\nordinary\n"
	at := strings.Index(source, "Last continuation.") + len("Last continuation.")
	validateCase(t, source, []Change{{Start: at, End: at, New: " → [answer](note.side/answer.md)"}}, "")
}

func TestCompileRetiredSourceOwnership(t *testing.T) {
	cases := []string{
		"---\nprocessing-mode: free-form\nprocessing-mode: unknown\n---\n# A\n",
		"---\r\n'title': [opaque\r\n'presentation-mode': direct\r\n\"collaboration-mode\": chat\r\n---\r\n# A\r\n",
		"---\npresentation-mode: *unknown\nprocessing-instruction: [invalid\n---\n# A\n",
	}
	for _, source := range cases {
		policy, err := Compile([]byte(source))
		if err != nil {
			t.Fatalf("retired setting refused: %v", err)
		}
		if len(policy.Owned) != 2 {
			t.Fatalf("lost owned settings: %+v", policy.Owned)
		}
		for _, owned := range policy.Owned {
			if string(owned.Raw) != source[owned.Range.Start:owned.Range.End] {
				t.Fatalf("owned bytes changed: %+v", owned)
			}
			at := owned.Range.Start
			_, err = ValidateEdits([]byte(source), policy, []Change{{Start: at, End: at + 1, New: "x"}})
			var pe *PolicyError
			if !errors.As(err, &pe) || pe.Code != "operator-owned" {
				t.Fatalf("retired ownership change admitted: %v", err)
			}
		}
	}
}
