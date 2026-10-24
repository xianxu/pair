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

- [ ] Compare the unchanged focused count=1 /15 and /30 test at #827 landing 25905e0ecc875ce080ab401cfb445f54fcfcc20b and pre-#819 46270746147eb5d9048ab1107b1df19339f7b42e; retain logs and follow the first failing boundary.
- [ ] Trace the failing deadline/drive through Leave, parkJoined, completion observation and store coordination; establish one measured root cause before choosing a fix. Ask #827's owner about budget intent.
- [ ] Add a deterministic failing control for that cause, make the smallest correction preserving production deadlines, retry cap and uncertainty handling, then run focused N/2N and race controls.
- [ ] Audit reader inventory, push, run recorded affected, close through mandatory fresh review and land. Reconcile #841 only after main is green.

Quick-flow investigation; if the root cause requires a new mechanism or exceeds the shell, revise the plan and obtain TL approval before code. ARCH-PURE/ARCH-ORDER: prefer existing injected deadline/IO seams for controlled ordering rather than real-time assertions. ARCH-CONSTRAINTS: preserve production 60s whole-quit and per-completion budgets, concurrency cap and counted N/2N drive assertions; no timeout bump or flake registration. ARCH-DRY: reuse the existing park coordinator/futures and stateful lifecycle fixture. No persisted-format or user-facing state changes are currently proposed; production behavior and resource lifetimes remain the subject of investigation.

- [ ]

## Log

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
