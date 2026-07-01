package opener

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

type viewerCall struct {
	lua, file string
	env       []string
}

// fakeRuntime is a scriptable Runtime for the opener orchestration tests.
type fakeRuntime struct {
	sleeps      int
	pid         string
	alive       map[string]bool
	files       map[string]string
	wrote       map[string]string
	removed     []string
	sizes       map[string]int64
	touched     []string
	executable  map[string]bool
	rendered    []string // "raw|events|ansi"
	renderErr   error
	agentPaneID string
	dumpByPane  map[string]string
	detached    []string // scripts passed to StartDetached
	detachedPID string
	detachedEnv []string
	viewer      *viewerCall
}

func newFake() *fakeRuntime {
	return &fakeRuntime{
		alive: map[string]bool{}, files: map[string]string{}, wrote: map[string]string{},
		sizes: map[string]int64{}, executable: map[string]bool{}, dumpByPane: map[string]string{},
	}
}

func (f *fakeRuntime) Sleep(time.Duration)          { f.sleeps++ }
func (f *fakeRuntime) Getpid() string               { return f.pid }
func (f *fakeRuntime) ProcessAlive(pid string) bool { return f.alive[pid] }
func (f *fakeRuntime) ReadFile(p string) (string, error) {
	if v, ok := f.files[p]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no such file: %s", p)
}
func (f *fakeRuntime) WriteFile(p, d string) error { f.wrote[p] = d; return nil }
func (f *fakeRuntime) Remove(p string)             { f.removed = append(f.removed, p) }
func (f *fakeRuntime) FileSize(p string) (int64, bool) {
	s, ok := f.sizes[p]
	return s, ok
}
func (f *fakeRuntime) Touch(p string) error     { f.touched = append(f.touched, p); return nil }
func (f *fakeRuntime) Executable(p string) bool { return f.executable[p] }
func (f *fakeRuntime) RenderScrollback(raw, events, ansi string) error {
	f.rendered = append(f.rendered, raw+"|"+events+"|"+ansi)
	return f.renderErr
}
func (f *fakeRuntime) AgentPaneID() string { return f.agentPaneID }
func (f *fakeRuntime) DumpScreen(paneID string) (string, error) {
	if v, ok := f.dumpByPane[paneID]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no dump for %s", paneID)
}
func (f *fakeRuntime) StartDetached(script string, extraEnv []string, statusPath string) (string, error) {
	f.detached = append(f.detached, script)
	f.detachedEnv = extraEnv
	return f.detachedPID, nil
}
func (f *fakeRuntime) RunViewer(lua, file string, extraEnv []string) error {
	f.viewer = &viewerCall{lua: lua, file: file, env: extraEnv}
	return nil
}

func scrollbackOpts() Options {
	return Options{Tag: "t", Agent: "claude", DataDir: "/dd", PairHome: "/h"}
}

func TestRunScrollbackRendersAndOpensViewer(t *testing.T) {
	rt := newFake()
	rt.pid = "100"
	rt.sizes["/dd/scrollback-t-claude.raw"] = 42
	ansi := ansiFixture()
	rt.files["/dd/scrollback-t-claude.ansi"] = strings.Join(ansi, "\n")
	rt.agentPaneID = "5"
	rt.dumpByPane["5"] = strings.Join(ansi[10:15], "\n") // user scrolled to line 11

	code := RunScrollback(scrollbackOpts(), rt, io.Discard)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if want := "/dd/scrollback-t-claude.raw|/dd/scrollback-t-claude.events.jsonl|/dd/scrollback-t-claude.ansi"; len(rt.rendered) != 1 || rt.rendered[0] != want {
		t.Fatalf("rendered = %v", rt.rendered)
	}
	if rt.wrote["/dd/scrollback-t-claude.viewport"] != "11\n" {
		t.Fatalf("viewport = %q, want 11", rt.wrote["/dd/scrollback-t-claude.viewport"])
	}
	if rt.viewer == nil || rt.viewer.lua != "/h/nvim/scrollback.lua" || rt.viewer.file != "/dd/scrollback-t-claude.ansi" {
		t.Fatalf("viewer = %+v", rt.viewer)
	}
	if !hasEnv(rt.viewer.env, "PAIR_NVIM_PID_FILE=/dd/nvim-pid-t-scrollback") {
		t.Fatalf("viewer env missing pid file: %v", rt.viewer.env)
	}
	// Lock written (our pid) then cleared on return.
	if rt.wrote["/dd/scrollback-t-claude.openlock"] != "100\n" {
		t.Fatalf("openlock = %q", rt.wrote["/dd/scrollback-t-claude.openlock"])
	}
	if len(rt.removed) != 1 || rt.removed[0] != "/dd/scrollback-t-claude.openlock" {
		t.Fatalf("lock not cleared: %v", rt.removed)
	}
}

