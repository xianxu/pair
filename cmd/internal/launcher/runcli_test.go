package launcher

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// `pair --help` / `pair help` prints the native usage to stdout and exits 0 (#99
// M5c — the shell that used to own help is retired).
func TestLaunchNativeHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "help"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := LaunchNative([]string{arg}, "/pair", &stdout, &stderr)
			if err != nil || code != 0 {
				t.Fatalf("%s: code=%d err=%v", arg, code, err)
			}
			if !strings.Contains(stdout.String(), "USAGE") || stderr.Len() != 0 {
				t.Fatalf("%s: stdout=%q stderr=%q", arg, stdout.String(), stderr.String())
			}
		})
	}
}

func TestLaunchNativeConsumesRepoDefaultPolicyBeforeEarlyReturn(t *testing.T) {
	t.Setenv("PAIR_USE_REPO_DEFAULT", "1")
	var stdout, stderr bytes.Buffer
	code, err := LaunchNative([]string{"help"}, "/pair", &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, ok := os.LookupEnv("PAIR_USE_REPO_DEFAULT"); ok {
		t.Fatal("PAIR_USE_REPO_DEFAULT remains set after LaunchNative entry")
	}
}

func TestConsumeRepoDefaultPolicy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "exact one", value: "1", want: true},
		{name: "absent or empty", value: "", want: false},
		{name: "word true", value: "true", want: false},
		{name: "numeric lookalike", value: "01", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unsetCalls := 0
			got := consumeRepoDefaultPolicy(
				func(key string) string {
					if key != "PAIR_USE_REPO_DEFAULT" {
						t.Fatalf("get key = %q", key)
					}
					return tc.value
				},
				func(key string) error {
					if key != "PAIR_USE_REPO_DEFAULT" {
						t.Fatalf("unset key = %q", key)
					}
					unsetCalls++
					return nil
				},
			)
			if got != tc.want {
				t.Fatalf("consumeRepoDefaultPolicy(%q) = %v, want %v", tc.value, got, tc.want)
			}
			if unsetCalls != 1 {
				t.Fatalf("unset calls = %d, want 1", unsetCalls)
			}
		})
	}
}

func TestNewLaunchOptionsAppliesRepoDefaultPolicy(t *testing.T) {
	getenv := func(string) string { return "" }
	for _, want := range []bool{false, true} {
		opts := newLaunchOptions(LaunchArgs{}, Env{}, "/pair", "/data", want, getenv, 5)
		if opts.SkipConfigPicker != want {
			t.Fatalf("SkipConfigPicker = %v, want %v", opts.SkipConfigPicker, want)
		}
	}
}

func TestApplyCouchLaunchEnvironmentRequiresMatchingRepoDefaultMarker(t *testing.T) {
	args, err := ParseArgs([]string{"resume", "couch-0102030405060708"})
	if err != nil {
		t.Fatal(err)
	}
	pathRaw := `{"schema_version":1,"tag":"couch-0102030405060708","agent":"codex","argv":["--search"],"agent_source":"path","argv_source":"path"}`
	defaultRaw := `{"schema_version":1,"tag":"couch-0102030405060708","agent":"codex","argv":["--search"],"agent_source":"root","argv_source":"repo-default"}`
	if _, err := applyCouchLaunchEnvironment(args, pathRaw, true); err == nil {
		t.Fatal("path argv accepted with repo-default marker")
	}
	if _, err := applyCouchLaunchEnvironment(args, defaultRaw, false); err == nil {
		t.Fatal("repo-default argv accepted without repo-default marker")
	}
	got, err := applyCouchLaunchEnvironment(args, pathRaw, false)
	if err != nil || got.Agent != "codex" || !got.AgentArgsFromCouch {
		t.Fatalf("path profile = %+v, %v", got, err)
	}
}

