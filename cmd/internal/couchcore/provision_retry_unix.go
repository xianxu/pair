//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import "time"

// creationLeaseRetry returns a bounded delay after a busy flock. A released
// owner permits scheduling slack for inherited fork copies; a live or unknown
// owner retains the short refusal. elapsed always starts at the first attempt,
// so changing hints or owners cannot renew either budget.
func creationLeaseRetry(elapsed time.Duration, attempt int, released bool) time.Duration {
	budget := hostCreationSettle
	if released {
		budget = 2 * time.Second
	}
	remaining := budget - elapsed
	if remaining <= 0 {
		return 0
	}
	delay := 2 * time.Millisecond
	for i := 0; i < attempt && delay < 20*time.Millisecond; i++ {
		delay *= 2
	}
	if delay > 20*time.Millisecond {
		delay = 20 * time.Millisecond
	}
	if delay > remaining {
		delay = remaining
	}
	return delay
}
