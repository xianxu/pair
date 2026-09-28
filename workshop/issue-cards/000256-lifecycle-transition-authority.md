---
id: '000256'
status: done
started: 2026-09-16T20:08:18-07:00
created: 2026-09-15
updated: 2026-09-17
estimate_hours: 5.44
actual_hours: 11.84
---

# Enforce lifecycle transition authority and outcome uncertainty

## Problem

Split from #255 on 2026-09-15 when the operator narrowed that issue to terminal abstraction. Preserve the non-terminal findings of the 2026-09-14 audit; baseline was HEAD `5ebb381f` plus then-uncommitted #250 work. Revalidate all cited code against current main before design. These findings do not establish the unexplained disconnect's cause.

2. **Record validation is not transition authority.** `ThreadStore.UpdateExistingThread` (`cmd/internal/couchcore/threadstore.go:264`) accepts arbitrary mutation callbacks. CAS, immutable-field checks and final validation protect coherent records, but do not require an authorized state/event transition. Production continuation recovery mutates lifecycle fields through this door. Direct mutations already existed in HEAD; #250 adds more guarded reconciliation. Model these as explicit transitions rather than assuming every guarded mutation is a bug.
4. **Observation uncertainty is lost in projection.** `ProcOps` defines Live/Dead/Unknown, but `ObserveRecordedProcesses` (`couchcore/actionableinventory.go:568`) drops unknown and identity-read errors. `ThreadEvidence.Live` retains only positive evidence; `ClassifyThread` can therefore present an unobservable incarnation as stale (`:269–283`). #250's new execution path preserves uncertainty better than this diagnostic projection. Unknown must not be treated as confirmed absence.

The same audit found loss of process/transport outcome detail: ptychild/child.go discarded PTY read errors, procutil and launcher reduced termination to integer exit codes, and launcher cleanup outcomes were not uniformly consumed. Coordinate diagnostic evidence with #253. Terminal input/output failure handling and terminal-state consequences belong to #255; this issue owns propagation into attachment/process lifecycle decisions. Avoid separate competing outcome types.
