//go:build darwin

package wrapcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/nativelock"
	"golang.org/x/sys/unix"
)

// The native writer never opens /dev/tty. Ack fd3 establishes that every
// child-owned slave descriptor closed before the first parent master read.
func TestWrapPTYNativeChild(t *testing.T) {
	mode := os.Getenv("PAIR_WRAP_PTY_CHILD")
	if mode == "" {
		return
	}
	code, _ := strconv.Atoi(os.Getenv("PAIR_WRAP_PTY_EXIT"))
	if mode == "large" {
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), 128*1024))
	} else {
		_, _ = os.Stdout.Write([]byte("wrap-final-marker"))
	}
	for _, fd := range []int{0, 1, 2} {
		_ = syscall.Close(fd)
	}
	ack := os.NewFile(3, "ack")
	_, _ = ack.Write([]byte{1})
	if mode == "inspect" {
		cwd, _ := os.Getwd()
		_, _ = fmt.Fprintf(ack, "%s|%s|%s", os.Args[0], os.Getenv("WRAP_VALUE"), cwd)
	}
	if mode == "live" {
		var b [1]byte
		hold := os.NewFile(4, "hold")
		for {
			if _, err := hold.Read(b[:]); err != nil {
				break
			}
			// Answer only after a post-stdio-close request. A zombie cannot
			// produce this acknowledgement, unlike a successful signal-zero.
			_, _ = ack.Write(b[:])
		}
	}
	_ = ack.Close()
	os.Exit(code)
}

func wrapPTYNativeWriter(t *testing.T, code int) (*exec.Cmd, *os.File, *os.File) {
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
	cmd := exec.Command(exe, "-test.run=^TestWrapPTYNativeChild$")
	cmd.Env = append(os.Environ(), "PAIR_WRAP_PTY_CHILD=writer", "PAIR_WRAP_PTY_EXIT="+strconv.Itoa(code))
	cmd.ExtraFiles = []*os.File{w}
	return cmd, r, w
}

func wrapPTYAwaitClosed(t *testing.T, r *os.File) {
	t.Helper()
	var ack [1]byte
	if _, err := io.ReadFull(r, ack[:]); err != nil {
		t.Fatalf("closed stdio acknowledgement: %v", err)
	}
}

func wrapPTYProbeLive(ack, hold *os.File) error {
	if _, err := hold.Write([]byte{2}); err != nil {
		return err
	}
	var response [1]byte
	if _, err := io.ReadFull(ack, response[:]); err != nil {
		return err
	}
	if response[0] != 2 {
		return fmt.Errorf("live probe reply=%d, want2", response[0])
	}
	return nil
}

// This watchdog only bounds broken native tests, never orders their events.
func wrapPTYGuard(t *testing.T, cmd *exec.Cmd, p *startedPTY) *ownedGroup {
	t.Helper()
	group := newOwnedGroup(cmd, time.Second)
	var fired atomic.Bool
	done := make(chan struct{})
	watchdog := time.AfterFunc(30*time.Second, func() { defer close(done); fired.Store(true); _ = cmd.Process.Kill(); _ = p.master.Close() })
	t.Cleanup(func() {
		if !watchdog.Stop() {
			<-done
		}
		_ = p.master.Close()
		_ = p.releaseSlave()
		if cmd.ProcessState == nil {
			_ = group.Wait()
		}
		if fired.Load() {
			t.Error("native PTY watchdog cleaned owned target/master")
		}
	})
	return group
}

func TestDarwinPTYFinalOutputAfterChildStdioClosed(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			cmd, r, w := wrapPTYNativeWriter(t, code)
			p, err := startPTY(cmd)
			if err != nil {
				t.Fatal(err)
			}
			group := wrapPTYGuard(t, cmd, p)
			_ = w.Close()
			wrapPTYAwaitClosed(t, r)
			got, readErr := io.ReadAll(p.master)
			if err := p.releaseSlave(); err != nil {
				t.Error(err)
			}
			_ = group.Wait()
			if readErr != nil {
				t.Errorf("master read: %v", readErr)
			}
			if !bytes.Equal(got, []byte("wrap-final-marker")) {
				t.Errorf("final bytes %q, want wrap-final-marker", got)
			}
			if cmd.ProcessState.ExitCode() != code {
				t.Errorf("exit=%v, want %d", cmd.ProcessState, code)
			}
		})
	}
}

