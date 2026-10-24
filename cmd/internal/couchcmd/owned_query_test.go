package couchcmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// The descendant owns a liveness connection independent of stdout/stderr.
// Parent completion cannot close it; only stopping the descendant produces EOF.
func TestCouchOwnedQueryProcessHelper(t *testing.T) {
	mode := os.Getenv("PAIR_COUCH_QUERY_TEST_MODE")
	if mode == "" {
		return
	}
	connect := func(role string) net.Conn {
		conn, err := net.Dial("unix", os.Getenv("PAIR_COUCH_QUERY_TEST_SOCKET"))
		if err != nil {
			os.Exit(81)
		}
		group, err := syscall.Getpgid(0)
		if err != nil {
			os.Exit(82)
		}
		fmt.Fprintf(conn, "%s %d %d\n", role, os.Getpid(), group)
		return conn
	}
	if strings.HasPrefix(mode, "child-") {
		if mode == "child-redirected" {
			os.Stdout.Close()
			os.Stderr.Close()
		}
		conn := connect("child")
		var token [1]byte
		_, _ = conn.Read(token[:])
		conn.Close()
		os.Exit(0)
	}
	childMode := "child-redirected"
	if mode == "pipe-holder" {
		childMode = "child-held"
	}
	child := exec.Command(os.Args[0], "-test.run=^TestCouchOwnedQueryProcessHelper$")
	child.Env = append(os.Environ(), "PAIR_COUCH_QUERY_TEST_MODE="+childMode)
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		os.Exit(83)
	}
	conn := connect("parent")
	var token [1]byte
	_, _ = conn.Read(token[:])
	conn.Close()
	fmt.Fprint(os.Stdout, os.Getenv("PAIR_COUCH_QUERY_TEST_REPLY"))
	if mode == "failure" {
		fmt.Fprint(os.Stderr, os.Getenv("PAIR_COUCH_QUERY_TEST_DIAGNOSTIC"))
		code, err := strconv.Atoi(os.Getenv("PAIR_COUCH_QUERY_TEST_EXIT"))
		if err != nil {
			os.Exit(84)
		}
		os.Exit(code)
	}
	os.Exit(0)
}

type queryFixturePeer struct {
	role       string
	pid, group int
	conn       net.Conn
}

func TestCouchQueriesCollectOwnedDescendants(t *testing.T) {
	failure := zellijtest.Case(t, "couch.owned.failure")
	for _, entry := range []struct {
		name, reply string
		query       func(context.Context, string) ([]byte, error)
	}{
		{"clients", zellijtest.Case(t, "couch.owned.clients").Stdout, queryShortcutClients},
	} {
		for _, mode := range []string{"success", "failure", "cancel", "pipe-holder"} {
			t.Run(entry.name+"/"+mode, func(t *testing.T) {
				root := testpair.ShortRoot(t)
				listener, err := net.Listen("unix", filepath.Join(root, "gate"))
				if err != nil {
					t.Fatal(err)
				}
				binary, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
				shim := filepath.Join(root, "zellij")
				if err := os.WriteFile(shim, []byte("#!/bin/sh\nexec "+quote(binary)+" -test.run=^TestCouchOwnedQueryProcessHelper$ -- \"$@\"\n"), 0700); err != nil {
					t.Fatal(err)
				}
				testpair.ApplyEnvironment(t, t.TempDir(), "PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"), "PAIR_COUCH_QUERY_TEST_MODE="+mode, "PAIR_COUCH_QUERY_TEST_SOCKET="+filepath.Join(root, "gate"), "PAIR_COUCH_QUERY_TEST_REPLY="+entry.reply, "PAIR_COUCH_QUERY_TEST_DIAGNOSTIC="+failure.Stderr, "PAIR_COUCH_QUERY_TEST_EXIT="+strconv.Itoa(failure.Code))
				ctx, cancel := context.WithCancel(context.Background())
				type answer struct {
					out []byte
					err error
				}
				done := make(chan answer, 1)
				go func() { out, err := entry.query(ctx, "private-session"); done <- answer{out, err} }()
				peers := make(chan queryFixturePeer, 2)
				acceptDone := make(chan error, 1)
				go func() {
					defer close(peers)
					for i := 0; i < 2; i++ {
						conn, err := listener.Accept()
						if err != nil {
							acceptDone <- err
							return
						}
						p := queryFixturePeer{conn: conn}
						_, err = fmt.Fscanf(bufio.NewReader(conn), "%s %d %d\n", &p.role, &p.pid, &p.group)
						if err != nil {
							conn.Close()
							acceptDone <- err
							return
						}
						peers <- p
					}
					acceptDone <- nil
				}()
				var child, parent queryFixturePeer
				childIdentity := ""
				joined := false
				t.Cleanup(func() {
					cancel()
					listener.Close()
					if child.pid > 0 && childIdentity != "" && procutil.Identity(strconv.Itoa(child.pid)) == childIdentity {
						_ = syscall.Kill(child.pid, syscall.SIGKILL)
					}
					if child.conn != nil {
						child.conn.Close()
					}
					if parent.conn != nil {
						parent.conn.Close()
					}
					if !joined {
						select {
						case <-done:
						case <-testwait.After(t):
							t.Error("query did not join during cleanup")
						}
					}
					select {
					case <-acceptDone:
					case <-testwait.After(t):
						t.Error("fixture listener did not join")
					}
				})
				for i := 0; i < 2; i++ {
					select {
					case p, ok := <-peers:
						if !ok {
							t.Fatal("fixture listener ended before readiness")
						}
						if p.role == "child" {
							child = p
							childIdentity = procutil.Identity(strconv.Itoa(p.pid))
						} else {
							parent = p
						}
					case <-testwait.After(t):
						t.Fatal("query helper did not reach readiness")
					}
				}
				if mode == "cancel" {
					cancel()
				} else if _, err := parent.conn.Write([]byte{1}); err != nil {
					t.Fatal(err)
				}
				var result answer
				select {
				case result = <-done:
					joined = true
				case <-testwait.After(t):
					t.Fatal("production query did not return")
				}
				if mode == "success" {
					if result.err != nil || string(result.out) != entry.reply {
						t.Errorf("stdout=%q error=%v", result.out, result.err)
					}
				}
				if mode == "failure" {
					var exit *exec.ExitError
					if !errors.As(result.err, &exit) || exit.ExitCode() != 7 || string(exit.Stderr) != "owned query diagnostic" || string(result.out) != entry.reply {
						t.Errorf("lost failure output/status: stdout=%q error=%v exit=%+v", result.out, result.err, exit)
					}
				}
				if mode == "cancel" && result.err == nil {
					t.Error("canceled query succeeded")
				}
				if mode == "pipe-holder" && !errors.Is(result.err, exec.ErrWaitDelay) {
					t.Errorf("pipe-drain error=%v", result.err)
				}
				stopped := make(chan error, 1)
				go func() { var b [1]byte; _, err := child.conn.Read(b[:]); stopped <- err }()
				select {
				case err := <-stopped:
					if !errors.Is(err, io.EOF) {
						t.Errorf("descendant liveness=%v", err)
					}
				case <-time.After(time.Second):
					child.conn.Close()
					<-stopped
					t.Fatal("owned redirected descendant survived production query return")
				}
				if parent.pid <= 0 || parent.group != parent.pid || child.group != parent.group {
					t.Errorf("query lacks one private process group: parent=%+v child=%+v", parent, child)
				}
			})
		}
	}
}
