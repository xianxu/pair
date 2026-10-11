package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSelectedProcessLeaseManagedAndReusable(t *testing.T) {
	root := storeDirectory(t)
	env := map[string]string{"PAIR_DATA_DIR": root, "PAIR_TAG": "tag"}
	getenv := func(k string) string { return env[k] }
	lease, err := AcquireSelectedProcess(context.Background(), getenv, "watcher")
	if err != nil || lease == nil {
		t.Fatalf("acquire: %v %v", lease, err)
	}
	second, err := AcquireSelectedProcess(context.Background(), getenv, "watcher")
	if err != nil || second.ID != lease.ID {
		t.Fatalf("repeat creates another registration: %v %v", second, err)
	}
	state, err := lease.Coordinator.ReadOwner(lease.Owner)
	if err != nil || len(state.Processes) != 1 || state.Processes[0].Process.PID != os.Getpid() {
		t.Fatalf("wrong process: %+v %v", state, err)
	}
	before := state.Activity.LastUse
	other, err := AcquireSelectedProcess(context.Background(), getenv, "reader")
	if err != nil || other.ID == lease.ID {
		t.Fatalf("distinct role conflated: %v %v", other, err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	state, err = lease.Coordinator.ReadOwner(lease.Owner)
	if err != nil || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(before) {
		t.Fatalf("close refreshed use: %+v %v", state, err)
	}
}
func TestSelectedProcessLeaseUnmanagedAndInvalid(t *testing.T) {
	lease, err := AcquireSelectedProcess(context.Background(), func(string) string { return "" }, "watcher")
	if err != nil || lease != nil {
		t.Fatalf("unmanaged: %v %v", lease, err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	root := storeDirectory(t)
	for _, env := range []map[string]string{{"PAIR_DATA_DIR": root}, {"PAIR_TAG": "tag"}, {"PAIR_DATA_DIR": "relative", "PAIR_TAG": "tag"}, {"PAIR_DATA_DIR": root, "PAIR_TAG": "../tag"}} {
		if lease, err := AcquireSelectedProcess(context.Background(), func(k string) string { return env[k] }, "watcher"); err == nil || lease != nil {
			t.Fatalf("invalid managed env accepted: %v", env)
		}
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("invalid env created metadata")
	}
}
func TestSelectedProcessLeaseFailureRefusesCallerEffect(t *testing.T) {
	root := storeDirectory(t)
	outside := storeDirectory(t)
	if err := os.Symlink(outside, filepath.Join(root, ".retention")); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PAIR_DATA_DIR": root, "PAIR_TAG": "tag"}
	effect := false
	lease, err := AcquireSelectedProcess(context.Background(), func(k string) string { return env[k] }, "watcher")
	if err == nil {
		defer lease.Close()
		effect = true
	}
	if err == nil || effect {
		t.Fatal("failed registration permitted effect")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("failed registration wrote outside root")
	}
}

func TestRoleAcquisitionAcknowledgesStartAfterRegistration(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	start, err := c.ReserveStart(ctx, o, p, []string{"wrapper", "draft-editor"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MarkStartSpawned(ctx, o, start, p); err != nil {
		t.Fatal(err)
	}
	id, err := c.AcquireRoleProcess(ctx, o, p, "wrapper", start)
	if err != nil {
		t.Fatal(err)
	}
	again, err := c.AcquireRoleProcess(ctx, o, p, "wrapper", start)
	if err != nil || again != id {
		t.Fatalf("repeat changed registration: %s %s %v", id, again, err)
	}
	state, err := c.ReadOwner(o)
	if err != nil || len(state.Processes) != 1 || len(state.Starts) != 1 || state.Starts[0].Acknowledged["wrapper"] != p {
		t.Fatalf("registration/ack ordering: %+v %v", state, err)
	}
	if _, err := c.AcquireRoleProcess(ctx, o, p, "draft-editor", start); err != nil {
		t.Fatal(err)
	}
	state, err = c.ReadOwner(o)
	if err != nil || len(state.Processes) != 2 || len(state.Starts) != 0 {
		t.Fatalf("readiness did not settle: %+v %v", state, err)
	}
}

func TestRoleAcquisitionFailedAckRetainsStartAndRegistration(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	start, err := c.ReserveStart(ctx, o, p, []string{"wrapper"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MarkStartSpawned(ctx, o, start, p); err != nil {
		t.Fatal(err)
	}
	saves := 0
	c.BeforePersist = func() error {
		saves++
		if saves == 2 {
			return errors.New("ack disk failure")
		}
		return nil
	}
	if _, err := c.AcquireRoleProcess(ctx, o, p, "wrapper", start); err == nil {
		t.Fatal("failed ack authorized access")
	}
	state, err := c.ReadOwner(o)
	if err != nil || len(state.Processes) != 1 || len(state.Starts) != 1 || len(state.Starts[0].Acknowledged) != 0 {
		t.Fatalf("failure lost protection: %+v %v", state, err)
	}
}

func TestRoleTargetsKeepIndependentCaptureReaders(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(o.Directory(), "capture-a.raw")
	b := filepath.Join(o.Directory(), "capture-b.raw")
	first, err := c.AcquireRoleProcessTarget(ctx, o, p, "capture-reader", "", a)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.AcquireRoleProcessTarget(ctx, o, p, "capture-reader", "", b)
	if err != nil || first == second {
		t.Fatal("capture readers conflated", err)
	}
	again, err := c.AcquireRoleProcessTarget(ctx, o, p, "capture-reader", "", a)
	if err != nil || again != first {
		t.Fatal("reader identity unstable", err)
	}
	state, err := c.ReadOwner(o)
	if err != nil || len(state.Processes) != 2 || state.Processes[0].Target != a || state.Processes[1].Target != b {
		t.Fatalf("targets lost: %+v %v", state, err)
	}
	if _, err := c.AcquireRoleProcessTarget(ctx, o, p, "capture-reader", "", "../outside"); err == nil {
		t.Fatal("relative reader target accepted")
	}
	if _, err := c.AcquireRoleProcessTarget(ctx, o, p, "capture-reader", "", filepath.Join(t.TempDir(), "foreign.raw")); err == nil {
		t.Fatal("foreign reader target accepted")
	}
}

// pair#436: SelectedOwner resolves PAIR_DATA_DIR to its physical path, so a
// target named through a symlinked spelling of the same directory (macOS
// TMPDIR=/var/... → /private/var/...) is the owner's file, and is recorded in
// the physical frame the collector compares capture paths in.
func TestRoleTargetThroughSymlinkedDataDirIsRecordedPhysically(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(o.Directory(), 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(o.Directory(), alias); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AcquireRoleProcessTarget(ctx, o, p, "changelog-viewer", "", filepath.Join(alias, "changelog-t-claude.md")); err != nil {
		t.Fatalf("aliased owner target refused: %v", err)
	}
	state, err := c.ReadOwner(o)
	want := filepath.Join(o.Directory(), "changelog-t-claude.md")
	if err != nil || len(state.Processes) != 1 || state.Processes[0].Target != want {
		t.Fatalf("target = %+v %v, want %s", state.Processes, err, want)
	}
	escape := filepath.Join(o.Directory(), "escape")
	if err := os.Symlink(t.TempDir(), escape); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AcquireRoleProcessTarget(ctx, o, p, "capture-reader", "", filepath.Join(escape, "foreign.raw")); err == nil {
		t.Fatal("target escaping the owner through a symlink accepted")
	}
}

func TestOwnerFramePathResolvesTheExistingPrefixOnly(t *testing.T) {
	real := t.TempDir()
	if physical, err := filepath.EvalSymlinks(real); err == nil {
		real = physical
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	for in, want := range map[string]string{
		filepath.Join(alias, "log.md"):              filepath.Join(real, "log.md"),
		filepath.Join(alias, "repos", "k", "t.raw"): filepath.Join(real, "repos", "k", "t.raw"),
	} {
		if got, err := OwnerFramePath(in); err != nil || got != want {
			t.Fatalf("OwnerFramePath(%s) = %s, %v; want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{"relative/x", filepath.Join(alias, "..", "x") + "/"} {
		if _, err := OwnerFramePath(bad); err == nil {
			t.Fatalf("OwnerFramePath(%q) accepted a non-exact path", bad)
		}
	}
}
