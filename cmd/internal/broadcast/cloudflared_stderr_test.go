package broadcast

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// A small failure fits in the real pipe even when the reader has not run.
// Waiting first forces the ordering that used to close StderrPipe before the
// scanner could preserve the child's diagnostic. No scheduler race is needed.
func TestCloudflaredStderrSurvivesWaitBeforeDrain(t *testing.T) {
	const failure = "ERR failed to request quick Tunnel: Post https://api.trycloudflare.com/tunnel"
	bin, _ := fakeCloudflared(t, "echo '"+failure+"' >&2; exit 1")
	cmd := exec.Command(bin)
	stderr, writer, err := cloudflaredStderrPipe(cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close() })
	if writer != nil {
		t.Cleanup(func() { _ = writer.Close() })
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("fake failing process succeeded")
	}
	h := &cloudflaredHandle{}
	h.watchStderr(stderr, false, func(string) {})
	if got := h.lastLine(); got != failure {
		t.Fatalf("lost diagnostic after process wait: got %q, want %q", got, failure)
	}
}

func TestCloudflaredForcedCloseReleasesHeldStderr(t *testing.T) {
	for _, processExited := range []bool{false, true} {
		t.Run(map[bool]string{false: "running", true: "reaped"}[processExited], func(t *testing.T) {
			body := "echo ready >&2; exec sleep 300"
			if processExited {
				body = "echo ready >&2; exit 1"
			}
			bin, _ := fakeCloudflared(t, body)
			cmd := exec.Command(bin)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			stderr, writer, err := cloudflaredStderrPipe(cmd)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = stderr.Close(); _ = writer.Close() })
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			h := &cloudflaredHandle{cmd: cmd, stdin: stdin, stderr: stderr, guarded: true, exited: make(chan struct{})}
			ready, drained, waited := make(chan struct{}), make(chan struct{}), make(chan struct{})
			go func() {
				defer close(drained)
				defer stderr.Close()
				h.watchStderr(stderr, false, func(line string) {
					if line == "ready" {
						close(ready)
					}
				})
			}()
			go h.waitForExit(func() { close(waited); <-drained })
			t.Cleanup(func() {
				_ = writer.Close()
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				<-h.exited
			})
			awaitCloudflaredEvent(t, ready)
			if processExited {
				awaitCloudflaredEvent(t, waited)
			}
			// Deliberately retain the parent's writer, modeling an inherited FD.
			// EOF cannot complete the drain; the existing force deadline must
			// kill the running owned group and interrupt the reader before join.
			deadline := make(chan time.Time)
			close(deadline)
			closed := make(chan struct{})
			go func() { _ = h.closeWithDeadline(deadline); close(closed) }()
			awaitCloudflaredEvent(t, closed)
			if cmd.ProcessState == nil || (!processExited && cmd.ProcessState.Success()) {
				t.Fatal("owned process was not reaped after forced close")
			}
			if h.lastLine() != "ready" {
				t.Fatalf("lost completed line: %q", h.lastLine())
			}
			if _, err := writer.Write([]byte("still open")); err == nil {
				t.Fatal("forced close left the reader open")
			}
		})
	}
}

