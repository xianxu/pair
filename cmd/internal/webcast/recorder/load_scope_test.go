package recorder

import (
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
)

func scopeCell() loadCell {
	return loadCell{id: 1, epoch: 1, selection: loadSelection{"S", "T", "F"}, attached: true}
}
func scopeIntent(c *loadCell, serial uint64) *snapshotIntent {
	return &snapshotIntent{serial: serial, activation: serial, cell: c, activationStart: true}
}
func TestLoadScopeHistoricalIntentSurvivesNewSelection(t *testing.T) {
	c := scopeCell()
	first := scopeIntent(&c, 1)
	state, _ := nextLoadScope(loadState{}, c, loadLive{1, 1})
	state, _ = nextLoadScope(state, c, loadLive{2, 2})
	state, effects := nextLoadScope(state, c, loadStart{first})
	if len(effects) != 1 || state.active == nil || state.active.generation != 0 {
		t.Fatal("old FIFO intent lost historical capture authority")
	}
	a := state.active
	state, effects = nextLoadScope(state, c, loadSnapshotReady{a, loadEvidence{c.selection, &recording.PageLoad{Target: "T", Loader: "L", Root: 1}}})
	if len(effects) != 1 || state.reservation == nil {
		t.Fatal("old historical evidence did not reserve")
	}
	state, effects = nextLoadScope(state, c, loadCommitDone{state.reservation, true})
	if len(effects) != 1 {
		t.Fatal("old history attempted UI confirmation")
	}
}
func TestLoadScopeEpochRetryAndCommitOwnership(t *testing.T) {
	c := scopeCell()
	intent := scopeIntent(&c, 1)
	state, _ := nextLoadScope(loadState{}, c, loadLive{1, 1})
	state, _ = nextLoadScope(state, c, loadStart{intent})
	old := state.active
	c = nextLoadCell(c, loadCellChange{invalidate: true})
	state, effects := nextLoadScope(state, c, loadChanged{1})
	if len(effects) != 2 {
		t.Fatal("navigation must cancel IO and publish pending")
	}
	state, effects = nextLoadScope(state, c, loadSnapshotFailed{old})
	if len(effects) != 0 || state.active != nil {
		t.Fatal("epoch retry retired intent")
	}
	state, _ = nextLoadScope(state, c, loadStart{intent})
	a := state.active
	state, _ = nextLoadScope(state, c, loadSnapshotReady{a, loadEvidence{c.selection, &recording.PageLoad{Target: "T", Loader: "L2", Root: 2}}})
	r := state.reservation
	if r == nil {
		t.Fatal("retry did not reserve")
	}
	forged := *r
	next, effects := nextLoadScope(state, c, loadCommitDone{&forged, true})
	if next.reservation != r || len(effects) != 0 {
		t.Fatal("foreign reservation committed")
	}
	c = nextLoadCell(c, loadCellChange{invalidate: true})
	state, effects = nextLoadScope(state, c, loadChanged{1})
	if len(effects) != 1 {
		t.Fatal("reserved sink write must not be cancelled")
	}
	state, effects = nextLoadScope(state, c, loadCommitDone{r, true})
	if len(effects) != 1 || state.reservation != nil {
		t.Fatal("superseded commit must retire history only")
	}
	_, effects = nextLoadScope(state, c, loadCommitDone{r, true})
	if len(effects) != 0 {
		t.Fatal("reservation consumed twice")
	}
}
func TestLoadScopePauseAndCloseKeepDrain(t *testing.T) {
	c := scopeCell()
	intent := scopeIntent(&c, 1)
	state, _ := nextLoadScope(loadState{}, c, loadLive{1, 1})
	generation := state.generation
	state, effects := nextLoadScope(state, c, loadPause{})
	if len(effects) != 0 || state.generation != generation {
		t.Fatal("pause changed load")
	}
	state, _ = nextLoadScope(state, c, loadClose{})
	state, effects = nextLoadScope(state, c, loadStart{intent})
	if len(effects) != 1 || state.active == nil || state.active.generation != 0 {
		t.Fatal("close lost historical queue drain")
	}
}
