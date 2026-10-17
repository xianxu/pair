package procutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// This adapter is portable so fixture tests exercise the actual procfs reader.
func readLinuxCommand(root string, pid int) (CommandInfo, error) {
	if pid <= 0 {
		return CommandInfo{}, errors.New("invalid process ID")
	}
	dir := filepath.Join(root, strconv.Itoa(pid))
	executable, err := os.Readlink(filepath.Join(dir, "exe"))
	if err != nil {
		return CommandInfo{}, err
	}
	f, err := os.Open(filepath.Join(dir, "cmdline"))
	if err != nil {
		return CommandInfo{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxCommandBytes+1))
	if err != nil {
		return CommandInfo{}, err
	}
	args, err := parseLinuxCmdline(raw)
	if err != nil {
		return CommandInfo{}, err
	}
	return CommandInfo{Executable: executable, Args: args}, nil
}

// readLinuxTable keeps procfs enumeration portable for native-reader tests.
func readLinuxTable(root string) ([]Process, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Process
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, e.Name(), "stat"))
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
			continue // Positive disappearance between the listing and the read.
		}
		p := Process{PID: pid}
		if err == nil {
			if parsed, parseErr := parseLinuxStat(pid, raw); parseErr == nil {
				p = parsed
			}
		}
		// Failed reads and malformed metadata preserve the listed PID, with
		// unknown generation/name/state. They cannot establish process exit.
		out = append(out, p)
	}
	return out, nil
}
