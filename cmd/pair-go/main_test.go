package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
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

// A missing asset root reports both layout markers + recovery hints and never
// runs the launcher — there's no bin/pair-shell to blame anymore.
func TestRunLaunchReportsMissingRoot(t *testing.T) {
	rt := &fakeLegacyRuntime{executable: "/repo/bin/pair-go"}
	var stdout, stderr bytes.Buffer
	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{"pair-go launch", "main-2.kdl", "main-3.kdl", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
	if rt.launchNativeCalled {
		t.Fatal("launcher must not run without a valid root")
	}
}

func TestRunLaunchRejectsRootMissingEitherLayout(t *testing.T) {
	rt := &fakeLegacyRuntime{
		executable:    "/repo/bin/pair-go",
		roots:         map[string]bool{"/repo": true},
		missingMarker: "/zellij/layouts/main-3.kdl",
	}
	var stdout, stderr bytes.Buffer
	if code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if rt.launchNativeCalled {
		t.Fatal("root missing main-3.kdl must not launch")
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

func TestPublicPairCommandFamiliesIgnoreCouchStore(t *testing.T) {
	pairRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	scope, err := launcher.ResolveRepoScope(pairRoot)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	pair := filepath.Join(binDir, "pair")
	buildCommand(t, pair, ".")

	commands := []struct {
		name       string
		args       []string
		liveZellij bool
		extraEnv   []string
	}{
		{name: "launch-create", args: []string{"claude"}},
		{name: "launch-resume-attach", args: []string{"resume", "compiler-fix"}, liveZellij: true},
		{name: "launch-picker", liveZellij: true},
		{name: "list", args: []string{"list"}, liveZellij: true},
		{name: "rename", args: []string{"rename", "--restart-check", "compiler-fix", "compiler-fixed"}},
		{name: "continue", args: []string{"continue"}},
		{name: "restart", args: []string{"restart"}, extraEnv: []string{"ZELLIJ_SESSION_NAME=📁pair-compiler-fix", "PAIR_TAG=compiler-fix"}},
		{name: "quit", args: []string{"quit"}, extraEnv: []string{"ZELLIJ_SESSION_NAME=📁pair-compiler-fix", "PAIR_TAG=compiler-fix"}},
	}
	stores := []struct {
		name string
		seed func(*testing.T, string)
	}{
		{name: "valid-forward", seed: func(t *testing.T, root string) {
			writePublicPairFixture(t, filepath.Join(root, "threadstore", "manifest.json"), `{"schema_version":1,"generation":1,"threads":[],"legacy_migration_version":1}`)
		}},
		{name: "malformed", seed: func(t *testing.T, root string) {
			writePublicPairFixture(t, filepath.Join(root, "threadstore", "manifest.json"), `{not-json`)
		}},
		{name: "unreadable-fifo", seed: func(t *testing.T, root string) {
			threadStore := filepath.Join(root, "threadstore")
			if err := os.MkdirAll(threadStore, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(filepath.Join(threadStore, "manifest.json"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing", seed: func(t *testing.T, root string) {}},
	}

	for _, command := range commands {
		for _, store := range stores {
			t.Run(command.name+"/"+store.name, func(t *testing.T) {
				home := t.TempDir()
				dataDir := filepath.Join(home, "pair-data")
				storeDir := filepath.Join(home, "couch-store")
				stubDir := filepath.Join(home, "stubs")
				commandLog := filepath.Join(home, "external-commands.log")
				for _, dir := range []string{dataDir, storeDir, stubDir} {
					if err := os.MkdirAll(dir, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				store.seed(t, storeDir)
				writePublicPairCommandStubs(t, stubDir)
				if command.liveZellij {
					line := fmt.Sprintf(`{"session_name":"📁pair-compiler-fix","scope_key":%q,"repo_root":%q,"repo_name":"pair","tag":"compiler-fix"}`+"\n", scope.Key, scope.Root)
					writePublicPairFixture(t, filepath.Join(dataDir, "session-names.jsonl"), line)
					writePublicPairFixture(t, filepath.Join(dataDir, "workbench-layout-compiler-fix"), "layout2\n")
				}
				if command.name == "rename" {
					writePublicPairFixture(t, filepath.Join(dataDir, "draft-compiler-fix.md"), "rename fixture\n")
				}
				before := snapshotPublicPairNamespace(t, storeDir)

				zellijMode := "empty"
				if command.liveZellij {
					zellijMode = "existing"
				}
				env := append([]string{}, os.Environ()...)
				env = append(env,
					"HOME="+home,
					"PAIR_DATA_DIR="+dataDir,
					"PAIR_HOME="+pairRoot,
					"COUCH_STORE_DIR="+storeDir,
					"COUCH_TREE=opaque-tree-value",
					"PAIR_TEST_COMMAND_LOG="+commandLog,
					"PAIR_TEST_ZELLIJ_MODE="+zellijMode,
					"PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				)
				env = append(env, command.extraEnv...)

				result, timedOut := runPublicPairCommand(t, 8*time.Second, env, pair, command.args...)
				if timedOut {
					t.Errorf("public Pair command timed out; an unread Couch manifest was opened: args=%q", command.args)
				} else if result.code != 0 {
					t.Errorf("public Pair command depends on %s Couch state: args=%q code=%d stderr=%q", store.name, command.args, result.code, result.stderr)
				}

				after := snapshotPublicPairNamespace(t, storeDir)
				if !reflect.DeepEqual(after, before) {
					t.Errorf("public Pair command mutated Couch namespace\nbefore: %#v\nafter:  %#v", before, after)
				}
				if command.name == "launch-create" && store.name == "missing" {
					raw, err := os.ReadFile(commandLog)
					if err != nil {
						t.Fatalf("read external command log: %v", err)
					}
					if !strings.Contains(string(raw), "COUCH_TREE=opaque-tree-value") || !strings.Contains(string(raw), "COUCH_STORE_DIR="+storeDir) {
						t.Errorf("opaque Couch environment was not propagated to the normal child boundary:\n%s", raw)
					}
				}
			})
		}
	}
}

func runPublicPairCommand(t *testing.T, timeout time.Duration, env []string, name string, args ...string) (commandResult, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := commandResult{stdout: stdout.String(), stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		return result, true
	}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("run %s: %v", name, err)
		}
		result.code = exit.ExitCode()
	}
	return result, false
}

func writePublicPairCommandStubs(t *testing.T, dir string) {
	t.Helper()
	zellij := `#!/bin/sh
printf 'zellij %s COUCH_TREE=%s COUCH_STORE_DIR=%s\n' "$*" "$COUCH_TREE" "$COUCH_STORE_DIR" >> "$PAIR_TEST_COMMAND_LOG"
case "$*" in
  "list-sessions --short")
    [ "$PAIR_TEST_ZELLIJ_MODE" = existing ] && printf '📁pair-compiler-fix\n'
    ;;
  "list-sessions --no-formatting")
    [ "$PAIR_TEST_ZELLIJ_MODE" = existing ] && printf '📁pair-compiler-fix [Created 1s ago]\n'
    ;;
  *"action list-clients"*)
    [ "$PAIR_TEST_ZELLIJ_MODE" = existing ] && printf 'test-client\n'
    ;;
esac
exit 0
`
	fzf := `#!/bin/sh
printf 'fzf %s COUCH_TREE=%s COUCH_STORE_DIR=%s\n' "$*" "$COUCH_TREE" "$COUCH_STORE_DIR" >> "$PAIR_TEST_COMMAND_LOG"
tr '\000' '\n' | head -n 1
`
	for name, body := range map[string]string{"zellij": zellij, "fzf": fzf, "claude": "#!/bin/sh\nexit 0\n"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writePublicPairFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotPublicPairNamespace(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%s:%#o", info.Mode().Type(), info.Mode().Perm())
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += ":" + string(raw)
		}
		snapshot[rel] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
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
	missingMarker   string
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
	if f.missingMarker != "" && strings.HasSuffix(path, f.missingMarker) {
		return os.ErrNotExist
	}
	for _, marker := range []string{"/zellij/layouts/main-2.kdl", "/zellij/layouts/main-3.kdl"} {
		if strings.HasSuffix(path, marker) && f.roots != nil {
			if f.roots[strings.TrimSuffix(path, marker)] {
				return nil
			}
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
