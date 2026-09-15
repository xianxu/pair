---
id: 000256
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Enforce lifecycle transition authority and outcome uncertainty

## Problem

Split from #255 on 2026-09-15 when the operator narrowed that issue to terminal abstraction. Preserve the non-terminal findings of the 2026-09-14 audit; baseline was HEAD `5ebb381f` plus then-uncommitted #250 work. Revalidate all cited code against current main before design. These findings do not establish the unexplained disconnect's cause.

2. **Record validation is not transition authority.** `ThreadStore.UpdateExistingThread` (`cmd/internal/couchcore/threadstore.go:264`) accepts arbitrary mutation callbacks. CAS, immutable-field checks and final validation protect coherent records, but do not require an authorized state/event transition. Production continuation recovery mutates lifecycle fields through this door. Direct mutations already existed in HEAD; #250 adds more guarded reconciliation. Model these as explicit transitions rather than assuming every guarded mutation is a bug.
4. **Observation uncertainty is lost in projection.** `ProcOps` defines Live/Dead/Unknown, but `ObserveRecordedProcesses` (`couchcore/actionableinventory.go:568`) drops unknown and identity-read errors. `ThreadEvidence.Live` retains only positive evidence; `ClassifyThread` can therefore present an unobservable incarnation as stale (`:269–283`). #250's new execution path preserves uncertainty better than this diagnostic projection. Unknown must not be treated as confirmed absence.

The same audit found loss of process/transport outcome detail: ptychild/child.go discarded PTY read errors, procutil and launcher reduced termination to integer exit codes, and launcher cleanup outcomes were not uniformly consumed. Coordinate diagnostic evidence with #253. Terminal input/output failure handling and terminal-state consequences belong to #255; this issue owns propagation into attachment/process lifecycle decisions. Avoid separate competing outcome types.

## Spec

Make lifecycle transitions authoritative without flattening thread, native-session, process, incarnation and attachment into one global state machine. Map each resource to actual symbols, exact identity, transition owner, accepted events, effects and termination/recovery rules. Preserve independently surviving resources.

- Replace general lifecycle-field mutation with named transition APIs; validation, revision CAS and storage atomicity do not themselves authorize transitions.
- Separate desired state, observation and operation outcome. Preserve Live/Dead/Unknown and identity-read errors through projection, classification and recovery.
- Carry structured process/attachment outcomes and exact attempt identities; distinguish confirmed success, confirmed failure and unconfirmed outcomes, including partial progress.
- Retain existing reducers, transaction journals, supervisor leases and PID/start identities. Reconcile against completed #250 recovery and #253 telemetry instead of duplicating them.

## Done when

- Canonical resource/ownership mapping explains independent thread, session, process and attachment lifetimes.
- Lifecycle mutations use named transition APIs, with tests proving arbitrary callbacks cannot bypass transition rules.
- Unknown observations cannot become confirmed absence or authorize destructive recovery; stale attempts cannot mutate replacements.
- Process/attachment errors and partial outcomes preserve evidence and lead to defined reconciliation behavior.
- Composed lifecycle tests establish attachment loss does not imply session death and retain existing recovery guarantees; #253 and #255 outcome ownership is explicit.

## Plan

- [ ] Revalidate the preserved audit findings against current code and coordinate #250/#253/#255.
- [ ] Claim/start-plan and obtain approval for a durable lifecycle design and scoped implementation boundaries.
- [ ] Implement named transitions and structured outcomes with production-boundary sequence/fault tests.
- [ ] Verify, document ownership and close through SDLC.

## Log

### 2026-09-15 — Scope extracted from #255

Preserved the generic lifecycle authority, observation uncertainty and process/attachment outcome findings while #255 becomes the terminal-abstraction issue. No implementation, diagnosis of the past disconnect, or new blocking dependency is asserted by this split.
