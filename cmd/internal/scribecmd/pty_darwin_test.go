//go:build darwin

package scribecmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// This child never opens /dev/tty. Its pipe acknowledgement proves that ALL
// child-owned slave descriptors closed before the parent can read the master.
func TestScribeNativeChild(t *testing.T) {
	mode := os.Getenv("PAIR_SCRIBE_TEST_CHILD")
	if mode == "" {
		return
	}
	if mode == "descendant" {
		signal.Ignore(syscall.SIGHUP)
		ready := os.NewFile(3, "ready")
		_, _ = ready.Write([]byte{1})
		_ = ready.Close()
		hold := os.NewFile(4, "hold")
		var b [1]byte
		_, _ = hold.Read(b[:])
		os.Exit(0)
	}
	if mode == "blocked" {
		signal.Ignore(syscall.SIGHUP)
		ack := os.NewFile(3, "ack")
		_, _ = ack.Write([]byte{1})
		_ = ack.Close()
		// A live child that cannot be stopped by master hangup alone.
		hold := os.NewFile(4, "hold")
		var b [1]byte
		_, _ = hold.Read(b[:])
		os.Exit(0)
	}
	if mode == "interactive" {
		// Report a real terminal resize, then accept and echo a line. No tty reopen.
		resized := make(chan os.Signal, 1)
		signal.Notify(resized, syscall.SIGWINCH)
		var ws *pty.Winsize
		for {
			var err error
			ws, err = pty.GetsizeFull(os.Stdin)
			if err != nil {
				os.Exit(99)
			}
			if ws.Rows != 0 && ws.Cols != 0 {
				break
			}
			<-resized
		}

		fmt.Fprintf(os.Stdout, "size=%dx%d\n", ws.Rows, ws.Cols)
		var b [80]byte
		n, _ := os.Stdin.Read(b[:])
		_, _ = os.Stdout.Write(b[:n])
		os.Exit(0)
	}
	if mode == "large" {
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), 128*1024))
		os.Exit(0)
	}
	if mode == "signal" {
		signal.Reset(syscall.SIGTERM, syscall.SIGHUP)
		_ = syscall.Kill(os.Getpid(), syscall.Signal(atoi(os.Getenv("PAIR_SCRIBE_TEST_SIGNAL"))))
		time.Sleep(time.Hour)
		os.Exit(98)
	}
	var descendant int
	if mode == "leader" {
		r, w, err := os.Pipe()
		if err != nil {
			os.Exit(91)
		}
		exe, _ := os.Executable()
		child := exec.Command(exe, "-test.run=^TestScribeNativeChild$")
		child.Env = append(os.Environ(), "PAIR_SCRIBE_TEST_CHILD=descendant")
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.ExtraFiles = []*os.File{w, os.NewFile(4, "hold")}
		if child.Start() != nil {
			os.Exit(92)
		}
		_ = w.Close()
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			os.Exit(93)
		}
		_ = r.Close()
		descendant = child.Process.Pid
	}

	code, _ := strconv.Atoi(os.Getenv("PAIR_SCRIBE_TEST_EXIT"))
	_, _ = os.Stdout.Write([]byte("final-marker"))
	for _, fd := range []int{0, 1, 2} {
		_ = syscall.Close(fd)
	}
	ack := os.NewFile(3, "ack")
	_, _ = ack.Write([]byte{1})
	if descendant != 0 {
		_, _ = fmt.Fprintln(ack, descendant)
	}
	_ = ack.Close()
	if mode == "live-closed" {
		var b [1]byte
		_, _ = os.NewFile(4, "hold").Read(b[:])
	}
	os.Exit(code)
}

func nativeWriter(t *testing.T, code int) (*exec.Cmd, *os.File, *os.File) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	cmd := exec.Command(exe, "-test.run=^TestScribeNativeChild$")
	cmd.Env = append(os.Environ(), "PAIR_SCRIBE_TEST_CHILD=writer", "PAIR_SCRIBE_TEST_EXIT="+strconv.Itoa(code))
	cmd.ExtraFiles = []*os.File{w}
	return cmd, r, w
}

// This watchdog only bounds a broken native test and closes its exact owned
// master/child. It is not a production deadline and never establishes ordering.
func guardNative(t *testing.T, cmd *exec.Cmd, p *startedPTY) {
	t.Helper()
	var fired atomic.Bool
	done := make(chan struct{})
	timer := time.AfterFunc(5*time.Second, func() {
		defer close(done)
		fired.Store(true)
		_ = cmd.Process.Kill()
		_ = p.master.Close()
	})
	t.Cleanup(func() {
		if !timer.Stop() {
			<-done
		}
		if fired.Load() {
			t.Error("native PTY did not finish; watchdog cleaned owned child/master")
		}
		if cmd.ProcessState == nil {
			_ = abortPTY(cmd, p)
		}
	})
}

