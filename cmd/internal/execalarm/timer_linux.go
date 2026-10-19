package execalarm

import "golang.org/x/sys/unix"

func armTimer(seconds int) error {
	timer := unix.Itimerval{Interval: unix.NsecToTimeval(int64(seconds) * 1e9), Value: unix.NsecToTimeval(int64(seconds) * 1e9)}
	_, err := unix.Setitimer(unix.ITIMER_REAL, timer)
	return err
}

func unblockAlarm() error {
	var mask unix.Sigset_t
	mask.Val[0] |= 1 << (unix.SIGALRM - 1)
	return unix.PthreadSigmask(unix.SIG_UNBLOCK, &mask, nil)
}