func wrapPTYRunPaths(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PAIR_TAG", "")
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_LAUNCH_ORDINAL", "")
	t.Setenv("PAIR_WRAP_REMAP_RETURN", "0")
	t.Setenv("PAIR_SCROLLBACK_EVENTS_PATH", filepath.Join(dir, "events"))
	return filepath.Join(dir, "raw")
}

func TestDarwinWrapRunDrainsBeforeOwnedWait(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			raw := wrapPTYRunPaths(t)
			template, r, w := wrapPTYNativeWriter(t, code)
			var out, diag bytes.Buffer
			var child *exec.Cmd
			status, err := runWithPTY([]string{"--scrollback-log", raw, "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
				child = cmd
				cmd.Env = template.Env
				cmd.ExtraFiles = template.ExtraFiles
				p, err := startPTY(cmd)
				if err != nil {
					return nil, err
				}
				wrapPTYGuard(t, cmd, p)
				_ = w.Close()
				wrapPTYAwaitClosed(t, r)
				return p, nil
			})
			body, readErr := os.ReadFile(raw)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if err != nil || status != code || out.String() != wrapPTYBanner+"wrap-final-marker" || string(body) != "wrap-final-marker" || diag.Len() != 0 {
				t.Fatalf("status=%d err=%v stdout=%q raw=%q diag=%q", status, err, out.String(), body, diag.String())
			}
			if child == nil || child.ProcessState == nil || child.ProcessState.ExitCode() != code {
				t.Fatalf("target was not reaped with status%d: %+v", code, child)
			}
		})
	}
}

const wrapPTYBanner = "\x1b[7m ⚙  pair-wrapped · Alt+h for help · Happy coding!                           \x1b[27m\r\n\r\n"

type wrapPTYMarkerOutput struct {
	bytes.Buffer
	ready chan struct{}
	once  sync.Once
}

func (w *wrapPTYMarkerOutput) Write(data []byte) (int, error) {
	n, err := w.Buffer.Write(data)
	if bytes.Contains(w.Buffer.Bytes(), []byte("wrap-final-marker")) {
		w.once.Do(func() { close(w.ready) })
	}
	return n, err
}

func TestDarwinWrapRunRetainsLiveClosedStdioTarget(t *testing.T) {
	raw := wrapPTYRunPaths(t)
	template, r, w := wrapPTYNativeWriter(t, 0)
	holdR, holdW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer holdR.Close()
	defer holdW.Close()
	out := &wrapPTYMarkerOutput{ready: make(chan struct{})}
	var diag bytes.Buffer
	var child *exec.Cmd
	ownerDone := make(chan struct{})
	checked := make(chan error, 1)
	status, runErr := runWithPTY([]string{"--scrollback-log", raw, "--", template.Path, template.Args[1]}, strings.NewReader(""), out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
		child = cmd
		cmd.Env = append(template.Env, "PAIR_WRAP_PTY_CHILD=live")
		cmd.ExtraFiles = []*os.File{w, holdR}
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		wrapPTYGuard(t, cmd, p)
		_ = w.Close()
		wrapPTYAwaitClosed(t, r)
		go func() {
			defer holdW.Close()
			select {
			case <-out.ready:
				// This cannot happen until production has read the master, after
				// the child's stdio-close acknowledgement. The hold is still live.
				checked <- wrapPTYProbeLive(r, holdW)
			case <-ownerDone:
				checked <- errors.New("wrapper returned before draining the live target")
			}
		}()
		return p, nil
	})
	close(ownerDone)
	if child == nil {
		t.Fatalf("target did not start: %v", runErr)
	}
	if err := <-checked; err != nil {
		t.Fatalf("live target lost before hold release: %v", err)
	}
	body, readErr := os.ReadFile(raw)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if status != 0 || runErr != nil || out.String() != wrapPTYBanner+"wrap-final-marker" || string(body) != "wrap-final-marker" || diag.Len() != 0 {
		t.Fatalf("status=%d err=%v stdout=%q raw=%q diag=%q", status, runErr, out.String(), body, diag.String())
	}
	if child.ProcessState == nil || child.ProcessState.ExitCode() != 0 {
		t.Fatal("live target was not reaped after hold release")
	}
}

