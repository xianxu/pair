package zellijexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The helper's child keeps a private liveness pipe open while blocked on a
// gate. EOF therefore proves it stopped; parent exit alone cannot satisfy it.
func TestQueryProcessHelper(t *testing.T) {
	mode := os.Getenv("PAIR_QUERY_PROCESS_HELPER")
	if mode == "" {
		return
	}
	if mode == "output" {
		fmt.Fprint(os.Stdout, "query output")
		fmt.Fprint(os.Stderr, "query diagnostics")
		os.Exit(0)
	}
	ready := os.NewFile(3, "ready")
	alive := os.NewFile(4, "alive")
	childGate := os.NewFile(5, "child-gate")
	parentGate := os.NewFile(6, "parent-gate")
	if mode == "child-closed" || mode == "child-held" {
		if mode == "child-closed" {
			os.Stdout.Close()
			os.Stderr.Close()
		}
		group, err := syscall.Getpgid(0)
		if err != nil {
			os.Exit(80)
		}
		fmt.Fprintf(ready, "%d %d\n", os.Getppid(), group)
		ready.Close()
		var token [1]byte
		_, _ = childGate.Read(token[:])
		alive.Close()
		os.Exit(0)
	}
	childMode := "child-closed"
	if mode == "held" {
		childMode = "child-held"
	}
	child := exec.Command(os.Args[0], "-test.run=^TestQueryProcessHelper$")
	child.Env = append(os.Environ(), "PAIR_QUERY_PROCESS_HELPER="+childMode)
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	child.ExtraFiles = []*os.File{ready, alive, childGate, parentGate}
	if err := child.Start(); err != nil {
		os.Exit(81)
	}
	ready.Close()
	alive.Close()
	childGate.Close()
	var token [1]byte
	_, _ = parentGate.Read(token[:])
	if mode == "failure" {
		os.Exit(7)
	}
	os.Exit(0)
}

func TestRunPreservesCallerConfiguration(t *testing.T) {
	for _, delay := range []time.Duration{0, 40 * time.Millisecond, 2 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestQueryProcessHelper$")
			cmd.Env = append(os.Environ(), "PAIR_QUERY_PROCESS_HELPER=output")
			credential := &syscall.Credential{Uid: uint32(os.Getuid()), Gid: uint32(os.Getgid()), NoSetGroups: true}
			attr := &syscall.SysProcAttr{Credential: credential, Pgid: 12345}
			cmd.SysProcAttr = attr
			cmd.WaitDelay = delay
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := Run(cmd); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != "query output" || stderr.String() != "query diagnostics" {
				t.Fatalf("lost output: %q, %q", &stdout, &stderr)
			}
			if cmd.SysProcAttr.Credential != credential || !cmd.SysProcAttr.Setpgid || cmd.SysProcAttr.Pgid != 0 {
				t.Fatalf("query process attributes: %+v", cmd.SysProcAttr)
			}
			if attr.Setpgid || attr.Pgid != 12345 {
				t.Fatalf("modified caller's shared attributes: %+v", attr)
			}
			want := delay
			if want <= 0 || want > time.Second {
				want = time.Second
			}
			if cmd.WaitDelay != want {
				t.Fatalf("WaitDelay=%s, want %s", cmd.WaitDelay, want)
			}
		})
	}
}

func TestRunPreservesStartAndContextErrors(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		cmd := exec.CommandContext(context.Background(), t.TempDir()+"/absent-zellij-query")
		if err := Run(cmd); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("lost start error: %v", err)
		}
		if cmd.Process != nil {
			t.Fatal("failed start created a process")
		}
	})
	t.Run("already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestQueryProcessHelper$")
		if err := Run(cmd); !errors.Is(err, context.Canceled) {
			t.Fatalf("lost context error: %v", err)
		}
		if cmd.Process != nil {
			t.Fatal("pre-canceled command created a process")
		}
	})
}

// Fixed-count control for very short shell queries and their cat children.
// Check both the native exec path and group ownership under the same budget;
// never retry a failed invocation or discard a cleanup error.
func TestRunFastExitScriptControl(t *testing.T) {
	const count = 100
	scripts := []struct{ name, body string }{
		{"shell", "exit 0\n"},
		{"foreground-child", "cat <<'EOF'\nquery\nEOF\n"},
		{"background-child", "cat <<'EOF' &\nquery\nEOF\nwait\n"},
	}
	for _, script := range scripts {
		path := filepath.Join(t.TempDir(), "query")
		if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' \"$$\"\n"+script.body), 0700); err != nil {
			t.Fatal(err)
		}
		for _, runner := range []struct {
			name string
			run  func(*exec.Cmd) error
		}{{"plain", (*exec.Cmd).Run}, {"group", Run}} {
			t.Run(script.name+"/"+runner.name, func(t *testing.T) {
				failures := 0
				for i := 0; i < count; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					cmd := exec.CommandContext(ctx, path)
					var stdout, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &stdout, &stderr
					cmd.WaitDelay = time.Second
					err := runner.run(cmd)
					cancel()
					var reportedPID int
					_, identityErr := fmt.Fscan(strings.NewReader(stdout.String()), &reportedPID)
					if err != nil || identityErr != nil || cmd.Process == nil || cmd.Process.Pid <= 0 || reportedPID != cmd.Process.Pid || cmd.ProcessState == nil || cmd.ProcessState.Pid() != reportedPID {
						failures++
						t.Errorf("iteration=%d error=%v identity-error=%v process=%+v state=%v stdout=%q stderr=%q", i, err, identityErr, cmd.Process, cmd.ProcessState, &stdout, &stderr)
					}
				}
				t.Logf("fixed control: invocations=%d failures=%d", count, failures)
			})
		}
	}
}

