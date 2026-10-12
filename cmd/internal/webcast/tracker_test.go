package webcast

import (
	"math/rand/v2"
	"testing"
)

func created(id string) TargetCreated { return TargetCreated{ID: id, URL: "https://" + id, Title: id} }

func TestTrackerActiveTab(t *testing.T) {
	cases := []struct {
		name   string
		inputs []TrackerInput
		want   string // "" = no active tab
	}{
		{"no pages", nil, ""},
		{"single tab", []TrackerInput{created("a")}, "a"},
		{"single tab reports visible unfocused", []TrackerInput{created("a"), PageReport{"a", true, false}}, "a"},
		{"new tab takes over", []TrackerInput{created("a"), PageReport{"a", true, false}, created("b")}, "b"},
		{"switch back", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", true, true},
			PageReport{"b", false, false}, PageReport{"a", true, true},
		}, "a"},
		// Order matters: the hidden report for the tab left behind may arrive
		// after the shown one for the tab switched to.
		{"switch back, reports reversed", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", true, true},
			PageReport{"a", true, true}, PageReport{"b", false, false},
		}, "a"},
		{"background open", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", false, false},
		}, "a"},
		{"two windows, last focused wins", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", true, true},
			PageReport{"a", true, false},
		}, "b"},
		{"two windows, focus moves back", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", true, true},
			PageReport{"b", true, false}, PageReport{"a", true, true},
		}, "a"},
		{"active destroyed falls back", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", true, true},
			PageReport{"a", false, false}, TargetDestroyed{"b"}, PageReport{"a", true, false},
		}, "a"},
		{"all destroyed", []TrackerInput{created("a"), created("b"), TargetDestroyed{"a"}, TargetDestroyed{"b"}}, ""},
		{"newtab never reports", []TrackerInput{created("a"), PageReport{"a", true, true}, created("ntp"), PageReport{"a", false, false}}, "ntp"},
		{"minimized keeps last active", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", false, false},
			PageReport{"a", false, false},
		}, "a"},
		{"report for unknown page ignored", []TrackerInput{created("a"), PageReport{"zz", true, true}}, "a"},
		{"blur does not demote", []TrackerInput{
			created("a"), PageReport{"a", true, true}, created("b"), PageReport{"b", true, true},
			PageReport{"b", true, false},
		}, "b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tr Tracker
			for _, in := range tc.inputs {
				tr.Apply(in)
			}
			tab, ok := tr.Active()
			if tc.want == "" {
				if ok {
					t.Fatalf("active %q, want none", tab.ID)
				}
				return
			}
			if !ok || tab.ID != tc.want {
				t.Fatalf("active %q (ok=%v), want %q", tab.ID, ok, tc.want)
			}
		})
	}
}

func TestTrackerURLTitleFollowChanges(t *testing.T) {
	var tr Tracker
	tr.Apply(TargetCreated{ID: "a", URL: "about:blank"})
	tr.Apply(TargetChanged{ID: "a", URL: "http://localhost:3000/", Title: "Shop"})
	tab, _ := tr.Active()
	if tab != (Tab{ID: "a", URL: "http://localhost:3000/", Title: "Shop"}) {
		t.Fatalf("tab %+v", tab)
	}
	tr.Apply(TargetCreated{ID: "a", URL: "http://localhost:3000/cart", Title: "Cart"})
	if tab, _ := tr.Active(); tab.Title != "Cart" || tr.Pages() != 1 {
		t.Fatalf("repeat create: tab %+v pages %d", tab, tr.Pages())
	}
	// Chrome is slow to report a new <title>; the document's own wins.
	tr.Apply(TargetChanged{ID: "a", URL: "data:x", Title: "data:x"})
	tr.Apply(PageTitle{ID: "a", Title: "one"})
	if tab, _ := tr.Active(); tab.Title != "one" || tab.URL != "data:x" {
		t.Fatalf("page title: %+v", tab)
	}
	tr.Apply(TargetChanged{ID: "a", URL: "data:x", Title: "data:x"})
	if tab, _ := tr.Active(); tab.Title != "one" {
		t.Fatalf("stale target title overrode the page's: %+v", tab)
	}
	// A document without a title falls back to Chrome's.
	tr.Apply(PageTitle{ID: "a", Title: ""})
	if tab, _ := tr.Active(); tab.Title != "data:x" {
		t.Fatalf("fallback: %+v", tab)
	}
	tr.Apply(TargetChanged{ID: "gone", URL: "x"})
	if tr.Pages() != 1 {
		t.Fatalf("change for unknown page created one")
	}
}

// TestTrackerInvariants drives generated input sequences against invariants
// stated independently of the rule's implementation: there is an active tab
// exactly when a page is open; the active tab is an open page; it is visible
// or unreported whenever such a page exists; and a page just created, or just
// reported visible and focused, is active at once.
func TestTrackerInvariants(t *testing.T) {
	ids := []string{"a", "b", "c", "d"}
	for seed := uint64(0); seed < 500; seed++ {
		r := rand.New(rand.NewPCG(seed, 454))
		var tr Tracker
		open := map[string]bool{}
		shown := map[string]bool{} // visible, or never reported
		for step := 0; step < 60; step++ {
			id := ids[r.IntN(len(ids))]
			var in TrackerInput
			switch r.IntN(4) {
			case 0:
				in = created(id)
			case 1:
				in = TargetDestroyed{id}
			case 2:
				in = PageReport{id, r.IntN(2) == 0, r.IntN(2) == 0}
			default:
				in = TargetChanged{ID: id, URL: "u", Title: "t"}
			}
			isNew := !open[id]
			tr.Apply(in)
			switch in := in.(type) {
			case TargetCreated:
				if !open[id] {
					shown[id] = true
				}
				open[id] = true
			case TargetDestroyed:
				delete(open, id)
				delete(shown, id)
			case PageReport:
				if open[id] {
					shown[id] = in.Visible
				}
			}
			tab, ok := tr.Active()
			if ok != (len(open) > 0) || tr.Pages() != len(open) {
				t.Fatalf("seed %d step %d: active=%v with %d open (Pages %d)", seed, step, ok, len(open), tr.Pages())
			}
			if !ok {
				continue
			}
			if !open[tab.ID] {
				t.Fatalf("seed %d step %d: active %q is not open", seed, step, tab.ID)
			}
			anyShown := false
			for id := range shown {
				anyShown = anyShown || shown[id]
			}
			if anyShown && !shown[tab.ID] {
				t.Fatalf("seed %d step %d: active %q is hidden while a shown page exists", seed, step, tab.ID)
			}
			switch in := in.(type) {
			case TargetCreated:
				if isNew && tab.ID != in.ID {
					t.Fatalf("seed %d step %d: just-created %q not active", seed, step, in.ID)
				}
			case PageReport:
				if open[in.ID] && in.Visible && in.Focused && tab.ID != in.ID {
					t.Fatalf("seed %d step %d: focused visible %q not active", seed, step, in.ID)
				}
			}
		}
	}
}
