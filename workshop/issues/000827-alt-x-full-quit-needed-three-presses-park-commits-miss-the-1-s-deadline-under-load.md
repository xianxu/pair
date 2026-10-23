---
id: 000827
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bd6abadefc220a7959f7305cfc730a60b72a6435' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T23:31:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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
