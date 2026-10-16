package nativelock

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"golang.org/x/sys/unix"
)

func testKey(t *testing.T, id string) Key {
	t.Helper()
	k, err := NewKey("codex", id)
	if err != nil {
		t.Fatal(err)
	}
	return k
}
func lockPath(t *testing.T, home string, k Key) string {
	t.Helper()
	paths, err := artifactpath.ResolvePairCache(home)
	if err != nil {
		t.Fatal(err)
	}
	path, err := paths.NativeLock(k.Agent(), k.ID())
	if err != nil {
		t.Fatal(err)
	}
	return path
}
func TestKeysAndExactPath(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "agy", "muse", "qoder", "grok"} {
		if _, err := NewKey(agent, "conversation-世界:1"); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"", ".", "..", "../id", "a/b", `a\b`, "a\x00b", "a\nb", "a b", "a\u202eb", string([]byte{0xff}), strings.Repeat("x", 513)} {
		if _, err := NewKey("codex", id); err == nil {
			t.Errorf("accepted %q", id)
		}
	}
	if _, err := NewKey("other", "id"); err == nil {
		t.Fatal("unknown agent")
	}
	home := t.TempDir()
	paths := map[string]bool{}
	for _, id := range []string{"a:b", "a_b", "A", "a", "é", "e\u0301", strings.Repeat("x", 512)} {
		key := testKey(t, id)
		path := lockPath(t, home, key)
		if paths[path] || path != lockPath(t, home, key) {
			t.Fatal("path aliases or is unstable", path)
		}
		paths[path] = true
		if filepath.Dir(path) != filepath.Join(home, ".cache", "pair", "native-locks") || len(filepath.Base(path)) > 255 {
			t.Fatal(path)
		}
	}
	if _, err := (artifactpath.PairCachePaths{}).NativeLock("codex", "id"); err == nil {
		t.Fatal("zero path accepted")
	}
	if _, err := Acquire(home, Key{}, "pair:2"); err == nil {
		t.Fatal("zero key accepted")
	}
}
func TestContentionStableInodeAndClose(t *testing.T) {
	home := t.TempDir()
	k := testKey(t, "exact")
	h, err := Acquire(home, k, "pair:2")
	if err != nil {
		t.Fatal(err)
	}
	path := lockPath(t, home, k)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if len(data) > 1024 || ParseHolder(data) != "pair:2" {
		t.Fatalf("holder %q", data)
	}
	if _, err := Acquire(home, k, "pair:3"); err == nil || !strings.Contains(err.Error(), "is in use by pair:2") || !strings.Contains(err.Error(), "Start fresh") {
		t.Fatalf("contention %v", err)
	}
	afterData, _ := os.ReadFile(path)
	if !bytes.Equal(data, afterData) {
		t.Fatal("loser mutated metadata")
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Acquire(home, k, "pair:3")
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	after, _ := os.Stat(path)
	if !os.SameFile(before, after) {
		t.Fatal("inode replaced")
	}
	if after.Mode().Perm() != 0600 {
		t.Fatal(after.Mode())
	}
	stat := after.Sys().(*syscall.Stat_t)
	t.Logf("per conversation: metadata=%d bytes, allocated=%d bytes, inode=%d (filesystem-dependent)", after.Size(), stat.Blocks*512, stat.Ino)
	flags, err := unix.FcntlInt(next.file.Fd(), unix.F_GETFD, 0)
	if err != nil || flags&unix.FD_CLOEXEC == 0 {
		t.Fatalf("CLOEXEC flags=%d err=%v", flags, err)
	}
}
func TestRefuseUnsafeFiles(t *testing.T) {
	for _, kind := range []string{"symlink", "fifo", "directory", "permissions", "directory-symlink"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			k := testKey(t, "id")
			path := lockPath(t, home, k)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "symlink":
				target := filepath.Join(home, "target")
				os.WriteFile(target, []byte("preserved"), 0600)
				os.Symlink(target, path)
			case "fifo":
				if err := unix.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				os.Mkdir(path, 0700)
			case "permissions":
				os.WriteFile(path, []byte("private"), 0644)
			case "directory-symlink":
				os.Remove(filepath.Dir(path))
				os.Symlink(home, filepath.Dir(path))
			}
			if h, err := Acquire(home, k, "pair:2"); err == nil {
				h.Close()
				t.Fatal("unsafe path accepted")
			}
		})
	}
}
func TestHolderMetadataCannotChangeKernelDecision(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("bad\nvalue"), []byte("\x1b[31m"), bytes.Repeat([]byte("x"), 1025)} {
		t.Run(fmt.Sprintf("%d", len(data)), func(t *testing.T) {
			home := t.TempDir()
			k := testKey(t, "id")
			h, err := Acquire(home, k, "pair:2")
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			path := lockPath(t, home, k)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if ParseHolder(data) != "another Pair session" {
				t.Fatalf("unsafe holder accepted %q", data)
			}
			if _, err := Acquire(home, k, "pair:3"); err == nil || !strings.Contains(err.Error(), "in use by another Pair session") {
				t.Fatalf("%v", err)
			}
			got, _ := os.ReadFile(path)
			if !bytes.Equal(got, data) {
				t.Fatal("metadata mutated")
			}
		})
	}
	home := t.TempDir()
	if _, err := Acquire(home, testKey(t, "id"), strings.Repeat("x", 1025)); err == nil {
		t.Fatal("oversize holder accepted")
	}
}
func TestLockHelper(t *testing.T) {
	if os.Getenv("PAIR_NATIVELOCK_HELPER") != "1" {
		return
	}
	key, err := NewKey("codex", "subprocess")
	if err != nil {
		panic(err)
	}
	h, err := Acquire(os.Getenv("PAIR_NATIVELOCK_HOME"), key, "scratch:7")
	if err != nil {
		panic(err)
	}
	fmt.Println("ready")
	io.Copy(io.Discard, os.Stdin)
	h.Close()
	os.Exit(0)
}
func TestSubprocessRelease(t *testing.T) {
	for _, kill := range []bool{false, true} {
		t.Run(fmt.Sprintf("kill=%v", kill), func(t *testing.T) {
			home := t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLockHelper$")
			cmd.Env = append(os.Environ(), "PAIR_NATIVELOCK_HELPER=1", "PAIR_NATIVELOCK_HOME="+home)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
			ready := make([]byte, len("ready\n"))
			if _, err := io.ReadFull(stdout, ready); err != nil || string(ready) != "ready\n" {
				t.Fatalf("barrier %q %v", ready, err)
			}
			k := testKey(t, "subprocess")
			if _, err := Acquire(home, k, "scratch:8"); err == nil || !strings.Contains(err.Error(), "scratch:7") {
				t.Fatalf("%v", err)
			}
			if kill {
				cmd.Process.Kill()
			} else {
				stdin.Close()
			}
			err = cmd.Wait()
			if !kill && err != nil {
				t.Fatal(err)
			}
			h, err := Acquire(home, k, "scratch:8")
			if err != nil {
				t.Fatal(err)
			}
			h.Close()
		})
	}
}