func awaitClosedStdio(t *testing.T, r *os.File) {
	t.Helper()
	done := make(chan error, 1)
	go func() { var b [1]byte; _, err := io.ReadFull(r, b[:]); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("child stdio-close barrier: %v", err)
		}
	case <-time.After(5 * time.Second):
		_ = r.Close()
		<-done
		t.Fatal("child did not close stdio")
	}
}

func TestDarwinPTYFinalOutputAfterSlaveClose(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			cmd, r, w := nativeWriter(t, code)
			p, err := startPTY(cmd)
			if err != nil {
				t.Fatal(err)
			}
			guardNative(t, cmd, p)
			master := p.master
			defer master.Close()
			defer p.releaseSlave()
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			_ = w.Close()
			awaitClosedStdio(t, r)
			got, err := io.ReadAll(master)
			if err != nil {
				t.Fatal(err)
			}
			if err := p.releaseSlave(); err != nil {
				t.Fatal(err)
			}
			werr := cmd.Wait()
			if !bytes.Equal(got, []byte("final-marker")) {
				t.Errorf("final bytes = %q, want final-marker", got)
			}
			if code == 0 && werr != nil {
				t.Fatal(werr)
			}
			if cmd.ProcessState.ExitCode() != code {
				t.Fatalf("exit = %v, want %d", cmd.ProcessState, code)
			}
		})
	}
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func TestDarwinRunDrainsBeforeWait(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			template, r, w := nativeWriter(t, code)
			log := filepath.Join(t.TempDir(), "log")
			var out, diag bytes.Buffer
			result := run([]string{"-log", log, "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
				cmd.Env = template.Env
				cmd.ExtraFiles = template.ExtraFiles
				p, err := startPTY(cmd)
				if err != nil {
					return nil, err
				}
				guardNative(t, cmd, p)
				_ = w.Close()
				awaitClosedStdio(t, r)
				return p, nil
			})
			body, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			if result != code || out.String() != "final-marker" || string(body) != "final-marker" || diag.Len() != 0 {
				t.Fatalf("result=%d out=%q log=%q diag=%q", result, out.String(), body, diag.String())
			}
		})
	}
}

func TestDarwinRunDescendantHoldingSlave(t *testing.T) {
	template, r, w := nativeWriter(t, 7)
	holdR, holdW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer holdR.Close()
	defer holdW.Close()
	var descendant int
	defer func() {
		if descendant != 0 {
			_ = syscall.Kill(descendant, syscall.SIGKILL)
		}
	}()
	log := filepath.Join(t.TempDir(), "log")
	var out, diag bytes.Buffer
	result := run([]string{"-log", log, "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
		cmd.Env = append(template.Env, "PAIR_SCRIBE_TEST_CHILD=leader")
		cmd.ExtraFiles = []*os.File{w, holdR}
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		guardNative(t, cmd, p)
		_ = w.Close()
		awaitClosedStdio(t, r)
		if _, err := fmt.Fscan(r, &descendant); err != nil {
			_ = abortPTY(cmd, p)
			return nil, err
		}
		return p, nil
	})
	body, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if result != 7 || out.String() != "final-marker" || string(body) != "final-marker" || diag.Len() != 0 {
		t.Fatalf("result=%d out=%q log=%q diag=%q", result, out.String(), body, diag.String())
	}
	if descendant == 0 {
		t.Fatal("no descendant identity")
	}
	if err := syscall.Kill(descendant, 0); err != nil {
		t.Fatalf("descendant did not hold slave through run return: %v", err)
	}
	// This exact descendant is test-owned. Release its control pipe after run
	// returns; cleanup also signals it on every failure path.
	_ = holdW.Close()
}

