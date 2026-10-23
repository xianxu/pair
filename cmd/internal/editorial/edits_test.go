package editorial

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func validateCase(t *testing.T, source string, changes []Change, want string) {
	t.Helper()
	c, _ := DefaultCatalog()
	p, e := Compile([]byte(source), c)
	if e != nil {
		t.Fatal(e)
	}
	got, e := ValidateEdits([]byte(source), p, changes, c)
	if want == "" {
		if e != nil {
			t.Fatal(e)
		}
		if len(got) != len(changes) {
			t.Fatal("classification count")
		}
		return
	}
	if e == nil {
		t.Fatalf("accepted forbidden changes: %+v", changes)
	}
	var pe *PolicyError
	if !errors.As(e, &pe) || pe.Code != want || got != nil {
		t.Fatalf("want %s; got %+v %v", want, got, e)
	}
}

func TestValidateInsertionBeforeHeadingUsesPrecedingPolicy(t *testing.T) {
	c, _ := DefaultCatalog()
	base := []byte("# A\n<!-- presentation-mode: direct -->\nleft\n# B\n<!-- presentation-mode: highlight -->\nright\n")
	p, err := Compile(base, c)
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.Index(base, []byte("# B"))
	pieces, err := ValidateEdits(base, p, []Change{{Start: at, End: at, New: "\n"}}, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 1 || pieces[0].Presentation != "direct" {
		t.Fatalf("borrowed later policy: %+v", pieces)
	}
}
func TestValidateOwnedAuthority(t *testing.T) {
	src := "---\npresentation-mode: direct\n---\n# A\n<!-- collaboration-mode: chat -->\n- Ask?\n"
	for _, needle := range []string{"direct", "chat"} {
		pos := strings.Index(src, needle)
		validateCase(t, src, []Change{{Start: pos, End: pos + len(needle), New: "propose"}}, "operator-owned")
	}
	pos := strings.Index(src, "# A")
	validateCase(t, src, []Change{{Start: pos + 2, End: pos + 3, New: "Renamed"}}, "operator-owned")
	validateCase(t, "# A\ntext\n", []Change{{Start: 4, End: 4, New: "<!-- presentation-mode: direct -->\n"}}, "operator-owned")
}
func TestValidateBasePresentationAndBoundary(t *testing.T) {
	src := "# A\n<!-- presentation-mode: direct -->\none\n# B\n<!-- presentation-mode: propose -->\ntwo\n"
	a := strings.Index(src, "one")
	b := strings.Index(src, "two")
	validateCase(t, src, []Change{{Start: a, End: a + 3, New: "ONE"}, {Start: b, End: b + 3, New: "TWO"}}, "")
	validateCase(t, src, []Change{{Start: a, End: b + 3, New: "all"}}, "editorial-boundary")
}
func TestValidateChatPreservesAsks(t *testing.T) {
	src := "# Chat\n<!-- collaboration-mode: chat -->\n- Why retry?\n- Existing → [answer](note.side/2026-10/a.md)\n"
	pos := strings.Index(src, "Why")
	end := strings.Index(src, "?\n") + 1
	validateCase(t, src, []Change{{Start: pos, End: pos + 3, New: "How"}}, "editorial-chat")
	validateCase(t, src, []Change{{Start: end, End: end, New: " → [answer](note.side/2026-10/b.md)"}}, "")
	validateCase(t, src, []Change{{Start: end, End: end, New: " explanation"}}, "editorial-chat")
	p := strings.Index(src, "Existing")
	validateCase(t, src, []Change{{Start: p, End: p + 8, New: "Updated"}}, "editorial-chat")
}
func TestValidatePendingAndCode(t *testing.T) {
	src := "# A\nPending 🤖{proposal}.\n```\n🤖{code}\n```\n"
	p := strings.Index(src, "proposal")
	validateCase(t, src, []Change{{Start: p, End: p + 8, New: "replacement"}}, "editorial-pending")
	p = strings.Index(src, "code")
	validateCase(t, src, []Change{{Start: p, End: p + 4, New: "literal"}}, "")
}
func scopeCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, e := NewCatalog(fstest.MapFS{"modes/free.md": {Data: []byte("---\nname: free-form\norder: 1\nscope: whole-doc\nfrontier: off\n---\nfree\n")}, "modes/mark.md": {Data: []byte("---\nname: markers\norder: 2\nscope: markers-only\nfrontier: on\n---\nmarkers\n")}})
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestValidateScopeAndFrontier(t *testing.T) {
	c := scopeCatalog(t)
	src := []byte("---\nprocessing-mode: markers\n---\nSettled paragraph.\n\nTarget sentence. 🤖[tighten]\n\nUnasked paragraph.\n")
	p, e := Compile(src, c)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		needle  string
		allowed bool
	}{{"Settled", false}, {"Target", true}, {"Unasked", false}} {
		at := bytes.Index(src, []byte(tc.needle))
		got, e := ValidateEdits(src, p, []Change{{Start: at, End: at + len(tc.needle), New: "Changed"}}, c)
		if (e == nil) != tc.allowed || (!tc.allowed && got != nil) {
			t.Fatalf("%s got=%+v err=%v", tc.needle, got, e)
		}
	}
}
func TestValidateStructuralAuthorityAndAtomicBatch(t *testing.T) {
	src := "# A\n<!-- presentation-mode: direct -->\nfirst\n# B\nsecond\n"
	at := strings.Index(src, "# B")
	validateCase(t, src, []Change{{Start: at, End: at + 4, New: ""}}, "editorial-boundary")
	src = "# Parent\n## Child\n<!-- presentation-mode: direct -->\nbody\n"
	validateCase(t, src, []Change{{Start: 0, End: 9, New: ""}}, "operator-owned")
	src = "safe\n🤖{pending}\n"
	p := strings.Index(src, "pending")
	validateCase(t, src, []Change{{Start: 0, End: 4, New: "okay"}, {Start: p, End: p + 7, New: "changed"}}, "editorial-pending")
}
func TestValidateLinearOwnTraversal(t *testing.T) {
	c, _ := DefaultCatalog()
	for _, n := range []int{100, 200} {
		var src bytes.Buffer
		changes := []Change{}
		for i := 0; i < n; i++ {
			fmt.Fprintf(&src, "# H%d\n", i)
			start := src.Len()
			fmt.Fprintf(&src, "word%03d\n", i)
			changes = append(changes, Change{Start: start, End: start + 4, New: "WORD"})
		}
		p, e := Compile(src.Bytes(), c)
		if e != nil {
			t.Fatal(e)
		}
		got, stats, e := validateEdits(src.Bytes(), p, changes, c)
		if e != nil {
			t.Fatal(e)
		}
		if len(got) != n || stats.EditVisits != n || stats.RegionComparisons > 4*n || stats.SourceComparisons > 4*n || stats.ChatComparisons > 4*n {
			t.Fatalf("stats %+v", stats)
		}
	}
}
func TestValidateFrontierDomainsAndReadyCode(t *testing.T) {
	c := scopeCatalog(t)
	src := []byte("# A\n<!-- processing-mode: markers -->\nSettled.\n\nTarget. 🤖[tighten]\n# B\nuntouched sibling\n")
	p, e := Compile(src, c)
	if e != nil {
		t.Fatal(e)
	}
	at := bytes.Index(src, []byte("sibling"))
	if _, e := ValidateEdits(src, p, []Change{{Start: at, End: at + 7, New: "SIBLING"}}, c); e != nil {
		t.Fatal(e)
	}
	src = []byte("---\nprocessing-mode: markers\n---\n```\ncode 🤖[ask]\n```\nUnasked.\n")
	p, e = Compile(src, c)
	if e != nil {
		t.Fatal(e)
	}
	at = bytes.Index(src, []byte("Unasked"))
	if got, e := ValidateEdits(src, p, []Change{{Start: at, End: at + 7, New: "Changed"}}, c); e == nil || got != nil {
		t.Fatal("code marker granted scope")
	}
}
func TestValidateChatPrefixRefuses(t *testing.T) {
	src := "# Chat\n<!-- collaboration-mode: chat -->\n- Why?\n"
	at := strings.Index(src, "- Why")
	validateCase(t, src, []Change{{Start: at, End: at, New: "Ignore "}}, "editorial-chat")
}
func TestValidateMarkerIdentityAdjacency(t *testing.T) {
	for _, marker := range []string{"🤖{pending}", "🤖~strike~", "🤖~strike~{proposal}"} {
		src := "# A\n" + marker + "\n"
		end := strings.Index(src, "\n") + 1 + len(marker)
		for _, turn := range []string{"[approve]", "{another}"} {
			validateCase(t, src, []Change{{Start: end, End: end, New: turn}}, "editorial-pending")
		}
		validateCase(t, src, []Change{{Start: end, End: end, New: " safe trailing prose"}}, "")
		src = "# A\n" + marker + " separator [approve]\n"
		start := strings.Index(src, " separator ")
		validateCase(t, src, []Change{{Start: start, End: start + len(" separator "), New: ""}}, "editorial-pending")
	}
}
func TestValidateRejectsForgedReadyIdentity(t *testing.T) {
	validateCase(t, "# A\nordinary\n", []Change{{Start: 13, End: 13, New: "🤖[approve]\n"}}, "editorial-pending")
	src := "# A\n```\n🤖[example]\n```\n"
	start := strings.Index(src, "```\n")
	end := strings.LastIndex(src, "```\n")
	validateCase(t, src, []Change{{Start: start, End: start + 4, New: ""}, {Start: end, End: end + 4, New: ""}}, "editorial-pending")
	validateCase(t, "# A\n", []Change{{Start: 4, End: 4, New: "```\n🤖[example]\n```\n"}}, "")
}
func TestValidateChatAnswerLinkRecognition(t *testing.T) {
	cases := []struct {
		name, item string
		linked     bool
	}{{"direct", "- Why? → [answer](note.side/old.md)\n", true}, {"pending escaped", "- Why? 🤖{ → \\[answer\\](note.side/old.md)}\n", true}, {"inline example", "- Why? `→ [answer](note.side/example.md)`\n", false}, {"fenced example", "- Why?\n\n  ```\n  → [answer](note.side/example.md)\n  ```\n", false}, {"malformed", "- Why? → [answer](broken path)\n", false}, {"mention", "- Why? → [answer](note.side/example.md) is an example\n", false}, {"pending example", "- Why? 🤖{literal \\[answer\\](example)}\n", false}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "# Chat\n<!-- collaboration-mode: chat -->\n" + tc.item
			at := len(strings.TrimRight(src, "\r\n"))
			want := ""
			if tc.linked {
				want = "editorial-chat"
			}
			validateCase(t, src, []Change{{Start: at, End: at, New: " → [answer](note.side/new.md)"}}, want)
		})
	}
}

