package layoutcmd

import (
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestExpandRecordRoundTripAndLegacy(t *testing.T) {
	for _, tc := range []struct {
		line string
		want ExpandRecord
	}{
		{"", ExpandRecord{}},
		{"3", ExpandRecord{Return: "3"}},
		{"3 swap=third-split order=2,5", ExpandRecord{Return: "3", Swap: "third-split", Order: []string{"2", "5"}}},
		{"3 swap=BASE", ExpandRecord{Return: "3", Swap: "BASE"}},
		{"3 future=x swap=minimized", ExpandRecord{Return: "3", Swap: "minimized"}},
	} {
		got := DecodeExpandRecord(tc.line)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("decode %q = %+v, want %+v", tc.line, got, tc.want)
		}
		if tc.line != "" && got.Encode() != DecodeExpandRecord(got.Encode()).Encode() {
			t.Fatalf("encode not stable for %q", tc.line)
		}
	}
	if got := (ExpandRecord{Return: "3", Swap: "BASE", Order: []string{"2", "5"}}).Encode(); got != "3 swap=BASE order=2,5" {
		t.Fatalf("encode = %q", got)
	}
	if got := (ExpandRecord{Return: "3"}).Encode(); got != "3" {
		t.Fatalf("bare record must stay legacy-compatible: %q", got)
	}
	// The record rides FullscreenReturnStore: the encoding must satisfy its validation.
	store := workbenchshortcut.FullscreenReturnStore{DataDir: t.TempDir(), Tag: "rec"}
	line := ExpandRecord{Return: "terminal_12", Swap: "small-split", Order: []string{"terminal_3", "terminal_9"}}.Encode()
	if err := store.Write(line); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Read(); err != nil || got != line {
		t.Fatalf("store round trip %q %v", got, err)
	}
}

func TestRestoreTarget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		halves int
		want   string
	}{
		{"", 1, ""},
		{"BASE", 1, "BASE"},
		{"third", 1, "third"},
		{"BASE", 2, "small-split"},
		{"third", 2, "third-split"},
		{"minimized-split", 2, "minimized-split"},
		{"small-split", 1, "BASE"},
		{"third-split", 0, "third"},
	} {
		if got := restoreTarget(tc.name, tc.halves); got != tc.want {
			t.Fatalf("restoreTarget(%q,%d)=%q want %q", tc.name, tc.halves, got, tc.want)
		}
	}
}

func TestParseTabLayout(t *testing.T) {
	name, dirty, err := ParseTabLayout([]byte(`{"active_swap_layout_name":"third","is_swap_layout_dirty":true,"tab_id":1}`))
	if err != nil || name != "third" || !dirty {
		t.Fatalf("%q %v %v", name, dirty, err)
	}
	name, dirty, err = ParseTabLayout([]byte(`{"active_swap_layout_name":null,"is_swap_layout_dirty":false}`))
	if err != nil || name != "" || dirty {
		t.Fatalf("null name: %q %v %v", name, dirty, err)
	}
	if _, _, err := ParseTabLayout([]byte(`not json`)); err == nil {
		t.Fatal("accepted invalid JSON")
	}
}

func modePanes() []zellijpane.Pane {
	off := false
	return []zellijpane.Pane{
		{ID: "0", IsPlugin: true, IsFocused: true},
		{ID: "1", Title: "agent", TerminalCommand: "pair wrap codex", IsFullscreen: &off, Y: 0},
		{ID: "2", Title: "draft", TerminalCommand: "nvim -u /pair/nvim/init.lua", IsFocused: true, IsFullscreen: &off, Y: 12},
		{ID: "3", Title: "terminal", IsFullscreen: &off, X: 40, Y: 0},
	}
}

func TestObserveRightPaneMode(t *testing.T) {
	on := true
	panes := modePanes()
	if mode, _, err := ObserveRightPaneMode(panes, nil); err != nil || mode != ModeNormal {
		t.Fatalf("tiled: %v %v", mode, err)
	}
	panes[3].IsFloating = true
	if mode, id, err := ObserveRightPaneMode(panes, nil); err != nil || mode != ModeFocus || id != "3" {
		t.Fatalf("floating: %v %s %v", mode, id, err)
	}
	if !FocusModeActive(panes, nil) {
		t.Fatal("FocusModeActive disagrees with ObserveRightPaneMode")
	}
	// A floating pane with unknown fullscreen state is still focus mode: the
	// floating observation alone decides it.
	panes[3].IsFullscreen = nil
	if mode, _, err := ObserveRightPaneMode(panes, nil); err != nil || mode != ModeFocus {
		t.Fatalf("floating nil-fs: %v %v", mode, err)
	}
	panes = modePanes()
	panes[3].IsFullscreen = &on
	if mode, id, err := ObserveRightPaneMode(panes, nil); err != nil || mode != ModeMaximize || id != "3" {
		t.Fatalf("fullscreen: %v %s %v", mode, id, err)
	}
	if FocusModeActive(panes, nil) {
		t.Fatal("maximize reported as focus")
	}
	panes = append(modePanes(), zellijpane.Pane{ID: "4", Title: "terminal 2", IsFloating: true})
	panes[3].IsFloating = true
	if _, _, err := ObserveRightPaneMode(panes, nil); err == nil {
		t.Fatal("two floating right terminals accepted")
	}
	if FocusModeActive(panes, nil) {
		t.Fatal("ambiguous observation reported as focus")
	}
	// Floating panes that are not right terminals (review, key help) are not focus mode.
	panes = append(modePanes(), zellijpane.Pane{ID: "9", Title: "review", TerminalCommand: "nvim review", IsFloating: true})
	if mode, _, err := ObserveRightPaneMode(panes, nil); err != nil || mode != ModeNormal {
		t.Fatalf("foreign floating: %v %v", mode, err)
	}
}