func TestDarwinRunAbortsChildOnRawModeFailure(t *testing.T) {
	for _, mode := range []string{"blocked", "writer"} {
		t.Run(mode, func(t *testing.T) {
			template, r, w := nativeWriter(t, 7)
			holdR, holdW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer holdR.Close()
			defer holdW.Close()
			input, err := os.CreateTemp(t.TempDir(), "not-a-tty")
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			var child *exec.Cmd
			var started *startedPTY
			var diag bytes.Buffer
			code := run([]string{"-log", filepath.Join(t.TempDir(), "log"), "--", template.Path, template.Args[1]}, input, io.Discard, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
				child = cmd
				cmd.Env = append(template.Env, "PAIR_SCRIBE_TEST_CHILD="+mode)
				cmd.ExtraFiles = []*os.File{w, holdR}
				p, err := startPTY(cmd)
				if err != nil {
					return nil, err
				}
				started = p
				guardNative(t, cmd, p)
				_ = w.Close()
				awaitClosedStdio(t, r)
				return p, nil
			})
			if code != 1 || !strings.Contains(diag.String(), "MakeRaw:") {
				t.Fatalf("code=%d diagnostic=%q", code, diag.String())
			}
			if child == nil || child.ProcessState == nil {
				t.Fatal("child not reaped")
			}
			if err := child.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
				t.Fatalf("child still owned: %v", err)
			}
			if _, err := started.master.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("master not closed: %v", err)
			}
			if err := started.slave.(*os.File).Close(); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("slave not closed: %v", err)
			}
		})
	}
}

func TestDarwinStartClosesDescriptorsOnExecFailure(t *testing.T) {
	for _, path := range []string{filepath.Join(t.TempDir(), "absent"), t.TempDir()} {
		var master, slave *os.File
		cmd := exec.Command(path)
		_, err := startPTYWithOpen(cmd, func() (*os.File, *os.File, error) {
			var e error
			master, slave, e = pty.Open()
			return master, slave, e
		})
		if err == nil {
			t.Fatal("exec should fail synchronously")
		}
		if cmd.Process != nil {
			t.Fatal("failed exec published a process")
		}
		for _, f := range []*os.File{master, slave} {
			if err := f.Close(); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("descriptor not closed: %v", err)
			}
		}
	}
	want := errors.New("allocation failed")
	if _, err := startPTYWithOpen(exec.Command("/bin/true"), func() (*os.File, *os.File, error) { return nil, nil, want }); !errors.Is(err, want) {
		t.Fatal(err)
	}
}

func TestDarwinLeaseResourceLifetimes(t *testing.T) {
	for _, n := range []int{8, 16} {
		leases := make([]*startedPTY, 0, n)
		cmds := make([]*exec.Cmd, 0, n)
		defer func() {
			for i, p := range leases {
				if cmds[i].ProcessState == nil {
					_ = abortPTY(cmds[i], p)
				}
			}
		}()
		for i := 0; i < n; i++ {
			cmd, r, w := nativeWriter(t, 0)
			p, err := startPTY(cmd)
			if err != nil {
				t.Fatal(err)
			}
			guardNative(t, cmd, p)
			leases = append(leases, p)
			cmds = append(cmds, cmd)
			_ = w.Close()
			awaitClosedStdio(t, r)
			if _, err := unix.FcntlInt(p.slave.(*os.File).Fd(), unix.F_GETFD, 0); err != nil {
				t.Fatalf("lease invalid before drain: %v", err)
			}
		}
		for i, p := range leases {
			if _, err := io.Copy(io.Discard, p.master); err != nil {
				t.Fatal(err)
			}
			if err := p.releaseSlave(); err != nil {
				t.Fatal(err)
			}
			if err := p.releaseSlave(); err != nil {
				t.Fatal(err)
			}
			if err := cmds[i].Wait(); err != nil {
				t.Fatal(err)
			}
			_ = p.master.Close()
			if _, err := unix.FcntlInt(p.slave.(*os.File).Fd(), unix.F_GETFD, 0); !errors.Is(err, syscall.EBADF) {
				t.Fatalf("lease survived join: %v", err)
			}
		}
	}
}

func TestDarwinRunLargeOutputAndSignals(t *testing.T) {
	for _, mode := range []string{"large", "signal"} {
		signals := []syscall.Signal{syscall.SIGTERM}
		if mode == "signal" {
			signals = append(signals, syscall.SIGHUP)
		}
		for _, sig := range signals {
			t.Run(fmt.Sprintf("%s-%d", mode, sig), func(t *testing.T) {
				template, _, _ := nativeWriter(t, 0)
				var out, diag bytes.Buffer
				log := filepath.Join(t.TempDir(), "log")
				result := run([]string{"-log", log, "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
					cmd.Env = append(template.Env, "PAIR_SCRIBE_TEST_CHILD="+mode, "PAIR_SCRIBE_TEST_SIGNAL="+strconv.Itoa(int(sig)))
					p, err := startPTY(cmd)
					if err == nil {
						guardNative(t, cmd, p)
					}
					return p, err
				})
				body, err := os.ReadFile(log)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "large" {
					want := bytes.Repeat([]byte("x"), 128*1024)
					if result != 0 || !bytes.Equal(out.Bytes(), want) || !bytes.Equal(body, want) {
						t.Fatalf("result=%d stdout=%d log=%d", result, out.Len(), len(body))
					}
				} else if result != -1 {
					t.Fatalf("signal exit=%d want -1", result)
				}
				if diag.Len() != 0 {
					t.Fatal(diag.String())
				}
			})
		}
	}
}

