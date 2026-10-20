package vmcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestPlanSnapshotRejectsEscapeAndReservedMetadata(t *testing.T) {
	for _, path := range []string{"", ".", "../outside", "/outside", "a/../b", ".git/config", ".GIT/config", ".couch/socket"} {
		if _, err := PlanSnapshot("/source", "/destination", []string{path}); err == nil {
			t.Fatalf("admitted unsafe snapshot path %q", path)
		}
	}
	for _, dst := range []string{"/source", "/source/child", "/"} {
		if _, err := PlanSnapshot("/source", dst, nil); err == nil {
			t.Fatalf("admitted overlapping snapshot root %q", dst)
		}
	}
}

func TestSnapshotCheckoutCopiesOnlyTrackedCurrentBytes(t *testing.T) {
	source, dst := snapshotFixture(t)
	writeSnapshotFixture(t, filepath.Join(source, "tracked"), "current tracked bytes")
	if err := os.Remove(filepath.Join(source, "deleted")); err != nil {
		t.Fatal(err)
	}
	writeSnapshotFixture(t, filepath.Join(source, "private-token"), "ignored secret")
	writeSnapshotFixture(t, filepath.Join(source, "untracked"), "untracked secret")
	report, err := SnapshotCheckout(context.Background(), source, dst)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dst, "tracked"))
	if err != nil || string(raw) != "current tracked bytes" {
		t.Fatalf("tracked snapshot: %q %v", raw, err)
	}
	for _, omitted := range []string{"deleted", "private-token", "untracked", ".git/commondir", ".git/objects/info/alternates"} {
		if _, err := os.Lstat(filepath.Join(dst, omitted)); !os.IsNotExist(err) {
			t.Fatalf("exported absent/private/linked input %s: %v", omitted, err)
		}
	}
	if got := strings.TrimSpace(snapshotGitFixture(t, dst, "rev-parse", "HEAD")); got != report.Head {
		t.Fatalf("Git HEAD=%s, report=%s", got, report.Head)
	}
	if raw, err := os.ReadFile(filepath.Join(source, "tracked")); err != nil || string(raw) != "current tracked bytes" {
		t.Fatal("snapshot changed source")
	}
}

func TestSnapshotCheckoutNormalizesLinkedWorktreeGit(t *testing.T) {
	source, dst := snapshotFixture(t)
	linked := filepath.Join(filepath.Dir(dst), "linked")
	snapshotGitFixture(t, source, "worktree", "add", "--detach", linked, "HEAD")
	if _, err := SnapshotCheckout(context.Background(), linked, dst); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(dst, ".git"))
	if err != nil || !info.IsDir() {
		t.Fatalf("snapshot retains external gitfile: %v %v", info, err)
	}
	if got := strings.TrimSpace(snapshotGitFixture(t, dst, "rev-parse", "--show-toplevel")); got != dst {
		t.Fatalf("snapshot resolves outside itself: %s", got)
	}
}

func TestSnapshotCheckoutRefusesCanonicalOverlapAndParentSymlinkEscape(t *testing.T) {
	source, dst := snapshotFixture(t)
	alias := filepath.Join(filepath.Dir(dst), "alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotCheckout(context.Background(), source, filepath.Join(alias, "snapshot")); err == nil {
		t.Fatal("snapshot wrote through alias into source")
	}
	if _, err := os.Lstat(filepath.Join(source, "snapshot")); !os.IsNotExist(err) {
		t.Fatal("overlap refusal occurred after mutation")
	}
	writeSnapshotFixture(t, filepath.Join(source, "nested", "tracked"), "original")
	snapshotGitFixture(t, source, "add", "nested/tracked")
	snapshotGitFixture(t, source, "commit", "-m", "nested")
	if err := os.Remove(filepath.Join(source, "nested", "tracked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(source, "nested")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeSnapshotFixture(t, filepath.Join(outside, "tracked"), "outside secret")
	if err := os.Symlink(outside, filepath.Join(source, "nested")); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotCheckout(context.Background(), source, dst); err == nil {
		t.Fatal("snapshot followed tracked parent into outside data")
	}
}

func TestSnapshotCheckoutBoundsFallbackAndCountsLinearWork(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			source, dst := snapshotFixture(t)
			for i := 0; i < n; i++ {
				writeSnapshotFixture(t, filepath.Join(source, fmt.Sprintf("file-%d", i)), "hello")
			}
			snapshotGitFixture(t, source, "add", ".")
			snapshotGitFixture(t, source, "commit", "-m", "files")
			calls := 0
			clone := func(_, _ *os.File, _ string) error { calls++; return syscall.EXDEV }
			report, err := snapshotCheckout(context.Background(), source, dst, clone, SnapshotBudget{MaxFiles: 100, MaxBytes: 1024})
			if err != nil {
				t.Fatal(err)
			}
			if calls != n+3 || report.Enumerations != 1 || report.Files != n+3 || report.ClonedBytes != 0 || report.CopiedBytes != int64(n*5+15) {
				t.Fatalf("copy count not one per tracked file: %+v calls=%d", report, calls)
			}
		})
	}
	source, dst := snapshotFixture(t)
	for _, injected := range []error{syscall.EPERM, errors.New("native clone failed")} {
		if _, err := snapshotCheckout(context.Background(), source, dst, func(_, _ *os.File, _ string) error { return injected }, SnapshotBudget{MaxFiles: 100, MaxBytes: 1024}); err == nil {
			t.Fatal("non-filesystem clone error silently fell back")
		}
		if err := os.RemoveAll(dst); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := snapshotCheckout(context.Background(), source, dst, func(_, _ *os.File, _ string) error { t.Fatal("copy started before budget check"); return nil }, SnapshotBudget{MaxFiles: 1, MaxBytes: 1}); err == nil {
		t.Fatal("oversized snapshot admitted")
	}
}

func snapshotFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	snapshotGitFixture(t, source, "init", "--quiet")
	snapshotGitFixture(t, source, "config", "user.name", "Fixture")
	snapshotGitFixture(t, source, "config", "user.email", "fixture@example.invalid")
	writeSnapshotFixture(t, filepath.Join(source, "tracked"), "old")
	writeSnapshotFixture(t, filepath.Join(source, "deleted"), "old")
	writeSnapshotFixture(t, filepath.Join(source, ".gitignore"), "private*\n")
	snapshotGitFixture(t, source, "add", ".")
	snapshotGitFixture(t, source, "commit", "-m", "fixture")
	return source, filepath.Join(root, "snapshot")
}

func snapshotGitFixture(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := snapshotGit(context.Background(), root, args...)
	if err != nil {
		t.Fatalf("git fixture %v: %v (%s)", args, err, out)
	}
	return string(out)
}

func writeSnapshotFixture(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}
