package procutil

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// Keep the exited child unreaped until cleanup so its table row remains while
// its command storage has gone. Pipe EOF proves the child has closed its files;
// nativeCommand may lag briefly as exit completes.
func exitedUnreapedChild(t *testing.T) int {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestNativeProcessHelper$")
	cmd.Env = append(os.Environ(), "PROCUTIL_HELPER=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	if _, err := io.Copy(io.Discard, stdout); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if _, err := nativeCommand(cmd.Process.Pid); err != nil {
			t.Logf("exited unreaped PID %d: command error %v", cmd.Process.Pid, err)
			return cmd.Process.Pid
		}
	}
	t.Fatal("exited child still has command evidence")
	return 0
}

func TestSnapshotExcludesExitedUnreapedChild(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native inspection unsupported")
	}
	pid := exitedUnreapedChild(t)
	rows, err := Table()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range rows {
		if p.PID == pid {
			found = true
			t.Logf("table retains exited child: %+v", p)
		}
	}
	if !found {
		t.Fatal("unreaped child missing from native table")
	}
	observations, err := Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range observations {
		if p.PID == pid {
			t.Fatalf("snapshot presents terminated process as possible live process: %+v", p)
		}
	}
}

func TestSnapshotSkipsOnlyConfirmedZombieRows(t *testing.T) {
	w := &snapshotWorld{
		rows:       []Process{{PID: 1, Identity: "dead", Name: "zellij", Zombie: true}, {PID: 2, Identity: "live", Name: "zellij"}},
		commands:   map[int]CommandInfo{},
		identities: map[int]string{1: "dead", 2: "live"},
	}
	rows, err := snapshot(context.Background(), w)
	if err != nil || len(rows) != 1 || rows[0].PID != 2 || rows[0].Err == nil {
		t.Fatalf("lost live uncertainty or retained zombie: %+v %v", rows, err)
	}
	if len(w.calls) != 1 || w.calls[0] != 2 {
		t.Fatalf("read argv for confirmed zombie: %v", w.calls)
	}
}

func TestLinuxStatTerminalStates(t *testing.T) {
	for _, state := range []string{"Z", "X", "x", "R", "S", "T", "D"} {
		raw := []byte("41 (zellij) " + state + " 3 0 0 0 0 0 0 0 0 0 0 0 0 0 0 7 0 0 123")
		p, err := parseLinuxStat(41, raw)
		if err != nil {
			t.Fatal(err)
		}
		want := state == "Z" || state == "X" || state == "x"
		if p.Zombie != want {
			t.Errorf("state %s zombie=%v, want %v", state, p.Zombie, want)
		}
	}
}
