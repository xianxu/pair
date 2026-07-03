package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

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
