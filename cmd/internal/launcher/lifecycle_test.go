package launcher

import (
	"reflect"
	"strings"
	"testing"
)

// `pair resume <livetag>` decides attach: runAttach fires (AttachSession, tag
// export, title/tty/cmux/poller refresh) and NO create happens.
func TestRunLaunchAttach(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessions = []Session{{Name: "pair-work-live", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "pair-work-live",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "live",
	}}}
	rt.blocksReuse["pair-work-live"] = true // live → decision resolves to attach
	rt.inferAgent["live"] = "codex"         // title agent comes from the on-disk record
	rt.attachCode = 7
	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "live"}), rt)
	if err != nil {
		t.Fatalf("attach unexpected err: %v", err)
	}
	if code != 7 {
		t.Fatalf("attach code = %d, want the AttachSession code 7", code)
	}
	if !reflect.DeepEqual(rt.attached, []string{"pair-work-live"}) {
		t.Fatalf("attached = %v", rt.attached)
	}
	if rt.launched != "" || rt.launchCount != 0 {
		t.Fatalf("attach must not create: launched=%q count=%d", rt.launched, rt.launchCount)
	}
	if len(rt.watchers) != 0 {
		t.Fatalf("attach must not spawn a session watcher: %v", rt.watchers)
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
	rt.quitMarkers["pair-work-bugfix"] = true
	// A non-empty raw scrollback gates the nudge; the create-flow mint writes the
	// config-bugfix-claude.json (session_id SID) that drives the resume hint.
	rt.files["/data/scrollback-bugfix-claude.raw"] = "some captured bytes"

	var stderr strings.Builder
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !reflect.DeepEqual(rt.deleted, []string{"pair-work-bugfix"}) {
		t.Fatalf("DeleteSession = %v", rt.deleted)
	}
	if !reflect.DeepEqual(rt.reaped, []string{"bugfix"}) {
		t.Fatalf("ReapNvim = %v", rt.reaped)
	}
	if !reflect.DeepEqual(rt.killedPollers, []string{"bugfix"}) {
		t.Fatalf("KillTitlePoller = %v", rt.killedPollers)
	}
	// Park-nudge prompted + parked (move mode).
	if !reflect.DeepEqual(rt.parkPrompts, []string{"pair-work-bugfix"}) {
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
	// Resume hint on stderr.
	if !strings.Contains(stderr.String(), "pair resume pair-work-bugfix") || !strings.Contains(stderr.String(), "session id:  SID") {
		t.Fatalf("resume hint missing: %q", stderr.String())
	}
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
	rt.quitMarkers["pair-work-work"] = true
	rt.files["/data/scrollback-work-claude.raw"] = "bytes"
	// Alt+n restart pending → park-nudge must be skipped in cleanup.
	rt.restartMarkers["pair-work-work"] = RestartMarker{Tag: "work", Agent: "claude"}
	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rt.parkPrompts) != 0 {
		t.Fatalf("park nudge must be skipped when a restart is pending: %v", rt.parkPrompts)
	}
}

// Alt+n restart: after the quit cleanup the restart marker drives a second,
// in-process handoff that resumes the prior session (composed --resume token).
func TestRunLaunchRestartLoopAltN(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"} // iteration 1 mints; iteration 2 resumes (no mint)
	rt.quitMarkers["pair-work-work"] = true
	rt.restartMarkers["pair-work-work"] = RestartMarker{Tag: "work", Agent: "claude"}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launchCount != 2 {
		t.Fatalf("restart loop should hand off twice, got %d", rt.launchCount)
	}
	// Iteration 2's env reflects the resume onto the minted id.
	if rt.env["PAIR_SESSION_ID"] != "MINT" {
		t.Fatalf("resumed session id = %q, want MINT", rt.env["PAIR_SESSION_ID"])
	}
	if !strings.Contains(rt.env["PAIR_AGENT_ARGS"], "--resume MINT") {
		t.Fatalf("PAIR_AGENT_ARGS = %q (want the resume token)", rt.env["PAIR_AGENT_ARGS"])
	}
}

