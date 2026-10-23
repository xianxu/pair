package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// wakeFixture is two hubs and two workers, every row parked; asleep names the
// labels a full quit put to sleep.
func wakeFixture(t *testing.T, asleep ...string) MenuState {
	t.Helper()
	at := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC)
	ops, bench := presentationOrdinary("/src/ops", "main"), presentationOrdinary("/src/bench", "extra")
	worker, idle := presentationSlot("/src/pair", 7), presentationSlot("/src/pair", 8)
	rows := map[string]*couchcore.ActionableThreadSummary{"ops": &ops, "bench": &bench, "pair:7": &worker, "pair:8": &idle}
	for _, row := range rows {
		row.State = couchcore.ThreadParked
	}
	for _, label := range asleep {
		rows[label].AsleepAt = &at
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{ops, bench, worker, idle}, couchcore.ThreadAddress{})
	state.HubOwnership = multiHubFixture(t)
	state.Hub = "/src/ops"
	return state
}

func wakeResumes(effects []MenuEffect) []string {
	var tags []string
	for _, e := range effects {
		if e.Operation == "resume" {
			tags = append(tags, e.Args["tag"])
		}
	}
	return tags
}

// The gesture wakes exactly the asleep hubs, main hub last so the operator
// lands on it; an asleep worker waits for its hub and a plain parked slot is
// never touched (pair#830).
func TestWakeResumesOnlyTheAsleepHubs(t *testing.T) {
	state := wakeFixture(t, "ops", "bench", "pair:7")
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "wake"})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation || state.CurrentFrame().Action != "wake" {
		t.Fatalf("wake did not ask first: frame %+v effects %+v", state.CurrentFrame(), effects)
	}
	items := confirmationMenuItems(state, state.CurrentFrame())
	if len(items) != 2 || items[1] != "wake 2 asleep hubs: bench, ops" {
		t.Fatalf("items = %q", items)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if got := strings.Join(wakeResumes(effects), ","); got != "extra,main" {
		t.Fatalf("resumed %q, want the two hubs, main last (effects %+v)", got, effects)
	}
	if len(state.Frames) != 1 || !strings.Contains(state.Notice.Text, "waking bench, ops") {
		t.Fatalf("frames %d notice %q", len(state.Frames), state.Notice.Text)
	}
}

// A hub parked on its own stays put; with no asleep hub the key says so and
// opens nothing.
func TestWakeLeavesPlainParkedHubsAndSaysWhenNothingSleeps(t *testing.T) {
	state := wakeFixture(t, "ops")
	if got := wakeTargets(state); len(got) != 1 || got[0].Label != "ops" {
		t.Fatalf("targets = %+v", got)
	}
	none := wakeFixture(t, "pair:7")
	next, effects := ReduceMenu(none, MenuEvent{Kind: MenuEventParkHotkey, Operation: "wake"})
	if len(effects) != 0 || len(next.Frames) != 1 || next.Notice.Text != "no asleep hubs to wake" {
		t.Fatalf("frames %d notice %q effects %+v", len(next.Frames), next.Notice.Text, effects)
	}
}

// The set is re-read at Enter: a hub woken (or resumed by hand) after the
// confirmation opened is not resumed twice.
func TestWakeRereadsTheAsleepSetAtEnter(t *testing.T) {
	state := wakeFixture(t, "ops", "bench")
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "wake"})
	rows := menuRows(state)
	for i := range rows {
		if rows[i].Address.Tag == "extra" {
			rows[i].AsleepAt, rows[i].State = nil, couchcore.ThreadLive
		}
	}
	state, _ = replaceMenuInventory(state, rows)
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if got := strings.Join(wakeResumes(effects), ","); got != "main" {
		t.Fatalf("resumed %q, want only the still-asleep main hub", got)
	}
}
