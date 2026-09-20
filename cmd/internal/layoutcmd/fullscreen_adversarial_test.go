package layoutcmd

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestFullscreenTransitionExhaustiveShortSequences(t *testing.T) {
	// Literal rows from the plan's ordering table, not a second implementation
	// of the production switch. Six events reach every terminal state and then
	// exercise late completions, duplicate Begin, and unknown events.
	type edge struct {
		phase fullscreenPhase
		event FullscreenEvent
	}
	type result struct {
		phase  fullscreenPhase
		effect FullscreenEffect
	}
	for _, tc := range []struct {
		name string
		plan FullscreenPlan
		rows map[edge]result
	}{
		{"expand", FullscreenPlan{FullscreenExpand, "4", "2"}, map[edge]result{
			{fullscreenInitial, FullscreenBegin}:      {fullscreenSaving, FullscreenSave},
			{fullscreenSaving, FullscreenConfirmed}:   {fullscreenToggling, FullscreenToggle},
			{fullscreenToggling, FullscreenConfirmed}: {fullscreenDone, FullscreenNothing},
		}},
		{"collapse-return", FullscreenPlan{FullscreenCollapse, "4", "2"}, map[edge]result{
			{fullscreenInitial, FullscreenBegin}:      {fullscreenToggling, FullscreenToggle},
			{fullscreenToggling, FullscreenConfirmed}: {fullscreenFocusing, FullscreenFocus},
			{fullscreenFocusing, FullscreenConfirmed}: {fullscreenClearing, FullscreenClear},
			{fullscreenClearing, FullscreenConfirmed}: {fullscreenDone, FullscreenNothing},
		}},
		{"collapse-no-return", FullscreenPlan{FullscreenCollapse, "4", ""}, map[edge]result{
			{fullscreenInitial, FullscreenBegin}:      {fullscreenToggling, FullscreenToggle},
			{fullscreenToggling, FullscreenConfirmed}: {fullscreenClearing, FullscreenClear},
			{fullscreenClearing, FullscreenConfirmed}: {fullscreenDone, FullscreenNothing},
		}},
		{"collapse-same-terminal", FullscreenPlan{FullscreenCollapse, "4", "4"}, map[edge]result{
			{fullscreenInitial, FullscreenBegin}:      {fullscreenToggling, FullscreenToggle},
			{fullscreenToggling, FullscreenConfirmed}: {fullscreenClearing, FullscreenClear},
			{fullscreenClearing, FullscreenConfirmed}: {fullscreenDone, FullscreenNothing},
		}},
		{"noop", FullscreenPlan{}, map[edge]result{
			{fullscreenInitial, FullscreenBegin}: {fullscreenDone, FullscreenNothing},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, phase := range []fullscreenPhase{fullscreenSaving, fullscreenToggling, fullscreenFocusing, fullscreenClearing} {
				for _, event := range []FullscreenEvent{FullscreenFailed, FullscreenUnconfirmed} {
					tc.rows[edge{phase, event}] = result{fullscreenStopped, FullscreenNothing}
				}
			}
			var shortest []FullscreenEvent
			var first string
			checks, mismatches := 0, 0
			var visit func(FullscreenState, fullscreenPhase, []FullscreenEvent)
			visit = func(actual FullscreenState, model fullscreenPhase, prefix []FullscreenEvent) {
				if len(prefix) == 6 {
					return
				}
				for _, event := range []FullscreenEvent{FullscreenBegin, FullscreenConfirmed, FullscreenFailed, FullscreenUnconfirmed, 255} {
					seq := append(append([]FullscreenEvent(nil), prefix...), event)
					want, ok := tc.rows[edge{model, event}]
					if !ok {
						want = result{model, FullscreenNothing}
					}
					next, effect := FullscreenTransition(actual, event)
					checks++
					if next.plan != tc.plan || next.phase != want.phase || effect != want.effect {
						mismatches++
						if shortest == nil || len(seq) < len(shortest) {
							shortest = seq
							first = fmt.Sprintf("got phase=%v effect=%v plan=%+v; want phase=%v effect=%v plan=%+v", next.phase, effect, next.plan, want.phase, want.effect, tc.plan)
						}
					}
					visit(next, want.phase, seq)
				}
			}
			visit(NewFullscreenState(tc.plan), fullscreenInitial, nil)
			if checks != 19530 {
				t.Fatalf("sequence enumeration incomplete: %d checks", checks)
			}
			if mismatches != 0 {
				t.Fatalf("%d/%d transition mismatches; shortest sequence=%v: %s", mismatches, checks, shortest, first)
			}
		})
	}
}