func TestDarwinPTYConcurrentDrainLargeOutput(t *testing.T) {
	cmd, r, w := wrapPTYNativeWriter(t, 0)
	cmd.Env = append(cmd.Env, "PAIR_WRAP_PTY_CHILD=large")
	p, err := startPTY(cmd)
	if err != nil {
		t.Fatal(err)
	}
	group := wrapPTYGuard(t, cmd, p)
	_ = w.Close()
	// Drain while the child is writing: the payload exceeds native queue capacity.
	got, err := io.ReadAll(p.master)
	if err != nil {
		t.Fatal(err)
	}
	wrapPTYAwaitClosed(t, r)
	if err := p.releaseSlave(); err != nil {
		t.Fatal(err)
	}
	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bytes.Repeat([]byte("x"), 128*1024)) {
		t.Fatalf("large payload bytes=%d, want128KiB", len(got))
	}
}

func TestDarwinWrapRunDrainsLargeOutput(t *testing.T) {
	raw := wrapPTYRunPaths(t)
	template, r, w := wrapPTYNativeWriter(t, 0)
	var out, diag bytes.Buffer
	status, err := runWithPTY([]string{"--scrollback-log", raw, "--", template.Path, template.Args[1]}, strings.NewReader(""), &out, &diag, func(cmd *exec.Cmd) (*startedPTY, error) {
		cmd.Env = append(template.Env, "PAIR_WRAP_PTY_CHILD=large")
		cmd.ExtraFiles = template.ExtraFiles
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		wrapPTYGuard(t, cmd, p)
		_ = w.Close()
		// Do not wait for the closed-stdio acknowledgement here: production
		// must drain concurrently so a queue-sized write can complete at all.
		return p, nil
	})
	wrapPTYAwaitClosed(t, r)
	body, readErr := os.ReadFile(raw)
	if readErr != nil {
		t.Fatal(readErr)
	}
	payload := bytes.Repeat([]byte("x"), 128*1024)
	wantOut := append([]byte(wrapPTYBanner), payload...)
	if status != 0 || err != nil || !bytes.Equal(out.Bytes(), wantOut) || !bytes.Equal(body, payload) || diag.Len() != 0 {
		t.Fatalf("status=%d err=%v stdoutbytes=%d rawbytes=%d diag=%q", status, err, out.Len(), len(body), diag.String())
	}
}

