package wrapcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestFreshAgentInvocationDropsRestoreAndPreservesWrapperAndUserArgs(t *testing.T) {
	data := t.TempDir()
	request, err := freshAgentInvocation(
		"/pair/bin/pair",
		"/data/scroll.raw",
		[]string{"codex", "--sandbox", "danger-full-access", "resume", "old-session", "--no-alt-screen"},
		[]string{"PAIR_DATA_DIR=" + data, "PAIR_TAG=work", "PAIR_SESSION_ID=old-session"},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/pair/bin/pair", "wrap", "--scrollback-log", "/data/scroll.raw",
		"codex", "--sandbox", "danger-full-access", "--no-alt-screen",
	}
	if !reflect.DeepEqual(request.argv, want) {
		t.Fatalf("argv = %v, want %v", request.argv, want)
	}
	if got := envValue(request.env, "PAIR_SESSION_ID"); got != "" {
		t.Fatalf("PAIR_SESSION_ID = %q, want cleared", got)
	}
	if got := envValue(request.env, "PAIR_AGENT_ARGS"); got != "--sandbox danger-full-access --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", got)
	}
}

func TestSIGUSR2ReExecsWrapperWithoutReplacingPaneProcess(t *testing.T) {
	data := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", data)
	t.Setenv("PAIR_TAG", "restart-test")
	t.Setenv("PAIR_AGENT", "sh")

	oldExec := execProcess
	defer func() { execProcess = oldExec }()
	execArgv := make(chan []string, 1)
	execProcess = func(_ string, argv, _ []string) error {
		execArgv <- append([]string(nil), argv...)
		return errors.New("test stops exec")
	}

	done := make(chan int, 1)
	go func() {
		done <- Run([]string{"/bin/sh", "-c", "sleep 30"}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	}()

	pidPath := filepath.Join(data, "pair-wrap-pid-restart-test")
	deadline := time.Now().Add(3 * time.Second)
	for {
		raw, err := os.ReadFile(pidPath)
		if err == nil {
			if pid, parseErr := strconv.Atoi(string(raw)); parseErr == nil && pid > 0 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("wrapper pid file did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR2); err != nil {
		t.Fatal(err)
	}

	select {
	case argv := <-execArgv:
		if len(argv) < 4 || argv[1] != "wrap" || argv[len(argv)-3] != "/bin/sh" {
			t.Fatalf("replacement argv = %v", argv)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wrapper did not request fresh exec")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("wrapper did not finish old child")
	}
}

func TestFreshClaudeInvocationMintsNewBindingButPersistsCleanArgs(t *testing.T) {
	data := t.TempDir()
	request, err := freshAgentInvocation(
		"/pair/bin/pair", "",
		[]string{"claude", "--model", "opus", "--resume", "old-session"},
		[]string{"PAIR_DATA_DIR=" + data, "PAIR_TAG=work"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := request.argv[len(request.argv)-2]; got != "--session-id" {
		t.Fatalf("argv = %v, want fresh --session-id", request.argv)
	}
	if got := envValue(request.env, "PAIR_SESSION_ID"); got == "" || got == "old-session" {
		t.Fatalf("PAIR_SESSION_ID = %q, want fresh id", got)
	} else if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(got) {
		t.Fatalf("PAIR_SESSION_ID = %q, want UUID", got)
	}
}
