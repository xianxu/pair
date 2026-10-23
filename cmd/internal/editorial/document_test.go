package editorial

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileInheritance(t *testing.T) {
	c, _ := DefaultCatalog()
	p, err := Compile([]byte("---\npresentation-mode: direct\n---\nintro\n# A\n<!-- collaboration-mode: chat -->\none\n## B\ntwo\n# C\nthree\n"), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Headings) != 3 || p.Regions[2].Effective.Collaboration != "chat" || p.Regions[3].Effective.Collaboration != "discussion" {
		t.Fatalf("%+v", p)
	}
}
func TestParseOwnedSourceDuplicate(t *testing.T) {
	if _, err := parseOwnedSource([]byte("---\nprocessing-mode: free-form\nprocessing-mode: proofreading\n---\n")); err == nil {
		t.Fatal("accepted duplicate")
	}
}
func TestFillCanonicalPinned(t *testing.T) {
	c, _ := DefaultCatalog()
	paths, _ := filepath.Glob("testdata/contract/examples/editorial-fill-*/before/note.md")
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(path))), func(t *testing.T) {
			before, _ := os.ReadFile(path)
			want, _ := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(path)), "after/note.md"))
			got, err := FillCanonical(before, c)
			if bytes.Contains(before, []byte("[copy-edit][fact-check]")) {
				if err == nil || got != nil {
					t.Fatal("multimode must refuse without candidate")
				}
				return
			}
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("err=%v\ngot %s\nwant %s", err, got, want)
			}
		})
	}
}

func TestCompileAdversarial(t *testing.T) {
	c, _ := DefaultCatalog()
	cases := []struct {
		name, src string
		bad       bool
		headings  int
	}{
		{"fences", "```md\n# Fake\n<!-- presentation-mode: typo -->\n```\n# Real\ntext\n", false, 1},
		{"detached", "# Real\n\n<!-- presentation-mode: typo -->\n", false, 1},
		{"ordinary", "# Real\n<!-- ordinary comment -->\n", false, 1},
		{"duplicate", "# Real\n<!-- presentation-mode: direct; presentation-mode: propose -->\n", true, 0},
		{"malformed", "# Real\n<!-- presentation-mode : direct -->\n", true, 0},
		{"multiple", "# Real\n<!-- presentation-mode: direct -->\n<!-- processing-mode: free-form -->\n", true, 0},
		{"empty", "# Real\n<!-- presentation-mode:  -->\n", true, 0},
		{"legacy selector only", "---\nprocess-instruction: [copy-edit]\n---\n# Real\n", false, 1},
		{"legacy invalid whole yaml", "---\nprocess-instruction: [copy-edit] Keep this: exactly.\nopaque: [invalid\n---\n# Real\n", false, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, e := Compile([]byte(tc.src), c)
			if (e != nil) != tc.bad {
				t.Fatalf("err=%v", e)
			}
			if !tc.bad && len(p.Headings) != tc.headings {
				t.Fatalf("headings %+v", p.Headings)
			}
		})
	}
}
func TestFillPreservesCRLFAndOpaque(t *testing.T) {
	c, _ := DefaultCatalog()
	src := []byte("---\r\ntitle: [opaque\r\npresentation-mode: direct\r\n---\r\n# Text\r\n")
	got, e := FillCanonical(src, c)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(got, []byte("title: [opaque\r\npresentation-mode: direct\r\n")) || bytes.Contains(bytes.ReplaceAll(got, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatalf("%q", got)
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
	c, _ := DefaultCatalog()
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 65536 {
			t.Skip()
		}
		p, e := Compile([]byte(s), c)
		if e != nil {
			return
		}
		for _, o := range p.Owned {
			if o.Range.Start < 0 || o.Range.End > len(s) || o.Range.Start > o.Range.End || !bytes.Equal(o.Raw, []byte(s[o.Range.Start:o.Range.End])) {
				t.Fatal("owned range drift")
			}
		}
		filled, e := FillCanonical([]byte(s), c)
		if e == nil {
			again, e := FillCanonical(filled, c)
			if e != nil || !bytes.Equal(filled, again) {
				t.Fatal("fill not idempotent", e)
			}
		}
	})
}

