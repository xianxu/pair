package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestAdvancePreviewScheduleKeepsOneRunningAndLatestPending(t *testing.T) {
	var state PreviewSchedule
	request1 := PreviewRequest{Generation: 1, Path: "/one", Agent: "claude"}
	request2 := PreviewRequest{Generation: 2, Path: "/two", Agent: "claude"}
	request3 := PreviewRequest{Generation: 3, Path: "/three", Agent: "codex"}

	state, effects := AdvancePreviewSchedule(state, PreviewScheduleEvent{Kind: PreviewRequested, Request: request1})
	if !reflect.DeepEqual(effects, []PreviewScheduleEffect{{Kind: PreviewStart, Request: request1}}) {
		t.Fatalf("first request effects = %+v", effects)
	}
	state, effects = AdvancePreviewSchedule(state, PreviewScheduleEvent{Kind: PreviewRequested, Request: request2})
	if !reflect.DeepEqual(effects, []PreviewScheduleEffect{{Kind: PreviewCancel, Generation: 1}}) {
		t.Fatalf("second request effects = %+v", effects)
	}
	state, effects = AdvancePreviewSchedule(state, PreviewScheduleEvent{Kind: PreviewRequested, Request: request3})
	if len(effects) != 0 || state.Pending == nil || state.Pending.Generation != 3 {
		t.Fatalf("latest request did not replace pending: state=%+v effects=%+v", state, effects)
	}

	stale := state
	state, effects = AdvancePreviewSchedule(state, PreviewScheduleEvent{Kind: PreviewFinished, Generation: 99})
	if len(effects) != 0 || !reflect.DeepEqual(state, stale) {
		t.Fatalf("stale completion changed schedule: state=%+v effects=%+v", state, effects)
	}
	state, effects = AdvancePreviewSchedule(state, PreviewScheduleEvent{Kind: PreviewFinished, Generation: 1})
	if !reflect.DeepEqual(effects, []PreviewScheduleEffect{{Kind: PreviewStart, Request: request3}}) || state.Running == nil || state.Running.Generation != 3 || state.Pending != nil {
		t.Fatalf("matching completion did not start latest: state=%+v effects=%+v", state, effects)
	}
	state, effects = AdvancePreviewSchedule(state, PreviewScheduleEvent{Kind: PreviewFinished, Generation: 1})
	if len(effects) != 0 || state.Running == nil || state.Running.Generation != 3 {
		t.Fatalf("duplicate completion retired current request: state=%+v effects=%+v", state, effects)
	}
}

func TestAdvancePreviewScheduleRejectsZeroGeneration(t *testing.T) {
	state, effects := AdvancePreviewSchedule(PreviewSchedule{}, PreviewScheduleEvent{
		Kind: PreviewRequested, Request: PreviewRequest{Path: "/repo", Agent: "claude"},
	})
	if state.Running != nil || state.Pending != nil || len(effects) != 0 {
		t.Fatalf("zero generation entered schedule: state=%+v effects=%+v", state, effects)
	}
}

func TestReduceMenuStartPreviewArmsOneGenerationBoundSubmit(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude", "codex"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	for _, r := range "/repo" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.Generation != state.CurrentFrame().Generation || effects[0].Preview.Path != "/repo" {
		t.Fatalf("leaving path did not request preview: state=%+v effects=%+v", state, effects)
	}
	generation := state.CurrentFrame().Generation

	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().SubmitGeneration != generation || state.Notice != "resolving" {
		t.Fatalf("pending submit = state %+v effects %+v", state, effects)
	}
	stale := couchcore.PreparedStart{Token: "stale", Resolution: couchcore.StartResolution{CanonicalPath: "/stale"}}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation - 1, Prepared: &stale})
	if len(effects) != 0 || state.CurrentFrame().SubmitGeneration != generation {
		t.Fatalf("stale preview consumed armed submit: state=%+v effects=%+v", state, effects)
	}

	prepared := couchcore.PreparedStart{Token: "accepted", Resolution: couchcore.StartResolution{
		CanonicalPath: "/repo", Profile: couchcore.LaunchProfile{Agent: "claude"},
	}}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, Prepared: &prepared})
	want := []MenuEffect{{Operation: "start", Attempt: 1, Args: map[string]string{
		"path": "/repo", "agent": "claude", "token": "accepted",
	}}}
	if !reflect.DeepEqual(effects, want) || state.CurrentFrame().SubmitGeneration != 0 {
		t.Fatalf("accepted preview did not submit once: state=%+v effects=%+v want=%+v", state, effects, want)
	}
	_, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, Prepared: &prepared})
	if len(effects) != 0 {
		t.Fatalf("duplicate preview redispatched start: %+v", effects)
	}
}

