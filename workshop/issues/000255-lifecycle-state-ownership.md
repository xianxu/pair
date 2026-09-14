---
id: 000255
status: open
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
---

# Enforce lifecycle and terminal state ownership

## Problem

The 2026-09-14 Pair/Couch audit found strong local state machines but incomplete structural ownership and cross-component behavioral enforcement. TTY bugs and an unexplained attachment disconnect motivated the audit. A concurrent output/switch test reproduced a data race; it does not establish the disconnect's cause.

Audit baseline: HEAD `5ebb381f`, plus the then-current uncommitted #250 recovery work. The principal terminal and store-boundary gaps predate that recovery delta. Recheck current code before implementation; cited lines locate the audited version.

### Findings

1. **Confirmed TTY data race and incomplete switch transaction.** `cmd/internal/couchtty/console.go:565` reads `p.replayCutoff` outside `mu`, while `onChunk` writes it under `mu` at line 1405. The temporary race test below failed on those exact accesses. `switchTo` publishes active/focus at lines 530–547 and performs takeover afterward; the operation worker reaches it while Run continues input/output handling (`operation_queue.go:64`, `console.go:2217`). A stale output-source decision can outlive a focus change. The data race is reproduced; stale-screen/input misrouting is an architectural interleaving risk, not a reproduced incident.
2. **Record validation is not transition authority.** `ThreadStore.UpdateExistingThread` (`cmd/internal/couchcore/threadstore.go:264`) accepts arbitrary mutation callbacks. CAS, immutable-field checks and final validation protect coherent records, but do not require an authorized state/event transition. Production continuation recovery mutates lifecycle fields through this door. Direct mutations already existed in HEAD; #250 adds more guarded reconciliation. Model these as explicit transitions rather than assuming every guarded mutation is a bug.
3. **Transport and process outcomes are discarded.** `ptychild/child.go:147` discards the terminal PTY read error; `procutil/procutil.go:122` and `launcher/osruntime.go:167` reduce process termination to integer exit codes. Console input EOF/error retires only its reader (`console.go:1547`); child input errors are discarded and host scanning advances without treating partial/failed writes as lifecycle outcomes (`console.go:747`, `:1159`). `launcher/lifecycle.go:102` also discards the typed cleanup return; durable receipts cover some paths but not every setup/direct failure.
4. **Observation uncertainty is lost in projection.** `ProcOps` defines Live/Dead/Unknown, but `ObserveRecordedProcesses` (`couchcore/actionableinventory.go:568`) drops unknown and identity-read errors. `ThreadEvidence.Live` retains only positive evidence; `ClassifyThread` can therefore present an unobservable incarnation as stale (`:269–283`). #250's new execution path preserves uncertainty better than this diagnostic projection. Unknown must not be treated as confirmed absence.
5. **Sequence coverage does not span the composition.** Local menu/reattach/orientation reducers have generated sequence tests; park tests include timeout followed by late success and stale attempts. The audit found no comparable composed console/lifecycle model spanning attach, switch, queued output, child exit, input failure, partial host write, operation completion and stop. Tests confined to a locked write transaction do not exercise the ownership decision made before entering it.

### Existing foundations to preserve

`ReduceMenu`, `AdvanceStartTransaction`, `AdvanceParkTransaction`, `AdvanceDelivery`, latest-request scheduling, generated reattach sequences, exact PID/start identities, supervisor lease, revision CAS/store journal, and Child-owned geometry are real enforcement. Console's current output paths already use `terminalMu`; this is not a claim that byte-level writes are all unlocked. `termcmd.paneWriter` provides a useful typed-door precedent, though its raw write result also needs explicit fault semantics if reused.

## Spec

Establish a shared vocabulary and make lifecycle/terminal models authoritative under ARCH-ORDER's structural, behavioral, and uncertainty requirements (ariadne#226 and ariadne#227). This ticket captures the audit and desired outcome; implementation requires a durable design and approval. Preserve independently surviving resources; avoid flattening the whole application into one global FSM.

### Vocabulary and ownership

Map each noun to actual symbols, identity, sole transition owner, states, accepted events, effects, invariants and termination/recovery behavior:

| Noun | Meaning |
| --- | --- |
| Thread | Durable logical work identity/history, surviving runtime processes. |
| Native session | Zellij workspace hosting panes and their processes. |
| Process instance | OS process identified by PID plus process-start identity. |
| Incarnation | Currently the recorded Pair helper instance, not the whole workspace. |
| Attachment | Helper/client/PTY relationship to an existing native session. |
| Terminal view | Selected input/output source, replay position, geometry and terminal modes. |
| Operation attempt | Correlated start, park, resume, continuation or recovery attempt. |
| Observation | Evidence about an exact external resource, including freshness and uncertainty. |

