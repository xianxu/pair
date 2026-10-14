//go:build darwin

package couchcore

import "golang.org/x/sys/unix"

// processGroupMembers resolves an ambiguous signal probe without reducing
// group ownership to the direct child. Include zombies conservatively too.
func processGroupMembers(pid int) (bool, error) {
	members, err := unix.SysctlKinfoProcSlice("kern.proc.pgrp", pid)
	return len(members) != 0, err
}
