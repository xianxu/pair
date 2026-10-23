---
id: 000830
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '25a30036bbf0cdc9e14a4abcf0fdafe25b0b2057' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T07:18:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d12da403", done: "f6148d87"}
---

# Global Alt+x marks parked slots 'asleep'; one 'wake all' brings back exactly those

## Problem

## Spec

Operator idea (10-22, night): when the operator does the **global Alt+x** (park everything and quit), Couch should
mark the slots it parks that way as **asleep** ("put to sleep" during a restart), a flag on the parked slot distinct
from a slot the operator parked individually. That also answers the operator's earlier observation that "parked"
mixes two kinds: slots not in use, and slots that were merely running when parked.

- Global Alt+x (and `couch --drain`-driven full shutdowns) records each parked slot as asleep, with the time; an
  individual park, or a slot already parked, stays plain parked.
- On the next start, asleep slots are what a **"wake all"** action (one explicit operator or TL gesture, consistent
  with pair#739's no-automatic-resume rule) brings back; plain parked slots stay parked. tl-lead's fleet resume
  (ariadne#564) uses the asleep set instead of guessing from issue ownership.
- Waking or resuming a slot clears its flag; the switcher shows asleep distinctly from parked.
- Related: pair#827 (Alt+x must finish in one press), pair#739 (cold start stays manual).


**Operator refinement (10-22, night): wake in two stages.** The operator's one gesture wakes only the **asleep hubs**.
Each hub then wakes its own asleep worker slots: **first those with pending work, then the remaining asleep slots**.
So "wake all" at the Couch level means "wake the asleep hubs"; the per-hub ordering lives in the hub's fleet resume
(ariadne#564).

## Done when

- After a global Alt+x, previously live slots show asleep and previously parked ones show parked (test).
- The operator's wake gesture resumes only asleep hubs (test); each hub then wakes its asleep workers, pending-work first (test with ariadne#564's resume); individually parked slots are untouched.
- The flag survives a full restart and clears on wake.

## Plan

Durable plan: `workshop/plans/000830-global-alt-x-marks-parked-slots-asleep-one-wake-all-brings-back-exactly-those-plan.md` (awaiting TL approval).

- [ ] T1 record field `asleep` (SleepMark, born additive)
- [ ] T2 pure `ThreadAsleep` predicate (nonce-bound)
- [ ] T3 BeginPark cause; Leave fresh drives mark asleep; resume/reattach clear
- [ ] T4 summary + switcher shows asleep
- [ ] T5 wake operation (hubs only): switcher key, `couch --wake`, `couch --asleep --json`
- [ ] T6 native fleet acceptance (Claude, Codex)
- [ ] T7 atlas, lessons, ariadne#564 contract

## Log

### 2026-10-23

- pair:6 claimed and planned. The mark is written in BeginPark's CAS and bound to the park nonce; asleep is derived (receipt nonce = mark nonce), so every park-ending path and older builds leave it inert (ARCH-ORDER, ARCH-EVOLVE). It is a top-level member because nested park types are strict. tl-lead reads `couch --asleep --json`. `couch --drain` (#722) is not implemented yet; the cause is a Leave parameter so it can choose.

### 2026-10-22
