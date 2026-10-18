package couchcore

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testgit"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

type observedReadCloser struct {
	io.Reader
	closed bool
}

func TestMain(m *testing.M) {
	if os.Getenv("PAIR_TEST_COUCH_TERMINAL_FIXTURE") == "1" && (filepath.Base(os.Args[0]) == "zellij" || filepath.Base(os.Args[0]) == "pair-wrap" || filepath.Base(os.Args[0]) == "claude") {
		os.Exit(runRegistrationTerminalFixture())
	}
	if os.Getenv("PAIR_TEST_COUCH_LAUNCH_NATIVE_SIDECAR") == "1" && len(os.Args) > 1 &&
		(os.Args[1] == "session-watch" || os.Args[1] == "title") {
		os.Exit(0)
	}
	if os.Getenv("PAIR_TEST_COUCH_PARK_HELPER") == "1" {
		os.Exit(runParkLifecycleHelper())
	}
	if os.Getenv("PAIR_TEST_RUNNER_HELPER") == "1" {
		os.Exit(LaunchHelperMain(os.Args[1:], os.Stderr))
	}
	defaultParallelForSubprocessTests()
	os.Exit(testgit.RunWithEmptyGitTemplate(func() int { return testpair.Run(m) }))
}

// defaultParallelForSubprocessTests raises -test.parallel unless the caller set
// it. couchcore's parallel tests mostly wait on git and helper subprocesses: at
// the GOMAXPROCS default the package kept about 2 of 12 cores busy, and 3x
// GOMAXPROCS took it from 236s to ~150s under a loaded host (pair#436).
func defaultParallelForSubprocessTests() {
	flag.Parse()
	set := false
	flag.Visit(func(f *flag.Flag) { set = set || f.Name == "test.parallel" })
	if !set {
		_ = flag.Set("test.parallel", strconv.Itoa(3*runtime.GOMAXPROCS(0)))
	}
}

func (r *observedReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestRunLaunchHelperExecsExactlyOnceAfterAcknowledgement(t *testing.T) {
	t.Parallel()

	ack := &observedReadCloser{Reader: bytes.NewReader([]byte{launchAckByte})}
	var calls int
	err := RunLaunchHelper(ack, time.Second, []string{"target", "arg"}, []string{"K=V"}, func(argv, env []string) error {
		calls++
		if !ack.closed {
			t.Fatal("ack descriptor remained open across target exec")
		}
		if strings.Join(argv, " ") != "target arg" || strings.Join(env, " ") != "K=V" {
			t.Fatalf("exec input = %v, %v", argv, env)
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("RunLaunchHelper = %v, calls=%d", err, calls)
	}
}

func TestRunLaunchHelperNeverExecsWithoutExactAcknowledgement(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		ack  io.Reader
	}{
		{name: "eof", ack: bytes.NewReader(nil)},
		{name: "wrong byte", ack: bytes.NewReader([]byte{'x'})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ack := &observedReadCloser{Reader: tt.ack}
			calls := 0
			err := RunLaunchHelper(ack, time.Second, []string{"target"}, nil, func([]string, []string) error {
				calls++
				return nil
			})
			if err == nil || calls != 0 || !ack.closed {
				t.Fatalf("RunLaunchHelper = %v, calls=%d, closed=%v", err, calls, ack.closed)
			}
		})
	}
}

func TestRunLaunchHelperTimeoutClosesAcknowledgementWithoutExec(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	calls := 0
	err := RunLaunchHelper(reader, 20*time.Millisecond, []string{"target"}, nil, func([]string, []string) error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") || calls != 0 {
		t.Fatalf("RunLaunchHelper = %v, calls=%d", err, calls)
	}
}

func TestLaunchHelperSubprocess(t *testing.T) {
	t.Parallel()

	mode := os.Getenv("PAIR_TEST_LAUNCH_HELPER_MODE")
	if mode == "helper" {
		ack := os.NewFile(3, "launch-ack")
		env := testpair.Environment(t, os.Getenv("HOME"), "PAIR_TEST_LAUNCH_HELPER_MODE=target", "PAIR_TEST_LAUNCH_HELPER_MARKER="+os.Getenv("PAIR_TEST_LAUNCH_HELPER_MARKER"))
		err := RunLaunchHelper(ack, time.Second, []string{os.Args[0], "-test.run=^TestLaunchHelperSubprocess$"}, env, execLaunchTarget)
		if err != nil {
			t.Fatalf("helper: %v", err)
		}
		return
	}
	if mode == "target" {
		if err := os.WriteFile(os.Getenv("PAIR_TEST_LAUNCH_HELPER_MARKER"), []byte("exec\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}

	for _, acknowledge := range []bool{false, true} {
		name := "eof"
		if acknowledge {
			name = "ack"
		}
		t.Run(name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "target-ran")
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestLaunchHelperSubprocess$")
			cmd.Env = testpair.Environment(t, filepath.Dir(marker), "PAIR_TEST_LAUNCH_HELPER_MODE=helper", "PAIR_TEST_LAUNCH_HELPER_MARKER="+marker)
			cmd.ExtraFiles = []*os.File{reader}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			_ = reader.Close()

			time.Sleep(75 * time.Millisecond)
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("target ran before acknowledgement: %v", err)
			}
			if acknowledge {
				if _, err := writer.Write([]byte{launchAckByte}); err != nil {
					t.Fatal(err)
				}
			}
			_ = writer.Close()
			err = cmd.Wait()
			if acknowledge {
				if err != nil {
					t.Fatalf("acknowledged helper: %v", err)
				}
				if raw, err := os.ReadFile(marker); err != nil || string(raw) != "exec\n" {
					t.Fatalf("target marker = %q, %v", raw, err)
				}
			} else {
				if err == nil {
					t.Fatal("EOF helper exited successfully")
				}
				if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("EOF helper ran target: %v", err)
				}
			}
		})
	}
}
