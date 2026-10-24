---
id: 000852
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
severity: high
card_mirror: 'f997a49a12deb5562c7a50db50023b9383b1ccda' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T18:13:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "f17ff17f", done: "c8321a29"}
---

# Park rollback retries fail on main

## Problem

TestLeaveParkRetriesProvenRollbackWithinOneQuit fails in isolation on both the #841 executable head and pure origin/main. It also blocked #841's recorded affected check. This is a main regression requiring root-cause investigation; #841 changes bell display rendering and does not change park code.

## Spec

Restore reliable proven-rollback retries within one Full quit, preserving the intended fresh/recovery drive contract. Identify the cause rather than increasing timeouts, suppressing errors or retrying the test. Investigate #827's park retry implementation and later interactions without assuming the cause from the symptoms.

## Done when

- The root cause is documented with a deterministic regression control.
- The focused test passes for both /15 and /30 with count=1, with expected parked outcomes and no unexpected recovery drives.
- Appropriate race and affected checks pass, without blanket ignored errors or timeout inflation.

## Plan

- [x] Compare the unchanged focused count=1 /15 and /30 test at #827 landing 25905e0ecc875ce080ab401cfb445f54fcfcc20b and pre-#819 46270746147eb5d9048ab1107b1df19339f7b42e; retain logs and follow the first failing boundary.
- [x] Trace the failing deadline/drive through Leave, parkJoined, completion observation and store coordination; establish one measured root cause before choosing a fix. Ask #827's owner about budget intent.
- [x] Add a deterministic failing control for that cause, make the smallest correction preserving production deadlines, retry cap and uncertainty handling, then run focused N/2N and race controls.
- [x] Audit reader inventory and prepare the verified test-only patch for push, recorded affected and close.

Downstream gates remain outstanding until their own result: recorded affected pass, mandatory fresh close review and landing. Reconcile #841 only after main is green.

Quick-flow investigation; if the root cause requires a new mechanism or exceeds the shell, revise the plan and obtain TL approval before code. ARCH-PURE/ARCH-ORDER: prefer existing injected deadline/IO seams for controlled ordering rather than real-time assertions. ARCH-CONSTRAINTS: preserve production 60s whole-quit and per-completion budgets, concurrency cap and counted N/2N drive assertions; no timeout bump or flake registration. ARCH-DRY: reuse the existing park coordinator/futures and stateful lifecycle fixture. No persisted-format or user-facing state changes are currently proposed; production behavior and resource lifetimes remain the subject of investigation.

- [ ]

## Log

### 2026-10-23 — Root measurement and approved test-only correction
- 2026-10-23: closed — Test-only fixture correction; no new architectural surface. Tiny-budget/missing-proof controls RED before 8.670s then GREEN6.154s; matching proof finalization traced at1.10-1.13s against real1s. Focused park/retry/joined-future and N15/30 exact counts PASS36.421s, race PASS34.706s; whole-budget and caller cancellation retained, lint PASS5.738s, inventory PASS immediately before close, diff clean. Production budgets untouched. Recorded affected running on pushed44cde4961; required pass before landing. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Both historical count=1 comparisons fail: #827 landing 25905e0ecc875ce080ab401cfb445f54fcfcc20b package43.368s (/15 pass16.42s; /30 unexpected recovery [1 2] instead of [1],26.58s), and pre-#819 46270746147eb5d9048ab1107b1df19339f7b42e package41.865s (same /30 failure25.44s; /15 pass16.14s). Couchcore has no diff between these pins. Logs: 827-repro.jsonl and pre819-repro.jsonl in the retained scratch root.

Disposable main-tree tracing then measured matching completion proof already present, followed by applyCompletion/FinalizePark taking1.125574875s (b006) and1.102143834s (b00a), both returning context deadline exceeded with the real1s completion context expired. The /15 assertion failed with [fresh fresh recovery] instead of [fresh fresh], package22.296s; main-finalization.jsonl preserves the trace. The fixture's injected2.5s commit clock is unrelated to this real deadline. #827 author confirms this test measures drive counts, not host IO latency. Root cause: incidental wall-clock expiration during serialized finalization in a synchronous-completion fixture.

