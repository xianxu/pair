package launcher

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// A single complete reply owns presence and state. There is no second
// independently timed listing whose name set could contradict it.
func TestSessionListingFullReplyDefinesPresence(t *testing.T) {
	for _, tc := range []struct {
		name, full string
		want       int
	}{
		{"selected absent", "pair-b [Created 1s ago]\n", 0},
		{"unrelated added", "pair-a [Created 1s ago]\npair-b [Created 1s ago]\n", 1},
		{"malformed", `[{"id":1}]`, -1},
		{"duplicate", "pair-a [Created 1s ago]\npair-a [Created 1s ago]\n", -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := installAdmissionFixture(t)
			f.SetModel(ownerFixtureModel("pair term"))
			f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &tc.full})
			got, err := (ZellijSource{Path: f.Path()}).SnapshotSessionsContext(t.Context(), []string{"pair-a"})
			if tc.want < 0 {
				if err == nil || got != nil {
					t.Fatalf("invalid full listing granted authority: %v %v", got, err)
				}
			} else if err != nil || len(got) != tc.want {
				t.Fatalf("full reply presence: %v %v", got, err)
			}
			want := 1
			if tc.want > 0 {
				want++
			}
			if len(f.Queries()) != want || len(f.Effects()) != 0 {
				t.Fatalf("queries=%v effects=%v", f.Queries(), f.Effects())
			}
		})
	}
}

func TestCachedMalformedSessionListingRefusesBeforeNativeQuery(t *testing.T) {
	f := installAdmissionFixture(t)
	f.SetModel(ownerFixtureModel("pair term"))
	p := &ListingPoller{now: time.Now, MaxStale: time.Second, list: func(context.Context) ([]byte, error) { return []byte(`[{"id":1}]`), nil }}
	p.Refresh(t.Context())
	got, err := (ZellijSource{Path: f.Path(), Listings: p}).SnapshotSessionsContext(t.Context(), []string{"pair-a"})
	if err == nil || got != nil || len(f.Queries()) != 0 || len(f.Effects()) != 0 {
		t.Fatalf("invalid cached reply: %v %v queries=%v", got, err, f.Queries())
	}
}

func TestSessionLivenessKeepsExactNativeNameAndExitedSuffix(t *testing.T) {
	f := installAdmissionFixture(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{
		" pair-foreign":         {},
		"pair-EXITED-is-a-name": {},
	}})
	got, err := (ZellijSource{Path: f.Path()}).LivenessContext(t.Context())
	if err != nil || len(got) != 1 || got[0].Name != "pair-EXITED-is-a-name" || got[0].State != SessionLive {
		t.Fatalf("name text became authority for a different name/state: sessions=%v err=%v", got, err)
	}
	if len(f.Queries()) != 1 || len(f.Effects()) != 0 {
		t.Fatalf("liveness budget changed: queries=%v effects=%v", f.Queries(), f.Effects())
	}
}

func TestSessionListingAcceptsReorderedNativeRows(t *testing.T) {
	f := installAdmissionFixture(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-a": {}, "pair-b": {}}})
	full := "pair-b [Created 2s ago] (current)\npair-a [Created 1s ago]\n"
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &full})
	got, err := (ZellijSource{Path: f.Path()}).SnapshotContext(t.Context())
	if err != nil || len(got) != 2 || got[0].Name != "pair-a" || got[1].Name != "pair-b" || got[0].State != SessionDetached || got[1].State != SessionDetached {
		t.Fatalf("compatible reordered pair refused: sessions=%v err=%v", got, err)
	}
	if len(f.Queries()) != 3 || len(f.Effects()) != 0 {
		t.Fatalf("healthy query/effect budget: queries=%v effects=%v", f.Queries(), f.Effects())
	}
	queries := f.Queries()
	if strings.Join(queries[1].Args, " ") != "--session pair-a action list-clients" || strings.Join(queries[2].Args, " ") != "--session pair-b action list-clients" {
		t.Fatalf("native query order must stay deterministic: %v", queries)
	}
}

func installAdmissionFixture(t *testing.T) *zellijtest.Fixture {
	t.Helper()
	f := zellijtest.New(t)
	for _, entry := range f.Env() {
		k, v, _ := strings.Cut(entry, "=")
		t.Setenv(k, v)
	}
	return f
}
func ownerFixtureModel(command string) zellijtest.Model {
	return zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-a": {Panes: []zellijtest.Pane{zellijtest.DefaultPane(1, command)}, Tab: zellijtest.DefaultTab()}}}
}
func TestOwnerAdmissionUsesStatefulNativeReplies(t *testing.T) {
	const owned = "nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"
	for _, fault := range []string{"wrong-kind", "changed-owner", "healthy"} {
		t.Run(fault, func(t *testing.T) {
			f := installAdmissionFixture(t)
			f.SetModel(ownerFixtureModel(owned))
			switch fault {
			case "wrong-kind":
				raw := `{"active_swap_layout_name":null,"is_swap_layout_dirty":false}`
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw}, zellijtest.Fault{Ordinal: 2, Reply: &raw}, zellijtest.Fault{Ordinal: 3, Reply: &raw})
			case "changed-owner":
				other := ownerFixtureModel("nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-2.md")
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &other})
			}
			io := &emptyReplyOwnerIO{source: ZellijSource{Path: f.Path()}, hasServer: true, server: SessionServerIdentity{PID: 42, Identity: "same-server", Session: "pair-a", Socket: "/fake/socket"}}
			p := SessionOwnerProbe{IO: io}
			got, err := p.Probe(t.Context(), "pair-a", "/tmp/data", "0123456789abcdef", "1-repo-1")
			if fault == "healthy" {
				if err != nil || got.State != SessionOwnerOwned {
					t.Fatalf("healthy owner: %+v,%v", got, err)
				}
			} else if got.State != SessionOwnerUnknown {
				t.Fatalf("fault granted authority: %+v,%v", got, err)
			}
			want := 2
			if fault == "wrong-kind" {
				want = 3
			}
			if len(f.Queries()) != want || len(f.Effects()) != 0 {
				t.Fatalf("queries=%v effects=%v", f.Queries(), f.Effects())
			}
		})
	}
}

func TestLiveLayoutRefusesChangedNativePaneSignature(t *testing.T) {
	f := installAdmissionFixture(t)
	panes := []zellijtest.Pane{zellijtest.DefaultPane(0, "pair wrap codex"), zellijtest.DefaultPane(1, "nvim /data/draft-work.md"), zellijtest.DefaultPane(3, "pair term")}
	for _, pane := range panes {
		pane["terminal_command"] = pane["pane_command"]
	}
	panes[1]["title"] = "draft"
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-a": {Panes: panes, Tab: zellijtest.DefaultTab()}}})
	next := zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-a": {Panes: panes[:2], Tab: zellijtest.DefaultTab()}}}
	f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &next})
	if mode, err := (OSRuntime{}).ProbeLiveLayout("pair-a"); err == nil {
		t.Fatalf("disagreeing layout admitted: %v", mode)
	}
	if len(f.Effects()) != 0 {
		t.Fatalf("preflight effects=%v", f.Effects())
	}
}