func TestDarwinStartPreservesCommand(t *testing.T) {
	dir := t.TempDir()
	output, err := os.CreateTemp(dir, "output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	cmd := exec.Command("/bin/sh", "-c", `printf '%s|%s|%s|%s' "$1" "$2" "$VALUE" "$PWD"`, "custom-argv0", "a b", "$(literal)")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "VALUE=hello world")
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	attrs := &syscall.SysProcAttr{Ctty: 0}
	cmd.SysProcAttr = attrs
	p, err := startPTY(cmd)
	if err != nil {
		t.Fatal(err)
	}
	guardNative(t, cmd, p)
	if cmd.Stdout != output || cmd.SysProcAttr != attrs || cmd.Args[0] != "/bin/sh" || cmd.Dir != dir {
		t.Fatal("provided command fields replaced")
	}
	_, _ = io.Copy(io.Discard, p.master)
	_ = p.releaseSlave()
	err = cmd.Wait()
	_ = p.master.Close()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	want := "a b|$(literal)|hello world|" + dir
	if string(body) != want {
		t.Fatalf("got %q want %q", body, want)
	}
}

func TestDarwinRunLiveClosedStdio(t *testing.T) {
	template, r, w := nativeWriter(t, 0)
	holdR, holdW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer holdR.Close()
	defer holdW.Close()
	var out, diag bytes.Buffer
	log := filepath.Join(t.TempDir(), "log")
	result := run([]string{"-log", log, "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
		cmd.Env = append(template.Env, "PAIR_SCRIBE_TEST_CHILD=live-closed")
		cmd.ExtraFiles = []*os.File{w, holdR}
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		guardNative(t, cmd, p)
		_ = w.Close()
		awaitClosedStdio(t, r)
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			t.Errorf("live target stopped: %v", err)
		}
		_ = holdW.Close()
		return p, nil
	})
	if result != 0 || out.String() != "final-marker" || diag.Len() != 0 {
		t.Fatalf("code=%d output=%q diag=%q", result, out.String(), diag.String())
	}
}

func TestDarwinRunInteractiveResizeAndRestore(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	before, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	template, _, _ := nativeWriter(t, 0)
	var out, diag bytes.Buffer
	log := filepath.Join(t.TempDir(), "log")
	// Set the outer size before startup; the child waits for Run's actual
	// initial SIGWINCH propagation before reporting the observed size.
	_ = pty.Setsize(master, &pty.Winsize{Rows: 31, Cols: 93})
	go func() { _, _ = master.Write([]byte("hello\n")) }()
	code := run([]string{"-log", log, "--", template.Path, template.Args[1]}, slave, &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
		cmd.Env = append(template.Env, "PAIR_SCRIBE_TEST_CHILD=interactive")
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		guardNative(t, cmd, p)
		return p, nil
	})
	after, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 || !strings.Contains(out.String(), "size=31x93") || !strings.Contains(out.String(), "hello") || diag.Len() != 0 {
		t.Fatalf("code=%d out=%q diag=%q", code, out.String(), diag.String())
	}
	if *before != *after {
		t.Fatal("outer terminal mode not restored")
	}
}

type reportingCloser struct {
	io.Closer
	err error
}

func (c reportingCloser) Close() error { _ = c.Closer.Close(); return c.err }

func TestDarwinRunCleanupDiagnosticPreservesExit(t *testing.T) {
	template, r, w := nativeWriter(t, 7)
	var out, diag bytes.Buffer
	code := run([]string{"-log", filepath.Join(t.TempDir(), "log"), "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
		cmd.Env = template.Env
		cmd.ExtraFiles = template.ExtraFiles
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		guardNative(t, cmd, p)
		p.slave = reportingCloser{p.slave, errors.New("reported close failure")}
		_ = w.Close()
		awaitClosedStdio(t, r)
		return p, nil
	})
	if code != 7 || out.String() != "final-marker" || !strings.Contains(diag.String(), "releasing slave: reported close failure") {
		t.Fatalf("code=%d out=%q diag=%q", code, out.String(), diag.String())
	}
}