func TestDarwinPTYClosesAllocationsOnStartFailure(t *testing.T) {
	dir := t.TempDir()
	nonexec := filepath.Join(dir, "nonexec")
	malformed := filepath.Join(dir, "malformed")
	if err := os.WriteFile(nonexec, []byte("not executable"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(malformed, []byte("not an executable image\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{filepath.Join(dir, "missing"), nonexec, malformed} {
		t.Run(filepath.Base(target), func(t *testing.T) {
			var master, slave *os.File
			cmd := exec.Command(target)
			p, err := startPTYWithOpen(cmd, func() (*os.File, *os.File, error) {
				var err error
				master, slave, err = pty.Open()
				return master, slave, err
			})
			if err == nil || p != nil || cmd.Process != nil {
				t.Fatalf("start failure returned p=%v err=%v process=%v", p, err, cmd.Process)
			}
			for _, f := range []*os.File{master, slave} {
				if f == nil {
					t.Fatal("allocation not reached")
				}
				if err := f.Close(); !errors.Is(err, os.ErrClosed) {
					t.Errorf("descriptor survived failed exec: %v", err)
				}
			}
		})
	}
	want := errors.New("original command resolution failure")
	cmd := exec.Command("/bin/true")
	cmd.Err = want
	if _, err := startPTY(cmd); !errors.Is(err, want) {
		t.Fatalf("command Err lost: %v", err)
	}
	for _, partial := range []bool{false, true} {
		var master, slave *os.File
		_, err := startPTYWithOpen(exec.Command("/bin/true"), func() (*os.File, *os.File, error) {
			if partial {
				master, slave, _ = os.Pipe()
			}
			return master, slave, want
		})
		if !errors.Is(err, want) {
			t.Fatalf("open error lost: %v", err)
		}
		if partial {
			for _, f := range []*os.File{master, slave} {
				if err := f.Close(); !errors.Is(err, os.ErrClosed) {
					t.Errorf("partial allocation leaked: %v", err)
				}
			}
		}
	}
}

func TestDarwinPTYStartPreservesCommand(t *testing.T) {
	cmd, r, w := wrapPTYNativeWriter(t, 0)
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cmd.Dir = dir
	cmd.Args[0] = "wrap-owned-custom-argv0"
	cmd.Env = append(cmd.Env, "PAIR_WRAP_PTY_CHILD=inspect", "WRAP_VALUE=hello $(literal)")
	attrs := &syscall.SysProcAttr{Ctty: 0}
	cmd.SysProcAttr = attrs
	path := cmd.Path
	p, err := startPTY(cmd)
	if err != nil {
		t.Fatal(err)
	}
	group := wrapPTYGuard(t, cmd, p)
	_ = w.Close()
	wrapPTYAwaitClosed(t, r)
	metadata, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(p.master)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.releaseSlave(); err != nil {
		t.Fatal(err)
	}
	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}
	if cmd.Path != path || cmd.Args[0] != "wrap-owned-custom-argv0" || cmd.SysProcAttr != attrs || cmd.Dir != dir || len(cmd.ExtraFiles) != 1 {
		t.Fatal("provided exec.Cmd fields replaced")
	}
	if string(metadata) != "wrap-owned-custom-argv0|hello $(literal)|"+dir || string(got) != "wrap-final-marker" {
		t.Fatalf("metadata=%q master=%q", metadata, got)
	}
}

func TestDarwinPTYLeaseResourceLifetimes(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			leases := make([]*startedPTY, 0, n)
			groups := make([]*ownedGroup, 0, n)
			for range n {
				cmd, r, w := wrapPTYNativeWriter(t, 0)
				p, err := startPTY(cmd)
				if err != nil {
					t.Fatal(err)
				}
				groups = append(groups, wrapPTYGuard(t, cmd, p))
				leases = append(leases, p)
				_ = w.Close()
				wrapPTYAwaitClosed(t, r)
				if _, err := unix.FcntlInt(p.slave.(*os.File).Fd(), unix.F_GETFD, 0); err != nil {
					t.Fatalf("lease closed before drain: %v", err)
				}
			}
			for i, p := range leases {
				got, err := io.ReadAll(p.master)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != "wrap-final-marker" {
					t.Fatalf("lease%d bytes=%q", i, got)
				}
				for range 2 {
					if err := p.releaseSlave(); err != nil {
						t.Fatal(err)
					}
				}
				if err := groups[i].Wait(); err != nil {
					t.Fatal(err)
				}
				_ = p.master.Close()
				if _, err := unix.FcntlInt(p.slave.(*os.File).Fd(), unix.F_GETFD, 0); !errors.Is(err, syscall.EBADF) {
					t.Fatalf("lease survived join: %v", err)
				}
			}
		})
	}
}

