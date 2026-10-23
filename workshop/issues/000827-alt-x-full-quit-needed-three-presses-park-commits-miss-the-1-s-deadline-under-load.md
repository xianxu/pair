---
id: 000827
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 2.691
collaboration-mode: collaboration
card_mirror: 'c206cdc111f9b4790d139a90dc1bc84fe4682722' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T23:31:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Alt+x full quit needed three presses: park commits miss the 1 s deadline under load

## Problem

## Spec

Operator report (10-22 full shutdown): **Alt+x (park everything and quit Couch) timed out on some slots; it took
three Alt+x presses to shut everything down.** The machine was loaded (load average 40+, 30-odd slots).

Lead (TL reading, unverified): `couchcore/park.go` sets `parkCommitSoftTarget = 100ms` and `parkCommitDeadline =
1s` for each slot's park commit; under load some slots likely miss the 1 s commit and are left unparked, so the
operator has to repeat Alt+x.

Fix: measure park commit times across a full fleet under realistic load (pair#738-style counted timing), derive the
deadline from the measurement with a re-measure note (ARCH-CONSTRAINTS), and make one Alt+x finish the job:
slots that miss the first deadline get retried within one bounded overall budget, and anything still unparked is
named in a single summary (with its recovery) instead of silently needing another press.


**Operator decision (10-22, night):** "there's no reason to set this this tight. I think 5 to 10 seconds is perfectly
fine." Set the per-slot park commit deadline in that range (measurement still recorded to confirm it). Current
parallelism, for context: parks fan out up to `LifecycleParallelism` = half the CPU cores (operator decision 10-07),
so slots park in parallel batches, not all at once nor one by one.

## Done when

- One Alt+x parks a fleet of 30 fake slots where some commits take 2–3 s (test), or names exactly the ones it
  couldn't park.
- The per-slot deadline is derived from measured park commit times, with the measurement recorded.

## Plan

Durable plan: [one-gesture full quit](../plans/000827-one-gesture-full-quit-plan.md).

- [ ] Measure private real-store commit timings at15/30 slots before changing the threshold; choose within operator5–10s, stop if evidence contradicts it.
- [ ] Retain fan-out/gates/nonce recovery; one proposed60s scheduling budget and at most3 safe drives per target. Synchronous durability cannot be forcibly interrupted.
- [ ] Confirmed Alt+x automatically handles eligible stragglers, exits only after verified success, or leaves one exact unresolved-slot/recovery report.
- [ ] Deterministic30-slot/N2N tests, physical shared-harness4/8-slot Claude/Codex check, normal recorded verification and close.


## Log

### 2026-10-22


### 2026-10-22 — claimed and plan proposed

TL ee975c2b dispatched #827 while #759 awaits operator review. Card was unowned;
claimed pair:6 and ran start-plan. #759 remains untouched at
6019be87f98b434429c17d2787802abf6c98ad2e. Start-plan catch-up refreshed sibling
ariadne and seeded an unrelated merge-check workflow diff (ariadne#557 class).
Saved it at /Volumes/TestScratch/p827-plan.ogi77579/merge-check-seeded.patch,
restored only that workflow, and reran start-plan successfully. No seeded change
is included on this issue branch.

Read-only trace confirms the1s limit is a POST-COMMIT elapsed check, followed by
known-unpublished rollback; it is not a preemptive timer. Leave has one pass and
no whole-operation deadline. Existing thread/park nonce ownership and Recover
must govern any retries. Raising the shared constant must not accidentally
change relaunch's separate1s diagnostic observation window.

Proposed full-flow plan records a measurement-first10s provisional threshold,
one60s overall scheduling/context budget, at most3 safe per-target drives with
current fan-out, same-identity/nonce fences, exact partial reporting and no force
quit on unresolved state. Non-cancellable durability remains joined and can
exceed the logical deadline; no hard wall-time guarantee. No measurement numbers
or operator-incident causal conclusion are invented. No implementation or native
run started. Plan goes to TL before change-code.


### 2026-10-22 — plan approved

TL3cbb3359 approved the measurement-first plan, conditional10s commit threshold,
60s overall scheduling budget and up to3 safe drives, exact partial recovery
summary, unchanged fan-out and separate1s relaunch diagnostics. The60s is not a
hard process-exit ceiling; synchronous durability stays owned/joined. Proceed to
change-code, then measure before changing the threshold. No #759 changes.


### 2026-10-22 — full plan-quality round1 findings

Pinned full flow because the planned controller exceeds the quick shell. Gate
round1 raised PQ-1 (cancellation ownership) and PQ-2 (function-level test strategy).
Both are valid. Existing parkFuture.Await can return before its producer; the
proposed Leave-only parkJoined/recoverJoined paths retain the exact submitted
future and use AwaitJoined, keeping the thread hold/fan-out unit until done.
Ordinary callers keep existing cancellation behavior. Updated the durable plan
with named risky functions, adversarial class strategies and explicit unreadable
snapshot reporting. No code/measurement started while the gate is blocked.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md. Method A only; calibration is marked stale/provisional by
sdlc estimate-source. Existing controller, store, lifecycle model, console and
native harness make this familiar territory; no novel library/stack discovery
or new dependency is needed.

Derivation from v2/v2.1 table, with thorough-spec design discount0.2 and v3.1
implementation scale0.40 applied once:

- Measurement/threshold extension: smaller Go base design0.2/impl0.5 ->0.04/0.20.
- Joined lifecycle retry integration: API integration base2.0/1.5 ->0.40/0.60.
- Existing console completion/reporting: TUI base1.0/0.8 ->0.20/0.32.
- Shared native fixture extension: smaller Go base0.3/0.5 ->0.06/0.20.
- Native key/park conformance discovery: real API base0/0.6 ->0/0.24.
- Atlas and measurement documentation: docs base0.1/0.2 ->0.02/0.08.
- One close review/fixes: review base0.1/0.5 ->0.02/0.20.

Design subtotal0.74 with15% buffer=0.851; implementation subtotal1.84 at familiar1.0;
combined2.691 focused ship hours. Machine FIFO time is not a primitive estimate.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.20
item: api-integration design=0.40 impl=0.60
item: tui-screen design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.691
```

## Revisions

### 2026-10-22 — plan gate and approved joined ownership refinement

TL f4a757ee confirmed the Leave-only exact-future joined adapters; ordinary waits
and recovery authority remain unchanged. Plan-quality round2 CLEAN, both PQ-1 and
PQ-2 addressed. Derivation above follows only after that plan acceptance. Full
implementation remains measurement-first; no production budget changed yet.