A process may host several state machines; an operation may coordinate several processes. Separate desired state, observed state and operation outcome where relevant. Define confirmed success, confirmed failure and unconfirmed outcome, permitted actions under uncertainty, and bounded reconciliation/escalation. Preserve partial progress; local rollback cannot undo completed external effects. Probes can fail and observations can become stale.

### Enforcement direction

- Give terminal focus, output eligibility, replay and takeover one defined transition ordering; restrict callers to that owner rather than adding only a lock around the one reported read.
- Restrict durable lifecycle updates to named transition APIs. Keep storage atomicity/CAS distinct from permission to transition.
- Return structured external outcomes as events, with exact process/attempt identities; preserve unknown through observation, projection and recovery.
- Test independent invariants after generated event sequences and force orderings at production scheduling boundaries. Reducer fuzzing alone cannot prove IO-shell wiring or cross-resource ordering.

### Related work

- #224 owns the typed console-writer door. Coordinate or expand its scope for whole focus/screen transactions; avoid a second parallel implementation.
- #250 owns stale-thread recovery and currently changes the audited surfaces. Reconcile this design against its final implementation.
- #253 owns default bounded attachment-disconnect telemetry. Reuse its structured outcomes; telemetry is evidence collection, not proof of the past incident's cause.

These are coordination references, not a claim that all work is blocked on all three. Set actual dependencies and child issues during design. No live-session repairs or production refactoring are authorized by this capture alone.

## Done when

- A canonical vocabulary/ownership map names real code components and explains independent thread, native-session, process and attachment lifetimes.
- The reproduced race is covered by a committed regression; the terminal transition design also prevents stale output/input ownership across switches.
- Authoritative lifecycle changes pass through named transition APIs; callers cannot use general mutation callbacks to bypass lifecycle rules.
- Failed/partial IO and unknown external outcomes retain their evidence and have explicit state transitions, safe actions and bounded reconciliation.
- Composed sequence tests enforce: attachment loss does not imply session death; stale attempts cannot mutate replacements; unknown cannot authorize destructive cleanup; partial effects are not erased by local rollback; released terminals accept no later output.
- Existing local FSM guarantees remain valid; focused integration/conformance evidence demonstrates production routing, and #224/#250/#253 ownership is resolved without duplicated work.

## Plan

- [ ] Revalidate findings against the latest implementation and coordinate with #224, #250 and #253.
- [ ] Claim/start-plan and author an approved durable design with vocabulary, owners, transition contracts, uncertainty semantics and implementation boundaries.
- [ ] Implement scoped changes with a permanent race regression and deterministic production-boundary sequence tests.
- [ ] Verify relevant race/integration/conformance checks, update atlas and close through SDLC.

## Log

### 2026-09-14 — Audit capture

Created at the user's request after the read-only audit. Left open; no implementation begun. Existing selected orientation, ptychild, termcmd and couchcore tests passed; the TTY audit's selected fake/reducer tests also passed. A temporary Go overlay test failed under the race detector with `onChunk` writing console.go:1405 and `switchTo` reading console.go:565. No Pair source or live sessions were changed by the audit.

The temporary files were `/tmp/pair-state-audit/{overlay.json,audit_test.go,race.log}`. They are not durable dependencies: the complete test source is preserved below. It uses existing package fixtures, concurrent fake-child output and forced switching; production's operation worker permits this concurrency. Promote it to a deterministic scheduling-boundary regression during implementation.

```sh
go test -race -overlay /tmp/pair-state-audit/overlay.json ./cmd/internal/couchtty -run '^TestAuditConcurrentOutputAndSwitch$' -count=1
```

```go
package couchtty

import (
 "strings"
 "sync"
 "testing"
)

func TestAuditConcurrentOutputAndSwitch(t *testing.T) {
 f := newFixture(t, 24, 80)
 waitFor(t, "console ready", func() bool { return strings.Contains(f.host.Written(), "\x1b[") })
 var wg sync.WaitGroup
 wg.Add(1)
 go func() {
  defer wg.Done()
  for i:=0;i<200;i++ { f.child.Feed([]byte("output\r\n")) }
 }()
 for i:=0;i<60;i++ { f.con.switchTo("c1",true,arrivalOrdinary) }
 wg.Wait()
}
```

The overlay maps an additional `cmd/internal/couchtty/audit_temp_test.go` to the temporary source above. Observed result: `WARNING: DATA RACE`, followed by test failure. This demonstrates the field race only; it neither reproduces the random disconnect nor proves every hypothesized interleaving.
