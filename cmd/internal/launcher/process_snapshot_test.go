package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestProcessAncestryUsesOneTableAndTerminatesCycles(t *testing.T) {
	rows := []procutil.Process{{PID: 10, PPID: 20, Name: "pair"}, {PID: 20, PPID: 30, Name: "shell"}, {PID: 30, PPID: 1, Name: "zellij"}}
	if !hasProcessAncestor(rows, 10, "zellij") {
		t.Fatal("missed zellij ancestor")
	}
	rows[2].Name = "not-zellij"
	rows[2].PPID = 10
	if hasProcessAncestor(rows, 10, "zellij") {
		t.Fatal("cycle fabricated zellij")
	}
	if hasProcessAncestor(rows, 99, "zellij") {
		t.Fatal("missing row fabricated ancestor")
	}
}

func TestServerProcessesKeepsSnapshotIdentityAndArgvAuthority(t *testing.T) {
	rows := []procutil.Observation{{Process: procutil.Process{PID: 10, Name: "test-helper", Identity: "generation"}, CommandInfo: procutil.CommandInfo{Executable: "/tmp/test-helper", Args: []string{"/opt/bin/zellij", "--server", "/tmp/session"}}}}
	got, err := serverProcesses(rows)
	if err != nil || len(got) != 1 || got[0].Identity != "generation" || got[0].Socket != "/tmp/session" {
		t.Fatalf("servers=%+v err=%v", got, err)
	}
	rows[0].Args = append(rows[0].Args, "extra")
	got, err = serverProcesses(rows)
	if err != nil || len(got) != 0 {
		t.Fatalf("non-exact server accepted: %+v %v", got, err)
	}
}

func TestServerProcessesRefusesUnreadablePotentialServer(t *testing.T) {
	rows := []procutil.Observation{{Process: procutil.Process{PID: 10, Name: "zellij", Identity: "generation"}, Err: errors.New("denied")}}
	if _, err := serverProcesses(rows); err == nil {
		t.Fatal("unreadable zellij became absence")
	}
	rows[0].Name = "unrelated"
	if got, err := serverProcesses(rows); err != nil || len(got) != 0 {
		t.Fatalf("unrelated unreadable row=%v %v", got, err)
	}
	rows[0].Name = "zellij"
	rows[0].Err = nil
	rows[0].CommandInfo = procutil.CommandInfo{Args: []string{"zellij", "--server", "/tmp/session"}}
	rows[0].Identity = ""
	if _, err := serverProcesses(rows); err == nil {
		t.Fatal("unidentified server accepted")
	}
}

func TestNativeServerObservationHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (OSServerStates{}).ServerStates(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("server states cancellation: %v", err)
	}
	if _, err := (newOSSessionQuiescenceOps()).SessionServers(ctx, "unused"); !errors.Is(err, context.Canceled) {
		t.Fatalf("session servers cancellation: %v", err)
	}
}

// The process is a test helper with zellij argv, matching the existing live
// quiescence sentinel. Its executable/kernel name must not hide valid argv.
func TestNativeSnapshotFindsArgvServer(t *testing.T) {
	dir := t.TempDir()
	session := "pair547-controlled-server"
	child := &exec.Cmd{Path: os.Args[0], Args: []string{filepath.Join(dir, "zellij"), "--server", filepath.Join(dir, session)}, Env: append(os.Environ(), "PAIR_ZELLIJ_KILL_SENTINEL=1")}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	ops := newOSSessionQuiescenceOps()
	servers, err := ops.SessionServers(t.Context(), session)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, server := range servers {
		if server.PID != child.Process.Pid {
			continue
		}
		if server.Identity == "" {
			t.Fatal("missing native identity")
		}
		found = true
		killed := false
		ops.killProcess = func(pid int) error {
			if pid != child.Process.Pid {
				t.Fatalf("wrong signal target %d", pid)
			}
			killed = true
			return nil
		}
		if err := ops.KillServer(server); err != nil || !killed {
			t.Fatalf("native command authorization: killed=%v err=%v", killed, err)
		}
	}
	if !found {
		t.Fatal("native inventory missed argv-spoofed helper")
	}
	states, err := (OSServerStates{}).ServerStates(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state := states[session]; state.Server.PID != child.Process.Pid || state.Server.Identity == "" || state.Verdict != ServerOrphaned {
		t.Fatalf("native server state: %+v", state)
	}
}

func TestServerProcessesSkipsGonePossibleServersAndZombies(t *testing.T) {
	for _, name := range []string{"zellij", ""} {
		for _, gone := range []error{syscall.EINVAL, syscall.ESRCH, syscall.ENOENT} {
			row := procutil.Observation{Process: procutil.Process{PID: 10, Name: name, Identity: "generation"}, Err: fmt.Errorf("process command: %w", gone)}
			if got, err := serverProcesses([]procutil.Observation{row}); err != nil || len(got) != 0 {
				t.Fatalf("gone %q %v: servers=%v err=%v", name, gone, got, err)
			}
		}
	}
	row := procutil.Observation{Process: procutil.Process{PID: 10, Name: "zellij", Identity: "generation", Zombie: true}, CommandInfo: procutil.CommandInfo{Args: []string{"zellij", "--server", "/tmp/session"}}}
	if got, err := serverProcesses([]procutil.Observation{row}); err != nil || len(got) != 0 {
		t.Fatalf("zombie server=%v err=%v", got, err)
	}
}
func TestServerProcessesPermissionAndUnknownStayFailClosed(t *testing.T) {
	for _, name := range []string{"zellij", ""} {
		for _, failure := range []error{syscall.EPERM, syscall.EACCES, errors.New("unrecognized process read")} {
			row := procutil.Observation{Process: procutil.Process{PID: 10, Name: name, Identity: "generation"}, Err: fmt.Errorf("process command: %w", failure)}
			if _, err := serverProcesses([]procutil.Observation{row}); err == nil {
				t.Fatalf("%q %v became absence", name, failure)
			}
		}
	}
}
