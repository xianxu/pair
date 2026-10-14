//go:build linux

package procutil

func nativeCommand(pid int) (CommandInfo, error) { return readLinuxCommand("/proc", pid) }
