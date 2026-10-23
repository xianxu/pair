package editorial

import (
	"strings"
	"testing"
)

func TestScanMarkersMalformedLinear(t *testing.T) {
	for _, n := range []int{1000, 2000} {
		src := strings.Repeat("🤖[", n) + " later 🤖[valid]"
		got, stats := scanMarkers(src)
		if len(got) != 1 || !got[0].Ready || src[got[0].Range.Start:got[0].Range.End] != "🤖[valid]" {
			t.Fatalf("lost later valid marker: %+v", got)
		}
		if stats.Bytes > len(src) || stats.Robots > n+1 || stats.BracketLookups > 2*(n+1) {
			t.Fatalf("nonlinear scanner: %+v bytes=%d", stats, len(src))
		}
	}
}
func TestScanMarkerBracketEscapesAndCrossings(t *testing.T) {
	for _, src := range []string{`🤖[nested [one] and \] escaped]`, `🤖<quote [nested]>{answer}`, `🤖[cross { ] }`} {
		got := ScanMarkers(src)
		if len(got) != 1 {
			t.Fatalf("%q => %+v", src, got)
		}
	}
	if _, _, ok := ReadyMarkerSpan("🤖~struck~[note]"); ok {
		t.Fatal("strike changed readiness")
	}
}