func TestPlanRightPaneCycle(t *testing.T) {
	on := true
	split := func() []zellijpane.Pane {
		off := false
		return append(modePanes(), zellijpane.Pane{ID: "4", Title: "custom", IsFullscreen: &off, X: 40, Y: 12})
	}
	registered := []string{"3", "4"}
	floating := func(p []zellijpane.Pane, id string) []zellijpane.Pane {
		for i := range p {
			if p[i].ID == id {
				p[i].IsFloating = true
			}
		}
		return p
	}
	fullscreen := func(p []zellijpane.Pane, id string) []zellijpane.Pane {
		for i := range p {
			if p[i].ID == id {
				p[i].IsFullscreen = &on
			}
		}
		return p
	}
	for _, tc := range []struct {
		name   string
		in     RightPaneInput
		from   RightPaneMode
		to     RightPaneMode
		record ExpandRecord
		steps  []Step
	}{
		{"normal-to-focus", RightPaneInput{Panes: modePanes(), Caller: "2", Swap: "third"}, ModeNormal, ModeFocus,
			ExpandRecord{Return: "2", Swap: "third"},
			[]Step{{StepSave, ""}, {StepFloat, "3"}, {StepPlace, "3"}, {StepShow, ""}, {StepNudge, ""}}},
		{"normal-to-focus-split-caller-half", RightPaneInput{Panes: split(), Caller: "4", Last: "3", Registered: registered, Swap: "small-split"}, ModeNormal, ModeFocus,
			ExpandRecord{Return: "4", Swap: "small-split", Order: []string{"3", "4"}},
			[]Step{{StepSave, ""}, {StepFloat, "4"}, {StepPlace, "4"}, {StepShow, ""}, {StepNudge, ""}}},
		{"focus-to-maximize", RightPaneInput{Panes: floating(modePanes(), "3"), Caller: "3", Record: ExpandRecord{Return: "2", Swap: "third"}}, ModeFocus, ModeMaximize,
			ExpandRecord{Return: "2", Swap: "third"},
			[]Step{{StepEmbed, "3"}, {StepRestore, "3"}, {StepFullscreen, "3"}, {StepNudge, ""}}},
		{"focus-to-maximize-split", RightPaneInput{Panes: floating(split(), "4"), Caller: "4", Registered: registered, Record: ExpandRecord{Return: "2", Swap: "BASE", Order: []string{"3", "4"}}}, ModeFocus, ModeMaximize,
			ExpandRecord{Return: "2", Swap: "BASE", Order: []string{"3", "4"}},
			[]Step{{StepEmbed, "4"}, {StepRestore, "4"}, {StepFullscreen, "4"}, {StepNudge, ""}}},
		{"maximize-to-normal", RightPaneInput{Panes: fullscreen(modePanes(), "3"), Caller: "3", Record: ExpandRecord{Return: "2", Swap: "third"}}, ModeMaximize, ModeNormal,
			ExpandRecord{Return: "2", Swap: "third"},
			[]Step{{StepFullscreen, "3"}, {StepFocus, "2"}, {StepClear, ""}}},
		{"maximize-to-normal-same-pane", RightPaneInput{Panes: fullscreen(modePanes(), "3"), Caller: "3", Record: ExpandRecord{Return: "3"}}, ModeMaximize, ModeNormal,
			ExpandRecord{Return: "3"},
			[]Step{{StepFullscreen, "3"}, {StepClear, ""}}},
		{"no-right-terminal", RightPaneInput{Panes: modePanes()[:3], Caller: "2"}, ModeNormal, ModeNormal, ExpandRecord{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanRightPane(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if plan.From != tc.from || plan.To != tc.to || !reflect.DeepEqual(plan.Record, tc.record) || !reflect.DeepEqual(plan.Steps, tc.steps) {
				t.Fatalf("got %+v\nwant from=%v to=%v record=%+v steps=%+v", plan, tc.from, tc.to, tc.record, tc.steps)
			}
		})
	}
	if _, err := PlanRightPane(RightPaneInput{Panes: modePanes(), Caller: "99"}); err == nil {
		t.Fatal("stale caller accepted")
	}
}
