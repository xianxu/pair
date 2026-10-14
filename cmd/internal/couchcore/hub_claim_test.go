package couchcore

import (
	"strings"
	"testing"
)

func TestHubClaimsParse(t *testing.T) {
	doc, err := ParseHubClaims([]byte("# team\npair\nparley.nvim:7\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Claims()) != 2 || doc.Claims()[0].Slot != -1 || doc.Claims()[1].Slot != 7 {
		t.Fatalf("claims: %+v", doc.Claims())
	}
	for _, raw := range []string{"../pair", "/work/pair", "pair:-1", "pair:01", "pair:1:2", "!pair:7", "pair\x00", string([]byte{0xff}), strings.Repeat("x", 65537)} {
		if _, err := ParseHubClaims([]byte(raw)); err == nil {
			t.Errorf("accepted %q", raw[:min(len(raw), 40)])
		}
	}
}

func hubFixture(t *testing.T, claims ...string) []HubDefinition {
	t.Helper()
	roots := []string{"/work/ops", "/work/bench", "/work/other"}
	out := make([]HubDefinition, len(claims))
	for i, raw := range claims {
		doc, err := ParseHubClaims([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		out[i] = HubDefinition{Root: roots[i], Workspace: "/work", Slot: HubSlot{Repository: roots[i], Number: 0}, Claims: doc}
	}
	return out
}

func TestHubSpecificityAndFallback(t *testing.T) {
	hubs := hubFixture(t, "", "pair\n", "pair:7\n")
	state, err := NewHubOwnership(hubs)
	if err != nil {
		t.Fatal(err)
	}
	for n, want := range map[int]string{1: "/work/bench", 7: "/work/other"} {
		got := state.Resolve(HubSlot{Repository: "/work/pair", Number: n})
		if got.Hub != want || len(got.Conflicts) != 0 {
			t.Fatalf("slot %d: %+v", n, got)
		}
	}
	if got := state.Resolve(HubSlot{Repository: "/work/unknown", Number: 9}); got.Hub != "/work/ops" {
		t.Fatal(got)
	}
	hubs = hubFixture(t, "", "pair:7\n", "pair:7\n")
	state, _ = NewHubOwnership(hubs)
	got := state.Resolve(HubSlot{Repository: "/work/pair", Number: 7})
	if got.Hub != "/work/bench" || len(got.Conflicts) != 1 {
		t.Fatal(got)
	}
	state, _ = NewHubOwnership(hubs[:1])
	if got := state.Resolve(HubSlot{Repository: "/work/pair", Number: 7}); got.Hub != "/work/ops" {
		t.Fatal(got)
	}
}

func TestHubPersistentEditsKeepBroadClaimAndMoveBack(t *testing.T) {
	hubs := hubFixture(t, "# main\n", "# bench\npair\n", "# other\n")
	slot := HubSlot{Repository: "/work/pair", Number: 7}
	edits, err := PlanHubClaimEdits(hubs, []HubSlot{slot}, "/work/other")
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 1 || edits[0].Root != "/work/other" || string(edits[0].After) != "# other\npair:7\n" {
		t.Fatalf("%+v", edits)
	}
	hubs[2].Claims, _ = ParseHubClaims(edits[0].After)
	again, err := PlanHubClaimEdits(hubs, []HubSlot{slot}, "/work/other")
	if err != nil || len(again) != 0 {
		t.Fatalf("repeat %+v %v", again, err)
	}
	back, err := PlanHubClaimEdits(hubs, []HubSlot{slot}, "/work/bench")
	if err != nil || len(back) != 1 || string(back[0].After) != "# other\n" {
		t.Fatalf("back %+v %v", back, err)
	}
	if string(hubs[1].Claims.Bytes()) != "# bench\npair\n" {
		t.Fatal("mutated source")
	}
	main, err := PlanHubClaimEdits(hubs, []HubSlot{slot}, "/work/ops")
	if err != nil || len(main) != 2 || main[0].Root != "/work/ops" {
		t.Fatalf("main %+v %v", main, err)
	}
}

func TestHubMoveAuthorityRevisionAndDurability(t *testing.T) {
	hubs := hubFixture(t, "", "pair\n", "")
	state, _ := NewHubOwnership(hubs)
	slot := HubSlot{Repository: "/work/pair", Number: 7}
	move := HubMove{Slots: []HubSlot{slot}, Target: "/work/other", Caller: "/work/other", Revision: state.Revision()}
	if _, err := state.Apply(move); err == nil {
		t.Fatal("third hub took bench slot")
	}
	move.Caller = "/work/bench"
	next, err := state.Apply(move)
	if err != nil {
		t.Fatal(err)
	}
	if next.Resolve(slot).Hub != "/work/other" || state.Resolve(slot).Hub != "/work/bench" {
		t.Fatal("not immutable")
	}
	if _, err := next.Apply(move); err == nil {
		t.Fatal("stale move accepted")
	}
	edits, err := PlanHubClaimEdits(next.Hubs(), []HubSlot{slot}, "/work/other")
	if err != nil || len(edits) != 1 {
		t.Fatalf("ephemeral move hid persistence: %+v %v", edits, err)
	}
	restarted, _ := NewHubOwnership(next.Hubs())
	if restarted.Resolve(slot).Hub != "/work/bench" {
		t.Fatal("override persisted")
	}
}

func FuzzHubClaims(f *testing.F) {
	for _, s := range []string{"pair\n", "pair:7\n", "../bad", "# hi\n"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		doc, err := ParseHubClaims(raw)
		if err != nil {
			return
		}
		if string(doc.Bytes()) != string(raw) {
			t.Fatal("lost raw")
		}
		for _, c := range doc.Claims() {
			if c.Repository == "" || c.Slot < -1 {
				t.Fatal(c)
			}
		}
	})
}

func TestHubClaimLineBudgetNamesRepair(t *testing.T) {
	if _, err := ParseHubClaims([]byte(strings.Repeat("# comment\n", 1024))); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseHubClaims([]byte(strings.Repeat("# comment\n", 1025))); err == nil || !strings.Contains(err.Error(), "shorten the file and retry") {
		t.Fatalf("budget: %v", err)
	}
}
