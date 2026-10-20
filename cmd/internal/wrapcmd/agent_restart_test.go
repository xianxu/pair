package wrapcmd

import (
	"bytes"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
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
	command, err := launcher.DecodeAgentCommand(envValue(request.env, launcher.AgentCommandEnv))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(command.Argv, " "); got != "--sandbox danger-full-access --no-alt-screen" {
		t.Fatalf("AgentCommand argv = %q", got)
	}
}

func TestSIGUSR2ReExecsWrapperWithoutReplacingPaneProcess(t *testing.T) {
	isolateNotificationSockets(t)
	data := filepath.Join(t.TempDir(), "repos", "scope")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_RETENTION_START_ID", "")
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
	startWatcherProcess = func(_ []string, _ []string, _ ...*os.File) (*ownedGroup, error) { return nil, nil }

	done := make(chan int, 1)
	var stderr bytes.Buffer
	go func() {
		done <- Run([]string{fakeCodex, "-c", "sleep 30"}, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
	}()

	pidPath := filepath.Join(data, "pair-wrap-pid-restart-test")
	deadline := time.Now().Add(testwait.Budget(t))
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
	case <-testwait.After(t):
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
	case <-testwait.After(t):
		t.Fatal("wrapper did not finish old child")
	}
}

func TestFreshPinAgentsMintSessionIDButKeepRecoveryProvisional(t *testing.T) {
	for _, agent := range []string{"claude", "qoder"} {
		t.Run(agent, func(t *testing.T) {
			data := t.TempDir()
			request, err := freshAgentInvocation(
				"/pair/bin/pair", "",
				[]string{agent, "--model", "opus", "--resume", "old-session"},
				[]string{"PAIR_DATA_DIR=" + data, "PAIR_TAG=work", "PAIR_SCOPE_KEY=scope"},
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
			target, err := sessioninventory.QueryResumeTarget(sessioninventory.NewOSRuntime("", data), "scope", "work", sessioninventory.Agent(agent))
			if err != nil || target.Status != sessioninventory.BindingProvisional || target.NativeID != "" || target.RequestedNativeID != envValue(request.env, "PAIR_SESSION_ID") {
				t.Fatalf("fresh chosen target = %+v, error = %v", target, err)
			}
			if _, err := os.Stat(filepath.Join(data, "config-work-"+agent+".json")); !os.IsNotExist(err) {
				t.Fatalf("fresh provisional launch wrote config: %v", err)
			}
		})
	}
}

// The fresh path spawns no watcher of its own (#329): the replacement wrap does,
// from the ordinal this request hands it. A watcher here too would be a second
// one for the same launch.
func TestFreshAgentInvocationHandsTheWatcherToTheReplacementWrap(t *testing.T) {
	bound := time.Date(2026, 8, 19, 9, 31, 0, 456, time.UTC)
	for _, agent := range launcher.AgentInventory() {
		t.Run(agent, func(t *testing.T) {
			request, err := freshAgentInvocation("/pair", "", []string{agent, "--flag"}, []string{
				"PAIR_DATA_DIR=" + t.TempDir(), "PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", "HOME=/home/me",
			})
			if err != nil {
				t.Fatal(err)
			}
			command, err := launcher.DecodeAgentCommand(envValue(request.env, launcher.AgentCommandEnv))
			if err != nil {
				t.Fatal(err)
			}
			ordinal := envValue(request.env, "PAIR_LAUNCH_ORDINAL")
			watcher := launchWatcherArgv("/pair", append([]string{agent}, command.Argv...), request.env, bound)
			if len(watcher) == 0 {
				t.Fatalf("replacement wrap would spawn no watcher: ordinal=%q", ordinal)
			}
			if !containsArgPair(watcher, "--launch-ordinal", ordinal) || !containsArgPair(watcher, "--pid-not-before", bound.Format(time.RFC3339Nano)) {
				t.Fatalf("watcher argv = %v, want ordinal %s and generation bound", watcher, ordinal)
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

// A launch prompt (#522) is said once: the wrapper keeps the argv without it,
// so an in-pane fresh start re-derives a command that carries no prompt, while
// the exec of this launch carries it exactly once.
func TestAnInPaneFreshStartDropsTheLaunchPrompt(t *testing.T) {
	const prompt = `Continue from the saved continuation at "/s/c.md": read it.`
	raw, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: "claude", Argv: []string{"--model", "opus"}, Prompt: prompt})
	if err != nil {
		t.Fatal(err)
	}
	argv, got, err := launchEnvCommand(raw)
	if err != nil || got != prompt || slices.Contains(argv, prompt) {
		t.Fatalf("kept argv %q, prompt %q, %v", argv, got, err)
	}
	if exec := launcher.PromptArgs(argv[0], argv[1:], got); strings.Count(strings.Join(exec, "\x00"), prompt) != 1 {
		t.Fatalf("exec argv %q", exec)
	}
	request, err := freshAgentInvocation("/pair", "", argv, []string{"PAIR_DATA_DIR=" + t.TempDir(), "PAIR_TAG=work", "PAIR_SCOPE_KEY=scope", launcher.AgentCommandEnv + "=" + raw})
	if err != nil {
		t.Fatal(err)
	}
	next, err := launcher.DecodeAgentCommand(envValue(request.env, launcher.AgentCommandEnv))
	if err != nil || next.Prompt != "" || slices.Contains(next.Argv, prompt) || slices.Contains(request.argv, prompt) {
		t.Fatalf("the fresh start carries the prompt: %+v / %q, %v", next, request.argv, err)
	}
}
