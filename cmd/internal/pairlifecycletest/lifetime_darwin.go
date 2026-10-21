//go:build darwin

package pairlifecycletest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

// The stand-in inherits the real target's slave descriptors. A socket and an
// inherited readiness pipe order leader exit after descendant startup without
// timing assumptions. Escaped descendants are released by the test, not wrap.
// WrapLifetimeChild is a subprocess stand-in for the real wrapped target.
func WrapLifetimeChild(t *testing.T, prefix, helperTest string) {
	role := os.Getenv(prefix + "_ROLE")
	if role == "" {
		t.Skip("subprocess helper")
	}
	conn, err := net.Dial("tcp", os.Getenv(prefix+"_ADDR"))
	if err != nil {
		os.Exit(90)
	}
	defer conn.Close()
	mode := os.Getenv(prefix + "_MODE")
	if role == "descendant" {
		// Same-group cleanup must escalate past TERM. The socket remains
		// blocked until the test releases an escaped daemon or wrap kills it.
		signal.Ignore(syscall.SIGHUP, syscall.SIGTERM)
		fmt.Fprintf(conn, "%d\n", os.Getpid())
		ready := os.NewFile(3, "descendant-ready")
		if _, err := ready.Write([]byte{1}); err != nil {
			os.Exit(91)
		}
		ready.Close()
		var release [1]byte
		_, _ = conn.Read(release[:])
		_, _ = conn.Write([]byte{'A'})
		os.Exit(0)
	}
	if mode != "normal" {
		reader, writer, err := os.Pipe()
		if err != nil {
			os.Exit(92)
		}
		child := exec.Command(os.Args[0], "-test.run=^"+helperTest+"$")
		child.Env = append(os.Environ(), prefix+"_ROLE=descendant")
		child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
		child.ExtraFiles = []*os.File{writer}
		if mode == "daemon" {
			child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		}
		if err := child.Start(); err != nil {
			os.Exit(93)
		}
		writer.Close()
		var ready [1]byte
		if _, err := io.ReadFull(reader, ready[:]); err != nil {
			os.Exit(94)
		}
		reader.Close()
	}
	fmt.Fprint(os.Stdout, "pair769-final-marker")
	os.Exit(7)
}

// AssertWrapLifetime checks final bytes/status, group cleanup and escaped-daemon ownership.
// prefix isolates the helper protocol from other native subprocess fixtures.
func AssertWrapLifetime(t *testing.T, cmd *exec.Cmd, mode, capture, prefix string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	listener.(*net.TCPListener).SetDeadline(time.Now().Add(15 * time.Second))
	cmd.Env = append(cmd.Env, prefix+"_ROLE=leader", prefix+"_ADDR="+listener.Addr().String(), prefix+"_MODE="+mode)
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	cmd.Stdin = slave
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var waitErr error
	go func() { waitErr = cmd.Wait(); close(done) }()
	t.Cleanup(func() { _ = cmd.Process.Kill(); <-done })
	// The leader connects first, before it creates the descendant.
	leader, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer leader.Close()
	var descendant net.Conn
	if mode != "normal" {
		descendant, err = listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer descendant.Close()
		descendant.SetReadDeadline(time.Now().Add(15 * time.Second))
		line, err := bufio.NewReader(descendant).ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || pid <= 0 {
			t.Fatalf("descendant pid %q: %v", line, err)
		}
		t.Cleanup(func() { _, _ = descendant.Write([]byte{1}); _ = descendant.Close() })
	}
	select {
	case <-done:
		err = waitErr
	case <-time.After(15 * time.Second):
		t.Fatal("wrapper did not finish after leader exit")
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 7 {
		t.Fatalf("exit=%v, output=%q", err, output.String())
	}
	if strings.Count(output.String(), "pair769-final-marker") != 1 {
		t.Fatalf("final output=%q", output.String())
	}
	raw, err := os.ReadFile(capture)
	if err != nil || string(raw) != "pair769-final-marker" {
		t.Fatalf("capture=%q err=%v", raw, err)
	}
	if descendant != nil {
		var b [1]byte
		if mode == "daemon" {
			// The escaped descendant still owns the slave, yet leader exit must finish
			// wrap. A synchronous reply to release proves it was not adopted/killed.
			if _, err := descendant.Write([]byte{1}); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(descendant, b[:]); err != nil || b[0] != 'A' {
				t.Fatalf("escaped daemon did not acknowledge release: %q %v", b, err)
			}
		}
		if _, err := descendant.Read(b[:]); err != io.EOF {
			t.Fatalf("descendant did not close: %v", err)
		}
	}
}