func TestCompileGrowth(t *testing.T) {
	c, _ := DefaultCatalog()
	for _, n := range []int{100, 200} {
		var src bytes.Buffer
		src.WriteString("---\ncollaboration-mode:\n  chat:\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&src, "    - H%d\n", i)
		}
		src.WriteString("---\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&src, "# H%d\nbody\n", i)
		}
		p, e := Compile(src.Bytes(), c)
		if e != nil {
			t.Fatal(e)
		}
		if p.Stats.HeadingVisits != n || p.Stats.LegacyAssignments != n || p.Stats.LineAdvances > len(sourceLines(src.Bytes())) {
			t.Fatalf("nonlinear own traversal: %+v", p.Stats)
		}
	}
}
func TestATXNamePreservesLiteralHash(t *testing.T) {
	c, _ := DefaultCatalog()
	p, e := Compile([]byte("# C#\n# C ##\n"), c)
	if e != nil {
		t.Fatal(e)
	}
	if p.Headings[0].Name != "C#" || p.Headings[1].Name != "C" {
		t.Fatalf("%+v", p.Headings)
	}
}
func TestQuotedOwnedKeysAndCodeComments(t *testing.T) {
	c, _ := DefaultCatalog()
	src := []byte("---\n'presentation-mode': direct\n\"collaboration-mode\": chat\n---\n# A\n    <!-- presentation-mode: typo -->\n")
	p, e := Compile(src, c)
	if e != nil {
		t.Fatal(e)
	}
	if p.Defaults.Presentation != "direct" || p.Defaults.Collaboration != "chat" || len(p.Owned) != 2 {
		t.Fatalf("%+v", p)
	}
	if _, e := Compile([]byte("---\n'presentation-mode': direct\npresentation-mode: propose\n---\n"), c); e == nil {
		t.Fatal("quoted duplicate accepted")
	}
	if _, e := Compile([]byte("---\npresentation-mode: *unknown\n---\n"), c); e == nil {
		t.Fatal("unresolved alias accepted")
	}
}
func TestAncestorCommentOutranksLegacyName(t *testing.T) {
	c, _ := DefaultCatalog()
	p, e := Compile([]byte("---\ncollaboration-mode: {chat: [Child]}\n---\n# Parent\n<!-- collaboration-mode: collaboration -->\n## Child\ntext\n"), c)
	if e != nil {
		t.Fatal(e)
	}
	if p.Regions[2].Effective.Collaboration != "collaboration" {
		t.Fatal("legacy map overrode inherited comment")
	}
}
func TestCompileCodeMasks(t *testing.T) {
	c, _ := DefaultCatalog()
	src := []byte("# A `literal`\n\n    indented 🤖[ask]\n\n```lang\nfenced 🤖[ask]\n```\n")
	p, e := Compile(src, c)
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
func TestCompilerReviewBoundaryMutations(t *testing.T) {
	c, _ := DefaultCatalog()
	for _, bad := range []string{"<!-- presentation-mode\t: direct -->", "<!-- presentation-mode direct -->"} {
		if _, e := Compile([]byte("# A\n"+bad+"\n"), c); e == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	p, e := Compile([]byte("---\n  presentation-mode: direct\n  collaboration-mode: chat\n---\n# A\n"), c)
	if e != nil || p.Defaults.Presentation != "direct" || p.Defaults.Collaboration != "chat" || len(p.Owned) != 2 {
		t.Fatalf("p=%+v e=%v", p, e)
	}
	p, e = Compile([]byte("---\nprocessing-mode: free-form\nprocessing-instruction: canonical\nprocess-instruction: '[invalid] obsolete'\n---\n"), c)
	if e != nil || p.Defaults.Instruction != "canonical" {
		t.Fatalf("canonical override failed: %v", e)
	}
	p, e = Compile([]byte("---\nprocessing-mode: free-form\nprocess-instruction: '[invalid] obsolete'\n---\n"), c)
	if e != nil || p.Defaults.Instruction != "obsolete" {
		t.Fatalf("independent instruction fallback failed: %v", e)
	}
	if _, e := Compile([]byte("---\nprocessing-instruction: canonical\nprocess-instruction: '[invalid] obsolete'\n---\n"), c); e == nil {
		t.Fatal("unresolved legacy mode accepted")
	}
	for _, transform := range []func([]byte, *Catalog) ([]byte, error){FillCanonical, func(b []byte, c *Catalog) ([]byte, error) {
		m, e := PlanMigration(b, c)
		if e != nil {
			return nil, e
		}
		return m.After, nil
	}} {
		got, e := transform([]byte("---\nprocessing-mode: free-form\nprocessing-instruction: canonical\nprocess-instruction: '[invalid][also-invalid] obsolete'\n---\n"), c)
		if e == nil || got != nil {
			t.Fatal("overridden multimode fill returned candidate")
		}
	}

}
func TestFillMixedNewlineAndIndentedHeader(t *testing.T) {
	c, _ := DefaultCatalog()
	for _, src := range []string{"---\ntitle: keep\n---\n# A\r\nbody\r\n", "---\r\ntitle: keep\r\n---\r\n# A\nbody\n", "---\n  presentation-mode: direct\n  title: keep\n---\n# A\n"} {
		filled, e := FillCanonical([]byte(src), c)
		if e != nil {
			t.Fatal(e)
		}
		again, e := FillCanonical(filled, c)
		if e != nil || !bytes.Equal(filled, again) {
			t.Fatalf("invalid/non-idempotent fill %q err=%v", filled, e)
		}
	}
	if _, e := Compile([]byte("# A\n<!-- Tip: use presentation-mode: direct for quiet edits -->\n"), c); e != nil {
		t.Fatal(e)
	}
}

// generatedTree builds its oracle as it emits headings; it never resolves source
// through Compile to obtain expected inheritance.
func generatedTree(data []byte) ([]byte, [][3]string) {
	if len(data) == 0 {
		data = []byte{0}
	}
	var src bytes.Buffer
	src.WriteString("---\ncollaboration-mode: discussion\npresentation-mode: highlight\nprocessing-mode: free-form\n---\npreamble\n")
	expected := [][3]string{{"discussion", "highlight", "free-form"}}
	type frame struct {
		level int
		axes  [3]string
	}
	stack := []frame{}
	collabs := []string{"discussion", "collaboration", "chat"}
	presentations := []string{"highlight", "direct", "propose"}
	processing := []string{"free-form", "proofreading", "fact-check"}
	for i, b := range data {
		level := int(b%6) + 1
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}
		axes := [3]string{"discussion", "highlight", "free-form"}
		if len(stack) > 0 {
			axes = stack[len(stack)-1].axes
		}
		fmt.Fprintf(&src, "%s H%d\n", strings.Repeat("#", level), i)
		decl := []string{}
		if b&8 != 0 {
			axes[0] = collabs[int(b/3)%3]
			decl = append(decl, "collaboration-mode: "+axes[0])
		}
		if b&16 != 0 {
			axes[1] = presentations[int(b/5)%3]
			decl = append(decl, "presentation-mode: "+axes[1])
		}
		if b&32 != 0 {
			axes[2] = processing[int(b/7)%3]
			decl = append(decl, "processing-mode: "+axes[2])
		}
		if len(decl) > 0 {
			fmt.Fprintf(&src, "<!-- %s -->\n", strings.Join(decl, "; "))
		}
		src.WriteString("body\n")
		expected = append(expected, axes)
		stack = append(stack, frame{level, axes})
	}
	return src.Bytes(), expected
}
func assertGeneratedInheritance(t *testing.T, data []byte) {
	t.Helper()
	c, _ := DefaultCatalog()
	src, want := generatedTree(data)
	p, e := Compile(src, c)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Regions) != len(want) {
		t.Fatal("region count mismatch")
	}
	for i, axes := range want {
		got := p.Regions[i].Effective
		if got.Collaboration != axes[0] || got.Presentation != axes[1] || got.Processing != axes[2] {
			t.Fatalf("region %d got %+v want %v", i, got, axes)
		}
	}
}
func TestCompileIndependentInheritanceOracle(t *testing.T) {
	for seed := int64(0); seed < 40; seed++ {
		data := make([]byte, 100)
		rand.New(rand.NewSource(seed)).Read(data)
		assertGeneratedInheritance(t, data)
	}
}
func FuzzCompileGeneratedInheritance(f *testing.F) {
	f.Add([]byte{63, 2, 31, 1, 18, 0})
	paths, _ := filepath.Glob("testdata/contract/examples/editorial-*/before/*.md")
	for _, path := range paths {
		raw, e := os.ReadFile(path)
		if e == nil {
			f.Add(raw)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 512 {
			t.Skip()
		}
		assertGeneratedInheritance(t, data)
	})
}
func TestCompileBlockAndListFacts(t *testing.T) {
	c, _ := DefaultCatalog()
	src := []byte("# Chat\n- Ask\n  - Nested\n\n- Second\n\n```\n- code\n```\n")
	p, e := Compile(src, c)
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
	c, _ := DefaultCatalog()
	for _, tc := range []struct{ header, want string }{{"  processing-instruction: |2-\n    line one\n    line two\n", "line one\nline two"}, {"  processing-instruction: |2\n    line one\n    line two\n", "line one\nline two\n"}, {"  processing-instruction: >2-\n    line one\n    line two\n", "line one line two"}, {"processing-instruction: |2-\n  line one\n    indented\n", "line one\n  indented"}} {
		p, e := Compile([]byte("---\n"+tc.header+"---\n"), c)
		if e != nil || p.Defaults.Instruction != tc.want {
			t.Fatalf("%q instruction=%q want=%q err=%v", tc.header, p.Defaults.Instruction, tc.want, e)
		}
	}
}
func TestOwnedBlockScalarKeepChomp(t *testing.T) {
	c, _ := DefaultCatalog()
	p, e := Compile([]byte("---\n  processing-instruction: |2+\n    line\n\n\n---\n"), c)
	if e != nil {
		t.Fatal(e)
	}
	if p.Defaults.Instruction != "line\n\n\n" {
		t.Fatalf("instruction %q", p.Defaults.Instruction)
	}
}

func TestCompileListItemCompleteSourceMembership(t *testing.T) {
	c, _ := DefaultCatalog()
	prefix := "---\ncollaboration-mode: chat\n---\n# Chat\n"
	cases := []struct{ name, item, protected string }{{"nonempty fence", "- Why?\n\n  ```\n  example\n  ```\n", "```"}, {"empty fence", "- Why?\n\n  ```\n  ```\n", "```"}, {"fence only", "- ```\n  ```\n", "```"}, {"thematic break", "- Why?\n\n  ---\n", "---"}, {"empty heading", "- Why?\n\n  #\n", "#"}, {"empty item", "-\n", "-"}, {"nested empty fence", "- Why?\n  - Nested\n\n    ```\n    ```\n", "```"}, {"lazy continuation", "- Why?\nlazy continuation\n\n  ```\n  ```\n", "```"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := prefix + tc.item + "\n# Next\nordinary\n"
			p, e := Compile([]byte(source), c)
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
			at := len(prefix) + strings.LastIndex(tc.item, tc.protected)
			_, e = ValidateEdits([]byte(source), p, []Change{{Start: at, End: at + len(tc.protected), New: "~~~"}}, c)
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
