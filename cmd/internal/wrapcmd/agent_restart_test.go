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
		[]string{"PAIR_DATA_DIR=" + data, "PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", "PAIR_SESSION_ID=old-session"},
		time.Date(2026, 8, 19, 9, 30, 0, 123, time.UTC),
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
	t.Setenv("PAIR_SCOPE_KEY", "scope")
	t.Setenv("PAIR_AGENT", "codex")
	t.Setenv("HOME", t.TempDir())
	fakeCodex := filepath.Join(t.TempDir(), "codex")
	if err := os.Symlink("/bin/sh", fakeCodex); err != nil {
		t.Fatal(err)
	}

	oldExec := execProcess
	oldStartWatcher := startWatcherProcess
	defer func() {
		execProcess = oldExec
		startWatcherProcess = oldStartWatcher
	}()
	execArgv := make(chan []string, 1)
	execProcess = func(_ string, argv, _ []string) error {
		execArgv <- append([]string(nil), argv...)
		return errors.New("test stops exec")
	}
	startWatcherProcess = func([]string, []string) error { return nil }

	done := make(chan int, 1)
	var stderr bytes.Buffer
	go func() {
		done <- Run([]string{fakeCodex, "-c", "sleep 30"}, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
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
		if len(argv) < 4 || argv[1] != "wrap" || argv[len(argv)-3] != fakeCodex {
			t.Fatalf("replacement argv = %v", argv)
		}
	case <-time.After(3 * time.Second):
		events, _ := os.ReadFile(filepath.Join(data, "wrap-events-restart-test.jsonl"))
		select {
		case code := <-done:
			t.Fatalf("wrapper exited %d before fresh exec: %s events=%s", code, stderr.String(), events)
		default:
			t.Fatalf("wrapper did not request fresh exec: %s events=%s", stderr.String(), events)
		}
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("wrapper did not finish old child")
	}
}

func TestFreshClaudeInvocationMintsInvocationIDButKeepsRecoveryProvisional(t *testing.T) {
	data := t.TempDir()
	request, err := freshAgentInvocation(
		"/pair/bin/pair", "",
		[]string{"claude", "--model", "opus", "--resume", "old-session"},
		[]string{"PAIR_DATA_DIR=" + data, "PAIR_TAG=work", "PAIR_SCOPE_KEY=scope"},
		time.Date(2026, 8, 19, 9, 30, 0, 123, time.UTC),
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
	if _, err := os.Stat(filepath.Join(data, "config-work-claude.json")); !os.IsNotExist(err) {
		t.Fatalf("fresh provisional launch wrote config: %v", err)
	}
}

func TestFreshAgentInvocationWatcherMatchesAsyncAgentRegistry(t *testing.T) {
	bound := time.Date(2026, 8, 19, 9, 31, 0, 456, time.UTC)
	for _, tc := range []struct {
		agent string
		watch bool
	}{
		{agent: "codex", watch: true},
		{agent: "agy", watch: true},
		{agent: "muse", watch: true},
		{agent: "claude", watch: true},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			request, err := freshAgentInvocation("/pair", "", []string{tc.agent, "--flag"}, []string{
				"PAIR_DATA_DIR=" + t.TempDir(), "PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", "HOME=/home/me",
			}, bound)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(request.watcherArgv) > 0; got != tc.watch {
				t.Fatalf("watcher present = %v, want %v: %v", got, tc.watch, request.watcherArgv)
			}
			if tc.watch && !containsArgPair(request.watcherArgv, "--pid-not-before", bound.Format(time.RFC3339Nano)) {
				t.Fatalf("watcher argv = %v, want generation bound", request.watcherArgv)
			}
		})
	}
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
