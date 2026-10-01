package couchmessage

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSocketCrashHelper(t *testing.T) {
	dir := os.Getenv("PAIR_MESSAGE_CRASH_TEST_DIR")
	if dir == "" {
		return
	}
	binding := protocolBinding("pair:0")
	binding.PID = os.Getpid()
	socket, err := EndpointSocket("/test", binding)
	if err != nil {
		t.Fatal(err)
	}
	socket = filepath.Join(dir, filepath.Base(socket))
	server, err := StartServer(context.Background(), socket, socketCleanupEcho)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	fmt.Println(socket)
	for {
		time.Sleep(time.Hour)
	}
}

func socketCleanupEcho(_ context.Context, raw []byte) ([]byte, error) { return raw, nil }

func TestSocketStartupCollectsCrashedOwnerPreservesLive(t *testing.T) {
	dir := filepath.Dir(transportSocket(t))
	cmd := exec.Command(os.Args[0], "-test.run=^TestSocketCrashHelper$")
	cmd.Env = append(os.Environ(), "PAIR_MESSAGE_CRASH_TEST_DIR="+dir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			ready <- scanner.Text()
		} else {
			ready <- ""
		}
	}()
	var crashed string
	select {
	case crashed = <-ready:
		if crashed == "" {
			t.Fatal("helper did not publish socket")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("helper readiness timeout")
	}
	binding := protocolBinding("pair:1")
	binding.PID = os.Getpid()
	livePath, err := EndpointSocket("/test", binding)
	if err != nil {
		t.Fatal(err)
	}
	livePath = filepath.Join(dir, filepath.Base(livePath))
	live, err := StartServer(context.Background(), livePath, socketCleanupEcho)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	// Startup must not remove either owner's socket while both are alive.
	var reply any
	if err := Call(context.Background(), crashed, struct{}{}, &reply); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("helper did not crash")
	}
	if _, err := os.Lstat(crashed); err != nil {
		t.Fatalf("no crash residue: %v", err)
	}
	replacement, err := StartServer(context.Background(), filepath.Join(dir, "replacement.sock"), socketCleanupEcho)
	if err != nil {
		t.Fatal(err)
	}
	defer replacement.Close()
	if _, err := os.Lstat(crashed); !os.IsNotExist(err) {
		t.Fatalf("crashed owner's socket retained: %v", err)
	}
	if err := Call(context.Background(), livePath, struct{}{}, &reply); err != nil {
		t.Fatalf("live socket removed: %v", err)
	}
}

func TestSocketCleanupPreservesUnknownOwnerAndReplacement(t *testing.T) {
	for _, outcome := range []string{"live", "permission-denied", "unknown", "replacement"} {
		t.Run(outcome, func(t *testing.T) {
			dir := filepath.Dir(transportSocket(t))
			binding := protocolBinding("pair:0")
			path, err := EndpointSocket("/test", binding)
			if err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(dir, filepath.Base(path))
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			listener.SetUnlinkOnClose(false)
			defer listener.Close()
			var replacement *net.UnixListener
			calls := 0
			err = collectDeadEndpointSockets(dir, func(pid int) error {
				calls++
				if pid != binding.PID {
					t.Fatalf("probed PID %d, want %d", pid, binding.PID)
				}
				switch outcome {
				case "live":
					return nil
				case "permission-denied":
					return syscall.EPERM
				case "unknown":
					return syscall.EIO
				default:
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					replacement, err = net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { replacement.Close() })
					return syscall.ESRCH
				}
			})
			if err != nil || calls != 1 {
				t.Fatalf("cleanup err=%v probes=%d", err, calls)
			}
			conn, err := net.Dial("unix", path)
			if err != nil {
				t.Fatalf("removed %s socket: %v", outcome, err)
			}
			conn.Close()
		})
	}
}

func TestSocketCleanupIgnoresUnownedArtifacts(t *testing.T) {
	dir := filepath.Dir(transportSocket(t))
	binding := protocolBinding("pair:0")
	path, err := EndpointSocket("/test", binding)
	if err != nil {
		t.Fatal(err)
	}
	// Neither an owner-shaped regular file nor an opaque broker socket proves
	// authority to delete. Malformed PID/hash names must also remain untouched.
	names := []string{filepath.Base(path), "wrapper-0-0000000000000000000000000000000000000000.sock", "wrapper-42-invalid.sock", "broker.sock"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	err = collectDeadEndpointSockets(dir, func(int) error { t.Error("probed unowned artifact"); return syscall.ESRCH })
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if _, err := os.Lstat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("removed %s: %v", name, err)
		}
	}
}
