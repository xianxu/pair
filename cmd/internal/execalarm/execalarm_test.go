//go:build darwin || linux

package execalarm

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestParse(t *testing.T) {
	for _, args := range [][]string{nil, {"1"}, {"1", "--"}, {"0", "--", "sleep"}, {"601", "--", "sleep"}, {"1.5", "--", "sleep"}, {"-1", "--", "sleep"}, {"1", "sleep"}, {"1", "--", ""}} {
		if _, _, err := parse(args); err == nil {
			t.Errorf("parse(%q) succeeded", args)
		}
	}
	for _, seconds := range []int{1, 600} {
		n, argv, err := parse([]string{strconv.Itoa(seconds), "--", "sleep", "30"})
		if err != nil || n != seconds || len(argv) != 2 || argv[1] != "30" {
			t.Fatalf("parse = %d %q %v", n, argv, err)
		}
	}
}

func TestRunMissingProgram(t *testing.T) {
	if err := Run([]string{"1", "--", filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("missing program succeeded")
	}
}

func TestAlarmSurvivesExec(t *testing.T)                   { testAlarm(t, "run") }
func TestConsumedAlarmBeforeExecStillExpires(t *testing.T) { testAlarm(t, "consumed") }
func TestIgnoredAlarmStillExpires(t *testing.T)            { testAlarm(t, "ignored") }
func TestBlockedAlarmStillExpires(t *testing.T)            { testAlarm(t, "blocked") }

func testAlarm(t *testing.T, mode string) {
	t.Helper()
	lease, err := os.Create(filepath.Join(t.TempDir(), "lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := unix.Flock(int(lease.Fd()), unix.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestExecAlarmHelper$")
	cmd.Env = append(os.Environ(), "PAIR_EXECALARM_TEST="+mode)
	cmd.ExtraFiles = []*os.File{lease}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	lease.Close()
	probe, err := os.OpenFile(lease.Name(), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected alarm death, got %v", err)
		}
		status := exit.Sys().(syscall.WaitStatus)
		if !status.Signaled() || status.Signal() != syscall.SIGALRM {
			t.Fatalf("status %v", status)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("exec child outlived alarm")
	}
	if err := unix.Flock(int(probe.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("lease survived child: %v", err)
	}
}

func TestOwnerDeathCannotReleaseExecLease(t *testing.T) {
	lease, err := os.Create(filepath.Join(t.TempDir(), "lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := unix.Flock(int(lease.Fd()), unix.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	// These pipes belong to the root test. Reaping the owner must not close the
	// orphan's input or output before the root releases its controlled hold.
	input, control, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer control.Close()
	output, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	defer writer.Close()
	owner := exec.Command(os.Args[0], "-test.run=^TestExecAlarmHelper$")
	owner.Env = append(os.Environ(), "PAIR_EXECALARM_TEST=owner")
	owner.ExtraFiles = []*os.File{lease}
	owner.Stdin = input
	owner.Stdout = writer
	owner.Stderr = os.Stderr
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	defer owner.Process.Kill()
	lease.Close()
	input.Close()
	writer.Close()
	events := make(chan string, 4)
	go func() {
		s := bufio.NewScanner(output)
		for s.Scan() {
			events <- s.Text()
		}
		events <- "EOF"
	}()
	await := func(want string) {
		t.Helper()
		select {
		case line := <-events:
			if line != want {
				t.Fatalf("exec fixture: got %q, want %q", line, want)
			}
		case <-time.After(20 * time.Second):
			t.Fatalf("exec fixture did not report %q", want)
		}
	}
	await("held")
	if err := owner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = owner.Wait()
	exit, ok := err.(*exec.ExitError)
	if !ok || !exit.Sys().(syscall.WaitStatus).Signaled() || exit.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatalf("owner not killed by test: %v", err)
	}
	// Require two real alarm deliveries after owner death while the fixture
	// deliberately holds the lease. This crosses the one-second budget using
	// kernel events, rather than relying on the parent being scheduled promptly.
	for range 2 {
		if _, err := fmt.Fprintln(control, "probe"); err != nil {
			t.Fatal(err)
		}
		await("held-after-alarm")
	}
	probe, err := os.OpenFile(lease.Name(), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	if err := unix.Flock(int(probe.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != unix.EWOULDBLOCK {
		t.Fatalf("owner death released held exec lease: %v", err)
	}
	if _, err := fmt.Fprintln(control, "release"); err != nil {
		t.Fatal(err)
	}
	await("alarm")
	await("EOF")
	if err := unix.Flock(int(probe.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("lease survived exec fixture exit: %v", err)
	}
}

func TestExecAlarmHelper(t *testing.T) {
	mode := os.Getenv("PAIR_EXECALARM_TEST")
	if mode == "" {
		return
	}
	if mode == "owner" {
		child := exec.Command(os.Args[0], "-test.run=^TestExecAlarmHelper$")
		child.Env = append(os.Environ(), "PAIR_EXECALARM_TEST=orphan")
		child.ExtraFiles = []*os.File{os.NewFile(3, "lease")}
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			panic(err)
		}
		_ = child.ExtraFiles[0].Close()
		_ = child.Wait()
		os.Exit(0)
	}
	if mode == "held-exec" {
		alarms := make(chan os.Signal, 1)
		signal.Notify(alarms, syscall.SIGALRM)
		fmt.Println("held")
		commands := bufio.NewScanner(os.Stdin)
		for commands.Scan() {
			// Discard any alarm queued before the root's command, then require a
			// fresh kernel expiry for each receipt. Go intentionally catches ALRM in
			// this fixture so the lease can stay held for arbitrarily delayed roots.
			select {
			case <-alarms:
			default:
			}
			<-alarms
			switch commands.Text() {
			case "probe":
				fmt.Println("held-after-alarm")
			case "release":
				fmt.Println("alarm")
				os.Exit(0)
			default:
				panic("unknown held-exec command")
			}
		}
		panic("held-exec input closed before release")
	}
	if mode == "consumed" {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGALRM)
		if err := armTimer(1); err != nil {
			panic(err)
		}
		<-ch // Deliberately consume the first expiry before replacing this process.
		path, err := exec.LookPath("sleep")
		if err != nil {
			panic(err)
		}
		if err := syscall.Exec(path, []string{"sleep", "30"}, os.Environ()); err != nil {
			panic(err)
		}
	}
	if mode == "ignored" || mode == "blocked" {
		if mode == "ignored" {
			signal.Ignore(syscall.SIGALRM)
		} else if err := blockAlarmForTest(); err != nil {
			panic(err)
		}
		// Re-enter a fresh Go runtime under the inherited ignored/blocked state.
		if err := os.Setenv("PAIR_EXECALARM_TEST", "run"); err != nil {
			panic(err)
		}
		if err := syscall.Exec(os.Args[0], []string{os.Args[0], "-test.run=^TestExecAlarmHelper$"}, os.Environ()); err != nil {
			panic(err)
		}
	}
	args := []string{"1", "--", "sleep", "30"}
	if mode == "orphan" {
		if err := os.Setenv("PAIR_EXECALARM_TEST", "held-exec"); err != nil {
			panic(err)
		}
		args = []string{"1", "--", os.Args[0], "-test.run=^TestExecAlarmHelper$"}
	}
	if err := Run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
