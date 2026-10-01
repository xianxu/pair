package couchmessage

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// collectDeadEndpointSockets runs on server startup in the private runtime
// directory. Only ESRCH proves the named process is gone: permission errors,
// reused PIDs, and all other uncertain observations preserve the socket. The
// incarnation hash prevents a replacement wrapper from reusing the old name.
// No sidecar or persistent registration is needed to collect crash residue.
func collectDeadEndpointSockets(dir string, probe func(int) error) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		pid, ok := endpointSocketPID(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if info.Mode()&os.ModeSocket == 0 || !ok || stat.Uid != uint32(os.Getuid()) {
			continue
		}
		if !errors.Is(probe(pid), syscall.ESRCH) {
			continue
		}
		// The pathname may have been replaced while ownership was checked.
		if err := transportRemoveOwned(path, info); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func endpointSocketPID(name string) (int, bool) {
	if !strings.HasPrefix(name, "wrapper-") || !strings.HasSuffix(name, ".sock") {
		return 0, false
	}
	pidText, digest, ok := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(name, "wrapper-"), ".sock"), "-")
	if !ok || len(digest) != 40 {
		return 0, false
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(pidText)
	return pid, err == nil && pid > 0 && strconv.Itoa(pid) == pidText
}
