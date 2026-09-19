package launcher

import (
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/panebirth"
)

// #287: the title poller makes no zellij call until this launch's agent pane
// has written its sidecar (zellij 0.45.1 panics when a connection accepted
// before the first client initialized the session closes). The poller's
// evidence is only fresh because the create path clears the previous launch's
// sidecar first -- before the poller, the session watcher or zellij can start.
// A stale file surviving to the spawn would pass the gate at once and put the
// poller's first list-sessions back in the new server's birth window.
func TestCreateClearsTheAgentPaneSidecarBeforeSpawningAnything(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINTED-1"}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"})
	paths, err := artifactpath.ResolveScoped(opts.Env.DataDir, "bugfix")
	if err != nil {
		t.Fatal(err)
	}
	// What the poller this create spawns will wait for: the file that must be
	// gone when it starts.
	stale, err := panebirth.Evidence(opts.Env.DataDir, "bugfix", "claude")
	if err != nil {
		t.Fatal(err)
	}
	twin := paths.Pane("codex")
	rt.files[stale] = `{"pane_id":"3","cwd":"/old"}` + "\n"
	rt.files[twin] = `{"pane_id":"3","cwd":"/old"}` + "\n"

	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("create = %d, %v", code, err)
	}
	if len(rt.filesAtPollerSpawn) != 1 || len(rt.filesAtLaunch) != 1 {
		t.Fatalf("spawns = %d, launches = %d, want 1 each", len(rt.filesAtPollerSpawn), len(rt.filesAtLaunch))
	}
	if _, ok := rt.filesAtPollerSpawn[0][stale]; ok {
		t.Fatalf("the previous launch's %s survived to the poller's spawn", stale)
	}
	if _, ok := rt.filesAtLaunch[0][stale]; ok {
		t.Fatalf("the previous launch's %s survived to LaunchSession", stale)
	}
	// Only this agent's sidecar is the poller's evidence; a twin left by
	// another agent is not the create path's to remove.
	if _, ok := rt.filesAtPollerSpawn[0][twin]; !ok {
		t.Fatalf("the create path removed another agent's sidecar %s", twin)
	}
}

// Attach never clears: the session is live, its pane wrote the sidecar at its
// own birth and will not write it again, so the attach-path poller's gate is
// that file. Removing it would leave the poller waiting for a pane that already
// ran, and the session without titles.
func TestAttachLeavesTheLivePaneSidecar(t *testing.T) {
	rt := newFakeRuntime()
	opts := baseOpts(LaunchArgs{})
	paths, err := artifactpath.ResolveScoped(opts.Env.DataDir, "live")
	if err != nil {
		t.Fatal(err)
	}
	live := paths.Pane("claude")
	rt.files[live] = `{"pane_id":"1","cwd":"/home/u/work"}` + "\n"

	if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "📁work-live", "claude"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(rt.removed, live) {
		t.Fatalf("attach removed the live pane's sidecar %s", live)
	}
	if len(rt.filesAtPollerSpawn) != 1 {
		t.Fatalf("title pollers spawned = %d, want 1", len(rt.filesAtPollerSpawn))
	}
	if _, ok := rt.filesAtPollerSpawn[0][live]; !ok {
		t.Fatalf("the attach-path poller would find no pane sidecar to pass its gate")
	}
}
