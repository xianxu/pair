---
id: '000255'
status: done
started: 2026-09-15T09:20:10-07:00
created: 2026-09-14
updated: 2026-09-15
estimate_hours: 18.025
actual_hours: 25.48
---

# Establish a faithful terminal abstraction for Couch and Pair

## Problem

Couch and Pair must present a defined terminal abstraction to inner programs. Composing UI, switching children and routing input must preserve that abstraction. The current combination of selective parsing, passthrough, interleaved control writes and bounded replay has incomplete contracts for terminal state and transformations. Correctness requires more than locks or consistent ownership of existing fields.

Concrete failures motivate the design: #252 reproduces control insertion inside split UTF-8 characters; #207 captures click-only mode emission followed by takeover without restored mouse modes. The historical terminal audit also identified:

1. **Confirmed TTY data race and incomplete switch transaction.** `cmd/internal/couchtty/console.go:565` reads `p.replayCutoff` outside `mu`, while `onChunk` writes it under `mu` at line 1405. The temporary race test below failed on those exact accesses. `switchTo` publishes active/focus at lines 530–547 and performs takeover afterward; the operation worker reaches it while Run continues input/output handling (`operation_queue.go:64`, `console.go:2217`). A stale output-source decision can outlive a focus change. The data race is reproduced; stale-screen/input misrouting is an architectural interleaving risk, not a reproduced incident.
5. **Sequence coverage does not span the composition.** Local menu/reattach/orientation reducers have generated sequence tests; park tests include timeout followed by late success and stale attempts. The audit found no comparable composed console/lifecycle model spanning attach, switch, queued output, child exit, input failure, partial host write, operation completion and stop. Tests confined to a locked write transaction do not exercise the ownership decision made before entering it.

Those audit findings refer to baseline `5ebb381f` plus then-uncommitted #250 recovery work; revalidate before implementation. The race was reproduced, but neither it nor the other terminal findings establishes the disconnect's cause.