func TestRunScrollbackReentrancyDefers(t *testing.T) {
	rt := newFake()
	rt.files["/dd/scrollback-t-claude.openlock"] = "77\n"
	rt.alive["77"] = true // a viewer is already up
	code := RunScrollback(scrollbackOpts(), rt, io.Discard)
	if code != 0 || len(rt.rendered) != 0 || rt.viewer != nil {
		t.Fatalf("should defer to live viewer: code=%d rendered=%v viewer=%v", code, rt.rendered, rt.viewer)
	}
}

func TestRunScrollbackNoScrollback(t *testing.T) {
	rt := newFake() // no size for raw ⇒ FileSize returns (0,false)
	code := RunScrollback(scrollbackOpts(), rt, io.Discard)
	if code != 0 || len(rt.rendered) != 0 || rt.viewer != nil {
		t.Fatalf("empty scrollback should no-op: code=%d rendered=%v", code, rt.rendered)
	}
	if rt.sleeps == 0 {
		t.Fatal("expected the UX message sleep")
	}
}

func TestRunScrollbackJumpThreadsToViewer(t *testing.T) {
	rt := newFake()
	rt.pid = "100"
	rt.sizes["/dd/scrollback-t-claude.raw"] = 42
	opts := scrollbackOpts()
	opts.Jump = "prev"
	RunScrollback(opts, rt, io.Discard)
	if rt.viewer == nil || !hasEnv(rt.viewer.env, "PAIR_SCROLLBACK_JUMP=prev") {
		t.Fatalf("jump not threaded: %+v", rt.viewer)
	}
}

func TestRunChangelogLaunchesDetachedDistillerAndViewer(t *testing.T) {
	rt := newFake()
	rt.pid = "100"
	rt.sizes["/dd/scrollback-t-claude.raw"] = 50
	rt.executable["/h/bin/pair"] = true
	rt.detachedPID = "999"
	opts := Options{Tag: "t", Agent: "claude", DataDir: "/dd", PairHome: "/h", SessionID: "sid1"}

	code := RunChangelog(opts, rt, io.Discard)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	base := "/dd/changelog-t-claude-sid1"
	if len(rt.touched) != 1 || rt.touched[0] != base+".md" {
		t.Fatalf("log not touched: %v", rt.touched)
	}
	if len(rt.detached) != 1 || rt.detached[0] != distillerInner {
		t.Fatalf("distiller not launched: %v", rt.detached)
	}
	if rt.wrote[base+".distill.lock"] != "999\n" {
		t.Fatalf("dlock = %q, want distiller pid", rt.wrote[base+".distill.lock"])
	}
	if !hasEnv(rt.detachedEnv, "PCL_LOG="+base+".md") || !hasEnv(rt.detachedEnv, "PCL_BIN=/h/bin/pair") {
		t.Fatalf("distiller env wrong: %v", rt.detachedEnv)
	}
	if rt.viewer == nil || rt.viewer.lua != "/h/nvim/changelog.lua" || rt.viewer.file != base+".md" {
		t.Fatalf("viewer = %+v", rt.viewer)
	}
}

func TestRunChangelogSkipsDistillerWhenRunning(t *testing.T) {
	rt := newFake()
	rt.pid = "100"
	rt.sizes["/dd/scrollback-t-claude.raw"] = 50
	rt.executable["/h/bin/pair"] = true
	rt.files["/dd/changelog-t-claude.distill.lock"] = "555\n"
	rt.alive["555"] = true // a distiller is already running
	code := RunChangelog(Options{Tag: "t", Agent: "claude", DataDir: "/dd", PairHome: "/h"}, rt, io.Discard)
	if code != 0 || len(rt.detached) != 0 {
		t.Fatalf("must not double-spawn the distiller: detached=%v", rt.detached)
	}
	if rt.viewer == nil { // viewer still opens to watch the running build
		t.Fatal("viewer should still open")
	}
}

func TestRunChangelogSessionKeyFallsBackToConfig(t *testing.T) {
	rt := newFake()
	rt.pid = "100"
	rt.files["/dd/config-t-claude.json"] = `{"agent":"claude","session_id":"cfgsid"}`
	// no raw size ⇒ distiller skipped; we only assert the resolved base path.
	RunChangelog(Options{Tag: "t", Agent: "claude", DataDir: "/dd", PairHome: "/h"}, rt, io.Discard)
	if rt.viewer == nil || rt.viewer.file != "/dd/changelog-t-claude-cfgsid.md" {
		t.Fatalf("config-keyed base wrong: %+v", rt.viewer)
	}
}

func hasEnv(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
