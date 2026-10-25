package sessionsync

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestApplyShadowProjection(t *testing.T) {
	raw, err := os.ReadFile("testdata/apply-shadow/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Shadow, Base, Next, Want string
		OK                             bool
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got, ok, _ := applyShadow(c.Shadow, c.Base, c.Next)
			if got != c.Want || ok != c.OK {
				t.Fatalf("got %q/%v want %q/%v", got, ok, c.Want, c.OK)
			}
		})
	}
}

func TestApplyShadowLinearMapping(t *testing.T) {
	var previousVisits, previousBytes int
	for _, n := range []int{8, 16} {
		var shadow, base, next strings.Builder
		for i := 0; i < n; i++ {
			fmt.Fprintf(&shadow, "heading%04d\nold%04d\nwork%04d\n", i, i, i)
			fmt.Fprintf(&base, "heading%04d\nunread%04d\nwork%04d\n", i, i, i)
			fmt.Fprintf(&next, "heading%04d\nunread%04d\ndone%04d\n", i, i, i)
		}
		got, ok, stats := applyShadow(shadow.String(), base.String(), next.String())
		if !ok || got != strings.ReplaceAll(shadow.String(), "work", "done") {
			t.Fatal("mapping failed")
		}
		if stats.SpanVisits > 4*n || stats.EditVisits != n || stats.CopiedBytes != len(got) {
			t.Fatalf("stats %+v", stats)
		}
		if previousVisits > 0 && (stats.SpanVisits > 2*previousVisits+2 || stats.CopiedBytes > 2*previousBytes) {
			t.Fatalf("superlinear %+v", stats)
		}
		previousVisits, previousBytes = stats.SpanVisits, stats.CopiedBytes
	}
}

func TestApplyShadowDiffCapDoesNotGrantAuthority(t *testing.T) {
	old := maxDiffEdits
	maxDiffEdits = 2
	defer func() { maxDiffEdits = old }()
	shadow := "prefix work\na\nb\nc\nd\nsuffix work\n"
	base := "prefix work\nw\nx\ny\nz\nsuffix work\n"
	for _, tc := range []struct {
		next, want string
		ok         bool
	}{
		{strings.Replace(base, "prefix work", "prefix done", 1), strings.Replace(shadow, "prefix work", "prefix done", 1), true},
		{strings.Replace(base, "x\n", "changed\n", 1), shadow, false},
	} {
		got, ok, _ := applyShadow(shadow, base, tc.next)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("%q %v", got, ok)
		}
	}
}

func FuzzApplyShadowPreservesUnread(f *testing.F) {
	f.Add([]byte{0, 1, 2}, true)
	f.Add([]byte{255, 0, 7}, false)
	f.Fuzz(func(t *testing.T, data []byte, crlf bool) {
		nl := "\n"
		if crlf {
			nl = "\r\n"
		}
		text := fmt.Sprintf("%x", data)
		shadow := "head" + nl + "old" + nl + "anchor" + nl + "work " + text + nl + "tail" + nl
		base := strings.Replace(shadow, "old"+nl, "operator 未読 "+text+nl, 1)
		next := strings.Replace(base, "work "+text, "done "+text+"追加", 1)
		want := strings.Replace(shadow, "work "+text, "done "+text+"追加", 1)
		got, ok, _ := applyShadow(shadow, base, next)
		if !ok || got != want {
			t.Fatalf("projection lost operator baseline: %q want %q mapped=%v", got, want, ok)
		}
	})
}
