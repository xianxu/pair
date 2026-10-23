package sessionsync

import (
	"os"
	"strings"
	"testing"
)

func TestEditorialSectionHashBindsAuthority(t *testing.T) {
	r := newRig(t)
	base := strings.Replace(statusDoc, "owner: ops:0", "owner: ops:0\npresentation-mode: direct", 1)
	r.put(r.p.File, base)
	h := r.hash("Trains")
	// An unrelated body edit retains the scope token.
	r.put(r.p.File, strings.Replace(base, "old", "operator text", 1))
	if r.hash("Trains") != h {
		t.Fatal("unrelated text staled a section")
	}
	// Identical section bytes with changed review authority must stale it.
	r.put(r.p.File, strings.Replace(base, "presentation-mode: direct", "presentation-mode: propose", 1))
	if r.hash("Trains") == h {
		t.Fatal("changed document policy retained section token")
	}
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "new trains", IfHash: h})
	wantRefusal(t, err, ExitStale, "stale")
}

func TestEditorialReadInvalidDoesNotAcknowledge(t *testing.T) {
	r := newRig(t)
	before := r.read(r.p.Shadow)
	r.put(r.p.File, strings.Replace(statusDoc, "## Needs you\n", "## Needs you\n<!-- presentation-mode: bad -->\n", 1))
	for _, scope := range []string{"", "Needs you"} {
		if _, err := r.sync.Read(r.p.File, scope); err == nil {
			t.Fatal("read accepted invalid settings")
		}
	}
	if r.read(r.p.Shadow) != before {
		t.Fatal("invalid read acknowledged snapshot")
	}
}

func TestEditorialEmptyApplyValidatesBeforeRetiringRound(t *testing.T) {
	doc := "# A\n<!-- processing-mode: bad -->\nbody\n"
	r := newFileRig(t, doc)
	oldRound := "old round bytes"
	if err := os.WriteFile(r.path, []byte(oldRound), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.apply(Hash(doc)); err == nil {
		t.Fatal("empty apply accepted invalid settings")
	}
	if r.read(r.path) != oldRound || r.read(r.doc) != doc {
		t.Fatal("refused empty apply had effects")
	}
}

func TestEditorialMixedPresentationUsesBasePolicy(t *testing.T) {
	base := "---\npresentation-mode: direct\n---\n# A\n<!-- presentation-mode: highlight -->\nold A\n# B\n<!-- presentation-mode: propose -->\nold B\n# C\nold C\n"
	e, err := BuildEdit(base, []EditRecord{{Old: "old A", New: "new A", Occurrence: 1}, {Old: "old B", New: "new B", Occurrence: 1}, {Old: "old C", New: "new C", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.Next, "new A") || !strings.Contains(e.Next, "🤖~old~{new} B") || !strings.Contains(e.Next, "new C") {
		t.Fatalf("mixed result: %s", e.Next)
	}
	if len(e.Round.Spans) != 3 {
		t.Fatalf("spans: %+v", e.Round.Spans)
	}
	for i, want := range []string{"highlight", "propose", "direct"} {
		if e.Round.Spans[i].Presentation != want {
			t.Errorf("span %d: %+v", i, e.Round.Spans[i])
		}
	}
}
