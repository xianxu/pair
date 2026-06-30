package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
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
	rt := &fakeLegacyRuntime{
		executable: "/repo/bin/pair-go",
	}
	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if rt.execPath != "/repo/bin/pair" {
		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
	}
	wantArgv := []string{"pair", "--help"}
	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
	}
}

func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
	t.Setenv("PAIR_TEST_ENV", "kept")
	rt := &fakeLegacyRuntime{
		executable: "/repo/bin/pair-go",
		execCode:   42,
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)

	if code != 42 {
		t.Fatalf("code = %d, want 42", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if rt.execPath != "/repo/bin/pair" {
		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
	}
	wantArgv := []string{"pair", "claude", "--", "--resume"}
	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
	}
	if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
		t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
	}
}

func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable: "/repo/bin/pair-go",
		statErr:    os.ErrNotExist,
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
	if rt.execPath != "" {
		t.Fatalf("execPath = %q, want empty", rt.execPath)
	}
}

type fakeLegacyRuntime struct {
	executable string
	statErr    error
	execCode   int

	execPath string
	execArgv []string
	execEnv  []string
}

func (f *fakeLegacyRuntime) Executable() (string, error) {
	if f.executable == "" {
		return "", errors.New("missing executable")
	}
	return f.executable, nil
}

func (f *fakeLegacyRuntime) Stat(_ string) error {
	return f.statErr
}

func (f *fakeLegacyRuntime) Environ() []string {
	return os.Environ()
}

func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
	f.execPath = path
	f.execArgv = append([]string(nil), argv...)
	f.execEnv = append([]string(nil), env...)
	return f.execCode
}

func containsEnv(env []string, want string) bool {
	for _, got := range env {
		if got == want {
			return true
		}
	}
	return false
}
