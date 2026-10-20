---
id: 000695
status: codecomplete
deps: [pair#683]
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 4.91
collaboration-mode: collaboration
card_mirror: '769a19986eb4f711caf12828198ec1ede837e97d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T22:57:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 7.84
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* The plan is thorough, so the design buffer is 15%. Impl values are 40% of the v2 table.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.1  impl=0.2
item: greenfield-go-module   design=0.3  impl=0.32
item: smaller-go-module      design=0.05 impl=0.16
item: milestone-review       design=0.0  impl=0.2
item: cross-cutting-refactor design=0.1  impl=0.2
item: greenfield-go-module   design=0.2  impl=0.32
item: greenfield-go-module   design=0.5  impl=0.32
item: smaller-go-module      design=0.05 impl=0.2
item: greenfield-go-module   design=0.2  impl=0.24
item: cross-cutting-refactor design=0.2  impl=0.2
item: smaller-go-module      design=0.05 impl=0.2
item: milestone-review       design=0.0  impl=0.2
item: atlas-docs             design=0.05 impl=0.08
design-buffer: 0.15
total: 4.91
```

The items, in order:
1. M1: the shared preserving codec in strictjson, with #683's codec migrated onto it.
2. M1: the ThreadRecord binding and Bind.
3. M1: `LifecycleAddress`.
4. M1: the milestone review.
5. M1: rewiring resume, relaunch and recovery onto Bind.
6. M2: the lifetime lease package and its spawn-site wiring.
7. M2: Teardown (the tty and zellij witnesses, the sweep, the proof, the failure details, `ProvenAt`).
8. M2: recording the wrapper's tty under Couch.
9. M2: the orphan registry, `--resources` and `--reap-orphans`.
10. M2: the archive and replace proof funnel and the sidecar codec.
11. M2: the real-pty conformance test.
12. M2: the close review.
13. The atlas.

Revised after estimate-quality (2026-10-19). The lease package, the rewiring of resume, relaunch and recovery,
and the wrapper's tty were folded into other items, so each is now itemized. The milestone-review rows are
already at the method's scaled ceiling, which is 40% of 0.5h. The reviewer warned that review rounds may exceed
it.

## Plan

Durable plan: `workshop/plans/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown-plan.md` (moved from pair#684's plan §2, §3 address type, M1–M2).

- [x] M1 — Transcript binding on the record; Resume binds by one stat (P2); the two-kind `LifecycleAddress`
- [x] M2 — Lifetime lease, agent tty witness, `Teardown` with `TeardownProof`; archive and replace require a proof (P1); orphan registry

## Log


- 2026-10-19: closed — main integration 011ccaa4e brought #712 importing procutil in couch_test.go; dropped the branch duplicate (2ac8c028e), fixing the CI build; couchcore env/lease/teardown/spawn/retire tests ok (fixed-then-ship; full suite: sdlc-local/suite success on 2ac8c028 (TestScratch RAM disk)); review verdict: SHIP
- 2026-10-19: closed — main integration 2e3c72644: couch.go and warm_attach.go conflicts additive (SlotEnvironment/Scratch fields beside the lifetime lease); couchcore launch/env/lease/teardown tests ok (fixed-then-ship; full suite: sdlc-local/suite success on cd3cf1ac (WF_TEST_SCRATCH=off: the registered #715 flake fails on the TestScratch RAM disk, also on plain main)); review verdict: SHIP
- 2026-10-19: merge integration. Main (88540c4db) conflicted in couch.go and warm_attach.go, both additive: main's SlotEnvironment and Scratch fields sit beside the lifetime lease. The suite on 2e3c72644 then failed in attempt 13c132ae on couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees. Its helper (launcher.TestCouchLaunchPromptTransportHelper) hit a TempDir cleanup 'directory not empty', and sdlc mis-attributed it to the launcher package, so it was judged unattributable. The test is the registered flake #715 and is specific to the TestScratch RAM disk: on plain origin/main it failed 4/4 there and passed on the normal TMPDIR. The rerun on 2e3c72644 passed, but the earlier attempt still blocks that head, so this entry lands the explanation on a new head.

- 2026-10-19: closed — sdlc local-check suite: sdlc-local/suite success on f9d4f031 (full make test, APFS scratch, 8m); M1 FIX-THEN-SHIP, M2 SHIP after 5 rounds; TestTeardownProvesARealPtyTree live 6/6; mutation-checked fold, agent-witness, tty-guard and proof-stamp tests; review verdict: SHIP
- 2026-10-19: suite. Attempt 98a0a59b on 84b58903c failed in ten tests across couchcmd, historyarchive, launcher, sessioninventory, workbenchperftest, wrapcmd and zellijtest, plus write budgets. Cause: the TestScratch RAM disk was HFS+, with 1-second mtimes and no clone-based write accounting. Measured: the same tests failed there and passed on the normal TMPDIR on the same commit. The hub confirmed it (ariadne#474: sdlc now refuses a non-APFS scratch disk). After the disk was reformatted as APFS, the suite on 84b58903c passed with 0 failures, but sdlc keeps the earlier unexplained attempt against that head, so this entry lands the explanation on a new head.



- 2026-10-19: closed M2 — go test unsandboxed on 2f62802d9 (+accessor drop 3e0 dead-symbol fix): couchcore couchcmd threadrecord gccmd gcruntime historyarchive lifetimelease procutil readiness wrapcmd launcher strictjson ok; artifactpath dead-symbol ok, coverage fails only on 3 stale git-ignored local runtimebundle lua files; BR-11/BR-12 tests mutation-checked; TestTeardownProvesARealPtyTree live; review verdict: SHIP
- 2026-10-19: closed M1 — Round 3. BR-7 fixed by class: TranscriptBinding decodes owned members strictly and preserves unknown ones via its own UnmarshalJSON/MarshalJSON on strictjson (TestTranscriptBindingNestedUnknownMembersSurvive; duplicate/wrong type inside still refuse). Key narrowing pinned in TestLifecycleAddressSlotKeyMatchesFailureSlotKey. BR-4 deferral recorded at the receiver: pair#684 plan M2 (5d3d7f22f). Unsandboxed: couchcore 379s ok, couchcmd 114s ok, couchtty 62s ok, threadrecord ok, strictjson ok.; review verdict: FIX-THEN-SHIP
### 2026-10-18

- 2026-10-18: split from pair#684 on the hub's decision (delivery 51ef8069): #684's M1–M2 become this issue so pair#694's team lifecycle and #684's Converge both build from main. The design (already operator-approved under #684) moved here unchanged; #684's Revisions keep its history.
