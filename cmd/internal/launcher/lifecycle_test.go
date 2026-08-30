package launcher

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// `pair resume <livetag>` decides attach: runAttach fires (AttachSession, tag
// export, title/tty/cmux/poller refresh) and NO create happens.
func TestRunLaunchAttach(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	savedPath := "/data/config-live-codex.json"
	defaultPath := "/data/agent-default-codex.json"
	rt.files[savedPath] = `{"agent":"codex","args":["--poison-saved"],"session_id":"POISON"}`
	rt.files[defaultPath] = `{"agent":"codex","args":["--poison-default"]}`
	savedBefore, defaultBefore := rt.files[savedPath], rt.files[defaultPath]
	rt.sessions = []Session{{Name: "📁work-live", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-live",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "live",
	}}}
	rt.blocksReuse["📁work-live"] = true // live → decision resolves to attach
	rt.inferAgent["live"] = "codex"     // title agent comes from the on-disk record
	rt.attachCode = 7
	opts := baseOpts(LaunchArgs{ForcedTag: "live"})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if err != nil {
		t.Fatalf("attach unexpected err: %v", err)
	}
	if code != 7 {
		t.Fatalf("attach code = %d, want the AttachSession code 7", code)
	}
	if !reflect.DeepEqual(rt.attached, []string{"📁work-live"}) {
		t.Fatalf("attached = %v", rt.attached)
	}
	if rt.launched != "" || rt.launchCount != 0 {
		t.Fatalf("attach must not create: launched=%q count=%d", rt.launched, rt.launchCount)
	}
	if len(rt.watchers) != 0 {
		t.Fatalf("attach must not spawn a session watcher: %v", rt.watchers)
	}
	if _, ok := rt.env["PAIR_AGENT_ARGS"]; ok {
		t.Fatalf("attach exported create args: %q", rt.env["PAIR_AGENT_ARGS"])
	}
	if rt.files[savedPath] != savedBefore || rt.files[defaultPath] != defaultBefore {
		t.Fatalf("attach mutated create inputs: saved=%q default=%q", rt.files[savedPath], rt.files[defaultPath])
	}
	if rt.env["PAIR_TAG"] != "live" {
		t.Fatalf("PAIR_TAG = %q", rt.env["PAIR_TAG"])
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "live|codex" {
		t.Fatalf("title poller = %v (want the inferred codex agent)", rt.pollers)
	}
	if len(rt.titles) != 1 || len(rt.ttyRecorded) != 1 || len(rt.cmux) != 1 {
		t.Fatalf("attach refresh effects missing: %v %v %v", rt.titles, rt.ttyRecorded, rt.cmux)
	}
}

// Alt+x quit after a create: the quit marker present → full teardown (delete,
// reap, sidecar removal, poller kill, cmux reset) and the park-nudge fires
// (interactive tty + non-empty raw + no restart pending).
func TestRunLaunchQuitCleanup(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.isTTY = true
	rt.confirmPark = true
	rt.parkOK = true
	rt.cmuxOwned["bugfix"] = true
	rt.quitMarkers["📁work-bugfix"] = true
	// A non-empty raw scrollback gates the nudge; the create-flow mint writes the
	// config-bugfix-claude.json (session_id SID) that drives the resume hint.
	rt.files["/data/scrollback-bugfix-claude.raw"] = "some captured bytes"

	var stderr strings.Builder
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !reflect.DeepEqual(rt.deleted, []string{"📁work-bugfix"}) {
		t.Fatalf("DeleteSession = %v", rt.deleted)
	}
	if !reflect.DeepEqual(rt.reaped, []string{"bugfix"}) {
		t.Fatalf("ReapNvim = %v", rt.reaped)
	}
	if !reflect.DeepEqual(rt.killedPollers, []string{"bugfix"}) {
		t.Fatalf("KillTitlePoller = %v", rt.killedPollers)
	}
	// Park-nudge prompted + parked (move mode).
	if !reflect.DeepEqual(rt.parkPrompts, []string{"📁work-bugfix"}) {
		t.Fatalf("park prompts = %v", rt.parkPrompts)
	}
	if !reflect.DeepEqual(rt.parked, []string{"bugfix|claude|true"}) {
		t.Fatalf("ParkScrollback = %v", rt.parked)
	}
	// Parked → the raw capture is NOT removed; the .ansi always is.
	if contains(rt.removed, "/data/scrollback-bugfix-claude.raw") {
		t.Fatalf("parked raw must be preserved; removed=%v", rt.removed)
	}
	// #97: the agent's pane file is a per-(tag,agent) sidecar too — quit must
	// remove it (quitAgent falls back to step.agent="claude" here) so no stale
	// twin survives to mislead the frame poller.
	for _, want := range []string{"/data/outer-tty-bugfix", "/data/agent-bugfix", "/data/scrollback-bugfix-claude.ansi", "/data/adapt-bugfix.jsonl", "/data/pane-bugfix-claude.json"} {
		if !contains(rt.removed, want) {
			t.Fatalf("sidecar %q not removed; removed=%v", want, rt.removed)
		}
	}
	// cmux reset to the cwd basename + ownership released.
	last := rt.cmux[len(rt.cmux)-1]
	if last != "bugfix|work" { // baseOpts cwd is /home/u/work
		t.Fatalf("cmux reset = %q, want bugfix|work", last)
	}
	if rt.cmuxCleared != 1 {
		t.Fatalf("ClearCmuxOwner calls = %d", rt.cmuxCleared)
	}
	if strings.Contains(stderr.String(), "pair resume bugfix") || strings.Contains(stderr.String(), "session id:") {
		t.Fatalf("provisional pre-round launch advertised recovery: %q", stderr.String())
	}
}

