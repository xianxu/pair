package launcher

import (
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestRunLaunchUnrecordedTagDefaultsToLayout2(t *testing.T) {
	rt := newFakeRuntime()
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchLayout != "/pair/zellij/layouts/main-2.kdl" {
		t.Fatalf("layout = %q", rt.launchLayout)
	}
	if got := rt.files["/data/workbench-layout-work"]; got != "layout2\n" {
		t.Fatalf("record = %q, want layout2", got)
	}
	if rt.env["PAIR_WORKBENCH_LAYOUT"] != "layout2" {
		t.Fatalf("PAIR_WORKBENCH_LAYOUT = %q", rt.env["PAIR_WORKBENCH_LAYOUT"])
	}
}

func TestRunLaunchImplicitlyReusesRecordedLayout3(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/workbench-layout-work"] = "layout3\n"
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchLayout != "/pair/zellij/layouts/main-3.kdl" {
		t.Fatalf("layout = %q", rt.launchLayout)
	}
}

func TestRunLaunchExplicitLayoutWinsOnCreate(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/workbench-layout-work"] = "layout3\n"
	args := LaunchArgs{
		Agent:     "codex",
		ForcedTag: "work",
		Layout:    LayoutRequest{Mode: Layout2, Explicit: true},
	}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchLayout != "/pair/zellij/layouts/main-2.kdl" {
		t.Fatalf("layout = %q", rt.launchLayout)
	}
	if got := rt.files["/data/workbench-layout-work"]; got != "layout2\n" {
		t.Fatalf("record = %q", got)
	}
}

func TestRunLaunchLayoutRecordWriteFailureAbortsBeforeHandoff(t *testing.T) {
	rt := newFakeRuntime()
	rt.writeFailAt = "/data/workbench-layout-work"
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "work"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("launched despite record failure: %q", rt.launched)
	}
}

func TestRunLaunchImmediateErrorRestoresPriorLayoutRecord(t *testing.T) {
	for _, tc := range []struct {
		name     string
		previous string
		want     string
		present  bool
	}{
		{"restore previous", "layout3\n", "layout3\n", true},
		{"remove newly created", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := newFakeRuntime()
			if tc.previous != "" {
				rt.files["/data/workbench-layout-work"] = tc.previous
			}
			rt.launchErr = errors.New("exec failed")
			args := LaunchArgs{
				Agent:     "codex",
				ForcedTag: "work",
				Layout:    LayoutRequest{Mode: Layout2, Explicit: true},
			}
			code, err := run(t, baseOpts(args), rt)
			if err != nil || code != 1 {
				t.Fatalf("code=%d err=%v", code, err)
			}
			got, ok := rt.files["/data/workbench-layout-work"]
			if ok != tc.present || got != tc.want {
				t.Fatalf("record = %q present=%v, want %q present=%v", got, ok, tc.want, tc.present)
			}
		})
	}
}

func TestRunLaunchFailedPreflightDoesNotWriteLayoutRecord(t *testing.T) {
	rt := newFakeRuntime()
	rt.appendLedgerErr = errors.New("ledger failed")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "work"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, ok := rt.files["/data/workbench-layout-work"]; ok {
		t.Fatal("layout record written before preflight completed")
	}
}

