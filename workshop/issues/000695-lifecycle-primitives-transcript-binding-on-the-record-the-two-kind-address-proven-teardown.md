---
id: 000695
status: working
deps: [pair#683]
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '585cb33f5fef979dc7bcdb0ed95b4e80d2591c6b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T22:57:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown

## Problem

pair#684 (one lifecycle mechanism for every slot path) was planned with five milestones. Its first two are the shared primitives: the transcript binding stored on the record, the two-kind lifecycle address, and proven teardown. pair#694 (Couch teams) needs them on main before its team-lifecycle milestone. The hub decided (2026-10-18) to split them into their own issue. That keeps one issue per branch, with no landing in the middle of an issue, and lets both #684's remaining milestones and #694's M3 build from main.

## Spec

The design is pair#684's plan §2 (P1 teardown, P2 binding, §2a record compatibility) and §3's `LifecycleAddress`, moved into this issue's durable plan unchanged. In short:

- **Transcript binding (P2):** `ThreadRecord.Transcript` stores the agent's exact native ID. Resume binds by one stat, and Bind reads the tag's ledger file once to adopt a newer root. The record decoders tolerate and preserve unknown fields.
- **The address type:** `LifecycleAddress` has two kinds, a slot (`repo:N`) and a team (`TeamIdentity{CheckoutRoot, TeamDir, Name}`), with one `Key()` for the lease, store, failure records and orphan origin.
- **Proven teardown (P1):** a per-address lifetime lease held only by pair's own processes, the agent's controlling tty as a second witness, and zellij absence. `Teardown` returns a `TeardownProof`, and archive and replace require one. `LeftRunning` agent children go to an orphan registry, which `couch --resources` and `--reap-orphans` read.
- **Couch-only activation:** plain pair behaves exactly as today (the lease, tty and Inherit switch on only under `COUCH_*` env).

Prerequisite: pair#683 (failure persistence) lands first.

## Done when

- Resume binds by the stored exact ID (counted: 1 read, 1 stat, 0 listings), and a legacy record binds from its ledger once and then stores the ID.
- `Teardown` returns a proof or `ErrTeardownUnproven` with the holders named. Archive and Fresh replace refuse without a proof, and archive refuses an open park.
- `LifecycleAddress` validates both kinds, and two teams in one checkout get distinct keys.
- `TestPlainPairUnchangedWithoutCouch` passes, and older-build rewrites preserve the new fields.
- The tests named in the plan's M1 and M2 pass, and the full suite is green.

## Plan

Durable plan: `workshop/plans/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown-plan.md` (moved from pair#684's plan §2, §3 address type, M1–M2).

- [ ] M1 — Transcript binding on the record; Resume binds by one stat (P2); the two-kind `LifecycleAddress`
- [ ] M2 — Lifetime lease, agent tty witness, `Teardown` with `TeardownProof`; archive and replace require a proof (P1); orphan registry

## Log

### 2026-10-18

- 2026-10-18: split from pair#684 on the hub's decision (delivery 51ef8069): #684's M1–M2 become this issue so pair#694's team lifecycle and #684's Converge both build from main. The design (already operator-approved under #684) moved here unchanged; #684's Revisions keep its history.
