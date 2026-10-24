package testcouch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRuntimeSelectedGitFetchesPrivateUpstream(t *testing.T) {
	git, err := resolveTool("git")
	if err != nil {
		t.Fatal(err)
	}
	query := exec.CommandContext(t.Context(), git, "--exec-path")
	query.Env = []string{"PATH=" + filepath.Dir(git), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}
	raw, err := query.Output()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		// Apple's launcher masks the physical executable's private-PATH failure.
		git, err = resolveTool(filepath.Join(strings.TrimSpace(string(raw)), "git-upload-pack"))
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GIT_EXEC_PATH", "/foreign/git-helpers")
	t.Setenv("GIT_SSH_COMMAND", "/foreign/ssh")
	b, _, _ := runtimeFilesFixture(t)
	b.Git = git
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(p.Root); err != nil {
			t.Error(err)
		}
	})
	helperEnv, err := buildEnvironment(filepath.Join(p.Root, "build"), p.Mutable["bin"], goCaches{GOCACHE: p.Mutable["cache"], GOMODCACHE: p.Mutable["cache"]})
	if err != nil {
		t.Fatal(err)
	}
	b.GitUploadPack, err = selectGitUploadPack(t.Context(), git, p.Root, helperEnv, buildDependencies{resolve: resolveTool, run: runCandidateBuild})
	if err != nil {
		t.Fatal(err)
	}
	if err := installRuntime(b, p); err != nil {
		t.Fatal(err)
	}
	env, err := runtimeEnvironment(p, p.Mutable["bin"], "claude", "git-fetch", runtimeSourceForTest(t), b.Couch)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range env {
		if strings.Contains(entry, "/foreign/") {
			t.Fatalf("ambient Git settings leaked: %s", entry)
		}
		if strings.HasPrefix(entry, "PATH=") && entry != "PATH="+p.Mutable["bin"] {
			t.Fatalf("ambient PATH: %s", entry)
		}
	}
	run := func(dir string, args ...string) string {
		t.Helper()
		tool := git
		if args[0] == "fetch" {
			tool = filepath.Join(p.Mutable["bin"], "git")
		}
		cmd := exec.CommandContext(t.Context(), tool, args...)
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	fleet := p.Mutable["fleet"]
	source, upstream, target := filepath.Join(fleet, "source"), filepath.Join(fleet, "upstream.git"), filepath.Join(fleet, "target")
	run(fleet, "init", "-b", "main", source)
	run(source, "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "fixture")
	want := run(source, "rev-parse", "HEAD")
	run(fleet, "clone", "--bare", source, upstream)
	run(fleet, "init", target)
	run(target, "fetch", "--verbose", "--porcelain", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--refmap=", upstream, "+refs/heads/main:refs/remotes/upstream/main")
	if got := run(target, "rev-parse", "refs/remotes/upstream/main"); got != want {
		t.Fatalf("fetched %s, want %s", got, want)
	}
}

func TestSelectGitUploadPackRejectsInvalidInstallation(t *testing.T) {
	for _, scenario := range []string{"query-error", "relative", "multiple-lines", "missing", "non-executable", "directory"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			helper := filepath.Join(root, "git-upload-pack")
			if err := os.WriteFile(helper, []byte("fixture"), 0700); err != nil {
				t.Fatal(err)
			}
			raw := root + "\n"
			switch scenario {
			case "relative":
				raw = "relative"
			case "multiple-lines":
				raw = root + "\n/another"
			case "missing":
				if err := os.Remove(helper); err != nil {
					t.Fatal(err)
				}
			case "non-executable":
				if err := os.Chmod(helper, 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(helper); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(helper, 0700); err != nil {
					t.Fatal(err)
				}
			}
			_, err := selectGitUploadPack(t.Context(), "/selected/git", root, []string{"PATH=/private/bin"}, buildDependencies{resolve: resolveTool, run: func(_ context.Context, program string, args []string, dir string, env []string) (string, error) {
				if program != "/selected/git" || len(args) != 1 || args[0] != "--exec-path" || dir != root || strings.Join(env, " ") != "PATH=/private/bin" {
					t.Fatalf("selection escaped caller: %s %v %s %v", program, args, dir, env)
				}
				if scenario == "query-error" {
					return "", os.ErrPermission
				}
				return raw, nil
			}})
			if err == nil {
				t.Fatal("invalid helper accepted")
			}
		})
	}
}