func TestClassifyLiveLayout(t *testing.T) {
	for _, tc := range []struct {
		name  string
		panes []zellijpane.Pane
		want  LayoutMode
		ok    bool
	}{
		{
			name: "layout2 agent and draft",
			panes: []zellijpane.Pane{
				{Title: "codex", TerminalCommand: "pair wrap codex"},
				{Title: "draft", TerminalCommand: "nvim /data/draft-work.md"},
			},
			want: Layout2,
			ok:   true,
		},
		{
			name: "layout3 tiled terminal",
			panes: []zellijpane.Pane{
				{Title: "codex", TerminalCommand: "pair wrap codex"},
				{Title: "draft", TerminalCommand: "nvim /data/draft-work.md"},
				{Title: "terminal", TerminalCommand: "pair term"},
			},
			want: Layout3,
			ok:   true,
		},
		{
			name: "layout3 tiled terminals after split",
			panes: []zellijpane.Pane{
				{Title: "codex", TerminalCommand: "pair wrap codex"},
				{Title: "draft", TerminalCommand: "nvim /data/draft-work.md"},
				{Title: "[terminal 1]", TerminalCommand: "sh -c exec pair term"},
				{Title: "[terminal 1]", TerminalCommand: "sh -c exec pair term"},
			},
			want: Layout3,
			ok:   true,
		},
		{
			name: "layout3 legacy filler and floating terminal",
			panes: []zellijpane.Pane{
				{Title: "codex", TerminalCommand: "pair wrap codex"},
				{Title: "draft", TerminalCommand: "nvim /data/draft-work.md"},
				{Title: "terminal-filler", TerminalCommand: "tail -f /dev/null"},
				{Title: "terminal", TerminalCommand: "pair term", IsFloating: true},
			},
			want: Layout3,
			ok:   true,
		},
		{
			name:  "unknown signature",
			panes: []zellijpane.Pane{{Title: "shell", TerminalCommand: "zsh"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ClassifyLiveLayout(tc.panes)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("ClassifyLiveLayout() = (%q,%v), want (%q,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func liveLayoutRuntime(t *testing.T, recorded string) *fakeRuntime {
	t.Helper()
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessions = []Session{{Name: "pair-work-work", State: SessionDetached}}
	rt.sessionIndex.Entries = []SessionNameEntry{{
		SessionName: "pair-work-work",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "work",
	}}
	rt.inferAgent["work"] = "codex"
	if recorded != "" {
		rt.files["/data/workbench-layout-work"] = recorded
	}
	return rt
}

func TestRunLaunchImplicitLiveLayoutAttachesWithoutPrompt(t *testing.T) {
	rt := liveLayoutRuntime(t, "layout3\n")
	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.attached) != 1 || len(rt.layoutPrompts) != 0 || len(rt.deleted) != 0 {
		t.Fatalf("attached=%v prompts=%v deleted=%v", rt.attached, rt.layoutPrompts, rt.deleted)
	}
}

func TestRunLaunchExplicitSameLiveLayoutAttachesWithoutPrompt(t *testing.T) {
	rt := liveLayoutRuntime(t, "layout3\n")
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout3, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.attached) != 1 || len(rt.layoutPrompts) != 0 {
		t.Fatalf("attached=%v prompts=%v", rt.attached, rt.layoutPrompts)
	}
}

func TestRunLaunchDeclinedLiveLayoutChangeIsInert(t *testing.T) {
	rt := liveLayoutRuntime(t, "layout3\n")
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout2, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.layoutPrompts) != 1 || len(rt.attached) != 0 || len(rt.deleted) != 0 || rt.launchCount != 0 {
		t.Fatalf("prompts=%v attached=%v deleted=%v launches=%d", rt.layoutPrompts, rt.attached, rt.deleted, rt.launchCount)
	}
	if got := rt.files["/data/workbench-layout-work"]; got != "layout3\n" {
		t.Fatalf("record changed on decline: %q", got)
	}
}

func TestRunLaunchConfirmedLiveLayoutChangeRelaunchesSameTag(t *testing.T) {
	rt := liveLayoutRuntime(t, "layout3\n")
	rt.confirmLayout = true
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout2, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.deleted) != 1 || len(rt.reaped) != 1 || len(rt.killedPollers) != 1 {
		t.Fatalf("deleted=%v reaped=%v pollers=%v", rt.deleted, rt.reaped, rt.killedPollers)
	}
	if len(rt.attached) != 0 || rt.launchCount != 1 || rt.launched != "pair-work-work" {
		t.Fatalf("attached=%v launches=%d launched=%q", rt.attached, rt.launchCount, rt.launched)
	}
	if rt.launchLayout != "/pair/zellij/layouts/main-2.kdl" {
		t.Fatalf("layout = %q", rt.launchLayout)
	}
}

func TestRunLaunchPreRecordLiveSessionUsesProbe(t *testing.T) {
	rt := liveLayoutRuntime(t, "")
	rt.liveLayouts["pair-work-work"] = Layout3
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout2, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.layoutPrompts) != 1 {
		t.Fatalf("prompts=%v, want detected layout3 conflict", rt.layoutPrompts)
	}
}

func TestRunLaunchImplicitPreRecordSessionPersistsDetectedLayout(t *testing.T) {
	rt := liveLayoutRuntime(t, "")
	rt.liveLayouts["pair-work-work"] = Layout3
	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.attached) != 1 {
		t.Fatalf("attached=%v", rt.attached)
	}
	if got := rt.files["/data/workbench-layout-work"]; got != "layout3\n" {
		t.Fatalf("migrated record = %q", got)
	}
}

func TestRunLaunchLiveProbeFailureAbortsExplicitOverride(t *testing.T) {
	rt := liveLayoutRuntime(t, "")
	rt.liveLayoutErr = errors.New("probe failed")
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout2, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.attached) != 0 || len(rt.deleted) != 0 {
		t.Fatalf("attached=%v deleted=%v", rt.attached, rt.deleted)
	}
}

func TestRunLaunchLiveLayoutDeleteFailureAborts(t *testing.T) {
	rt := liveLayoutRuntime(t, "layout3\n")
	rt.confirmLayout = true
	rt.deleteErr = errors.New("delete failed")
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout2, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchCount != 0 || len(rt.reaped) != 0 {
		t.Fatalf("launches=%d reaped=%v", rt.launchCount, rt.reaped)
	}
}

func TestRunLaunchLiveLayoutStillPresentAborts(t *testing.T) {
	rt := liveLayoutRuntime(t, "layout3\n")
	rt.confirmLayout = true
	rt.keepDeletedLive = true
	args := LaunchArgs{ForcedTag: "work", Layout: LayoutRequest{Mode: Layout2, Explicit: true}}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchCount != 0 || len(rt.reaped) != 0 {
		t.Fatalf("launches=%d reaped=%v", rt.launchCount, rt.reaped)
	}
}
