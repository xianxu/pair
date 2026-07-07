package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRunStreamingSubcommandRoutesChangelogToInjectedStderr(t *testing.T) {
	// changelog with no flags → usage error to the *injected* stderr (proves the
	// seam passes real stderr through, unlike the buffered Dispatch path).
	var stdout, stderr bytes.Buffer
	code := runStreamingSubcommand("changelog render", nil, strings.NewReader(""), &stdout, &stderr)
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
		"continuation",
		[]string{"--repo-root", root, "--slug", "s", "--agent", "claude", "--issues", "1", "--body-file", "-"},
		strings.NewReader("just a body, no next action\n"), &out, &errb)
	if code != 1 {
		t.Fatalf("code = %d, want 1 (stdin body missing NEXT ACTION)", code)
	}
	if !strings.Contains(errb.String(), "NEXT ACTION") {
		t.Fatalf("stderr should reject the stdin body for missing NEXT ACTION; got %q", errb.String())
	}
}

func TestRunStreamingSubcommandRoutesSessionWatch(t *testing.T) {
	// session-watch with no args → buildOptions rejects (<3 args) → exit 0,
	// proving the seam case is wired to sessionwatch.RunCLI.
	var stdout, stderr bytes.Buffer
	code := runStreamingSubcommand("session-watch", nil, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0 (missing args no-op)", code)
	}
}

func TestRunStreamingSubcommandUnknownIsProgrammingError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStreamingSubcommand("nope", nil, strings.NewReader(""), &stdout, &stderr)
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
	// wrap/scribe are now implemented streaming routes (#96), so use an
	// unknown command to exercise the buffered dispatcher's stderr + exit-2 path.
	code := run([]string{"definitely-not-a-command"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
	}
}

// The launch route (public `pair` and `pair-go launch`) resolves the asset root,
// then drives the native launcher in-process and returns its exit code — there is
// no shell to exec (#99 M5c, bin/pair-shell retired).
func TestLaunchDrivesNativeLauncher(t *testing.T) {
	for _, tc := range []struct {
		name string
		argv []string
		want []string
	}{
		{"pair-go launch", []string{"launch", "claude", "--", "--resume"}, []string{"claude", "--", "--resume"}},
		{"direct pair", []string{"claude", "--", "--resume"}, []string{"claude", "--", "--resume"}},
		{"pair --help", []string{"launch", "--help"}, []string{"--help"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exe := "/repo/bin/pair"
			if tc.argv[0] == "launch" {
				exe = "/repo/bin/pair-go"
			}
			rt := &fakeLegacyRuntime{executable: exe, roots: map[string]bool{"/repo": true}, launchNativeCode: 5}
			var stdout, stderr bytes.Buffer
			code := runWithLegacyRuntime(tc.argv, &stdout, &stderr, rt)
			if code != 5 {
				t.Fatalf("code = %d, want the native exit code 5", code)
			}
			if !rt.launchNativeCalled || !reflect.DeepEqual(rt.launchNativeArgs, tc.want) || rt.launchNativeRoot != "/repo" {
				t.Fatalf("native called=%v args=%#v root=%q", rt.launchNativeCalled, rt.launchNativeArgs, rt.launchNativeRoot)
			}
		})
	}
}

// A missing asset root reports the marker (main.kdl) + recovery hints and never
// runs the launcher — there's no bin/pair-shell to blame anymore.
func TestRunLaunchReportsMissingRoot(t *testing.T) {
	rt := &fakeLegacyRuntime{executable: "/repo/bin/pair-go"}
	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{"pair-go launch", "main.kdl", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
	if rt.launchNativeCalled {
		t.Fatal("launcher must not run without a valid root")
	}
}

func TestRunLaunchFallsBackToDefaultPairHome(t *testing.T) {
	rt := &fakeLegacyRuntime{executable: "/home/me/.local/bin/pair", defaultPairHome: "/repo", roots: map[string]bool{"/repo": true}}
	var stdout, stderr bytes.Buffer
	if code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !rt.launchNativeCalled || rt.launchNativeRoot != "/repo" {
		t.Fatalf("native called=%v root=%q, want /repo", rt.launchNativeCalled, rt.launchNativeRoot)
	}
}

func TestRunLaunchFallsBackToEmbeddedRuntime(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable:       "/home/me/.local/bin/pair",
		embeddedRoot:     "/data/pair/runtime/abc/pair-home",
		roots:            map[string]bool{"/data/pair/runtime/abc/pair-home": true},
		launchNativeCode: 9,
	}
	var stdout, stderr bytes.Buffer
	if code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt); code != 9 {
		t.Fatalf("code = %d, want 9", code)
	}
	if rt.launchNativeRoot != "/data/pair/runtime/abc/pair-home" {
		t.Fatalf("native root = %q, want the embedded pair-home", rt.launchNativeRoot)
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

// The pair-go dispatcher `help` command does not touch the launch route (no asset-
// root probe, no launcher call).
func TestRunPairGoHelperDoesNotProbeLaunchRoute(t *testing.T) {
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
	if rt.launchNativeCalled {
		t.Fatal("dispatcher help must not run the launcher")
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
	statCalls       int
	embeddedRoot    string
	embeddedErr     error

	// native launcher seam (#99 M5c — the sole launcher).
	launchNativeCode   int
	launchNativeCalled bool
	launchNativeArgs   []string
	launchNativeRoot   string
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
	const marker = "/zellij/layouts/main.kdl"
	if strings.HasSuffix(path, marker) && f.roots != nil {
		if f.roots[strings.TrimSuffix(path, marker)] {
			return nil
		}
	}
	return os.ErrNotExist
}

func (f *fakeLegacyRuntime) EmbeddedAssetRoot() (string, error) {
	return f.embeddedRoot, f.embeddedErr
}

func (f *fakeLegacyRuntime) LaunchNative(args []string, root string, stdout, stderr io.Writer) int {
	f.launchNativeCalled = true
	f.launchNativeArgs = append([]string(nil), args...)
	f.launchNativeRoot = root
	return f.launchNativeCode
}
