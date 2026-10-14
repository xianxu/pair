//go:build darwin

package procutil

import "golang.org/x/sys/unix"

func nativeCommand(pid int) (CommandInfo, error) {
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return CommandInfo{}, err
	}
	command, _, err := ParseDarwinProcArgs(raw)
	return command, err
}
