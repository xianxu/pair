package couchcmd

import (
	"context"
	"debug/buildinfo"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/drainqueue/lease_helper.go.txt
var realDrainQueueLeaseHelper []byte

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
	helper := filepath.Join(root, "helper")
	if err := os.Mkdir(helper, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helper, "main.go"), realDrainQueueLeaseHelper, 0600); err != nil {
		t.Fatal(err)
	}
	run(helper, "go", "mod", "init", "github.com/xianxu/ariadne/cmd/sdlc/pairdrainconformance")
	run(helper, "go", "mod", "edit", "-replace=github.com/xianxu/ariadne="+source)
	run(helper, "go", "mod", "edit", "-require=github.com/xianxu/ariadne@v0.0.0")
	run(helper, "go", "build", "-mod=mod", "-o", filepath.Join(root, "lease-helper"), ".")
	return realDrainQueueFixture{root: root, cwd: cwd, binary: binary, workerTree: workerTree}
}