func TestRunLaunchQuitCleanupPrintsOnlyEstablishedSessionID(t *testing.T) {
	rt := newFakeRuntime()
	rt.quitMarkers["📁work-bugfix"] = true
	rt.establishedSessions["bugfix|claude"] = "ESTABLISHED"
	rt.files["/data/config-bugfix-claude.json"] = `{"agent":"claude","args":[],"session_id":"STALE"}`

	var stderr strings.Builder
	runCleanup(Env{DataDir: "/data", Cwd: "/home/u/work"}, rt, launchStep{tag: "bugfix", agent: "claude", session: "📁work-bugfix"}, "scope", 0, &stderr)
	if !strings.Contains(stderr.String(), "session id:  ESTABLISHED") || strings.Contains(stderr.String(), "STALE") {
		t.Fatalf("resume hint used non-authoritative identity: %q", stderr.String())
	}
}

func TestRunCleanupUsesTypedContextBoundary(t *testing.T) {
	t.Run("cancelled before effects", func(t *testing.T) {
		rt := newFakeRuntime()
		rt.quitMarkers["pair-work"] = true
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, ran := runCleanupContext(ctx, Env{DataDir: "/data", Cwd: "/repo"}, rt, launchStep{tag: "work", agent: "claude", session: "pair-work"}, "scope", 0, &strings.Builder{})
		if !ran || result.Outcome != pairlifecycle.CompletionFailure || len(result.Failures) != 1 || result.Failures[0].Code != pairlifecycle.FailureTimeout {
			t.Fatalf("ran=%v result=%#v", ran, result)
		}
		if len(rt.deleted) != 0 || len(rt.reaped) != 0 {
			t.Fatalf("cancelled cleanup performed effects: delete=%v reap=%v", rt.deleted, rt.reaped)
		}
	})

	t.Run("quiescence failure gates destructive stages", func(t *testing.T) {
		rt := newFakeRuntime()
		rt.quitMarkers["pair-work"] = true
		rt.deleteErr = errors.New("absence unproved")
		result, ran := runCleanupContext(context.Background(), Env{DataDir: "/data", Cwd: "/repo"}, rt, launchStep{tag: "work", agent: "claude", session: "pair-work"}, "scope", 0, &strings.Builder{})
		if !ran || result.Outcome != pairlifecycle.CompletionFailure || result.Failures[0].Stage != pairlifecycle.StageSessionQuiescence {
			t.Fatalf("ran=%v result=%#v", ran, result)
		}
		if len(rt.reaped) != 0 || len(rt.removed) != 0 || len(rt.killedPollers) != 0 {
			t.Fatalf("failed quiescence leaked later effects: reap=%v remove=%v poller=%v", rt.reaped, rt.removed, rt.killedPollers)
		}
	})
}

// A detach (Alt+d) leaves no quit marker: cleanup is a complete no-op.
func TestRunLaunchDetachNoCleanup(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.isTTY = true
	rt.files["/data/scrollback-bugfix-claude.raw"] = "bytes"
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.deleted) != 0 || len(rt.reaped) != 0 || len(rt.parkPrompts) != 0 || len(rt.killedPollers) != 0 {
		t.Fatalf("detach must not clean up: del=%v reap=%v park=%v kill=%v",
			rt.deleted, rt.reaped, rt.parkPrompts, rt.killedPollers)
	}
}

