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
		roots:      map[string]bool{"/repo": true},
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
	if rt.execPath != "/repo/bin/pair-shell" {
		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
	}
	if rt.execLabel != "pair-go launch" {
		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
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
		roots:      map[string]bool{"/repo": true},
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
	if rt.execPath != "/repo/bin/pair-shell" {
		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
	}
	if rt.execLabel != "pair-go launch" {
		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
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
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"pair-go launch", "pair-shell", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
	if rt.execPath != "" {
		t.Fatalf("execPath = %q, want empty", rt.execPath)
	}
}

func TestRunDirectPairExecsLegacyShellWithAllArgs(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable: "/repo/bin/pair",
		roots:      map[string]bool{"/repo": true},
		execCode:   7,
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"claude", "--", "--resume"}, &stdout, &stderr, rt)

	if code != 7 {
		t.Fatalf("code = %d, want 7", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if rt.execPath != "/repo/bin/pair-shell" {
		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
	}
	if rt.execLabel != "pair" {
		t.Fatalf("execLabel = %q, want pair", rt.execLabel)
	}
	wantArgv := []string{"pair", "claude", "--", "--resume"}
	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
	}
}

func TestRunDirectPairFallsBackToDefaultPairHome(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable:      "/home/me/.local/bin/pair",
		defaultPairHome: "/repo",
		roots:           map[string]bool{"/repo": true},
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if rt.execPath != "/repo/bin/pair-shell" {
		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
	}
}

func TestRunPairGoHelperDoesNotProbeOrExecShellLauncher(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable: "/repo/bin/pair-go",
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"help"}, &stdout, &stderr, rt)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if rt.statCalls != 0 {
		t.Fatalf("statCalls = %d, want 0", rt.statCalls)
	}
	if rt.execPath != "" {
		t.Fatalf("execPath = %q, want empty", rt.execPath)
	}
	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
		t.Fatalf("stdout missing usage:\n%s", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

type fakeLegacyRuntime struct {
	executable      string
	pairHome        string
	defaultPairHome string
	roots           map[string]bool
	statErr         error
	execCode        int
	statCalls       int

	execPath  string
	execLabel string
	execArgv  []string
	execEnv   []string
}

func (f *fakeLegacyRuntime) Executable() (string, error) {
	if f.executable == "" {
		return "", errors.New("missing executable")
	}
	return f.executable, nil
}

func (f *fakeLegacyRuntime) PairHome() string {
	return f.pairHome
}

func (f *fakeLegacyRuntime) DefaultPairHome() string {
	return f.defaultPairHome
}

func (f *fakeLegacyRuntime) Stat(path string) error {
	f.statCalls++
	if f.statErr != nil {
		return f.statErr
	}
	if strings.HasSuffix(path, "/bin/pair-shell") && f.roots != nil {
		root := strings.TrimSuffix(path, "/bin/pair-shell")
		if f.roots[root] {
			return nil
		}
	}
	return os.ErrNotExist
}

func (f *fakeLegacyRuntime) Environ() []string {
	return os.Environ()
}

func (f *fakeLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
	f.execLabel = label
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
