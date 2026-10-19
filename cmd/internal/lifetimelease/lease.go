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
	"path/filepath"
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
	return filepath.Join(namespace, "leases", hex.EncodeToString(sum[:])[:16]+".lease"), nil
}

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
	path := getenv(Env)
	if path == "" {
		return nil, nil
	}
	return Hold(ctx, path)
}

// Hold takes a shared lock, waiting up to holdBound for a probe to finish.
// The descriptor is close-on-exec, so no child inherits it; each pair role
// takes its own.
func Hold(ctx context.Context, path string) (io.Closer, error) {
	if err := validPath(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(holdBound)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) || time.Now().After(deadline) || ctx.Err() != nil {
			_ = f.Close()
			return nil, fmt.Errorf("hold lifetime lease: %w", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
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
