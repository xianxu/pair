// Package notifytransport delivers bounded hook messages to the owning wrapper.
// It never opens a terminal or encodes terminal controls.
package notifytransport

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func canonicalBinding(binding string) (string, error) {
	if binding == "" {
		return "", errors.New("notification PID binding is empty")
	}
	absolute, err := filepath.Abs(binding)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func owned(info os.FileInfo) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && s.Uid == uint32(os.Getuid())
}

func privateDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("notification namespace must be absolute")
	}
	if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || !owned(info) || info.Mode().Perm() != 0700 {
		return fmt.Errorf("unsafe notification directory %q", path)
	}
	return nil
}

func rootDirectory() string {
	if override := os.Getenv("PAIR_NOTIFY_SOCKET_DIR"); override != "" {
		return filepath.Clean(override)
	}
	return filepath.Join("/tmp", fmt.Sprintf("pair-notify-%d", os.Getuid()))
}

// One permanent UID-private lock serializes publication and removal across
// cooperating wrapper processes. Keeping its inode avoids lock-file ABA races.
func lockDirectory() (func(), error) { return lockNamespace(rootDirectory()) }
func lockNamespace(root string) (func(), error) {
	if err := privateDirectory(root); err != nil {
		return nil, err
	}
	fd, err := unix.Open(filepath.Join(root, "lock"), unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "notification-directory-lock")
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || !owned(info) || info.Mode().Perm() != 0600 {
		f.Close()
		return nil, errors.New("unsafe notification directory lock")
	}
	deadline := time.Now().Add(SendTimeout)
	for {
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = f.Close() }, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			f.Close()
			return nil, err
		}
		if !time.Now().Before(deadline) {
			f.Close()
			return nil, errors.New("notification directory lock timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func address(binding string, pid int) (string, error) {
	return addressIn(rootDirectory(), binding, pid)
}
func addressIn(root, binding string, pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("notification wrapper PID must be positive")
	}
	canonical, err := canonicalBinding(binding)
	if err != nil {
		return "", err
	}
	// /tmp keeps AF_UNIX paths short even when TMPDIR names a long macOS cache.
	if err := privateDirectory(root); err != nil {
		return "", err
	}
	socket := socketAddressIn(root, canonical, pid)
	if len(socket) > 100 {
		return "", errors.New("notification namespace makes socket pathname too long")
	}
	return socket, nil
}

// socketAddressIn is the production pure identity mapping; callers validate
// the canonical binding, positive PID and private directory before filesystem IO.
func socketAddressIn(root, canonicalBinding string, pid int) string {
	digest := sha256.Sum256([]byte(canonicalBinding))
	return filepath.Join(root, fmt.Sprintf("%x-%d.sock", digest[:16], pid))
}

func readPID(binding string) (int, os.FileInfo, error) {
	fd, err := unix.Open(binding, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return 0, nil, err
	}
	f := os.NewFile(uintptr(fd), binding)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, nil, err
	}
	if !info.Mode().IsRegular() || !owned(info) {
		return 0, nil, errors.New("unsafe notification PID binding")
	}
	raw, err := io.ReadAll(io.LimitReader(f, 33))
	if err != nil {
		return 0, nil, err
	}
	text := strings.TrimSpace(string(raw))
	if len(raw) > 32 || text == "" {
		return 0, nil, errors.New("invalid notification PID binding")
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return 0, nil, errors.New("invalid notification wrapper PID")
		}
	}
	pid, err := strconv.Atoi(text)
	if err != nil || pid <= 0 {
		return 0, nil, errors.New("invalid notification wrapper PID")
	}
	return pid, info, nil
}

func alive(pid int) bool { return !errors.Is(unix.Kill(pid, 0), unix.ESRCH) }

// removeOwned requires the directory lock; all broker mutation uses that lock.
// Inode comparison additionally preserves independently replaced bindings.
func removeOwned(path string, expected os.FileInfo) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !owned(info) || !os.SameFile(info, expected) {
		return nil
	}
	return os.Remove(path)
}

func clearDeadBinding(root, binding string) error {
	pid, info, err := readPID(binding)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if alive(pid) {
		return fmt.Errorf("notification wrapper PID %d is still live", pid)
	}
	socket, err := addressIn(root, binding, pid)
	if err != nil {
		return err
	}
	socketInfo, err := os.Lstat(socket)
	if err == nil {
		if socketInfo.Mode()&os.ModeSocket == 0 || !owned(socketInfo) {
			return errors.New("unsafe stale notification socket")
		}
		// Recheck immediately before removing resources belonging to a dead PID.
		if alive(pid) {
			return errors.New("notification socket owner became live")
		}
		if err = removeOwned(socket, socketInfo); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return removeOwned(binding, info)
}

const maxNamespaceEntries = 1024

// socketOwnerPID admits only our exact generated filename grammar. Unknown
// files, malformed names and PID values outside the OS pid_t range are foreign.
func socketOwnerPID(name string) (int, bool) {
	if len(name) < 39 || !strings.HasSuffix(name, ".sock") || name[32] != '-' {
		return 0, false
	}
	hash := name[:32]
	if strings.ToLower(hash) != hash {
		return 0, false
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return 0, false
	}
	text := name[33 : len(name)-5]
	pid, err := strconv.ParseInt(text, 10, 32)
	if err != nil || pid <= 0 || strconv.FormatInt(pid, 10) != text {
		return 0, false
	}
	return int(pid), true
}

// sweepDeadSockets runs under the namespace lock, independently of PID binding
// survival. Read at most one sentinel beyond the bound; unknown entries count
// toward capacity but are never removed. No paths are reconstructed from hashes.
func sweepDeadSockets(root string, reserve int) error {
	dir, err := os.Open(root)
	if err != nil {
		return err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(maxNamespaceEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > maxNamespaceEntries {
		return errors.New("notification namespace exceeds 1024-entry capacity")
	}
	remaining := len(entries)
	for _, entry := range entries {
		pid, ok := socketOwnerPID(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			remaining--
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSocket == 0 || !owned(info) || alive(pid) {
			continue
		}
		if err = removeOwned(path, info); err != nil {
			return err
		}
		remaining--
	}
	if remaining+reserve > maxNamespaceEntries {
		return errors.New("notification namespace has no free socket capacity")
	}
	return nil
}
