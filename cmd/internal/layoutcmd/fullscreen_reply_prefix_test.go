package layoutcmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// Empty Focus permits only a uniquely observed owned pane; structural native
// mutations can move focus before Pair's explicit return-focus step runs.
type nativeReplyExpectedState struct {
	Floating, Positioned, ExactTiling bool
	Focus                             string
}

func nativeReplyExpectedPrefix(prior nativeReplyExpectedState, events []nativeReplyEvent, source, target string, tab int) (nativeReplyExpectedState, error) {
	state := prior
	var pending []string
	terminal := false
	for _, event := range events {
		if event.Unknown || event.Kind == "refusal" {
			terminal = true
		}
		switch event.Kind {
		case "mutation", "record", "nudge":
			if terminal {
				return state, fmt.Errorf("new %s after unknown/refusal", event.Kind)
			}
			if event.Kind == "mutation" {
				if pending != nil || !nativeReplyAllowedTarget(event.Args, source, target, tab) {
					return state, fmt.Errorf("unowned or overlapping mutation %v", event.Args)
				}
				pending = event.Args
			}
			// Clear is reached only after the executor's restore and return-focus
			// stages, even if clearing its local record subsequently fails.
			if event.Kind == "record" && len(event.Args) > 0 && event.Args[0] == "clear" {
				if pending != nil || state.Floating {
					return state, fmt.Errorf("record clear before restore completed")
				}
				state.ExactTiling = true
			}
		case "mutation-result":
			if pending == nil || !reflect.DeepEqual(pending, event.Args) {
				return state, fmt.Errorf("mutation result without matching dispatch")
			}
			pending = nil
			// Only the runtime can authenticate a semantic already-focused success.
			// Native diagnostics/intention strings alone are not completion evidence.
			if !event.Completed {
				return state, fmt.Errorf("mutation outcome is not known complete: %v", event.Args)
			}
			switch event.Args[0] {
			case "toggle-pane-embed-or-floating":
				state.Floating = !state.Floating
				state.Positioned = false
				state.ExactTiling = false
				state.Focus = ""
			case "change-floating-pane-coordinates":
				if !state.Floating {
					return state, fmt.Errorf("placement without floated target")
				}
				state.Positioned = true
			case "show-floating-panes":
				if !state.Floating || !state.Positioned {
					return state, fmt.Errorf("show before admitted Focus placement")
				}
				state.Focus = target
			case "next-swap-layout", "previous-swap-layout", "move-pane":
				state.ExactTiling = false
				state.Focus = ""
			case "focus-pane-id":
				state.Focus = strings.TrimPrefix(event.Args[1], "terminal_")
			default:
				return state, fmt.Errorf("unsupported completed prefix action %s", event.Args[0])
			}
		}
	}
	if pending != nil {
		return state, fmt.Errorf("dispatched mutation has no completion evidence")
	}
	return state, nil
}
func nativeReplyCheckExpected(initial, final map[string]zellijpane.Pane, state nativeReplyExpectedState, oracle []nativeReplyEvent) (string, error) {
	if len(initial) != 4 || len(final) != len(initial) {
		return "", fmt.Errorf("owned pane set changed")
	}
	focused := ""
	maxX, maxY := 0, 0
	for _, p := range initial {
		maxX = max(maxX, p.X+p.Columns)
		maxY = max(maxY, p.Y+p.Rows)
	}
	tiled := []zellijpane.Pane{}
	for name, want := range initial {
		got, ok := final[name]
		if !ok || got.ID != want.ID || got.IsPlugin || got.TabID == nil || want.TabID == nil || *got.TabID != *want.TabID || got.TabPosition == nil || want.TabPosition == nil || *got.TabPosition != *want.TabPosition || got.IsFullscreen == nil || *got.IsFullscreen {
			return "", fmt.Errorf("owned identity/tab/fullscreen changed: %s", name)
		}
		if got.Columns <= 0 || got.Rows <= 0 || got.X < 0 || got.Y < 0 || got.X+got.Columns > maxX || got.Y+got.Rows > maxY {
			return "", fmt.Errorf("invalid owned geometry: %s", name)
		}
		if got.IsFloating != (state.Floating && name == "terminal-bottom") {
			return "", fmt.Errorf("unexpected floating state: %s", name)
		}
		if !got.IsFloating {
			tiled = append(tiled, got)
		}
		if got.IsFocused {
			if focused != "" {
				return "", fmt.Errorf("ambiguous owned focus")
			}
			focused = name
		}
	}
	if focused == "" {
		return "", fmt.Errorf("no positively observed owned focus")
	}
	if state.Focus != "" && final[focused].ID != state.Focus {
		return "", fmt.Errorf("actual owned focus %s disagrees with completed explicit focus %s", final[focused].ID, state.Focus)
	}
	for i, p := range tiled {
		for _, q := range tiled[i+1:] {
			if p.X < q.X+q.Columns && q.X < p.X+p.Columns && p.Y < q.Y+q.Rows && q.Y < p.Y+p.Rows {
				return "", fmt.Errorf("owned tiled panes overlap")
			}
		}
	}
	if !state.Floating && state.ExactTiling {
		if err := nativeReplyGeometry(initial, final, false); err != nil {
			return "", err
		}
	}
	if state.Floating && state.Positioned && state.Focus != "" {
		if err := nativeReplyGeometry(initial, final, true, oracle); err != nil {
			return "", err
		}
	}
	return focused, nil
}

