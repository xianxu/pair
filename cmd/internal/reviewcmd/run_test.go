package reviewcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type spawnCall struct{ cwd, lua, absFile, nvimPid string }

type fakeRuntime struct {
	files    map[string]string
	wrote    map[string]string
	removed  []string
	sizes    map[string]int64
	alive    map[string]bool
	killed   []string
	gitFn    func(dir string, args []string) (string, error)
	gitCalls [][]string
	classify string
	classErr error
	spawn    *spawnCall
	codexSID string
}

func newFake() *fakeRuntime {
	return &fakeRuntime{files: map[string]string{}, wrote: map[string]string{}, sizes: map[string]int64{}, alive: map[string]bool{}}
}

func (f *fakeRuntime) ReadFile(p string) (string, error) {
	if v, ok := f.files[p]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no file: %s", p)
}
func (f *fakeRuntime) WriteFile(p, d string) error { f.wrote[p] = d; return nil }
func (f *fakeRuntime) Remove(p string)             { f.removed = append(f.removed, p) }
func (f *fakeRuntime) FileSize(p string) (int64, bool) {
	s, ok := f.sizes[p]
	return s, ok
}
func (f *fakeRuntime) ProcessAlive(pid string) bool { return f.alive[pid] }
func (f *fakeRuntime) Kill(pid string)              { f.killed = append(f.killed, pid) }
func (f *fakeRuntime) AbsFile(file string) string   { return file }
func (f *fakeRuntime) LogicalDir(file string) string {
	if i := strings.LastIndexByte(file, '/'); i >= 0 {
		return file[:i]
	}
	return "."
}
func (f *fakeRuntime) PhysicalDir(file string) string { return f.LogicalDir(file) }
func (f *fakeRuntime) Git(dir string, args ...string) (string, error) {
	f.gitCalls = append(f.gitCalls, append([]string{dir}, args...))
	if f.gitFn != nil {
		return f.gitFn(dir, args)
	}
	return "", nil
}
func (f *fakeRuntime) Classify(lua string, _ ReadinessFacts) (string, error) {
	return f.classify, f.classErr
}
func (f *fakeRuntime) SpawnReviewPane(cwd, lua, absFile, nvimPid string) error {
	f.spawn = &spawnCall{cwd, lua, absFile, nvimPid}
	return nil
}
func (f *fakeRuntime) ResolveCodexSessionID(dataDir, tag string) string { return f.codexSID }

func targetOf(t *testing.T, rt *fakeRuntime, tag string) targetDoc {
	t.Helper()
	var d targetDoc
	if err := json.Unmarshal([]byte(rt.wrote["/dd/review-target-"+tag+".json"]), &d); err != nil {
		t.Fatalf("target json: %v (%q)", err, rt.wrote["/dd/review-target-"+tag+".json"])
	}
	return d
}

func TestRunTargetSessionPriority(t *testing.T) {
	// env wins
	rt := newFake()
	RunTarget(TargetOptions{File: "/r/doc.md", Status: "ready", Tag: "t", Agent: "codex", DataDir: "/dd", SessionID: "envsid"}, rt, &bytes.Buffer{}, &bytes.Buffer{})
	if d := targetOf(t, rt, "t"); d.Session != "envsid" || d.File != "/r/doc.md" || d.Status != "ready" {
		t.Fatalf("env: %+v", d)
	}
	// config fallback
	rt = newFake()
	rt.files["/dd/config-t-codex.json"] = `{"session_id":"cfgsid"}`
	RunTarget(TargetOptions{File: "/r/doc.md", Status: "proposed", Tag: "t", Agent: "codex", DataDir: "/dd"}, rt, &bytes.Buffer{}, &bytes.Buffer{})
	if d := targetOf(t, rt, "t"); d.Session != "cfgsid" {
		t.Fatalf("config: %+v", d)
	}
	// codex lsof-walk fallback
	rt = newFake()
	rt.codexSID = "walksid"
	RunTarget(TargetOptions{File: "/r/doc.md", Status: "ready", Tag: "t", Agent: "codex", DataDir: "/dd"}, rt, &bytes.Buffer{}, &bytes.Buffer{})
	if d := targetOf(t, rt, "t"); d.Session != "walksid" {
		t.Fatalf("codex: %+v", d)
	}
}

