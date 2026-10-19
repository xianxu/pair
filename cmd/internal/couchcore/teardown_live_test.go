package couchcore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/lifetimelease"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Live conformance (pair#695 M2): a real tree on a real pty, torn down by the
// production seams -- the kernel process table, lsof for the lease's holders,
// and real signals. The lease holder and everything on the agent's terminal
// die; the setsid'd runner the agent started is named in LeftRunning and left
// alive. Unsandboxed only (pty children, lsof).
func TestTeardownProvesARealPtyTree(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no process table on this platform")
	}
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl unavailable")
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof unavailable")
	}
	env := newTestEnv(t, "/repo")
	c := env.Couch
	c.Proc, c.ProcessTable, c.LeaseHolders = OSProcOps{}, procutil.Table, lifetimelease.Holders
	thread := validThreadRecord(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	thread.StartingPath, thread.WorkingPath = root, root
	thread.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	entry, err := c.lifetimeLeaseEnv(thread)
	if err != nil {
		t.Fatal(err)
	}
	lease := entry[len(lifetimelease.Env)+1:]

	started := func(cmd *exec.Cmd) {
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		go func() { _ = cmd.Wait() }() // reap, so a killed child is not a live zombie
	}
	// The wrapper's stand-in: a process holding the lease shared.
	holder := exec.Command("perl", "-MFcntl=:flock", "-e", `open(F, "<", $ARGV[0]) or die; flock(F, LOCK_SH) or die; sleep 60`, lease)
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	started(holder)
	// The agent on its pty: a foreground tool, and a runner that setsid's off
	// the terminal and records its pid.
	runnerPID := filepath.Join(t.TempDir(), "runner.pid")
	agent := exec.Command("/bin/sh", "-c", `sleep 60 & perl -MPOSIX -e 'my $p = fork; if ($p == 0) { POSIX::setsid(); sleep 60; exit 0 } open(F, ">", $ARGV[0]); print F $p; close F; sleep 60' "$0" & wait`, runnerPID)
	master, err := pty.Start(agent)
	if err != nil {
		t.Skipf("pty unavailable here (sandbox?): %v", err)
	}
	defer master.Close()
	started(agent)

	var agentRow procutil.Process
	var runner int
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		table, err := procutil.Table()
		if err != nil {
			t.Fatal(err)
		}
		agentRow, runner = procutil.Process{}, 0
		onTTY := 0
		for _, p := range table {
			if p.PID == agent.Process.Pid {
				agentRow = p
			}
		}
		for _, p := range table {
			if agentRow.TTY != "" && p.TTY == agentRow.TTY {
				onTTY++
			}
		}
		if pid := readPIDFile(runnerPID); pid > 0 && agentRow.TTY != "" && onTTY >= 3 {
			runner = pid
			break
		}
	}
	if agentRow.TTY == "" || runner == 0 {
		t.Fatalf("tree did not come up: agent %+v runner %d", agentRow, runner)
	}
	t.Cleanup(func() { _ = syscall.Kill(runner, syscall.SIGKILL) })
	if held, err := lifetimelease.Free(lease); err != nil || held {
		t.Fatalf("the lease is not held before teardown: free=%v err=%v", held, err)
	}
	env.Artifacts.SetAgentTerminal(thread.Address, AgentTerminal{Process: ProcessIdentity{PID: agentRow.PID, Identity: agentRow.Identity}, TTY: agentRow.TTY})

	live := OSProcOps{}
	proof, err := c.Teardown(context.Background(), thread, "live")
	if err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if live.Exists(holder.Process.Pid) != Dead || live.Exists(agent.Process.Pid) != Dead {
		t.Fatalf("the lease holder or the agent survived a proven teardown")
	}
	table, err := procutil.Table()
	if err != nil {
		t.Fatal(err)
	}
	if left := procutil.TTYProcesses(table, agentRow.TTY); len(left) != 0 {
		t.Fatalf("processes still on the agent's terminal: %+v", left)
	}
	if len(proof.LeftRunning) != 1 || proof.LeftRunning[0].Process.PID != runner {
		t.Fatalf("LeftRunning = %+v, want the setsid'd runner %d", proof.LeftRunning, runner)
	}
	if live.Exists(runner) != Live {
		t.Fatal("teardown killed the runner that had left the terminal")
	}
}

func readPIDFile(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid := 0
	for _, b := range raw {
		if b < '0' || b > '9' {
			break
		}
		pid = pid*10 + int(b-'0')
	}
	return pid
}