// The park-nudge is skipped when a restart is pending (a restart keeps the work),
// even with an interactive tty + non-empty raw.
func TestRunLaunchParkSkippedOnRestart(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID", "SID2"}
	rt.isTTY = true
	rt.confirmPark = true
	rt.parkOK = true
	rt.quitMarkers["📁work"] = true
	rt.files["/data/scrollback-work-claude.raw"] = "bytes"
	// Alt+n restart pending → park-nudge must be skipped in cleanup.
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude"}
	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rt.parkPrompts) != 0 {
		t.Fatalf("park nudge must be skipped when a restart is pending: %v", rt.parkPrompts)
	}
}

// Alt+n restart: after the quit cleanup the restart marker drives a second,
// in-process handoff that starts fresh when the first launch never established.
func TestRunLaunchRestartLoopAltN(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"} // iteration 1 mints; iteration 2 resumes (no mint)
	rt.quitMarkers["📁work"] = true
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude"}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchCount != 2 {
		t.Fatalf("restart loop should hand off twice, got %d", rt.launchCount)
	}
	if rt.env["PAIR_SESSION_ID"] != "" || strings.Contains(rt.env["PAIR_AGENT_ARGS"], "--resume MINT") {
		t.Fatalf("pre-round restart claimed recovery: id=%q args=%q", rt.env["PAIR_SESSION_ID"], rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchRestartLoopAltNCodexUsesMarkerSessionID(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"} // iteration 1 mints; iteration 2 resumes from marker
	rt.quitMarkers["📁work"] = true
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "codex", SessionID: "SID-LIVE"}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchCount != 2 {
		t.Fatalf("restart loop should hand off twice, got %d", rt.launchCount)
	}
	if rt.env["PAIR_SESSION_ID"] != "SID-LIVE" {
		t.Fatalf("resumed session id = %q, want SID-LIVE", rt.env["PAIR_SESSION_ID"])
	}
	if rt.env["PAIR_AGENT_ARGS"] != "resume SID-LIVE --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q, want codex resume marker id", rt.env["PAIR_AGENT_ARGS"])
	}
}

// Shift+Alt+N restart: the saved config is dropped and the re-launch starts a
// fresh conversation (a newly minted id, no resume token).
func TestRunLaunchRestartLoopNewSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT1", "MINT2"}
	rt.quitMarkers["📁work"] = true
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", NewSession: true}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchCount != 2 {
		t.Fatalf("new-session restart should hand off twice, got %d", rt.launchCount)
	}
	if !contains(rt.removed, "/data/config-work-claude.json") {
		t.Fatalf("Shift+Alt+N must drop the saved config; removed=%v", rt.removed)
	}
	if rt.env["PAIR_SESSION_ID"] != "MINT2" {
		t.Fatalf("fresh session id = %q, want the second mint MINT2", rt.env["PAIR_SESSION_ID"])
	}
	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "--resume") {
		t.Fatalf("fresh conversation must carry no resume token: %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

// A rename_to restart re-entry (M5b): after the quit cleanup, the loop moves the
// tag-scoped sidecars old→new, then re-launches natively under the NEW tag with
// args derived from the (renamed) saved config — no shell fallback.
func TestRunLaunchRenameReentry(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"}
	rt.quitMarkers["📁work"] = true
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", RenameTo: "renamed"}
	rt.files["/data"] = ""                       // data dir exists (rename gate)
	rt.files["/data/draft-work.md"] = "the work" // a sidecar to move

	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	opts.Env.DataDir = "/data"
	_, err := run(t, opts, rt)
	if err != nil {
		t.Fatalf("rename re-entry should be native, got %v", err)
	}
	if rt.launchCount != 2 {
		t.Fatalf("expected two handoffs (work, then renamed), got %d", rt.launchCount)
	}
	if rt.launched != "📁work-renamed" {
		t.Fatalf("relaunch tag = %q, want 📁work-renamed", rt.launched)
	}
	if _, ok := rt.files["/data/draft-renamed.md"]; !ok {
		t.Fatalf("sidecar not renamed; files=%v", rt.files)
	}
}

func TestRunLaunchRenameReentryPreservesExactPrefixedTags(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"}
	rt.quitMarkers["📁work-pair-demo"] = true
	rt.restartMarkers["📁work-pair-demo"] = RestartMarker{Tag: "pair-demo", Agent: "claude", RenameTo: "pair-new"}
	rt.files["/data"] = ""
	rt.files["/data/draft-pair-demo.md"] = "exact prefixed work"

	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "pair-demo"})
	opts.Env.DataDir = "/data"
	_, err := run(t, opts, rt)
	if err != nil {
		t.Fatalf("rename re-entry should be native, got %v", err)
	}
	if rt.launchCount != 2 || rt.launched != "📁work-pair-new" {
		t.Fatalf("relaunch = %d handoffs, session %q; want 2 and exact pair-new tag", rt.launchCount, rt.launched)
	}
	if got := rt.files["/data/draft-pair-new.md"]; got != "exact prefixed work" {
		t.Fatalf("renamed draft = %q, want exact prefixed rename", got)
	}
}

