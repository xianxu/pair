package procutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
