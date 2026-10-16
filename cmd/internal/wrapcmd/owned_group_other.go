//go:build !darwin && !linux

package wrapcmd

import "syscall"

// groupHasLiveMember is conservative elsewhere: a zombie leader may answer, so
// cleanup can spend the whole grace before SIGKILL.
func groupHasLiveMember(pgid int) bool { return syscall.Kill(-pgid, 0) == nil }