func nativeReplyRecordedOriginal() map[string]zellijpane.Pane {
	tab := 1
	notFullscreen := false
	rows := map[string]zellijpane.Pane{}
	for _, v := range []struct {
		name, id string
		x, y     int
	}{{"agent", "1", 0, 0}, {"draft", "3", 0, 12}, {"terminal-top", "2", 40, 0}, {"terminal-bottom", "4", 40, 12}} {
		rows[v.name] = zellijpane.Pane{ID: v.id, TabID: &tab, TabPosition: &tab, IsFocused: v.name == "agent", IsFullscreen: &notFullscreen, X: v.x, Y: v.y, Columns: 40, Rows: 12}
	}
	return rows
}
func nativeReplyCompleted(args ...string) []nativeReplyEvent {
	return []nativeReplyEvent{{Kind: "mutation", Args: args}, {Kind: "mutation-result", Args: args, Completed: true}}
}
func TestNativeReplyRecordedReturnPrefixAllowsOwnedImplicitFocus(t *testing.T) {
	// Transcribed successful native results from the retained 2026-10-18 run,
	// /tmp/p647nr-3940343646/result.json, decision2. This is an OFFLINE model
	// control, not a replacement artifact or a retrospectively invented PTY event.
	events := nativeReplyCompleted("toggle-pane-embed-or-floating", "--pane-id", "4")
	for i := 0; i < 3; i++ {
		events = append(events, nativeReplyCompleted("next-swap-layout", "--tab-id", "1")...)
	}
	events = append(events, nativeReplyEvent{Kind: "query", Unknown: true, Error: "exit status 1", Stderr: "There is no active session!"}, nativeReplyEvent{Kind: "refusal"})
	prior := nativeReplyExpectedState{Floating: true, Positioned: true, Focus: "4"}
	state, err := nativeReplyExpectedPrefix(prior, events, "1", "4", 1)
	if err != nil {
		t.Fatal(err)
	}
	if state.Floating || state.Positioned || state.ExactTiling || state.Focus != "" {
		t.Fatalf("prefix invented rollback/restore/focus guarantee: %+v", state)
	}
	initial, final := nativeReplyRecordedOriginal(), nativeReplyRecordedOriginal()
	focus, err := nativeReplyCheckExpected(initial, final, state, nil)
	if err != nil || focus != "agent" {
		t.Fatalf("recorded original tiling/agent focus rejected: focus=%q err=%v", focus, err)
	}
	if err := nativeReplyGeometry(initial, final, false); err != nil {
		t.Fatalf("retained snapshot is original tiling: %v", err)
	}
	old := nativeReplyOutcome{Events: events, Refused: true, ProtectedBad: true, Broker: true, RecordUnchanged: true, Observed: false}
	if got := nativeReplyClassification(old); got != "stopped" {
		t.Fatalf("old missing final PTY witness relabeled: %s", got)
	}
}

func TestNativeReplyPrefixRequiresPositiveCompletionAndStopsAtUnknown(t *testing.T) {
	prior := nativeReplyExpectedState{Floating: true, Positioned: true, Focus: "4"}
	for _, tc := range []struct {
		name   string
		events []nativeReplyEvent
	}{
		{"intention only", []nativeReplyEvent{{Kind: "mutation", Args: []string{"toggle-pane-embed-or-floating", "--pane-id", "4"}}}},
		{"failed mutation", []nativeReplyEvent{{Kind: "mutation", Args: []string{"toggle-pane-embed-or-floating", "--pane-id", "4"}}, {Kind: "mutation-result", Args: []string{"toggle-pane-embed-or-floating", "--pane-id", "4"}, Error: "exit status 1"}}},
		{"nil-looking diagnostic is not completion", []nativeReplyEvent{{Kind: "mutation", Args: []string{"toggle-pane-embed-or-floating", "--pane-id", "4"}}, {Kind: "mutation-result", Args: []string{"toggle-pane-embed-or-floating", "--pane-id", "4"}, Error: "<nil>"}}},
		{"effect after unknown", append([]nativeReplyEvent{{Kind: "query", Unknown: true}}, nativeReplyCompleted("toggle-pane-embed-or-floating", "--pane-id", "4")...)},
		{"record after unknown", []nativeReplyEvent{{Kind: "query", Unknown: true}, {Kind: "record", Args: []string{"clear"}}}},
		{"wrong target", nativeReplyCompleted("toggle-pane-embed-or-floating", "--pane-id", "99")},
		{"wrong fullscreen action", nativeReplyCompleted("toggle-fullscreen", "--pane-id", "4")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := nativeReplyExpectedPrefix(prior, tc.events, "1", "4", 1); err == nil {
				t.Fatal("unproven prefix accepted")
			}
		})
	}
}

