package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestTerminalLeaseSubprocess(t *testing.T) {
	if os.Getenv("PAIR_TEST_TERMINAL_LEASE_HELPER") != "1" {
		return
	}
	path := os.Getenv("PAIR_TEST_TERMINAL_LEASE_PATH")
	ctx := context.Background()
	if os.Getenv("PAIR_TEST_TERMINAL_LEASE_TIMEOUT") == "1" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()
		l, err := AcquireTerminalLease(ctx, path)
		if l != nil {
			l.Close()
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("wanted exclusion until deadline, got %v", err)
		}
		return
	}
	lease, err := AcquireTerminalLease(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	fmt.Println("acquired")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
func leaseHelper(t *testing.T, path string, deadline bool) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminalLeaseSubprocess$")
	cmd.Env = append(os.Environ(), "PAIR_TEST_TERMINAL_LEASE_HELPER=1", "PAIR_TEST_TERMINAL_LEASE_PATH="+path)
	if deadline {
		cmd.Env = append(cmd.Env, "PAIR_TEST_TERMINAL_LEASE_TIMEOUT=1")
	}
	return cmd
}
func TestTerminalLeaseExclusionAndStableInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locks", "one.lock")
	lease, err := AcquireTerminalLease(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if output, err := leaseHelper(t, path, true).CombinedOutput(); err != nil {
		t.Fatalf("subprocess exclusion: %s %v", output, err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := AcquireTerminalLease(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("lock inode replaced: %v", err)
	}
}
func TestTerminalLeaseOwnerDeathReleases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lease.lock")
	cmd := leaseHelper(t, path, false)
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || line != "acquired\n" {
		t.Fatalf("helper: %q %v", line, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	lease, err := AcquireTerminalLease(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
}
func TestTerminalLeaseRejectsNonRegularFilesAndCancellation(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "symlink")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, symlink, fifo, "relative"} {
		if l, err := AcquireTerminalLease(context.Background(), path); err == nil {
			l.Close()
			t.Fatalf("accepted %s", path)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := AcquireTerminalLease(ctx, filepath.Join(dir, "cancelled")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
func TestTerminalLeaseCanonicalAliasesAndStripeCollisions(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "sockets")
	if err := os.Mkdir(realDir, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(realDir, alias); err != nil {
		t.Fatal(err)
	}
	cache, err := artifactpath.ResolvePairCache(root)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{}
	var collision string
	for i := 0; i <= 256; i++ {
		socket, err := canonicalSocket(filepath.Join(realDir, fmt.Sprintf("session-%d", i)))
		if err != nil {
			t.Fatal(err)
		}
		aliasSocket, err := canonicalSocket(filepath.Join(alias, filepath.Base(socket)))
		if err != nil {
			t.Fatal(err)
		}
		path, err := cache.TerminalLease(socket)
		if err != nil {
			t.Fatal(err)
		}
		aliasPath, err := cache.TerminalLease(aliasSocket)
		if err != nil || path != aliasPath {
			t.Fatalf("alias split lock: %s %s %v", path, aliasPath, err)
		}
		if previous, ok := paths[path]; ok {
			if previous == socket {
				t.Fatal("collision must be between distinct sockets")
			}
			collision = path
			break
		}
		paths[path] = socket
	}
	if collision == "" {
		t.Fatal("257 distinct sockets must share a stripe")
	}
	lease, err := AcquireTerminalLease(context.Background(), collision)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if output, err := leaseHelper(t, collision, true).CombinedOutput(); err != nil {
		t.Fatalf("stripe collision did not serialize: %s %v", output, err)
	}
}
