package procutil

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestLinuxStatCarriesTheControllingTTY(t *testing.T) {
	// Fields after the command: state ppid pgrp session tty_nr ...
	line := func(tty string) []byte {
		fields := strings.Fields("S 3 9 9 " + tty + " 0 0 0 0 0 0 0 0 0 0 0 7 0 0 123")
		return []byte("41 (sh) " + strings.Join(fields, " "))
	}
	on, err := parseLinuxStat(41, line("34817"))
	if err != nil || on.TTY != "34817" {
		t.Fatalf("with a terminal: %+v %v", on, err)
	}
	off, err := parseLinuxStat(41, line("0"))
	if err != nil || off.TTY != "" {
		t.Fatalf("without a terminal: %+v %v", off, err)
	}
	if _, err := parseLinuxStat(41, line("x")); err == nil {
		t.Fatal("accepted a non-numeric tty_nr")
	}
}

func TestTTYProcessesFiltersOneTableRead(t *testing.T) {
	table := []Process{{PID: 1, TTY: "7"}, {PID: 2}, {PID: 3, TTY: "7"}, {PID: 4, TTY: "8"}}
	got := TTYProcesses(table, "7")
	if len(got) != 2 || got[0].PID != 1 || got[1].PID != 3 {
		t.Fatalf("TTYProcesses = %+v", got)
	}
	if TTYProcesses(table, "") != nil {
		t.Fatal("an empty tty must match nothing")
	}
}

// Live: a child started on a pty is listed with that pty as its terminal, and
// its own child too, from one table read. Unsandboxed only (pty children).
func TestTableReportsAPtyChildsTerminal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no process table on this platform")
	}
	cmd := exec.Command("/bin/sh", "-c", "sleep 30 & wait")
	master, err := pty.Start(cmd)
	if err != nil {
		t.Skipf("pty unavailable here (sandbox?): %v", err)
	}
	defer master.Close()
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()
	var shell, sleeper Process
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		table, err := Table()
		if err != nil {
			t.Fatal(err)
		}
		shell, sleeper = Process{}, Process{}
		for _, p := range table {
			if p.PID == cmd.Process.Pid {
				shell = p
			}
		}
		for _, p := range table {
			if p.PPID == cmd.Process.Pid && p.PID != 0 {
				sleeper = p
			}
		}
		if shell.TTY != "" && sleeper.PID != 0 {
			break
		}
	}
	if shell.TTY == "" {
		t.Fatalf("pty child %d has no terminal in the table", cmd.Process.Pid)
	}
	if sleeper.TTY != shell.TTY {
		t.Fatalf("grandchild terminal %q, child %q", sleeper.TTY, shell.TTY)
	}
	table, _ := Table()
	pids := map[int]bool{}
	for _, p := range TTYProcesses(table, shell.TTY) {
		pids[p.PID] = true
	}
	if !pids[shell.PID] || !pids[sleeper.PID] {
		t.Fatalf("TTYProcesses(%s) = %v, want %d and %d", shell.TTY, pids, shell.PID, sleeper.PID)
	}
}
