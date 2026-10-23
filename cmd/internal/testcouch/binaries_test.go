package testcouch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestCandidatePathsRejectInvalidPlacement(t *testing.T) {
	for _, dir := range []string{"", ".", "relative/bin", "/", "/owned/../foreign"} {
		if _, err := candidatePaths(dir); err == nil {
			t.Errorf("accepted candidate directory %q", dir)
		}
	}
	root := t.TempDir()
	paths, err := candidatePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pair", "couch", "pair-launch-helper", "fixture-agent"} {
		if paths[name] != filepath.Join(root, name) {
			t.Fatalf("%s destination = %q", name, paths[name])
		}
	}
	if len(paths) != 4 {
		t.Fatalf("unexpected candidates: %v", paths)
	}
}

func TestValidateToolFactsRejectNonExecutableOrUnscopedTools(t *testing.T) {
	for _, tc := range []struct {
		path string
		mode os.FileMode
		want bool
	}{
		{"/tools/git", 0755, true}, {"relative/git", 0755, false}, {"", 0755, false},
		{"/tools/git", 0644, false}, {"/tools/git", os.ModeDir | 0755, false},
		{"/tools/git", os.ModeSymlink | 0755, false}, {"/tools/git", os.ModeNamedPipe | 0755, false},
	} {
		if err := validateToolFacts(tc.path, tc.mode); (err == nil) != tc.want {
			t.Errorf("validate(%q,%v) = %v", tc.path, tc.mode, err)
		}
	}
}

func TestDiscoverToolRequiresPhysicalExecutable(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "tool")
	if err := os.WriteFile(file, []byte("#!/bin/sh\nexit 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTool(file); err == nil {
		t.Fatal("accepted non-executable tool")
	}
	if err := os.Chmod(file, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(file, alias); err != nil {
		t.Fatal(err)
	}
	got, err := resolveTool(alias)
	physical, physicalErr := filepath.EvalSymlinks(file)
	if err != nil || physicalErr != nil || got != physical {
		t.Fatalf("physical tool = %q, %v; want %q (%v)", got, err, physical, physicalErr)
	}
	if _, err := resolveTool(filepath.Join(root, "missing")); err == nil {
		t.Fatal("accepted missing tool")
	}
}

func TestBuildCommandBoundsDiagnosticsAndHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	output, err := buildCommand(ctx, t.TempDir(), "/bin/sh", "-c", "printf '%070000d' 0; exit 7")
	if err == nil || len(output) > buildOutputLimit+100 || !strings.Contains(output, "truncated") {
		t.Fatalf("bounded failure: length=%d err=%v", len(output), err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := buildCommand(cancelled, t.TempDir(), "/bin/sh", "-c", "exit 0"); err == nil {
		t.Fatal("ignored build cancellation")
	}
}

func TestCommandWrappersJoinCanceledDescendants(t *testing.T) {
	for _, wrapper := range []string{"build", "provision"} {
		t.Run(wrapper, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "child.pid")
			observeCtx, observeCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer observeCancel()
			ctx, cancel := context.WithCancel(observeCtx)
			defer cancel()
			done := make(chan error, 1)
			script := `trap 'wait; exit' TERM; sleep 30 & child=$!; printf '%s' "$child" > "$1"; wait`
			go func() {
				var err error
				if wrapper == "build" {
					_, err = buildCommand(ctx, root, "/bin/sh", "-c", script, "fixture", marker)
				} else {
					_, err = runProvisionCommand(ctx, root, []string{"PATH=/usr/bin:/bin"}, "/bin/sh", "-c", script, "fixture", marker)
				}
				done <- err
			}()
			pid := 0
			generation := ""
			if err := waitUntil(observeCtx, func() (bool, error) {
				raw, err := os.ReadFile(marker)
				if err != nil {
					return false, nil
				}
				pid, err = strconv.Atoi(string(raw))
				if err != nil {
					return false, nil
				}
				generation = procutil.Identity(strconv.Itoa(pid))
				return generation != "", nil
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = (nativeProcesses{}).Signal(pid, generation, syscall.SIGKILL) })
			cancel()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("canceled command returned success")
				}
			case <-observeCtx.Done():
				t.Fatal("command did not join cancellation")
			}
			rows, _, err := (procutil.NativeResourceReader{}).Table(observeCtx)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				if row.PID == pid && row.Identity == generation && !row.Zombie {
					t.Fatalf("%s returned with owned descendant still running: pid=%d generation=%s", wrapper, pid, generation)
				}
			}
		})
	}
}

func TestDefaultBuildConfigCanonicalizesTempAliasWithoutScratchVolume(t *testing.T) {
	physical, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "temp-alias")
	if err := os.Symlink(physical, alias); err != nil {
		t.Fatal(err)
	}
	// This is Build's actual config-selection seam. The absent-volume fact is
	// supplied without unmounting the machine scratch disk or changing TMPDIR.
	cfg := defaultBuildConfig("/selected/checkout", nil, alias, false)
	if cfg.ScratchParent != physical || cfg.Source != "/selected/checkout" {
		t.Fatalf("default build selected unphysical temp: %+v; physical=%s", cfg, physical)
	}
	if err := validateScratchParent(cfg.ScratchParent); err != nil {
		t.Fatalf("selected default cannot enter strict owner: %v", err)
	}
}

func TestBuildOwnedExplicitScratchAliasStillRefusedBeforeEffects(t *testing.T) {
	source := t.TempDir()
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	physical, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "explicit-alias")
	if err := os.Symlink(physical, alias); err != nil {
		t.Fatal(err)
	}
	looked := false
	owner, err := buildOwned(t.Context(), BuildConfig{Source: source, ScratchParent: alias}, buildDependencies{resolve: func(string) (string, error) { looked = true; return "", fmt.Errorf("unexpected lookup") }})
	if owner != nil || err == nil || !strings.Contains(err.Error(), "physical scratch parent required") || looked {
		t.Fatalf("explicit alias reached build effects: owner=%v err=%v looked=%v", owner, err, looked)
	}
	entries, err := os.ReadDir(physical)
	if err != nil || len(entries) != 0 {
		t.Fatalf("explicit refusal created files: %v %v", entries, err)
	}
}
