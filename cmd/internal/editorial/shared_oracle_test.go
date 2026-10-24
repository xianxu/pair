package editorial

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/sidedoc"
)

// This oracle uses a smallest-extent search over the shared facts, independently
// of Compile's stack sweep and region coalescing. The shared owner alone parses
// tags, legacy declarations and item grammar.
func assertSharedIntervals(t *testing.T, source []byte) {
	t.Helper()
	scan := sidedoc.ChatSections(string(source))
	policy, err := Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Stats.SharedScans != 1 {
		t.Fatalf("snapshot scans: %+v", policy.Stats)
	}
	cursor := 0
	for _, region := range policy.Regions {
		if region.Range.Start != cursor || region.Range.End < cursor || region.Range.End > len(source) {
			t.Fatalf("nonpartition region: %+v, cursor %d", region, cursor)
		}
		cursor = region.Range.End
	}
	if cursor != len(source) {
		t.Fatalf("coverage %d/%d", cursor, len(source))
	}
	for line, r := range sourceLines(source) {
		var owner *sidedoc.ChatSection
		for i := range scan.Sections {
			section := &scan.Sections[i]
			if section.Heading < line+1 && section.End >= line+1 && (owner == nil || section.Heading > owner.Heading) {
				owner = section
			}
		}
		for _, region := range policy.Regions {
			if region.Range.Start <= r.Start && r.Start < region.Range.End {
				want := "collaboration"
				legacy := false
				if owner != nil {
					want, legacy = "chat", owner.Legacy
				}
				if region.Effective.Collaboration != want || region.Effective.Legacy != legacy || region.Effective.Presentation != "highlight" {
					t.Fatalf("line %d got %+v want %s legacy=%t\n%s", line+1, region.Effective, want, legacy, source)
				}
				break
			}
		}
	}
	for _, owned := range policy.Owned {
		if owned.Range.Start < 0 || owned.Range.End > len(source) || !bytes.Equal(owned.Raw, source[owned.Range.Start:owned.Range.End]) {
			t.Fatalf("raw ownership drift: %+v", owned)
		}
	}
}

func generatedSharedDocument(data []byte) []byte {
	var source strings.Builder
	source.WriteString("---\npresentation-mode: retired-unknown\n---\npreamble λ\n")
	for i, b := range data {
		if b&8 != 0 {
			source.WriteString("@@chat@@\n")
		}
		fmt.Fprintf(&source, "%s H%d\n", strings.Repeat("#", int(b%6)+1), i)
		if b&16 != 0 {
			source.WriteString("<!-- collaboration-mode: chat -->\n")
		}
		fmt.Fprintf(&source, "1. numbered%d\n\n- bullet%d\n\nbody λ\n", i, i)
		if b&32 != 0 {
			source.WriteString("```md\n@@chat@@\n# Fake\n1. code\n```\n")
		}
	}
	return []byte(source.String())
}

func TestCompileSharedIntervalOracle(t *testing.T) {
	for seed := int64(0); seed < 40; seed++ {
		data := make([]byte, 100)
		_, _ = rand.New(rand.NewSource(seed)).Read(data)
		assertSharedIntervals(t, generatedSharedDocument(data))
	}
}

func FuzzCompileSharedIntervalOracle(f *testing.F) {
	f.Add([]byte{63, 2, 31, 1, 18, 0})
	f.Add([]byte{0, 8, 16, 24, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			t.Skip()
		}
		assertSharedIntervals(t, generatedSharedDocument(data))
	})
}

func TestCompileSharedItemGrammar(t *testing.T) {
	cases := []struct {
		name, source string
		want         []string
	}{
		{"tag only", "@@chat@@\n# A\n1. numbered\n\n- bullet\n", []string{"1. numbered"}},
		{"tag and legacy", "@@chat@@\n# A\n<!-- collaboration-mode: chat -->\n1. numbered\n\n- bullet\n", []string{"1. numbered", "- bullet"}},
		{"nested tag overrides legacy grammar", "# A\n<!-- collaboration-mode: chat -->\n- outer\n\n@@chat@@\n## B\n- ignored\n\n1. inner\n\n# C\n- ordinary\n", []string{"- outer", "1. inner"}},
		{"whole legacy retains tags and preamble", "---\ncollaboration-mode: chat\n---\n- preamble\n\n@@chat@@\n# A\n- bullet\n", []string{"- preamble", "- bullet"}},
		{"code does not introduce items", "@@chat@@\n# A\n```\n1. hidden\n~~~\n2. hidden too\n```\n\n1. visible\n", []string{"1. visible"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.source)
			policy, err := Compile(source)
			if err != nil {
				t.Fatal(err)
			}
			items := selectChatItems(source, policy)
			if len(items) != len(tc.want) {
				t.Fatalf("items %+v, want %v", items, tc.want)
			}
			for i, item := range items {
				got := strings.TrimSpace(string(source[item.Range.Start:item.Range.End]))
				if got != tc.want[i] {
					t.Fatalf("item %d %q want %q", i, got, tc.want[i])
				}
			}
		})
	}
}

func TestCompileAdapterGrowth(t *testing.T) {
	for _, n := range []int{8, 16, 128, 256} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var source strings.Builder
			for i := 0; i < n; i++ {
				fmt.Fprintf(&source, "@@chat@@\n# H%d\n1. ask%d\n\n", i, i)
			}
			raw := []byte(source.String())
			policy, err := Compile(raw)
			if err != nil {
				t.Fatal(err)
			}
			stats := policy.Stats
			if stats.SharedScans != 1 || stats.HeadingVisits != n || stats.SectionEvents > 2*n || stats.ItemJoins > 2*n || stats.ItemSourceBytes > len(raw) || stats.LineAdvances > len(sourceLines(raw)) || stats.ListSourceEvents > 8*len(sourceLines(raw)) {
				t.Fatalf("adapter work exceeded linear bound: n=%d bytes=%d stats=%+v", n, len(raw), stats)
			}
			if len(selectChatItems(raw, policy)) != n {
				t.Fatalf("lost asks: %+v", stats)
			}
		})
	}
}
