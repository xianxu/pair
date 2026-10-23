package editorial

import (
	"bytes"
	"errors"
	"fmt"
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
