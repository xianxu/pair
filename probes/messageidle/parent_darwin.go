package main

import "golang.org/x/sys/unix"

// parentName reads the parent's command name from the kernel; running ps here
// would recurse into this shim.
func parentName(pid int) string {
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return ""
	}
	return unix.ByteSliceToString(info.Proc.P_comm[:])
}