func TestReduceMenuStartPreviewEditCancelsArmedSubmitAndFailurePreservesForm(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Preview == nil || state.CurrentFrame().SubmitGeneration == 0 {
		t.Fatalf("submit without preview did not arm and request: state=%+v effects=%+v", state, effects)
	}
	oldGeneration := state.CurrentFrame().Generation
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	if state.CurrentFrame().SubmitGeneration != 0 || state.CurrentFrame().Generation == oldGeneration {
		t.Fatalf("edit did not cancel armed generation: %+v", state.CurrentFrame())
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	generation := state.CurrentFrame().Generation
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, Error: "policy unavailable"})
	if len(effects) != 0 || state.CurrentFrame().Path != "x" || state.Notice != "policy unavailable" {
		t.Fatalf("preview failure did not preserve form: state=%+v effects=%+v", state, effects)
	}
}

func TestReduceMenuStartPreviewPreservesOptionalAgentAndAcceptedProvenance(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude", "codex"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	for _, r := range "/repo" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}

	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 1 || effects[0].Preview == nil || effects[0].Preview.Agent != "" {
		t.Fatalf("non-sticky preview made fallback agent explicit: %+v", effects)
	}
	generation := state.CurrentFrame().Generation
	prepared := couchcore.PreparedStart{Token: "accepted", Resolution: couchcore.StartResolution{
		CanonicalPath: "/repo", Profile: couchcore.LaunchProfile{Agent: "codex", Argv: []string{"--search"}},
		AgentSource: couchcore.AgentSourcePath, ArgvSource: couchcore.ArgvSourcePath,
	}}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, Prepared: &prepared})
	if len(effects) != 0 {
		t.Fatalf("unarmed preview dispatched: %+v", effects)
	}
	rendered := RenderMenu(state, 80, 20, time.Time{}, false)
	if !strings.Contains(rendered, "agent codex") || !strings.Contains(rendered, "args  path") {
		t.Fatalf("accepted start provenance not rendered: %q", rendered)
	}
}

func TestReduceMenuStartPreviewReusesAcceptedGeneration(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 1 || effects[0].Preview == nil {
		t.Fatalf("initial preview = %+v", effects)
	}
	generation := state.CurrentFrame().Generation
	prepared := couchcore.PreparedStart{Token: "accepted", Resolution: couchcore.StartResolution{
		CanonicalPath: "/repo", Profile: couchcore.LaunchProfile{Agent: "claude", Argv: []string{}},
		AgentSource: couchcore.AgentSourceRoot, ArgvSource: couchcore.ArgvSourceRepoDefault,
	}}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, Prepared: &prepared})
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, effects = reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 0 {
		t.Fatalf("accepted generation requested another grant: %+v", effects)
	}
}

func TestReduceMenuPreviewIdentitySurvivesEscapeAndReopen(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 1 || effects[0].Preview == nil {
		t.Fatalf("first form preview = %+v", effects)
	}
	first := *effects[0].Preview
	var schedule PreviewSchedule
	schedule, _ = AdvancePreviewSchedule(schedule, PreviewScheduleEvent{Kind: PreviewRequested, Request: first})

	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, effects = reduceKey(state, PanelKey{Kind: KeyTab})
	if len(effects) != 1 || effects[0].Preview == nil {
		t.Fatalf("reopened form preview = %+v", effects)
	}
	second := *effects[0].Preview
	if second.Generation == first.Generation {
		t.Fatalf("form lifetimes reused preview identity %d", first.Generation)
	}
	schedule, effectsSchedule := AdvancePreviewSchedule(schedule, PreviewScheduleEvent{Kind: PreviewRequested, Request: second})
	if schedule.Pending == nil || schedule.Pending.Generation != second.Generation || len(effectsSchedule) != 1 || effectsSchedule[0].Kind != PreviewCancel {
		t.Fatalf("reopened request did not supersede old running request: state=%+v effects=%+v", schedule, effectsSchedule)
	}

	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	old := couchcore.PreparedStart{Token: "old-token", Resolution: couchcore.StartResolution{
		CanonicalPath: "/old", Profile: couchcore.LaunchProfile{Agent: "claude"},
	}}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: first.Generation, Prepared: &old})
	frame := state.CurrentFrame()
	if len(effects) != 0 || frame.PreviewToken != "" || frame.PreviewAccepted != 0 || frame.SubmitGeneration != second.Generation {
		t.Fatalf("old form completion populated or launched reopened form: state=%+v effects=%+v", state, effects)
	}

	schedule, effectsSchedule = AdvancePreviewSchedule(schedule, PreviewScheduleEvent{Kind: PreviewFinished, Generation: first.Generation})
	if schedule.Running == nil || schedule.Running.Generation != second.Generation || len(effectsSchedule) != 1 || effectsSchedule[0].Kind != PreviewStart {
		t.Fatalf("old completion did not admit reopened request: state=%+v effects=%+v", schedule, effectsSchedule)
	}
}
