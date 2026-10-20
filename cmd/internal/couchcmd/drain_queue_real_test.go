package couchcmd

import (
	"context"
	"debug/buildinfo"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/internal/testbinary"
)

//go:embed testdata/drainqueue/lease_helper.go.txt
var realDrainQueueLeaseHelper []byte

var realDrainQueueLeaseCache testbinary.Cache
var realDrainQueueLeaseSource string

type realDrainQueueFixture struct {
	root, cwd, binary, workerTree string
}

// Opt-in only: no dependency executable is discovered from PATH, and no queue,
// capacity lock, or fleet discovery ever falls back to the machine's live roots.
func newRealDrainQueueFixture(t *testing.T) realDrainQueueFixture {
	t.Helper()
	binary, source := os.Getenv("PAIR_DRAIN_SDLC_BINARY"), os.Getenv("PAIR_DRAIN_SDLC_SOURCE")
	if binary == "" && source == "" {
		t.Skip("set PAIR_DRAIN_SDLC_BINARY and PAIR_DRAIN_SDLC_SOURCE for isolated real queue conformance")
	}
	for name, value := range map[string]string{"PAIR_DRAIN_SDLC_BINARY": binary, "PAIR_DRAIN_SDLC_SOURCE": source} {
		if !filepath.IsAbs(value) {
			t.Fatalf("%s must explicitly name an absolute path", name)
		}
	}
	run := func(dir, program string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, program, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", program, args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	revision, modified := "", ""
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	head := run(source, "git", "rev-parse", "HEAD")
	if revision == "" || revision != head || modified != "false" || run(source, "git", "status", "--porcelain", "--untracked-files=no") != "" {
		t.Fatalf("require clean matching binary/source: binary revision=%q modified=%q source=%q", revision, modified, head)
	}
	t.Logf("real SDLC binary=%s source=%s revision=%s", binary, source, revision)
	root := t.TempDir()
	cwd := filepath.Join(root, "fleet", "fixture")
	if err := os.MkdirAll(cwd, 0700); err != nil {
		t.Fatal(err)
	}
	run(cwd, "git", "init", "--quiet")
	workerTree := os.Getenv("COUCH_TREE")
	t.Setenv("COUCH_TREE", "")
	if err := os.Unsetenv("COUCH_TREE"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WF_SDLC_QUEUE_ROOT", filepath.Join(root, "queue"))
	t.Setenv("WF_TEST_LOCK_FILE", filepath.Join(root, "capacity"))
	t.Setenv("WF_SDLC_QUEUE_FLEET_ROOT", filepath.Join(root, "fleet"))
	t.Setenv("PAIR722_PROBE_ALLOWED_ROOT", root)
	t.Setenv("PAIR722_PROBE_CWD", cwd)
	// The helper creates real lifetime leases using the same source revision as
	// the binary. It never executes a suite, merge, landing, or application command.
	cacheKey := source + "@" + head
	helper, err := realDrainQueueLeaseCache.Directory(func(dir string) error {
		buildCtx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		realDrainQueueLeaseSource = cacheKey
		if err := os.WriteFile(filepath.Join(dir, "main.go"), realDrainQueueLeaseHelper, 0600); err != nil {
			return err
		}
		buildRun := func(args ...string) error {
			cmd := exec.CommandContext(buildCtx, "go", args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("go %v: %w\n%s", args, err, out)
			}
			return nil
		}
		for _, args := range [][]string{
			{"mod", "init", "github.com/xianxu/ariadne/cmd/sdlc/pairdrainconformance"},
			{"mod", "edit", "-replace=github.com/xianxu/ariadne=" + source},
			{"mod", "edit", "-require=github.com/xianxu/ariadne@v0.0.0"},
		} {
			if err := buildRun(args...); err != nil {
				return err
			}
		}
		//lint:allow test-go-build — the suite-owned cache builds this lease fixture once and retains it until TestMain cleanup
		cmd := exec.CommandContext(buildCtx, "go", "build", "-mod=mod", "-o", filepath.Join(dir, "lease-helper"), ".")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build real queue lease helper: %w\n%s", err, out)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if realDrainQueueLeaseSource != cacheKey {
		t.Fatal("real queue dependency changed during this test process; rerun with one fixed binary/source revision")
	}
	// Each private fixture aliases the suite-owned read-only executable; its
	// lifetime is not tied to the first consumer's temporary queue directory.
	if err := os.Symlink(filepath.Join(helper, "lease-helper"), filepath.Join(root, "lease-helper")); err != nil {
		t.Fatal(err)
	}

	return realDrainQueueFixture{root: root, cwd: cwd, binary: binary, workerTree: workerTree}
}