func TestCloudflaredCloseKillsDescendantAfterLeaderWait(t *testing.T) {
	for _, tc := range []struct {
		name           string
		guarded, force bool
	}{
		{"direct-term", false, false},
		{"direct-kill", false, true},
		{"guarded-kill", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The descendant retains stderr after its leader exits. Forced
			// cases ignore TERM even after exec, requiring group KILL.
			trap := ""
			if tc.force {
				trap = `trap "" TERM; `
			}
			bin, _ := fakeCloudflared(t, "/bin/sh -c '"+trap+"echo descendant $$ >&2; exec sleep 300' &\nexit 1")
			cmd := exec.Command(bin)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			stderr, writer, err := cloudflaredStderrPipe(cmd)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = stderr.Close(); _ = writer.Close() })
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			_ = writer.Close()
			h := &cloudflaredHandle{cmd: cmd, stdin: stdin, stderr: stderr, guarded: tc.guarded, exited: make(chan struct{})}
			ready, drained, waited := make(chan struct{}), make(chan struct{}), make(chan struct{})
			child := ""
			go func() {
				defer close(drained)
				defer stderr.Close()
				h.watchStderr(stderr, false, func(line string) {
					if strings.HasPrefix(line, "descendant ") {
						child = strings.TrimPrefix(line, "descendant ")
						close(ready)
					}
				})
			}()
			go h.waitForExit(func() { close(waited); <-drained })
			t.Cleanup(func() {
				// Leader reaping never releases group ownership.
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				_ = stderr.Close()
				<-h.exited
			})
			awaitCloudflaredEvent(t, ready)
			awaitCloudflaredEvent(t, waited)
			pid, err := strconv.Atoi(child)
			if err != nil || pid <= 0 || !procutil.Alive(child) {
				t.Fatalf("descendant is not live: %q %v", child, err)
			}
			if pgid, err := syscall.Getpgid(pid); err != nil || pgid != cmd.Process.Pid {
				t.Fatalf("descendant group=%d err=%v, want owned group %d", pgid, err, cmd.Process.Pid)
			}
			deadline := make(chan time.Time)
			if tc.force {
				close(deadline)
			}
			closed := make(chan struct{})
			go func() { _ = h.closeWithDeadline(deadline); close(closed) }()
			awaitCloudflaredEvent(t, closed)
			waitDead(t, child)
		})
	}
}

// This deadline only reports a stuck lifecycle; no elapsed time makes a test
// succeed. Ordering is established by process, scanner and completion events.
func awaitCloudflaredEvent(t *testing.T, event <-chan struct{}) {
	t.Helper()
	select {
	case <-event:
	case <-testwait.After(t):
		t.Fatal("cloudflared lifecycle did not complete")
	}
}

func TestCloudflaredFastFailureKeepsStderr(t *testing.T) {
	for _, guarded := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "guarded"}[guarded], func(t *testing.T) {
			const failure = "ERR exact cloudflared startup failure"
			bin, _ := fakeCloudflared(t, "echo '"+failure+"' >&2; exit 1")
			c := Cloudflared{Binary: bin, RunDir: shortDir(t)}
			if guarded {
				c.Guard = guardArgv(t)
			}
			l, err := c.Listen()
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			h, err := c.Open(context.Background(), l)
			if h != nil {
				_ = h.Close()
				t.Fatal("failing child opened a tunnel")
			}
			if err == nil || !strings.HasSuffix(err.Error(), ": "+failure) {
				t.Fatalf("lost exact failure: %v", err)
			}
		})
	}
}

func TestCloudflaredExitPublishesAfterStderrDrain(t *testing.T) {
	const failure = "ERR final child failure"
	bin, _ := fakeCloudflared(t, "echo '"+failure+"' >&2; exit 1")
	cmd := exec.Command(bin)
	stderr, writer, err := cloudflaredStderrPipe(cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close(); _ = writer.Close() })
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	h := &cloudflaredHandle{cmd: cmd, exited: make(chan struct{})}
	// The lifecycle owner's real wait runs first. Its drain effect performs the
	// real scan synchronously here, forcing the delayed-reader ordering without
	// a scheduling delay or an assertion on elapsed time.
	drained := false
	h.waitForExit(func() {
		select {
		case <-h.exited:
			t.Fatal("published exit before stderr drain")
		default:
		}
		h.watchStderr(stderr, false, func(string) {})
		drained = true
	})
	if !drained || h.lastLine() != failure {
		t.Fatalf("exit lost pending stderr: drained=%t last=%q", drained, h.lastLine())
	}
	select {
	case <-h.exited:
	default:
		t.Fatal("complete process and drain did not publish exit")
	}
}
