package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRunStreamingSubcommandRoutesChangelogToInjectedStderr(t *testing.T) {
	// changelog with no flags → usage error to the *injected* stderr (proves the
	// seam passes real stderr through, unlike the buffered Dispatch path).
	var stdout, stderr bytes.Buffer
	code := runStreamingSubcommand([]string{"changelog"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1 (usage error)", code)
	}
	if !strings.Contains(stderr.String(), "pair-changelog: usage:") {
		t.Fatalf("stderr missing changelog usage (seam not wired to injected stderr):\n%s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("changelog writes no stdout; got %q", stdout.String())
	}
}

func TestRunStreamingSubcommandRoutesContinuationStdin(t *testing.T) {
	// The body arrives on stdin (--body-file -). It lacks a '## NEXT ACTION'
	// section, so the writer rejects it — which proves the seam passed the real
	// stdin through to the runner (the buffered Dispatch path has no stdin).
	root := t.TempDir()
	var out, errb bytes.Buffer
	code := runStreamingSubcommand(
		[]string{"continuation", "--repo-root", root, "--slug", "s", "--agent", "claude", "--issues", "1", "--body-file", "-"},
		strings.NewReader("just a body, no next action\n"), &out, &errb)
	if code != 1 {
		t.Fatalf("code = %d, want 1 (stdin body missing NEXT ACTION)", code)
	}
	if !strings.Contains(errb.String(), "NEXT ACTION") {
		t.Fatalf("stderr should reject the stdin body for missing NEXT ACTION; got %q", errb.String())
	}
}

func TestRunStreamingSubcommandUnknownIsProgrammingError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStreamingSubcommand([]string{"nope"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "no runner wired") {
		t.Fatalf("code=%d stderr=%q, want 2 + 'no runner wired'", code, stderr.String())
	}
}

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

func TestRunDirectPairFallsBackToEmbeddedRuntimeAndSetsPairHome(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable:   "/home/me/.local/bin/pair",
		embeddedRoot: "/data/pair/runtime/abc/pair-home",
		roots:        map[string]bool{"/data/pair/runtime/abc/pair-home": true},
		execCode:     9,
		environ:      []string{"PATH=/bin", "PAIR_HOME=/old"},
	}

	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)

	if code != 9 {
		t.Fatalf("code = %d, want 9", code)
	}
	if rt.execPath != "/data/pair/runtime/abc/pair-home/bin/pair-shell" {
		t.Fatalf("execPath = %q, want embedded pair-shell", rt.execPath)
	}
	if !containsEnv(rt.execEnv, "PAIR_HOME=/data/pair/runtime/abc/pair-home") {
		t.Fatalf("execEnv missing embedded PAIR_HOME: %#v", rt.execEnv)
	}
	if containsEnv(rt.execEnv, "PAIR_HOME=/old") {
		t.Fatalf("execEnv kept old PAIR_HOME: %#v", rt.execEnv)
	}
}

func TestRuntimeDataDirPrefersPairDataDir(t *testing.T) {
	got := runtimeDataDir("/pair-data", "/home/me", "/xdg")
	if got != "/pair-data" {
		t.Fatalf("runtimeDataDir = %q, want PAIR_DATA_DIR", got)
	}
}

func TestRuntimeDataDirFallsBackToXDGPairDir(t *testing.T) {
	got := runtimeDataDir("", "/home/me", "/xdg")
	if got != "/xdg/pair" {
		t.Fatalf("runtimeDataDir = %q, want XDG pair dir", got)
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
	embeddedRoot    string
	embeddedErr     error
	environ         []string

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
	if f.environ != nil {
		return f.environ
	}
	return os.Environ()
}

func (f *fakeLegacyRuntime) EmbeddedAssetRoot() (string, error) {
	return f.embeddedRoot, f.embeddedErr
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
