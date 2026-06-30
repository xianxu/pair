package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/dispatcher"
)

func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
		t.Fatalf("stdout missing usage:\n%s", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"wrap"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "wrap is planned but not implemented") {
		t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
	}
}

func TestRunLaunchHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"launch", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
		t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
	if code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
}

func TestRunLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWithLauncherRuntime([]string{"launch"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
	if code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
}

func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
	return dispatcher.LauncherRuntime{
		Env:      dispatcher.LauncherEnv(home, xdg, cwd),
		Sessions: dispatcher.StaticSessions{},
		History:  dispatcher.StaticHistory{},
	}
}