func TestPlanFullscreenGeneratedInventorySafety(t *testing.T) {
	// Cartesian product: missing panes, inventory order, stale focus flags,
	// registry-only identification, ambiguous/unknown fullscreen, stale IDs and
	// prefixed caller/return IDs. The expected selector uses known fixture roles,
	// never RoleForPane/PlanFullscreen/pickRightTerminal as an oracle.
	off, on := false, true
	checks := 0
	for mask := 0; mask < 32; mask++ {
		for order := 0; order < 2; order++ {
			for focus := 0; focus < 3; focus++ {
				for mode := 0; mode < 6; mode++ {
					for registry := 0; registry < 2; registry++ {
						base := []zellijpane.Pane{
							{ID: "0", IsPlugin: true, Title: "terminal plugin", IsFocused: true},
							{ID: "1", TerminalCommand: "pair wrap codex"},
							{ID: "2", TerminalCommand: "nvim -u /pair/nvim/init.lua"},
							{ID: "3", Title: "terminal top", IsFullscreen: &off},
							{ID: "4", Title: "custom tab title", IsFullscreen: &off},
						}
						if mode == 1 || mode == 3 {
							base[3].IsFullscreen = &on
						}
						if mode == 2 || mode == 3 {
							base[4].IsFullscreen = &on
						}
						if mode == 4 {
							base[3].IsFullscreen = nil
						}
						if mode == 5 {
							base[4].IsFullscreen = nil
						}
						var panes []zellijpane.Pane
						for i, p := range base {
							if mask&(1<<i) == 0 {
								continue
							}
							p.IsFocused = p.IsPlugin || focus == 2 || (focus == 1 && p.ID == "2")
							panes = append(panes, p)
						}
						// A floating terminal lookalike must never become the target,
						// even when it reports fullscreen or appears first.
						panes = append(panes, zellijpane.Pane{ID: "5", Title: "terminal floating", IsFloating: true, IsFullscreen: &on})
						if order == 1 {
							for i, j := 0, len(panes)-1; i < j; i, j = i+1, j-1 {
								panes[i], panes[j] = panes[j], panes[i]
							}
						}
						var registered []string
						if registry == 1 {
							registered = []string{"4", "5", "99"}
						}
						eligible := map[string]bool{"3": true, "4": registry == 1}
						var terminals, owners []string
						live := map[string]bool{}
						var focused []string
						unknown := false
						for _, p := range panes {
							if !p.IsPlugin {
								live[p.ID] = true
								if p.IsFocused {
									focused = append(focused, p.ID)
								}
							}
							if eligible[p.ID] {
								terminals = append(terminals, p.ID)
								if p.IsFullscreen == nil {
									unknown = true
								} else if *p.IsFullscreen {
									owners = append(owners, p.ID)
								}
							}
						}
						for _, caller := range []string{"", "1", "terminal_2", "3", "4", "0", "99"} {
							for _, last := range []string{"", "3", "4", "99"} {
								for _, record := range []string{"", "terminal_1", "4", "0", "99"} {
									want := FullscreenPlan{}
									wantErr := unknown || len(owners) > 1
									if !wantErr && len(terminals) > 0 {
										if len(owners) == 1 {
											ret := strings.TrimPrefix(record, "terminal_")
											if !live[ret] {
												ret = ""
												if live["2"] {
													ret = "2"
												}
											}
											want = FullscreenPlan{FullscreenCollapse, owners[0], ret}
										} else {
											id := strings.TrimPrefix(caller, "terminal_")
											if id == "" && len(focused) == 1 {
												id = focused[0]
											}
											wantErr = !live[id]
											if !wantErr {
												selected, rank := "", -1
												for _, p := range panes {
													if !eligible[p.ID] {
														continue
													}
													r := 0
													if p.IsFocused {
														r = 1
													}
													if p.ID == last {
														r = 2
													}
													if p.ID == id {
														r = 3
													}
													if r > rank {
														selected, rank = p.ID, r
													}
												}
												want = FullscreenPlan{FullscreenExpand, selected, id}
											}
										}
									}
									got, err := PlanFullscreen(panes, caller, last, registered, record)
									checks++
									if (err != nil) != wantErr || (!wantErr && got != want) {
										t.Fatalf("mask=%05b order=%d focus=%d mode=%d registry=%d caller=%q last=%q record=%q: got=%+v err=%v; want=%+v error=%v", mask, order, focus, mode, registry, caller, last, record, got, err, want, wantErr)
									}
									if err == nil && got.Operation != FullscreenNoop && (!live[got.TerminalID] || !eligible[got.TerminalID] || (got.ReturnID != "" && !live[got.ReturnID])) {
										t.Fatalf("unsafe action: %+v inventory=%+v", got, panes)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("checked %d generated inventory/identity combinations", checks)
}

// Only pane IO is simulated: every invocation opens the real store's flock.
// Blocking the first list proves the lock covers observation, not just writes.
type overlappingFullscreenRuntime struct {
	*fullscreenWorld
	store   workbenchshortcut.FullscreenReturnStore
	entered chan struct{}
	release chan struct{}
	lists   atomic.Int32
}

func (r *overlappingFullscreenRuntime) FullscreenStore() workbenchshortcut.FullscreenStore {
	return r.store
}
func (r *overlappingFullscreenRuntime) ListPanesJSON() ([]byte, error) {
	if r.lists.Add(1) == 1 && r.entered != nil {
		close(r.entered)
		<-r.release
	}
	return r.fullscreenWorld.ListPanesJSON()
}

func TestFullscreenRuntimeOverlappingEntryUsesRealStoreLock(t *testing.T) {
	r := &overlappingFullscreenRuntime{
		fullscreenWorld: newFullscreenWorld(),
		store:           workbenchshortcut.FullscreenReturnStore{DataDir: t.TempDir(), Tag: "overlap"},
		entered:         make(chan struct{}), release: make(chan struct{}),
	}
	var once sync.Once
	unblock := func() { once.Do(func() { close(r.release) }) }
	first := make(chan int, 1)
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		first <- RunToggleFocused(nil, r, io.Discard)
	}()
	t.Cleanup(func() {
		unblock()
		select {
		case <-firstDone:
		case <-time.After(5 * time.Second):
			t.Error("first invocation failed to stop during cleanup")
		}
	})
	select {
	case <-r.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first invocation never reached pane observation")
	}
	// A separate runtime/store value models another CLI process opening its
	// own lock descriptor, without sharing any in-memory busy flag.
	secondRuntime := &overlappingFullscreenRuntime{fullscreenWorld: newFullscreenWorld(), store: r.store}
	second := make(chan int, 1)
	go func() { second <- RunToggleFocused(nil, secondRuntime, io.Discard) }()
	select {
	case code := <-second:
		if code != 0 || secondRuntime.lists.Load() != 0 || len(secondRuntime.ops) != 0 {
			t.Fatalf("overlapping invocation did work: code=%d lists=%d ops=%v", code, secondRuntime.lists.Load(), secondRuntime.ops)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("overlapping invocation queued behind the lock")
	}
	if record, err := r.store.Read(); err != nil || record != "" {
		t.Fatalf("busy entry changed return record before observation: %q %v", record, err)
	}
	unblock()
	select {
	case code := <-first:
		if code != 0 {
			t.Fatalf("first invocation: exit %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first invocation did not finish")
	}
	if record, err := r.store.Read(); err != nil || record != "2" {
		t.Fatalf("expansion lost return record: %q %v", record, err)
	}
	if !reflect.DeepEqual(r.ops, []string{"toggle-fullscreen --pane-id 4"}) || r.lists.Load() != 1 {
		t.Fatalf("expansion actions=%v lists=%d", r.ops, r.lists.Load())
	}
	// A subsequent press must acquire the released lock and collapse, proving
	// the ignored press wasn't queued and cannot cause a delayed inverse toggle.
	if code := RunToggleFocused(nil, r, io.Discard); code != 0 {
		t.Fatalf("collapse exit=%d", code)
	}
	want := []string{"toggle-fullscreen --pane-id 4", "toggle-fullscreen --pane-id 4", "focus-pane-id 2"}
	if !reflect.DeepEqual(r.ops, want) || r.current != "2" || r.lists.Load() != 2 {
		t.Fatalf("round trip actions=%v current=%q lists=%d", r.ops, r.current, r.lists.Load())
	}
	if record, err := r.store.Read(); err != nil || record != "" {
		t.Fatalf("collapse retained record: %q %v", record, err)
	}
}
