---
id: 000813
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
---

# bound the extent of scribe's stdin-to-pty copy goroutine

## Problem

`cmd/internal/scribecmd/scribecmd.go:163-166` starts the stdin-to-pty copy with `go func() { _, _ = io.Copy(ptmx, stdin) }()` and never joins it. It is the only goroutine in `run` whose extent is not lexically bounded:

- the SIGWINCH propagator (`:102-124`) closes its channel and waits on `winchDone` from a deferred func;
- the SIGUSR1/SIGUSR2 pause handler (`:126-149`) does the same with `pauseDone`, under a comment reading "Same shape as the winch goroutine: unstopped, it leaks per Run call."

The winch defer carries a comment naming exactly the hazard an unjoined goroutine creates: it is registered after the `ptmx.Close` defer so that it runs before it (defers are LIFO), because otherwise the goroutine outlives `run` and can issue `pty.InheritSize` on a closed `ptmx` — and since the kernel recycles descriptors, that ioctl could resize an unrelated open file.

The copy goroutine has no such ordering. `run`'s deferred cleanup closes the master at `:92` once draining completes, while the copy can still be blocked reading `stdin` or writing `ptmx`. `ARCH-ORDER` wants the extent of concurrent work bounded lexically; here it is bounded only by the runtime.

This is filed as an ordering/hygiene issue and not as a defect, because `poll.FD` refcounting makes a use-after-close return `os.ErrFileClosing` rather than reaching a recycled descriptor — so the hazard the winch comment describes is closed by the runtime on this path, not by `run`. No misbehaviour has been observed or reproduced.

Surfaced while investigating #809 (a transient `open("/dev/ptmx)` ENXIO under concurrent pty churn). It is not causal there: #809's failure is in allocation, before any goroutine starts. Filed separately rather than folded in, on the hub's ruling that it is out of #809's scope.