func TestNativeReplyPrefixOutcomeKeepsOwnedFocusAndCleanContracts(t *testing.T) {
	initial := nativeReplyRecordedOriginal()
	partial := nativeReplyExpectedState{} // completed embed/swap: no exact focus/tiling promise
	for _, tc := range []struct {
		name   string
		state  nativeReplyExpectedState
		change func(map[string]zellijpane.Pane)
		bad    bool
	}{
		{"allowed implicit agent", partial, func(map[string]zellijpane.Pane) {}, false},
		{"allowed implicit owned split", partial, func(m map[string]zellijpane.Pane) {
			p := m["agent"]
			p.IsFocused = false
			m["agent"] = p
			p = m["terminal-top"]
			p.IsFocused = true
			m["terminal-top"] = p
		}, false},
		{"wrong ID", partial, func(m map[string]zellijpane.Pane) { p := m["agent"]; p.ID = "99"; m["agent"] = p }, true},
		{"wrong tab", partial, func(m map[string]zellijpane.Pane) { p := m["agent"]; n := 99; p.TabID = &n; m["agent"] = p }, true},
		{"fullscreen", partial, func(m map[string]zellijpane.Pane) {
			p := m["agent"]
			yes := true
			p.IsFullscreen = &yes
			m["agent"] = p
		}, true},
		{"floating after embedded", partial, func(m map[string]zellijpane.Pane) {
			p := m["terminal-bottom"]
			p.IsFloating = true
			m["terminal-bottom"] = p
		}, true},
		{"overlap", partial, func(m map[string]zellijpane.Pane) { p := m["draft"]; p.Y = 0; m["draft"] = p }, true},
		{"ambiguous focus", partial, func(m map[string]zellijpane.Pane) { p := m["draft"]; p.IsFocused = true; m["draft"] = p }, true},
		{"explicit focus must match", nativeReplyExpectedState{Focus: "4"}, func(map[string]zellijpane.Pane) {}, true},
		{"clean normal exact tiling", nativeReplyExpectedState{ExactTiling: true, Focus: "1"}, func(m map[string]zellijpane.Pane) { p := m["agent"]; p.Columns--; m["agent"] = p }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			final := nativeReplyRecordedOriginal()
			tc.change(final)
			if _, err := nativeReplyCheckExpected(initial, final, tc.state, nil); (err != nil) != tc.bad {
				t.Fatalf("error=%v want reject=%v", err, tc.bad)
			}
		})
	}
	normal := nativeReplyExpectedState{ExactTiling: true, Focus: "1"}
	focusEvents := nativeReplyCompleted("toggle-pane-embed-or-floating", "--pane-id", "4")
	focusEvents = append(focusEvents, nativeReplyCompleted("change-floating-pane-coordinates", "--pane-id", "4", "--width", "75%")...)
	focusEvents = append(focusEvents, nativeReplyCompleted("show-floating-panes", "--tab-id", "1")...)
	focused, err := nativeReplyExpectedPrefix(normal, focusEvents, "1", "4", 1)
	if err != nil || !focused.Floating || !focused.Positioned || focused.ExactTiling || focused.Focus != "4" {
		t.Fatalf("clean Focus state=%+v err=%v", focused, err)
	}
	final := nativeReplyRecordedOriginal()
	p := final["agent"]
	p.IsFocused = false
	final["agent"] = p
	p = final["terminal-bottom"]
	p.IsFloating = true
	p.IsFocused = true
	p.Columns = 60
	p.Rows = 24
	p.X = 10
	p.Y = 0
	final["terminal-bottom"] = p
	oracle := []nativeReplyEvent{{Args: []string{"current-tab-info"}, Stdout: `{"display_area_columns":80,"display_area_rows":24}`}, {Args: []string{"are-floating-panes-visible"}, Stdout: "true"}}
	if got, err := nativeReplyCheckExpected(initial, final, focused, oracle); err != nil || got != "terminal-bottom" {
		t.Fatalf("Focus witness selection=%s err=%v", got, err)
	}
	p.Columns = 70
	final["terminal-bottom"] = p
	if _, err := nativeReplyCheckExpected(initial, final, focused, oracle); err == nil {
		t.Fatal("clean Focus width assertion weakened")
	}
	returned := nativeReplyCompleted("toggle-pane-embed-or-floating", "--pane-id", "4")
	returned = append(returned, nativeReplyCompleted("focus-pane-id", "terminal_1")...)
	returned = append(returned, nativeReplyEvent{Kind: "record", Args: []string{"clear"}})
	restored, err := nativeReplyExpectedPrefix(focused, returned, "1", "4", 1)
	if err != nil || restored.Floating || !restored.ExactTiling || restored.Focus != "1" {
		t.Fatalf("clean Normal state=%+v err=%v", restored, err)
	}
}
