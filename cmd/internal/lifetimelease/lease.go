// Package lifetimelease is a Couch slot's lifetime lease (pair#695 P1): an
// empty file that each long-lived pair process of the slot's tree holds with a
// shared flock for its whole life. The tree is gone iff an exclusive,
// non-blocking flock on a fresh open succeeds. It is active only under Couch
// (COUCH_LIFETIME_LEASE set); plain pair opens nothing.
package lifetimelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Env carries the lease path from Couch to the pair processes of the tree.
// The wrapper removes it from the agent's environment (WithoutLease).
const Env = "COUCH_LIFETIME_LEASE"

// holdBound is how long a holder waits out a probe's brief exclusive lock.
const holdBound = 2 * time.Second

// Path is the lease file for one lifecycle address key under a Couch
// namespace: <namespace>/leases/<digest>.lease. The file name is a digest, so
// a key's paths never become path components.
func Path(namespace, key string) (string, error) {
	if !filepath.IsAbs(namespace) || filepath.Clean(namespace) != namespace || key == "" {
		return "", fmt.Errorf("lifetime lease needs an absolute namespace and a key")
	}
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(Dir(namespace), hex.EncodeToString(sum[:])[:16]+".lease"), nil
}

// Dir is the folder that holds a namespace's lease files.
func Dir(namespace string) string { return filepath.Join(namespace, "leases") }

// Prepare creates the lease file (and its directory) before Couch launches
// the tree. It holds nothing.
func Prepare(path string) error {
	if err := validPath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// HoldFromEnv holds the lease named by Env for the caller's lifetime. Without
// Env (plain pair) it returns nil, nil and touches nothing.
func HoldFromEnv(ctx context.Context, getenv func(string) string) (io.Closer, error) {
	if !Active(getenv) {
		return nil, nil
	}
	return Hold(ctx, getenv(Env))
}

// Active reports whether this pair role runs under Couch's lifetime lease:
// Env is set only by Couch, so plain pair (no Env) holds nothing and
// publishes no terminal.
func Active(getenv func(string) string) bool { return getenv(Env) != "" }

// Hold takes a shared lock, waiting up to holdBound for a probe to finish.
// The descriptor is close-on-exec, so no child inherits it; each pair role
// takes its own. A lock taken on a file SweepFree unlinked in the meantime
// would be invisible to every witness, so Hold keeps only a lock on the file
// the path still names, and otherwise reopens.
func Hold(ctx context.Context, path string) (io.Closer, error) {
	if err := validPath(path); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(holdBound)
	for {
		f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o600)
		if err != nil {
			return nil, err
		}
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
		if err == nil {
			if current(f, path) {
				return f, nil
			}
			_ = f.Close()
			continue
		}
		_ = f.Close()
		if !errors.Is(err, syscall.EWOULDBLOCK) || time.Now().After(deadline) || ctx.Err() != nil {
			return nil, fmt.Errorf("hold lifetime lease: %w", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// current reports whether f is still the file at path.
func current(f *os.File, path string) bool {
	held, err := f.Stat()
	if err != nil {
		return false
	}
	named, err := os.Stat(path)
	return err == nil && os.SameFile(held, named)
}

// SweepFree removes the lease files under dir that no process holds: the
// leftovers of retirements that had no proof to remove them (an unreadable
// record, an adopted survivor's predecessor). Each is unlinked while its
// exclusive lock is held, so a holder either got there first (and the file
// stays) or reopens a fresh one (Hold). It returns how many it removed.
func SweepFree(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".lease" {
			continue
		}
		if ok, _ := RemoveFree(filepath.Join(dir, entry.Name())); ok {
			removed++
		}
	}
	return removed, nil
}

// RemoveFree unlinks the lease file at path only while holding its exclusive
// lock, so a holder that got there first keeps it, and a holder that comes
// after reopens a fresh file (Hold). A missing file counts as removed.
func RemoveFree(path string) (bool, error) {
	if err := validPath(path); err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !current(f, path) {
		return true, nil // already replaced: the file it named is gone
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, nil
}

// Free reports whether no process holds the lease: an exclusive, non-blocking
// lock on a fresh open succeeds (and is released at once). A missing file is
// free: no process of the tree ever opened it.
func Free(path string) (bool, error) {
	if err := validPath(path); err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// WithoutLease is env without Env: the agent and everything it runs never see
// the lease, so nothing an agent starts can hold the slot.
func WithoutLease(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, Env+"=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func validPath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Ext(path) != ".lease" {
		return fmt.Errorf("invalid lifetime lease path %q", path)
	}
	return nil
}

// Holders names the processes that have the lease open: the survivors to
// stop when it is not free. It runs `lsof -t` on the one file, bounded at 2s;
// lsof exits 1 with no output when nothing has it open.
func Holders(ctx context.Context, path string) ([]int, error) {
	if err := validPath(path); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, holdBound)
	defer cancel()
	cmd := exec.CommandContext(ctx, "lsof", "-t", "--", path)
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("list lease holders: %w", ctx.Err())
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 && len(out) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list lease holders: %w", err)
	}
	return parsePIDLines(string(out))
}

func parsePIDLines(out string) ([]int, error) {
	var pids []int
	seen := map[int]bool{}
	for _, line := range strings.Fields(out) {
		pid, err := strconv.Atoi(line)
		if err != nil || pid <= 0 {
			return nil, fmt.Errorf("unexpected lsof output %q", line)
		}
		if !seen[pid] {
			seen[pid] = true
			pids = append(pids, pid)
		}
	}
	return pids, nil
}