func TestRunLaunchRenameReentryIgnoresOtherScopeTargetTag(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"}
	rt.sessions = []Session{{Name: "pair-other-renamed", State: SessionAttached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "pair-other-renamed", ScopeKey: "scope2", Tag: "renamed"},
	}}
	rt.quitMarkers["📁work"] = true
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", RenameTo: "renamed"}
	rt.files["/global/repos/scope1"] = ""
	rt.files["/global/repos/scope1/draft-work.md"] = "the work"

	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	opts.GlobalDataDir = "/global"
	opts.Env.DataDir = "/global/repos/scope1"
	_, err := run(t, opts, rt)
	if err != nil {
		t.Fatalf("rename re-entry should be native, got %v", err)
	}
	if rt.launched != "📁work-renamed" {
		t.Fatalf("relaunch tag = %q, want 📁work-renamed", rt.launched)
	}
	if _, ok := rt.files["/global/repos/scope1/draft-renamed.md"]; !ok {
		t.Fatalf("sidecar not renamed; files=%v", rt.files)
	}
}

// A continue (#55 compaction) restart re-entry: drop the config, re-launch fresh
// under the same tag, and re-seed the draft from the continuation slug.
func TestRunLaunchContinueReentry(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"}
	rt.quitMarkers["📁work"] = true
	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", NewSession: true, Continue: "demo"}
	rt.continuationDocs = map[string][2]string{"demo": {"/repo/workshop/continuation/20260101-demo.md", "claude"}}

	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	opts.Env.DataDir = "/data"
	_, err := run(t, opts, rt)
	if err != nil {
		t.Fatalf("continue re-entry should be native, got %v", err)
	}
	if rt.launchCount != 2 || rt.launched != "📁work" {
		t.Fatalf("relaunch = %d handoffs, tag %q (want 2, 📁work)", rt.launchCount, rt.launched)
	}
	if draft := rt.files["/data/draft-work.md"]; !strings.Contains(draft, "20260101-demo.md") {
		t.Fatalf("draft not re-seeded from the continuation: %q", draft)
	}
}

// SweepOrphanNvim runs once at startup with the repo-local tags of every live
// session in the current scope (attached/detached/exited all count as live).
func TestRunLaunchSweepsOnce(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.sessions = []Session{
		{Name: "📁work-a", State: SessionAttached},
		{Name: "📁work-b", State: SessionExited},
		{Name: "pair-other-a", State: SessionAttached},
	}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "📁work-a", ScopeKey: "scope1", Tag: "a"},
		{SessionName: "📁work-b", ScopeKey: "scope1", Tag: "b"},
		{SessionName: "pair-other-a", ScopeKey: "scope2", Tag: "a"},
	}}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "fresh"})
	opts.GlobalDataDir = "/global"
	opts.Env.DataDir = "/global/repos/scope1"
	if _, err := run(t, opts, rt); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rt.swept) != 1 {
		t.Fatalf("sweep should run exactly once, got %d calls", len(rt.swept))
	}
	if !reflect.DeepEqual(rt.swept[0], []string{"a", "b"}) {
		t.Fatalf("swept live tags = %v, want [a b]", rt.swept[0])
	}
}

func TestRunLaunchRefusesUnreadableSessionIndexBeforeOrphanSweep(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessionIndexErr = errors.New("index unreadable")
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "fresh"})
	code, err := run(t, opts, rt)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(rt.swept) != 0 || rt.launched != "" {
		t.Fatalf("unreadable index must stop before sweep/launch: swept=%v launched=%q", rt.swept, rt.launched)
	}
}

func TestLiveTagsForSweep(t *testing.T) {
	index := SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "📁work-x", ScopeKey: "scope1", Tag: "x"},
		{SessionName: "📁work-y", ScopeKey: "scope2", Tag: "y"},
	}}
	got := liveTagsForSweep([]Session{{Name: "📁work-x"}, {Name: "📁work-y"}, {Name: "pair-legacy"}, {Name: "other"}}, index, "scope1")
	if !reflect.DeepEqual(got, []string{"x", "legacy"}) {
		t.Fatalf("liveTagsForSweep = %v", got)
	}
}