TL approved test-only correction (submitted 5300a5cd-9dec-47f7-861a-cdb52e7c3973): inject the existing completionDeadline and leaveParkDeadline seams, control cancellation and spend completion only when proof is missing; retain whole-budget/cancel controls and add tiny-budget/missing-proof controls. Production budgets and logic remain untouched. The shared lint catalog's sub-second deadline rule does not catch this1s instance; TL explicitly waived a peer lint extension for this issue.

### 2026-10-23 — Baseline comparison requested by ops:0

The same command ran once on each Git-pinned tree, after the standard make runtimebundle-generate preparation:

```sh
go test -json ./cmd/internal/couchcore -run '^TestLeaveParkRetriesProvenRollbackWithinOneQuit$' -count=1 -timeout=90s
```

- #841 executable head d30f7c517f25996e15b8bf190d2ad6327e7bd4a5: FAIL, package 65.102s. /15 failed after 18.23s at leave_park_test.go:299: couch-000000000000b000 drives=[1 1 2], want [1 1]. /30 failed after 46.52s with the same unexpected third recovery drive for couch-000000000000b00f. Values mean fresh, fresh, recovery; expected fresh, fresh.
- Pure origin/main f109640a6ad2477df1629ea9b1238d0b90a3f634: FAIL, package 61.055s. /15 failed after 20.72s at line284: Full quit incomplete, 14 parked and 1 unresolved (b001), context deadline exceeded after one drive. /30 failed after 39.97s: 28 parked and 2 unresolved (b00b, b017), same timeout after one drive.
- Both runs used isolated detached diagnostic trees and separate TMPDIR/GOTMPDIR under /Volumes/TestScratch/pair841-leave-repro-gub_n4rh. No source edits were made. JSON logs are leave-repro.jsonl and main-leave-repro.jsonl there; trees are tree and main-tree. Retained locally for inspection.
- The fixture uses alternatingClock with a 2500ms step, CompletionTimeout 1s, PollInterval 1ms, parallelism4 and one failed publication for every third session. These are observations, not a diagnosed cause.
- Earlier recorded affected attempt e5a7db4c973ffa5a5ff24dae9232b02b on d30f7c51 failed /15 at line284 (14 parked, 1 unresolved), package445.134s. Log SHA256 c3bacd32bd9aa1fd007370840ad07ec85903f8c8b8040d922ed1e362cff3592d. Terminal escapes preceding the failure header caused SDLC to classify it as unattributable-failure; ops:0 filed that independent parser gap as ariadne#582. Script groups did not run after the Go failure.
- Filed unclaimed at ops:0's direction for urgent independent dispatch. No flake registered, no affected retry and no park fix on #841.

### 2026-10-23 — Deterministic regression and focused qualification

Tiny-budget and missing-proof controls RED before fixture correction (8.670s): tiny duration caused [fresh fresh recovery], and missing proof exhausted real deadlines. Both GREEN after installing the existing injected seams with caller-preserving cancellation (6.154s). Completion wait now spends its own cancellation only on a missing-proof poll; whole-quit time is explicitly driven by the existing budget test. Production files, configured durations and retry policy unchanged. Broader LeavePark, retry invariants and joined-future controls PASS36.421s; same set with race PASS34.706s, including N=15/30 exact drive and trigger counts plus exhaustion, whole-budget end and caller cancellation. Lint catalog PASS5.738s, reader inventory PASS, diff whitespace clean. Lessons updated; no architectural/product surface added. Push and recorded affected precede landing; no passing recorded result claimed yet.

## Revisions

### 2026-10-23 — Fixture-only root cause and release gates

The measured defect is the test fixture's use of real completion/whole-quit deadlines, not a production state transition. Apply TL-approved existing deadline seams only, with missing-proof and caller-cancellation controls. Preserve original Done when and production timing policy. Split final preparation from downstream gates so their results are recorded only after execution.
