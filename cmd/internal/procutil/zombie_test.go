package procutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// waitForUnreapedChild keeps command failures as diagnostic evidence. They do
// not establish that the kernel has published a terminal process state.
func waitForUnreapedChild(pid int, table func() ([]Process, error), command func(int) (CommandInfo, error), timeout time.Duration) error {
	var commandErr error
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		rows, err := table()
		if err != nil {
			return fmt.Errorf("read PID %d terminal metadata: %w", pid, err)
		}
		_, commandErr = command(pid)
		for _, row := range rows {
			if row.PID == pid && row.Zombie {
				return nil
			}
		}
	}
	return fmt.Errorf("PID %d did not become terminal (last command error: %v)", pid, commandErr)
}

func TestUnreapedChildWaitsForTerminalMetadata(t *testing.T) {
	const pid = 41
	reads := 0
	table := func() ([]Process, error) {
		reads++
		return []Process{{PID: pid, Zombie: reads >= 2}}, nil
	}
	command := func(int) (CommandInfo, error) {
		return CommandInfo{}, errors.New("command storage already unavailable")
	}
	if err := waitForUnreapedChild(pid, table, command, time.Second); err != nil {
		t.Fatal(err)
	}
	if reads != 2 {
		t.Fatalf("fixture returned before terminal metadata: table reads=%d, want 2", reads)
	}
}

func TestUnreapedChildWaitHasBoundedDeadline(t *testing.T) {
	const pid = 41
	table := func() ([]Process, error) {
		return []Process{{PID: pid}}, nil
	}
	command := func(int) (CommandInfo, error) {
		return CommandInfo{}, errors.New("command storage already unavailable")
	}
	if err := waitForUnreapedChild(pid, table, command, 5*time.Millisecond); err == nil {
		t.Fatal("fixture accepted a nonterminal row after its deadline")
	}
}

// Keep the exited child unreaped until cleanup so its table row remains while
// its command storage has gone. Pipe EOF and unavailable command storage can
// precede the kernel terminal state, so readiness requires a zombie table row.
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
	if err := waitForUnreapedChild(cmd.Process.Pid, Table, nativeCommand, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
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
