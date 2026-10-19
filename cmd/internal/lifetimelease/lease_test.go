package lifetimelease

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func testPath(t *testing.T) string {
	t.Helper()
	path, err := Path(t.TempDir(), "slot:/w/pair")
	if err != nil {
		t.Fatal(err)
	}
	if err := Prepare(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathIsADigestUnderTheNamespace(t *testing.T) {
	ns := t.TempDir()
	a, err := Path(ns, "slot:/w/pair")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Path(ns, "team:abcd/teams/devinfra")
	if a == b || filepath.Dir(a) != filepath.Join(ns, "leases") || !strings.HasSuffix(a, ".lease") || strings.Contains(filepath.Base(a), "pair") {
		t.Fatalf("paths %q %q", a, b)
	}
	for _, bad := range [][2]string{{"", "slot:/w"}, {"relative", "slot:/w"}, {ns, ""}} {
		if _, err := Path(bad[0], bad[1]); err == nil {
			t.Errorf("Path(%q, %q) accepted", bad[0], bad[1])
		}
	}
}

func TestHoldMakesTheLeaseBusyUntilClosed(t *testing.T) {
	path := testPath(t)
	if free, err := Free(path); err != nil || !free {
		t.Fatalf("fresh lease free=%v err=%v", free, err)
	}
	a, err := Hold(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Hold(context.Background(), path)
	if err != nil {
		t.Fatalf("a second shared holder was refused: %v", err)
	}
	if free, _ := Free(path); free {
		t.Fatal("held lease reported free")
	}
	_ = a.Close()
	if free, _ := Free(path); free {
		t.Fatal("lease reported free while one holder remains")
	}
	_ = b.Close()
	if free, err := Free(path); err != nil || !free {
		t.Fatalf("released lease free=%v err=%v", free, err)
	}
}

func TestFreeOfAMissingLeaseIsFree(t *testing.T) {
	path, _ := Path(t.TempDir(), "slot:/w/pair")
	if free, err := Free(path); err != nil || !free {
		t.Fatalf("missing lease free=%v err=%v", free, err)
	}
}

// A probe holds the lease exclusively for an instant; a holder starting then
// waits within its bound rather than failing.
func TestHoldWaitsOutABriefProbe(t *testing.T) {
	path := testPath(t)
	probe, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := lockExclusive(probe); err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(100*time.Millisecond, func() { _ = probe.Close() })
	held, err := Hold(context.Background(), path)
	if err != nil {
		t.Fatalf("Hold did not wait out a brief probe: %v", err)
	}
	_ = held.Close()
}

// pair launched without Couch: no env, so nothing is opened, locked or
// created, and the agent's environment is untouched (pair#695: plain pair
// unchanged).
func TestPlainPairUnchangedWithoutCouch(t *testing.T) {
	ns := t.TempDir()
	plain := func(string) string { return "" }
	if Active(plain) {
		t.Fatal("plain pair reads as under Couch, so its wrapper would publish a terminal")
	}
	before, _ := os.ReadDir("/dev/fd")
	closer, err := HoldFromEnv(context.Background(), plain)
	if err != nil || closer != nil {
		t.Fatalf("HoldFromEnv without Couch = %v, %v; want nothing", closer, err)
	}
	if after, _ := os.ReadDir("/dev/fd"); len(after) != len(before) {
		t.Fatalf("plain pair opened a descriptor: %d -> %d", len(before), len(after))
	}
	if entries, _ := os.ReadDir(ns); len(entries) != 0 {
		t.Fatalf("plain pair created %v", entries)
	}
	env := []string{"HOME=/h", "PAIR_TAG=x"}
	if got := WithoutLease(env); strings.Join(got, ",") != strings.Join(env, ",") {
		t.Fatalf("WithoutLease changed a plain environment: %v", got)
	}
}

func TestAgentNeverSeesTheLease(t *testing.T) {
	env := []string{"HOME=/h", Env + "=/ns/leases/x.lease", "PAIR_TAG=x", Env + "=dup"}
	for _, kv := range WithoutLease(env) {
		if strings.HasPrefix(kv, Env+"=") {
			t.Fatalf("WithoutLease kept %q", kv)
		}
	}
	if got := WithoutLease(env); len(got) != 2 {
		t.Fatalf("WithoutLease dropped more than the lease: %v", got)
	}
}

func TestHoldFromEnvRefusesAMalformedPath(t *testing.T) {
	for _, bad := range []string{"relative.lease", "/tmp/x.txt", "/tmp/../tmp/x.lease"} {
		if _, err := HoldFromEnv(context.Background(), func(k string) string {
			if k == Env {
				return bad
			}
			return ""
		}); err == nil {
			t.Errorf("HoldFromEnv accepted %q", bad)
		}
	}
}

func lockExclusive(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func TestParsePIDLines(t *testing.T) {
	got, err := parsePIDLines("12\n7\n12\n")
	if err != nil || len(got) != 2 || got[0] != 12 || got[1] != 7 {
		t.Fatalf("parsePIDLines = %v, %v", got, err)
	}
	if _, err := parsePIDLines("12\nx\n"); err == nil {
		t.Fatal("accepted a non-pid line")
	}
}

// Live: the holder lsof names is this process while it holds the lease, and
// nobody once released. Unsandboxed (lsof).
func TestHoldersNamesTheHoldingProcess(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof unavailable")
	}
	path := testPath(t)
	held, err := Hold(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	pids, err := Holders(context.Background(), path)
	if err != nil {
		t.Skipf("lsof not usable here (sandbox?): %v", err)
	}
	if len(pids) != 1 || pids[0] != os.Getpid() {
		t.Fatalf("holders while held = %v, want [%d]", pids, os.Getpid())
	}
	_ = held.Close()
	if pids, err := Holders(context.Background(), path); err != nil || len(pids) != 0 {
		t.Fatalf("holders after release = %v, %v", pids, err)
	}
}

// The sweep removes only free lease files; a held one stays, and a Hold
// after the sweep holds the file the path names.
func TestSweepFreeRemovesOnlyFreeLeases(t *testing.T) {
	ns := t.TempDir()
	held, err := Path(ns, "slot:/w/a t/1")
	if err != nil {
		t.Fatal(err)
	}
	free, err := Path(ns, "slot:/w/b t/2")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{held, free} {
		if err := Prepare(p); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := Hold(context.Background(), held)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	removed, err := SweepFree(Dir(ns))
	if err != nil || removed != 1 {
		t.Fatalf("SweepFree = %d, %v; want the one free lease", removed, err)
	}
	if _, err := os.Stat(held); err != nil {
		t.Fatalf("a held lease was swept: %v", err)
	}
	if _, err := os.Stat(free); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a free lease survived: %v", err)
	}
	again, err := Hold(context.Background(), free)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if ok, _ := Free(free); ok {
		t.Fatal("a Hold after the sweep does not hold the file its path names")
	}
}

// RemoveFree never unlinks a held lease; a free one goes, and a missing one
// counts as removed.
func TestRemoveFreeLeavesAHeldLease(t *testing.T) {
	path, err := Path(t.TempDir(), "slot:/w/a t/1")
	if err != nil {
		t.Fatal(err)
	}
	if err := Prepare(path); err != nil {
		t.Fatal(err)
	}
	lock, err := Hold(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveFree(path); err != nil || removed {
		t.Fatalf("RemoveFree of a held lease = %v, %v", removed, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("a held lease was unlinked: %v", err)
	}
	_ = lock.Close()
	if removed, err := RemoveFree(path); err != nil || !removed {
		t.Fatalf("RemoveFree of a free lease = %v, %v", removed, err)
	}
	if removed, err := RemoveFree(path); err != nil || !removed {
		t.Fatalf("RemoveFree of a missing lease = %v, %v", removed, err)
	}
}
