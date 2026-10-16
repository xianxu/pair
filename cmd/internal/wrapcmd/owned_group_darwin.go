package wrapcmd

import "syscall"

// groupHasLiveMember reports whether the owned group still has a running
// member, so cleanup ends as soon as it is empty instead of always spending
// the grace. Darwin answers kill(-pgid, 0) with EPERM once only zombies remain
// (our unreaped leader, or members mid-exit) and ESRCH once it is empty.
func groupHasLiveMember(pgid int) bool { return syscall.Kill(-pgid, 0) == nil }
