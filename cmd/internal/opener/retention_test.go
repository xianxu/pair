package opener

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestViewerHandoffReservesAndPassesOwnerBeforeSpawn(t *testing.T) {
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		switch k {
		case "PAIR_DATA_DIR":
			return c.Root
		case "PAIR_TAG":
			return "t"
		}
		return ""
	}
	lease, err := storagegc.AcquireSelectedProcess(context.Background(), getenv, "scrollback-opener")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	opts := scrollbackOpts()
	opts.DataDir = c.Root

	paths, _ := artifactpath.ResolveScoped(c.Root, "t")
	source, _ := paths.ScrollbackArtifacts("claude")
	rt := newFake()
	rt.sizes[source.Raw] = 10
	rt.viewerHook = func(env []string) {
		state, err := c.ReadOwner(lease.Owner)
		if err != nil || len(state.Processes) != 1 {
			t.Fatalf("viewer not reserved before spawn: %+v %v", state, err)
		}
		for _, want := range []string{"PAIR_DATA_DIR=" + c.Root, "PAIR_TAG=t", "PAIR_RETENTION_TARGET=" + source.Raw, "PAIR_RETENTION_ROLE=scrollback-viewer"} {
			if !containsEnv(env, want) {
				t.Fatalf("missing viewer owner env %s in %v", want, env)
			}
		}
	}
	if code := RunScrollback(opts, rt, io.Discard); code != 0 {
		t.Fatal(code)
	}
}
func containsEnv(env []string, want string) bool {
	for _, entry := range env {
		if entry == want {
			return true
		}
	}
	return false
}
func TestOpenerCLIRefusesInvalidLeaseBeforeAnyAccess(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{"PAIR_DATA_DIR": root + "/missing", "PAIR_TAG": "t", "PAIR_AGENT": "claude"}
	for _, run := range []func([]string, func(string) string, io.Writer) int{RunScrollbackCLI, RunChangelogCLI} {
		if code := run(nil, func(k string) string { return env[k] }, io.Discard); code != 1 {
			t.Fatal(code)
		}
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("invalid lease wrote artifacts")
	}
}

func TestProtectedChildCannotRunIfRegistrationFails(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, root+"/.retention"); err != nil {
		t.Fatal(err)
	}
	marker := root + "/effect"
	cmd := exec.Command("sh", "-c", childGate+`printf effect > "$1"`, "child", marker)
	cmd.Env = []string{"PAIR_DATA_DIR=" + root, "PAIR_TAG=t", "PAIR_RETENTION_ROLE=scrollback-viewer", "PAIR_RETENTION_TARGET=" + root + "/source.raw"}
	if _, err := startProtected(cmd); err == nil {
		t.Fatal("unregistered child started")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("child performed effect before registration")
	}
}

func TestDetachedChildRetainsProtectionAfterLauncherExits(t *testing.T) {
	if os.Getenv("PAIR_TEST_OPENER_CHILD") == "1" {
		root := os.Getenv("PAIR_DATA_DIR")
		pid, err := (OSRuntime{}).StartDetached("exec sleep 30", []string{"PAIR_RETENTION_ROLE=changelog-distiller", "PAIR_RETENTION_TARGET=" + root + "/source.raw"}, root+"/status")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Print(pid)
		os.Exit(0)
	}
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetachedChildRetainsProtectionAfterLauncherExits$")
	cmd.Env = append(os.Environ(), "PAIR_TEST_OPENER_CHILD=1", "PAIR_DATA_DIR="+c.Root, "PAIR_SCOPE_KEY=", "PAIR_TAG=t", "PAIR_RETENTION_START_ID=")
	raw, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Kill()
	owner, _ := artifactpath.NewStorageOwner(c.Root, "", "t")
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 1 || state.Processes[0].Process.PID != pid {
		t.Fatalf("detached registration lost: %+v %v", state, err)
	}
	if got := (storagegc.OSProcessProbe{}).Inspect(state.Processes[0].Process); got != storagegc.ProcessAlive {
		t.Fatalf("child after parent exit: %s", got)
	}
}
