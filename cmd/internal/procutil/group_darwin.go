//go:build darwin

package procutil

import "golang.org/x/sys/unix"

// ProcessGroupMembers resolves an ambiguous signal result using only the
// specified group. Zombies count as members; an inspection error is unknown.
// Darwin killpg can report EPERM for an existing group with no signalable
// members: XNU bsd/kern/kern_sig.c killpg1 filters SZOMB before nfound's check.
func ProcessGroupMembers(pid int) (bool, error) {
	members, err := unix.SysctlKinfoProcSlice("kern.proc.pgrp", pid)
	return len(members) != 0, err
}