func TestRunTargetInvalidStatus(t *testing.T) {
	rt := newFake()
	if code := RunTarget(TargetOptions{File: "/r/d.md", Status: "bogus", DataDir: "/dd"}, rt, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

// gitScript maps a git subcommand (args[0]) to (out, err) — enough to drive the
// readiness fact-gathering + prepare effects.
func gitScript(m map[string]struct {
	out string
	err error
}) func(string, []string) (string, error) {
	return func(_ string, args []string) (string, error) {
		if len(args) == 0 {
			return "", nil
		}
		r, ok := m[args[0]]
		if !ok {
			return "", nil
		}
		return r.out, r.err
	}
}

func TestRunReadinessJSON(t *testing.T) {
	rt := newFake()
	rt.classify = "resume"
	rt.gitFn = gitScript(map[string]struct {
		out string
		err error
	}{
		"rev-parse": {out: "/repo\n"}, // both --is-inside-work-tree and --show-toplevel
		"ls-files":  {out: ""},        // tracked (exit 0)
		"branch":    {out: "review/doc\n"},
		"status":    {out: ""}, // clean
		"log":       {out: "doc.md\n"},
	})
	var stdout bytes.Buffer
	code := RunReadiness(ReadinessOptions{File: "/repo/doc.md", PairHome: "/h"}, rt, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	var d readinessDoc
	if err := json.Unmarshal(stdout.Bytes(), &d); err != nil {
		t.Fatalf("json: %v (%s)", err, stdout.String())
	}
	if d.Case != "resume" || !d.IsGit || !d.IsTracked || d.Branch != "review/doc" || !d.OnReviewBranch || !d.IsClean {
		t.Fatalf("doc = %+v", d)
	}
}

func TestRunReadinessPrepareNew(t *testing.T) {
	rt := newFake()
	rt.classify = "new"
	rt.gitFn = func(dir string, args []string) (string, error) {
		switch args[0] {
		case "rev-parse":
			return "/repo\n", nil
		case "ls-files":
			return "", nil // tracked
		case "branch":
			return "main\n", nil
		case "status":
			return "", nil // clean
		case "show-ref":
			return "", fmt.Errorf("no such ref") // branch does not exist yet
		}
		return "", nil
	}
	var stdout bytes.Buffer
	code := RunReadiness(ReadinessOptions{File: "/repo/doc.md", Prepare: true, PairHome: "/h", Tag: "t", DataDir: "/dd", SessionID: "sid"}, rt, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	// created the review branch
	if !gitCalled(rt, "checkout", "-q", "-b", "review/doc") {
		t.Fatalf("expected checkout -b review/doc; calls=%v", rt.gitCalls)
	}
	// marked the target ready
	if d := targetOf(t, rt, "t"); d.Status != "ready" || d.File != "/repo/doc.md" || d.Session != "sid" {
		t.Fatalf("target: %+v", d)
	}
	// ack instruction
	out := stdout.String()
	for _, frag := range []string{"review prepared:", "review/doc", "Do not load xx-fix for this ack", "load the full xx-fix skill", `Reply "ready".`} {
		if !strings.Contains(out, frag) {
			t.Fatalf("ack missing %q:\n%s", frag, out)
		}
	}
}

func TestRunReadinessPrepareStopAndInteract(t *testing.T) {
	for _, c := range []string{"stop", "interact"} {
		rt := newFake()
		rt.classify = c
		rt.gitFn = gitScript(map[string]struct {
			out string
			err error
		}{"rev-parse": {out: "/repo\n"}, "branch": {out: "main\n"}})
		var stdout bytes.Buffer
		code := RunReadiness(ReadinessOptions{File: "/repo/doc.md", Prepare: true, PairHome: "/h", Tag: "t", DataDir: "/dd"}, rt, &stdout, &bytes.Buffer{})
		if code != 1 {
			t.Fatalf("%s: code = %d, want 1", c, code)
		}
		if !strings.Contains(stdout.String(), "not prepared") {
			t.Fatalf("%s: want 'not prepared', got %q", c, stdout.String())
		}
		if _, wrote := rt.wrote["/dd/review-target-t.json"]; wrote {
			t.Fatalf("%s: must not mark ready", c)
		}
	}
}

func TestRunOpenReplacesLivePaneAndSpawns(t *testing.T) {
	rt := newFake()
	rt.sizes["/repo/doc.md"] = 10
	rt.files["/dd/review-t.open"] = "777\n"
	rt.alive["777"] = true
	code := RunOpen(OpenOptions{File: "/repo/doc.md", Tag: "t", DataDir: "/dd", PairHome: "/h"}, rt, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if len(rt.killed) != 1 || rt.killed[0] != "777" {
		t.Fatalf("expected kill of the live review pane, killed=%v", rt.killed)
	}
	if rt.spawn == nil || rt.spawn.lua != "/h/nvim/review.lua" || rt.spawn.absFile != "/repo/doc.md" {
		t.Fatalf("spawn = %+v", rt.spawn)
	}
	if rt.spawn.nvimPid != "/dd/nvim-pid-t-review" {
		t.Fatalf("nvim pid path = %q", rt.spawn.nvimPid)
	}
}

func TestRunOpenMissingFile(t *testing.T) {
	rt := newFake() // no size for the file ⇒ not found
	if code := RunOpen(OpenOptions{File: "/nope.md", Tag: "t", DataDir: "/dd", PairHome: "/h"}, rt, &bytes.Buffer{}); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if rt.spawn != nil {
		t.Fatal("must not spawn for a missing file")
	}
}

func gitCalled(rt *fakeRuntime, want ...string) bool {
	for _, call := range rt.gitCalls {
		if len(call) < 1+len(want) {
			continue
		}
		if strings.Join(call[1:1+len(want)], " ") == strings.Join(want, " ") {
			return true
		}
	}
	return false
}
