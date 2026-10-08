package broadcast

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// TestHelperGuardProcess is not a test: re-executed by the guard tests, it
// runs RunGuard as its own process, the way `couch __broadcast-guard` does.
func TestHelperGuardProcess(t *testing.T) {
	if os.Getenv("BROADCAST_GUARD_HELPER") != "1" {
		t.Skip("helper process")
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	os.Exit(RunGuard(args, os.Stdin, os.Stderr))
}

type guardRun struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	child  int
	done   chan struct{} // closed once the guard exited; code holds its exit code
	code   int
	stderr *bufio.Reader
}

// startGuard runs the guard over a shell script, as Couch runs it over
// cloudflared: in its own process group, holding a stdin pipe.
func startGuard(t *testing.T, guardArgs []string, script string) *guardRun {
	t.Helper()
	args := append([]string{"-test.run=^TestHelperGuardProcess$", "--"}, guardArgs...)
	args = append(args, "--", "/bin/sh", "-c", script)
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "BROADCAST_GUARD_HELPER=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	g := &guardRun{cmd: cmd, stdin: stdin, done: make(chan struct{}), stderr: bufio.NewReader(stderr)}
	t.Cleanup(func() {
		stdin.Close()
		select {
		case <-g.done:
		case <-time.After(10 * time.Second):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	})
	line, err := g.stderr.ReadString('\n')
	if err != nil {
		t.Fatalf("guard said nothing: %v", err)
	}
	pid, ok := ParseGuardChild(line)
	if !ok {
		t.Fatalf("guard's first line %q", line)
	}
	g.child = pid
	go func() {
		_, _ = io.Copy(io.Discard, g.stderr)
		state, _ := cmd.Process.Wait()
		g.code = state.ExitCode()
		close(g.done)
	}()
	return g
}

func waitFile(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(b))) > 0 {
			return strings.TrimSpace(string(b))
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never written", path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func waitDead(t *testing.T, pid string, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for procutil.Alive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("pid %s still alive after %v", pid, within)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestGuardStopsGroupWhenOwnerGoes(t *testing.T) {
	dir := t.TempDir()
	private := filepath.Join(dir, "private")
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	grandchildFile := filepath.Join(dir, "grandchild")
	g := startGuard(t, []string{"--remove", private}, "sleep 300 & echo $! > "+grandchildFile+"; wait")
	grandchild := waitFile(t, grandchildFile)
	// The owner going away, by exit, crash or SIGKILL alike, closes the pipe.
	g.stdin.Close()
	select {
	case <-g.done:
	case <-time.After(5 * time.Second):
		t.Fatal("guard did not stop")
	}
	waitDead(t, strconv.Itoa(g.child), 2*time.Second)
	waitDead(t, grandchild, 2*time.Second)
	if _, err := os.Stat(private); !os.IsNotExist(err) {
		t.Fatalf("private directory survived: %v", err)
	}
}

func TestGuardEscalatesToKill(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "ready")
	g := startGuard(t, []string{"--grace", "200ms"}, "trap '' TERM; echo ready > "+marker+"; while :; do sleep 1; done")
	waitFile(t, marker)
	start := time.Now()
	g.stdin.Close()
	select {
	case <-g.done:
	case <-time.After(5 * time.Second):
		t.Fatal("guard did not escalate past an ignored TERM")
	}
	if d := time.Since(start); d < 150*time.Millisecond {
		t.Fatalf("killed after %v, before the grace period", d)
	}
	waitDead(t, strconv.Itoa(g.child), 2*time.Second)
}

func TestGuardExitsWithChild(t *testing.T) {
	dir := t.TempDir()
	private := filepath.Join(dir, "private")
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	g := startGuard(t, []string{"--remove", private}, "exit 7")
	select {
	case <-g.done:
		if g.code != 7 {
			t.Fatalf("guard exit %d, want the child's 7", g.code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("guard outlived its child")
	}
	// The owner is alive and still serving on that directory; it cleans up.
	if _, err := os.Stat(private); err != nil {
		t.Fatalf("guard removed the directory while its owner lives: %v", err)
	}
}

func TestGuardRejectsBadArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"--"}, {"--remove"}, {"--grace", "soon", "--", "true"}, {"true"}} {
		if code := RunGuard(args, strings.NewReader(""), io.Discard); code != 2 {
			t.Errorf("RunGuard(%q) = %d, want 2", args, code)
		}
	}
}
