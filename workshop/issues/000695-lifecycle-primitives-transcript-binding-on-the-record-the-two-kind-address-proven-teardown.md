---
id: 000695
status: open
deps: [pair#683]
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '381e903e43e68acc38aa3fb63232bcb118067f35' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-18
