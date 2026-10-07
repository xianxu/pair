package couchcmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/crashreport"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func testConsole() *couchtty.Console {
	return couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
}

// A panic in the console owner must reach its crash file through the real
// wiring. The first version ended capture from a defer in
// runTypedOperationWithConsole, which Go runs while the panic unwinds -- before
// the runtime prints -- so the file was deleted empty (#397 BR-1). The child
// therefore panics from the console's finish callback, inside the production
// function, with that function's own defers pending.
func TestConsoleOwnerPanicLeavesItsCrashFile(t *testing.T) {
	if store := os.Getenv("COUCH_CRASH_TEST_STORE"); store != "" {
		rt := newRT(t, "/repo")
		// The parent owns the store: the testing runner deletes t.TempDir in
		// its panic cleanup, before the runtime writes the crash.
		ns, err := couchcore.ResolveCouchNamespace(store, "/unused")
		if err != nil {
			t.Fatal(err)
		}
		rt.namespace, rt.dir = ns, ns.Dir()
		master, slave, err := pty.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer master.Close()
		defer slave.Close()
		if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
			t.Fatal(err)
		}
		finish := func(*couchtty.Console, *couchcore.Couch, couchcore.StartResult, io.Writer) int {
			panic("couchcmd crash test")
		}
		op, _ := Resolve("start")
		var stdout, stderr bytes.Buffer
		runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, couchcore.Layout2, slave, slave, slave, &stdout, &stderr, rt, finish)
		t.Fatal("finish returned instead of panicking")
	}
	store, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestConsoleOwnerPanicLeavesItsCrashFile$")
	cmd.Env = append(os.Environ(), "COUCH_CRASH_TEST_STORE="+store)
	if err := cmd.Run(); err == nil {
		t.Fatal("child did not crash")
	}
	dir := crashreport.Dir(store)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("crash dir = %v, %v; want the crashing run's file", entries, err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if !strings.Contains(string(body), "panic: couchcmd crash test") {
		t.Fatalf("crash file lacks the panic:\n%s", body)
	}
}

// The console owner captures into its own store's crash dir and reports what
// the previous incarnation left there (#397).
func TestInstallCrashReportUsesTheStoreAndReportsThePrevious(t *testing.T) {
	store := t.TempDir()
	dir := crashreport.Dir(store)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// pid 2147483646 is not a live process, so the file is a dead run's.
	previous := filepath.Join(dir, "20261006T135600Z-2147483646.log")
	if err := os.WriteFile(previous, []byte("panic: boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	installCrashReport(testConsole(), store)
	if _, err := os.Stat(filepath.Join(dir, "20261006T135600Z-2147483646.crash")); err != nil {
		t.Fatalf("previous crash not reported from the store's crash dir: %v", err)
	}
	if err := crashreport.Finish(); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("crash dir after a normal finish = %v, want only the reported crash", entries)
	}
}