func TestLaunchNativeVersion(t *testing.T) {
	for _, arg := range []string{"--version", "version"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := LaunchNative([]string{arg}, "/pair", &stdout, &stderr)
			if err != nil || code != 0 {
				t.Fatalf("%s: code=%d err=%v", arg, code, err)
			}
			if !strings.Contains(stdout.String(), "pair") || stderr.Len() != 0 {
				t.Fatalf("%s: stdout=%q stderr=%q", arg, stdout.String(), stderr.String())
			}
		})
	}
}

// A leading flag that isn't help is a usage error → stderr + exit 2 (no shell to
// defer to).
func TestLaunchNativeBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := LaunchNative([]string{"--nope"}, "/pair", &stdout, &stderr)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v, want 2", code, err)
	}
	if !strings.Contains(stderr.String(), "not an agent") || stdout.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLaunchNativeRestartInfersAgentFromScopedDataDir(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "work", "pair")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	repo, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "")
	t.Setenv("ZELLIJ_SESSION_NAME", "pair-work")
	t.Setenv("PAIR_KILL_CMD", "__pair_no_such_command__")

	globalDataDir := filepath.Join(home, ".local", "share", "pair")
	if err := os.MkdirAll(globalDataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDataDir, "agent-work"), []byte("claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scopedDataDir := ScopedLaunchDataDir(globalDataDir, repo)
	if err := os.MkdirAll(scopedDataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line, err := BuildLedgerLine(LedgerEntry{
		Agent:      "codex",
		Started:    time.Unix(10, 0),
		LastActive: time.Unix(10, 0),
		RepoRoot:   repo,
		RepoName:   "pair",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scopedDataDir, "ledger-work.jsonl"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code, err := LaunchNative([]string{"restart"}, "/pair", &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%q", code, err, stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(home, ".cache", "pair", "restart-pair-work"))
	if err != nil {
		t.Fatal(err)
	}
	marker := parseRestartMarker(string(raw))
	if marker.Agent != "codex" {
		t.Fatalf("restart marker agent = %q, want scoped codex; raw marker:\n%s", marker.Agent, string(raw))
	}
}

func TestLaunchNativeUsesGitRootForScopedDataDirFromSubdir(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "work", "pair")
	subdir := filepath.Join(repo, "cmd", "pair")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", repo, "init").Run(); err != nil {
		t.Skipf("git init unavailable: %v", err)
	}
	if real, err := filepath.EvalSymlinks(repo); err == nil {
		repo = real
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("ZELLIJ_SESSION_NAME", "📁pair-work")
	t.Setenv("PAIR_KILL_CMD", "__pair_no_such_command__")

	globalDataDir := filepath.Join(home, ".local", "share", "pair")
	scopedDataDir := ScopedLaunchDataDir(globalDataDir, repo)
	if err := os.MkdirAll(scopedDataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line, err := BuildLedgerLine(LedgerEntry{
		Agent:      "codex",
		Started:    time.Unix(10, 0),
		LastActive: time.Unix(10, 0),
		RepoRoot:   repo,
		RepoName:   "pair",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scopedDataDir, "ledger-work.jsonl"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code, err := LaunchNative([]string{"restart"}, "/pair", &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%q", code, err, stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(home, ".cache", "pair", "restart-📁pair-work"))
	if err != nil {
		t.Fatal(err)
	}
	marker := parseRestartMarker(string(raw))
	if marker.Agent != "codex" {
		t.Fatalf("restart marker agent = %q, want scoped codex from repo root; raw marker:\n%s", marker.Agent, string(raw))
	}
}

func TestLaunchNativeRenameHonorsPairDataDirOverride(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "work", "pair")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	dataDir := filepath.Join(home, "explicit-data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "draft-old.md"), []byte("draft"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("PAIR_DATA_DIR", dataDir)

	var stdout, stderr bytes.Buffer
	code, err := LaunchNative([]string{"rename", "old", "new"}, "/pair", &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stdout=%q stderr=%q", code, err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dataDir, "draft-new.md")); err != nil {
		t.Fatalf("rename did not use PAIR_DATA_DIR override: %v", err)
	}
}