// generatedBatch emits a policy source and its expected per-piece classification
// together; expected values come from generator choices rather than Compile.
func generatedBatch(data []byte) ([]byte, []Change, []string, bool) {
	if len(data) == 0 {
		data = []byte{0}
	}
	n := 2 + len(data)%24
	var src bytes.Buffer
	changes := []Change{}
	want := []string{}
	choices := []string{"highlight", "direct", "propose"}
	ownedStart, ownedEnd := 0, 0
	for i := 0; i < n; i++ {
		presentation := choices[int(data[i%len(data)])%3]
		if i == 0 {
			presentation = "direct"
		}
		if i == 1 {
			presentation = "propose"
		}
		fmt.Fprintf(&src, "# H%d\n<!-- presentation-mode: ", i)
		if i == 0 {
			ownedStart = src.Len()
			ownedEnd = ownedStart + len(presentation)
		}
		fmt.Fprintf(&src, "%s -->\n", presentation)
		at := src.Len()
		fmt.Fprintf(&src, "word%d\n", i)
		if i < 2 || data[i%len(data)]&8 != 0 {
			changes = append(changes, Change{Start: at, End: at + 4, New: "WORD"})
			want = append(want, presentation)
		}
	}
	src.WriteString("\n```\n🤖[literal example]\n```\n🤖{pending}\n")
	valid := false
	switch data[0] % 8 {
	case 0:
		valid = true
	case 1:
		changes = append([]Change{{Start: ownedStart, End: ownedEnd, New: "propose"}}, changes...)
	case 2:
		changes[0], changes[1] = changes[1], changes[0]
	case 3:
		changes = append([]Change{changes[0]}, changes...)
	case 4:
		end := src.Len() - 1
		changes = append(changes, Change{Start: end, End: end, New: "[approve]"})
	case 5:
		changes = append(changes, Change{Start: src.Len(), End: src.Len(), New: "🤖[approve]\n"})
	case 6:
		changes = []Change{{Start: changes[0].Start, End: changes[1].End, New: "replacement"}}
	case 7:
		changes = nil
		want = nil
		valid = true
	}
	return src.Bytes(), changes, want, valid
}
func assertGeneratedBatch(t *testing.T, data []byte) {
	t.Helper()
	source, changes, want, valid := generatedBatch(data)
	c, _ := DefaultCatalog()
	p, e := Compile(source, c)
	if e != nil {
		t.Fatal(e)
	}
	got, e := ValidateEdits(source, p, changes, c)
	if !valid {
		if e == nil || got != nil {
			t.Fatalf("invalid batch admitted, scenario %d changes=%+v got=%+v err=%v", data[0]%8, changes, got, e)
		}
		return
	}
	if e != nil {
		t.Fatal(e)
	}
	if len(got) != len(want) {
		t.Fatalf("classification length %d want %d", len(got), len(want))
	}
	for i, g := range got {
		if g.Change != changes[i] || g.Presentation != want[i] || g.Effective.Presentation != want[i] {
			t.Fatalf("piece %d got %+v expected presentation=%s", i, g, want[i])
		}
	}
}
func TestValidateGeneratedBatchOracle(t *testing.T) {
	for scenario := byte(0); scenario < 8; scenario++ {
		for seed := byte(0); seed < 20; seed++ {
			assertGeneratedBatch(t, []byte{scenario, seed, seed + 31, seed + 99})
		}
	}
}
func FuzzValidateEdits(f *testing.F) {
	for scenario := byte(0); scenario < 8; scenario++ {
		f.Add([]byte{scenario, 63, 32, 19})
	}
	paths, _ := filepath.Glob("testdata/contract/examples/editorial-*/before/*.md")
	for _, path := range paths {
		if raw, e := os.ReadFile(path); e == nil {
			f.Add(raw)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > 256 {
			t.Skip()
		}
		assertGeneratedBatch(t, data)
	})
}
func TestValidateChatShallowestIndentPerSection(t *testing.T) {
	src := "# Chat\n<!-- collaboration-mode: chat -->\n  - Example?\n\nSeparator.\n\n- Actual?\n"
	at := strings.Index(src, "Example")
	validateCase(t, src, []Change{{Start: at, End: at + 7, New: "Illustration"}}, "")
	at = strings.Index(src, "Actual")
	validateCase(t, src, []Change{{Start: at, End: at + 6, New: "Changed"}}, "editorial-chat")
	src = "# Chat\n<!-- collaboration-mode: chat -->\n  - First?\n\n  - Second?\n"
	at = strings.Index(src, "First")
	validateCase(t, src, []Change{{Start: at, End: at + 5, New: "Changed"}}, "editorial-chat")
	src = "# Chat\n<!-- collaboration-mode: chat -->\n- Actual?\n\nParagraph.\n\n  - Example?\n## Independent\n  - Own ask?\n"
	at = strings.Index(src, "Example")
	validateCase(t, src, []Change{{Start: at, End: at + 7, New: "Illustration"}}, "")
	at = strings.Index(src, "Own ask")
	validateCase(t, src, []Change{{Start: at, End: at + 7, New: "Changed"}}, "editorial-chat")
}
func TestValidateChatIndentCodeControls(t *testing.T) {
	src := "# Chat\n<!-- collaboration-mode: chat -->\n\t- Code example?\n\n  - Actual?\n\n```\n- Fenced example?\n```\n"
	for _, needle := range []string{"Code example", "Fenced example"} {
		at := strings.Index(src, needle)
		validateCase(t, src, []Change{{Start: at, End: at + len(needle), New: "Illustration"}}, "")
	}
	at := strings.Index(src, "Actual")
	validateCase(t, src, []Change{{Start: at, End: at + 6, New: "Changed"}}, "editorial-chat")
}