// Shift+Alt+N restart: the saved config is dropped and the re-launch starts a
// fresh conversation (a newly minted id, no resume token).
func TestRunLaunchRestartLoopNewSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT1", "MINT2"}
	rt.quitMarkers["pair-work-work"] = true
	rt.restartMarkers["pair-work-work"] = RestartMarker{Tag: "work", Agent: "claude", NewSession: true}
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
	rt.quitMarkers["pair-work-work"] = true
	rt.restartMarkers["pair-work-work"] = RestartMarker{Tag: "work", Agent: "claude", RenameTo: "renamed"}
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
	if rt.launched != "pair-work-renamed" {
		t.Fatalf("relaunch tag = %q, want pair-work-renamed", rt.launched)
	}
	if _, ok := rt.files["/data/draft-renamed.md"]; !ok {
		t.Fatalf("sidecar not renamed; files=%v", rt.files)
	}
}

// A continue (#55 compaction) restart re-entry: drop the config, re-launch fresh
// under the same tag, and re-seed the draft from the continuation slug.
func TestRunLaunchContinueReentry(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINT"}
	rt.quitMarkers["pair-work-work"] = true
	rt.restartMarkers["pair-work-work"] = RestartMarker{Tag: "work", Agent: "claude", NewSession: true, Continue: "demo"}
	rt.continuationDocs = map[string][2]string{"demo": {"/repo/workshop/continuation/20260101-demo.md", "claude"}}

	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	opts.Env.DataDir = "/data"
	_, err := run(t, opts, rt)
	if err != nil {
		t.Fatalf("continue re-entry should be native, got %v", err)
	}
	if rt.launchCount != 2 || rt.launched != "pair-work-work" {
		t.Fatalf("relaunch = %d handoffs, tag %q (want 2, pair-work-work)", rt.launchCount, rt.launched)
	}
	if draft := rt.files["/data/draft-work.md"]; !strings.Contains(draft, "20260101-demo.md") {
		t.Fatalf("draft not re-seeded from the continuation: %q", draft)
	}
}

// SweepOrphanNvim runs once at startup with the bare tags of every live pair-*
// session (attached/detached/exited all count as live).
func TestRunLaunchSweepsOnce(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.sessions = []Session{
		{Name: "pair-a", State: SessionAttached},
		{Name: "pair-b", State: SessionExited},
	}
	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "fresh"}), rt); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rt.swept) != 1 {
		t.Fatalf("sweep should run exactly once, got %d calls", len(rt.swept))
	}
	if !reflect.DeepEqual(rt.swept[0], []string{"a", "b"}) {
		t.Fatalf("swept live tags = %v, want [a b]", rt.swept[0])
	}
}

func TestLiveTagsForSweep(t *testing.T) {
	got := liveTagsForSweep([]Session{{Name: "pair-x"}, {Name: "pair-y-2"}, {Name: "other"}})
	if !reflect.DeepEqual(got, []string{"x", "y-2", "other"}) {
		t.Fatalf("liveTagsForSweep = %v", got)
	}
}

func TestTagFromEmbedArgv(t *testing.T) {
	const dd = "/data"
	cases := map[string]string{
		"nvim --embed --headless /data/draft-work.md":             "work",
		"/usr/bin/nvim --embed /data/draft-my-tag.md --more":      "my-tag",
		"nvim --embed /data/scrollback-work-claude.ansi":          "work",
		"nvim --embed /data/scrollback-my-tag-codex.ansi":         "my-tag",
		"nvim --embed /some/other/file":                           "",
		"nvim --embed /data/scrollback-solo-claude.ansi trailing": "solo",
	}
	for argv, want := range cases {
		if got := tagFromEmbedArgv(argv, dd); got != want {
			t.Fatalf("tagFromEmbedArgv(%q) = %q, want %q", argv, got, want)
		}
	}
}