func TestStopGroupKeepsUncertainCleanupErrors(t *testing.T) {
	lookupErr := errors.New("group lookup unavailable")
	for _, tc := range []struct {
		name      string
		signalErr error
		present   bool
		lookupErr error
		want      error
		lookup    bool
	}{
		{"stopped", nil, true, nil, nil, false},
		{"absent", syscall.ESRCH, true, nil, nil, false},
		{"independently empty", syscall.EPERM, false, nil, nil, true},
		{"live or zombie", syscall.EPERM, true, nil, syscall.EPERM, true},
		{"unknown", syscall.EPERM, false, lookupErr, syscall.EPERM, true},
		{"other error", syscall.EIO, false, nil, syscall.EIO, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			looked := false
			err := stopGroup(123, func(group int, signal syscall.Signal) error {
				if group != -123 || signal != syscall.SIGKILL {
					t.Fatalf("signal=%d %v", group, signal)
				}
				return tc.signalErr
			}, func(group int) (bool, error) {
				if group != 123 {
					t.Fatalf("lookup group=%d", group)
				}
				looked = true
				return tc.present, tc.lookupErr
			})
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
			if tc.lookupErr != nil && !errors.Is(err, tc.lookupErr) {
				t.Fatalf("lost membership error: %v", err)
			}
			if looked != tc.lookup {
				t.Fatalf("membership lookup=%v, want %v", looked, tc.lookup)
			}
		})
	}
}

func TestRunCollectsQueryProcessGroup(t *testing.T) {
	for _, mode := range []string{"cancel", "success", "failure", "held"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			readyR, readyW := testPipe(t)
			aliveR, aliveW := testPipe(t)
			childR, childW := testPipe(t)
			parentR, parentW := testPipe(t)
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestQueryProcessHelper$")
			cmd.Env = append(os.Environ(), "PAIR_QUERY_PROCESS_HELPER="+mode)
			cmd.ExtraFiles = []*os.File{readyW, aliveW, childR, parentR}
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			cmd.WaitDelay = 40 * time.Millisecond
			done := make(chan error, 1)
			go func() { done <- Run(cmd) }()
			group := 0
			t.Cleanup(func() {
				cancel()
				if group > 0 {
					_ = syscall.Kill(-group, syscall.SIGKILL)
				}
				// Release only this fixture's gates on assertion failures.
				childW.Close()
				parentW.Close()
			})
			type identity struct {
				leader, group int
				err           error
			}
			identified := make(chan identity, 1)
			go func() {
				var v identity
				_, v.err = fmt.Fscanf(readyR, "%d %d\n", &v.leader, &v.group)
				identified <- v
			}()
			select {
			case v := <-identified:
				if v.err != nil {
					t.Fatal(v.err)
				}
				if v.group <= 0 || v.group != v.leader {
					t.Fatalf("query lacks private group: %+v", v)
				}
				group = v.group
			case <-time.After(5 * time.Second):
				t.Fatal("query child did not reach readiness barrier")
			}
			if mode == "cancel" {
				cancel()
			} else if _, err := parentW.Write([]byte{1}); err != nil {
				t.Fatal(err)
			}
			var runErr error
			select {
			case runErr = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Run did not join the direct child")
			}
			// Run completion also joins Start's descriptor inspection. Closing
			// these earlier races its Fd calls, even after child readiness.
			readyW.Close()
			aliveW.Close()
			childR.Close()
			parentR.Close()
			switch mode {
			case "success":
				if runErr != nil {
					t.Fatalf("successful query: %v", runErr)
				}
			case "cancel":
				var exit *exec.ExitError
				if !errors.As(runErr, &exit) {
					t.Fatalf("lost canceled command exit: %v", runErr)
				}
			case "failure":
				var exit *exec.ExitError
				if !errors.As(runErr, &exit) || exit.ExitCode() != 7 {
					t.Fatalf("lost exit 7: %v", runErr)
				}
			case "held":
				if !errors.Is(runErr, exec.ErrWaitDelay) {
					t.Fatalf("lost pipe-drain cutoff: %v", runErr)
				}
			}
			stopped := make(chan error, 1)
			go func() { var b [1]byte; _, err := aliveR.Read(b[:]); stopped <- err }()
			select {
			case err := <-stopped:
				if !errors.Is(err, io.EOF) {
					t.Fatalf("liveness pipe: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("query-owned descendant survived Run completion")
			}
			if cmd.ProcessState == nil || !cmd.ProcessState.Exited() && mode != "cancel" {
				t.Fatalf("direct child was not joined: %v", cmd.ProcessState)
			}
		})
	}
}

func testPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	return r, w
}