func TestDarwinWrapRunReapsOnTransportInitializationFailure(t *testing.T) {
	raw := wrapPTYRunPaths(t)
	template, r, w := wrapPTYNativeWriter(t, 0)
	holdR, holdW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer holdR.Close()
	defer holdW.Close()
	var child *exec.Cmd
	var started *startedPTY
	_, err = runWithPTY([]string{"--scrollback-log", raw, "--", template.Path, template.Args[1]}, strings.NewReader(""), io.Discard, io.Discard, func(cmd *exec.Cmd) (*startedPTY, error) {
		child = cmd
		cmd.Env = append(template.Env, "PAIR_WRAP_PTY_CHILD=live")
		cmd.ExtraFiles = []*os.File{w, holdR}
		p, err := startPTY(cmd)
		if err != nil {
			return nil, err
		}
		started = p
		wrapPTYGuard(t, cmd, p)
		_ = w.Close()
		wrapPTYAwaitClosed(t, r)
		if err := p.master.Close(); err != nil {
			return nil, err
		}
		return p, nil
	})
	if err == nil || !strings.Contains(err.Error(), "initialize child terminal transport") {
		t.Fatalf("missing transport initialization failure: %v", err)
	}
	if child == nil || child.ProcessState == nil {
		t.Fatal("ownedGroup did not reap target via Cmd.Wait")
	}
	if err := child.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("live target survived initialization failure: %v", err)
	}
	if err := started.slave.(*os.File).Close(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("slave lease survived initialization failure: %v", err)
	}
}

func TestDarwinPTYExplicitStopAndRestartOwnLiveClosedStdio(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGUSR2} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd, r, w := wrapPTYNativeWriter(t, 0)
			holdR, holdW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer holdR.Close()
			defer holdW.Close()
			cmd.Env = append(cmd.Env, "PAIR_WRAP_PTY_CHILD=live")
			cmd.ExtraFiles = []*os.File{w, holdR}
			home := t.TempDir()
			locks := newWrapperNativeLocks(home, "codex", "native-pty-owner")
			t.Cleanup(locks.Close) // registered before the owned group's cleanup
			if err := locks.Discover("closed-stdio"); err != nil {
				t.Fatal(err)
			}
			key, err := nativelock.NewKey("codex", "closed-stdio")
			if err != nil {
				t.Fatal(err)
			}
			assertHeld := func() {
				t.Helper()
				if unexpected, err := nativelock.Acquire(home, key, "contender"); err == nil {
					_ = unexpected.Close()
					t.Fatal("conversation lock released before group cleanup")
				}
			}
			started, err := startPTY(cmd)
			if err != nil {
				t.Fatal(err)
			}
			group := wrapPTYGuard(t, cmd, started)
			_ = w.Close()
			wrapPTYAwaitClosed(t, r)
			if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
				t.Fatalf("closed-stdio target was not live before explicit stop: %v", err)
			}
			if err := wrapPTYProbeLive(r, holdW); err != nil {
				t.Fatalf("closed-stdio target failed live acknowledgement: %v", err)
			}
			owner := &proxy{cmd: cmd, ptmx: started.master, nativeLocks: locks, ownedGroup: group}
			restart := owner.stopForSignal(sig, locks, group)
			assertHeld()
			got, readErr := io.ReadAll(started.master)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if err := started.releaseSlave(); err != nil {
				t.Fatal(err)
			}
			waitErr := group.Wait()
			if !owner.stoppedByWrapper.Load() || owner.restartFresh.Load() != (sig == syscall.SIGUSR2) || restart != (sig == syscall.SIGUSR2) {
				t.Fatalf("stop/restart flags for%v: stopped=%v restart=%v returned=%v", sig, owner.stoppedByWrapper.Load(), owner.restartFresh.Load(), restart)
			}
			if string(got) != "wrap-final-marker" || waitErr == nil || cmd.ProcessState == nil || !cmd.ProcessState.Sys().(syscall.WaitStatus).Signaled() {
				t.Fatalf("explicit stop drain=%q wait=%v state=%v", got, waitErr, cmd.ProcessState)
			}
			if err := cmd.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
				t.Fatalf("target not reaped after explicitstop: %v", err)
			}
			assertHeld()
			locks.Close()
			replacement, err := nativelock.Acquire(home, key, "replacement")
			if err != nil {
				t.Fatalf("lock survived owner cleanup: %v", err)
			}
			if err := replacement.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
