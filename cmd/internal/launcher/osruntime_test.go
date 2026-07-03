package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ResolveContinuationDoc / ScanContinuations do real glob+read IO the fake can't
// exercise. Critically, "newest doc wins" is `matches[len-1]` after sort — if it
// were `matches[0]` every fake-driven test still passes but the wrong doc seeds
// the draft, so pin it against real files in a non-git temp cwd (git rev-parse
// fails there → continuationDirPath falls back to cwd) (#99 M5b review, Important).
func TestOSRuntimeResolveContinuation(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)
	cdir := filepath.Join(dir, "workshop", "continuation")
	if err := os.MkdirAll(cdir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, agent, next string) {
		body := "---\nagent: " + agent + "\nissues: [#99]\n---\n## NEXT ACTION\n" + next + "\n"
		if err := os.WriteFile(filepath.Join(cdir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("20260101T000000-demo.md", "claude", "the old one")
	write("20260202T000000-demo.md", "codex", "the newest one") // newest by timestamp name
	write("20260101T000000-other.md", "claude", "other work")

	rt := NewOSRuntime(dir, "/pair")
	path, agent, ok := rt.ResolveContinuationDoc("demo")
	if !ok {
		t.Fatal("demo should resolve")
	}
	if filepath.Base(path) != "20260202T000000-demo.md" {
		t.Fatalf("newest-wins failed (got %s) — matches[0] would pick the 2026-01-01 doc", filepath.Base(path))
	}
	if agent != "codex" {
		t.Fatalf("agent = %q, want codex (from the newest doc)", agent)
	}
	if _, _, ok := rt.ResolveContinuationDoc("missing"); ok {
		t.Fatal("a missing slug must not resolve")
	}

	rows, gotDir := rt.ScanContinuations()
	if !strings.HasSuffix(gotDir, filepath.Join("workshop", "continuation")) {
		t.Fatalf("scan dir = %q", gotDir)
	}
	if len(rows) != 3 { // two demo docs + one other
		t.Fatalf("rows = %d (%+v), want 3", len(rows), rows)
	}
	var demoRows int
	for _, r := range rows {
		if r.Slug == "demo" {
			demoRows++
			if r.Issues != "[#99]" {
				t.Fatalf("issues = %q", r.Issues)
			}
		}
	}
	if demoRows != 2 {
		t.Fatalf("demo rows = %d, want 2", demoRows)
	}
}

// The OSRuntime lifecycle methods that do real filesystem IO (marker read-clear,
// scrollback park, cmux ownership, pidfile reaping) exercised against temp dirs —
// the process-level coverage the fake-Runtime loop tests can't give (#99 M3; the
// M2 review's lesson: don't ship OSRuntime IO untested). The exec-only seams
// (zellij attach/create/delete-session, the ps orphan sweep) are exercised by the
// M3 boundary smoke against a stub zellij (attach → cleanup → in-process re-create)
// — a one-time end-to-end verification recorded in the issue Log, not a committed
// unit test (the real zellij interaction has no in-test home).

func mkCacheDir(t *testing.T) (home, cacheDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	cacheDir = filepath.Join(home, ".cache", "pair")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return home, cacheDir
}

func TestOSRuntimeQuitMarker(t *testing.T) {
	_, cacheDir := mkCacheDir(t)
	rt := NewOSRuntime(t.TempDir(), "/pair")

	if rt.TakeQuitMarker("pair-x") {
		t.Fatal("absent quit marker should read false")
	}
	marker := filepath.Join(cacheDir, "quit-pair-x")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !rt.TakeQuitMarker("pair-x") {
		t.Fatal("present quit marker should read true")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("TakeQuitMarker must clear the marker")
	}
	if rt.TakeQuitMarker("pair-x") {
		t.Fatal("a cleared marker should read false")
	}
}

func TestOSRuntimeRestartMarker(t *testing.T) {
	_, cacheDir := mkCacheDir(t)
	rt := NewOSRuntime(t.TempDir(), "/pair")
	marker := filepath.Join(cacheDir, "restart-pair-x")
	if err := os.WriteFile(marker, []byte("tag=x\nagent=codex\nnew_session=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Peek must NOT clear (park-nudge skip reads it before Take does).
	if !rt.RestartMarkerPresent("pair-x") {
		t.Fatal("RestartMarkerPresent should see the marker")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("RestartMarkerPresent must not clear the marker")
	}

	m, ok := rt.TakeRestartMarker("pair-x")
	if !ok || m.Tag != "x" || m.Agent != "codex" || !m.NewSession {
		t.Fatalf("TakeRestartMarker = %+v ok=%v", m, ok)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("TakeRestartMarker must clear the marker")
	}
	if _, ok := rt.TakeRestartMarker("pair-x"); ok {
		t.Fatal("a cleared restart marker should read false")
	}
}

func TestOSRuntimeParkScrollbackMove(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-work-claude.raw")
	if err := os.WriteFile(raw, []byte("captured"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "scrollback-work-claude.events.jsonl"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, ok := rt.ParkScrollback("work", "claude", true)
	if !ok {
		t.Fatal("park should succeed with a non-empty raw")
	}
	if _, err := os.Stat(raw); !os.IsNotExist(err) {
		t.Fatal("move mode must remove the original raw")
	}
	for _, suffix := range []string{".raw", ".events.jsonl"} {
		if _, err := os.Stat(base + suffix); err != nil {
			t.Fatalf("parked %s missing: %v", suffix, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "parked-work")); err != nil {
		t.Fatal("ParkScrollback must touch the parked-<tag> marker")
	}
}

func TestOSRuntimeParkScrollbackCopyKeepsOriginal(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-c-claude.raw")
	if err := os.WriteFile(raw, []byte("live bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, ok := rt.ParkScrollback("c", "claude", false) // copy (compaction path)
	if !ok {
		t.Fatal("copy park should succeed")
	}
	if _, err := os.Stat(raw); err != nil {
		t.Fatal("copy mode must leave the original raw in place")
	}
	if _, err := os.Stat(base + ".raw"); err != nil {
		t.Fatal("copy park should still write the parked raw")
	}
}

func TestOSRuntimeParkScrollbackEmpty(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	if err := os.WriteFile(filepath.Join(dataDir, "scrollback-work-claude.raw"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := rt.ParkScrollback("work", "claude", true); ok {
		t.Fatal("an empty raw should not park")
	}
}

func TestOSRuntimeCmuxOwnership(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	t.Setenv("CMUX_WORKSPACE_ID", "ws1")
	owner := filepath.Join(dataDir, "cmux-owner-ws1")
	if err := os.WriteFile(owner, []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !rt.PairOwnsCmuxWorkspace("work") {
		t.Fatal("owner-file == tag should own")
	}
	if rt.PairOwnsCmuxWorkspace("other") {
		t.Fatal("owner-file mismatch should not own")
	}
	rt.ClearCmuxOwner()
	if _, err := os.Stat(owner); !os.IsNotExist(err) {
		t.Fatal("ClearCmuxOwner must remove the owner file")
	}

	t.Setenv("CMUX_WORKSPACE_ID", "")
	if rt.PairOwnsCmuxWorkspace("work") {
		t.Fatal("outside cmux (no CMUX_WORKSPACE_ID) nothing is owned")
	}
}

func TestOSRuntimeReapAndPollerRemovePidfiles(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	// A syntactically-valid but non-existent pid: kill(2) returns ESRCH, so this
	// can never signal a real process; the assertion is on the pidfile removal.
	const deadPid = "2147483646"
	for _, name := range []string{"nvim-pid-work-draft", "nvim-pid-work-scrollback", "title-pid-work"} {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(deadPid+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rt.ReapNvim("work")
	for _, name := range []string{"nvim-pid-work-draft", "nvim-pid-work-scrollback"} {
		if _, err := os.Stat(filepath.Join(dataDir, name)); !os.IsNotExist(err) {
			t.Fatalf("ReapNvim should clear the %s pidfile", name)
		}
	}
	rt.KillTitlePoller("work")
	if _, err := os.Stat(filepath.Join(dataDir, "title-pid-work")); !os.IsNotExist(err) {
		t.Fatal("KillTitlePoller should clear the title pidfile")
	}
}

// The sidecar spawn argv must target the Go binaries directly — #94 M2 retired
// the pair-title.sh/pair-session-watch.sh shims (no longer in the bundle), and
// spawnDetached swallows a start error, so a regression back to a ".sh" target
// would fail silently at runtime. Pin the base names (and the title poller's
// "<…>/pair-title <tag> <agent>" shape the single-instance guard matches).
func TestSidecarSpawnArgvTargetsGoBinaries(t *testing.T) {
	tp := titlePollerArgv("/pair", "work", "claude")
	if got := tp[0]; got != "/pair/bin/pair-title" {
		t.Fatalf("title poller argv[0] = %q, want /pair/bin/pair-title (no .sh)", got)
	}
	if len(tp) != 3 || tp[1] != "work" || tp[2] != "claude" {
		t.Fatalf("title poller argv = %v, want [.../pair-title work claude]", tp)
	}

	sw := sessionWatcherArgv("/pair", "codex", "work", "/cwd", []string{"--no-alt-screen"})
	if got := sw[0]; got != "/pair/bin/pair-session-watch" {
		t.Fatalf("session watcher argv[0] = %q, want /pair/bin/pair-session-watch (no .sh)", got)
	}
	if len(sw) != 5 || sw[1] != "codex" || sw[2] != "work" || sw[3] != "/cwd" || sw[4] != "--no-alt-screen" {
		t.Fatalf("session watcher argv = %v, want [.../pair-session-watch codex work /cwd --no-alt-screen]", sw)
	}

	// Guard the invariant explicitly: no sidecar target ends in ".sh".
	for _, argv := range [][]string{tp, sw} {
		if strings.HasSuffix(argv[0], ".sh") {
			t.Fatalf("sidecar spawn target must not be a .sh shim: %q", argv[0])
		}
	}
}
