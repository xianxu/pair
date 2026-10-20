package launcher

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// The selected session never changes. Two independently observed formats can
// nevertheless disagree when an unrelated session starts between them.
func TestSessionListingUnrelatedChurnUsesOneObservation(t *testing.T) {
	for _, cached := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "polled"}[cached], func(t *testing.T) {
			f := installAdmissionFixture(t)
			f.SetModel(ownerFixtureModel("pair term"))
			next := zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-a": {}, "pair-unrelated": {}}}
			f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &next})
			source := ZellijSource{Path: f.Path()}
			if cached {
				p := NewListingPoller(source, time.Hour, time.Minute)
				p.Refresh(t.Context())
				source.Listings = p
			}
			got, err := source.LivenessContext(t.Context())
			if err != nil || len(got) == 0 || got[0].Name != "pair-a" {
				t.Fatalf("unchanged selected session: sessions=%v err=%v queries=%v", got, err, f.Queries())
			}
			if len(f.Queries()) != 1 || len(f.Effects()) != 0 {
				t.Fatalf("one read-only observation required: queries=%v effects=%v", f.Queries(), f.Effects())
			}
		})
	}
}

func TestSessionListingCostAtEightAndSixteenSessions(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := installAdmissionFixture(t)
			sessions := make(map[string]zellijtest.Session, n)
			for i := range n {
				sessions[fmt.Sprintf("pair-%02d", i)] = zellijtest.Session{}
			}
			f.SetModel(zellijtest.Model{Sessions: sessions})
			source := ZellijSource{Path: f.Path()}
			p := NewListingPoller(source, time.Hour, time.Minute)
			p.Refresh(t.Context())
			source.Listings = p
			for range 2 * n {
				got, err := source.LivenessContext(t.Context())
				if err != nil || len(got) != n {
					t.Fatalf("snapshot: sessions=%d err=%v", len(got), err)
				}
			}
			if len(f.Queries()) != 1 || len(f.Effects()) != 0 {
				t.Fatalf("N=%d cached reads created work: queries=%v effects=%v", n, f.Queries(), f.Effects())
			}
		})
	}
}

func TestMalformedListingRefreshDoesNotFreshenLastGood(t *testing.T) {
	f := installAdmissionFixture(t)
	f.SetModel(ownerFixtureModel("pair term"))
	source := ZellijSource{Path: f.Path()}
	now := time.Unix(1000, 0)
	p := NewListingPoller(source, time.Hour, 5*time.Second)
	p.now = func() time.Time { return now }
	p.Refresh(t.Context())
	source.Listings = p
	malformed := `[{"id":1}]`
	f.SetFaults(zellijtest.Fault{Ordinal: len(f.Queries()) + 1, Reply: &malformed})
	now = now.Add(4 * time.Second)
	p.Refresh(t.Context())
	now = now.Add(2 * time.Second)
	if _, err := source.LivenessContext(t.Context()); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("malformed refresh renewed authority: %v", err)
	}
}

// The display runtime must not combine names from a later short reply with an
// earlier full reply, either. Client counting is a separate targeted observation.
func TestRuntimeListSessionsUsesOneInventoryObservation(t *testing.T) {
	f := installAdmissionFixture(t)
	f.SetModel(ownerFixtureModel("pair term"))
	next := zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-a": {}, "pair-unrelated": {}}}
	f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &next})
	dir := t.TempDir()
	rows, err := (OSRuntime{DataDir: dir, GlobalDataDir: dir}).ListSessions()
	if err != nil || len(rows) != 1 || rows[0].Session != "pair-a" {
		t.Fatalf("mixed inventory: rows=%v err=%v", rows, err)
	}
	queries := f.Queries()
	if len(queries) != 2 || strings.Join(queries[0].Args, " ") != "list-sessions --no-formatting" || strings.Join(queries[1].Args, " ") != "--session pair-a action list-clients" || len(f.Effects()) != 0 {
		t.Fatalf("inventory query budget: queries=%v effects=%v", queries, f.Effects())
	}
}
