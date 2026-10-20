---
id: 000709
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '2ecbc94060a802b3ee10565f1376e1697aae0f6a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T20:14:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "24f2b07f", done: "3d438487"}
---

# landing: pilot landing trains in pair (60-minute interval)

## Problem

ariadne#449 (landed, PR 324) adds landing trains: ready merges in one repo are batched and tested with one recorded suite and
merged together, instead of one suite per change. Trains are off unless a repo has `.sdlc/landing.json`. The operator approved
a pilot in pair (2026-10-19): pair's suites are the long pole in the landing queue.

## Spec

Add `.sdlc/landing.json` to pair with `train_interval_minutes: 60` (exact schema per ariadne#449's issue and
`sdlc landing --help`). The PR touches landing configuration, so the local-gates guard fails it until the operator applies
the `local-gates-verified` label; ask the hub, never apply the label yourself. After it lands, the first pair train with at
least 2 low-risk cars is the conformance check: for each car record `gh pr view N --json state,mergeCommit,headRefOid`, the
train PR's merge commit, and its `sdlc landing status` line, and record them in the TL-owned test-economy project (revised acceptance below).

## Done when

- The candidate adds only `.sdlc/landing.json` with a 60-minute interval and the existing default car cap; required suite and close review pass.
- Operator verification is obtained through the hub before landing; after landing, `sdlc landing status` on pair main shows trains on at 60 minutes.
- The first real ≥2-car train is handed to the TL as a follow-up, recorded in the test-economy project with per-car PR state/merge/head, train PR merge commit and landing status. The TL files a bug for any failure and owns follow-on ariadne/parley rollout.

## Plan

- [x] Add the single-field 60-minute config; retain the existing max-car default and document operator usage in README/atlas.
- [x] Verify JSON and existing provider config/status tests; publish the PR and record the required suite.
- [x] Prepare the single close review with suite evidence and operator local-gates verification obtained through the hub before landing.
- [x] Record the TL-owned post-landing status and real-train conformance handoff (landing follows close and operator verification).

## Log

### 2026-10-19
- 2026-10-19: closed — JSON validation and provider config/status/label controls pass. Recorded suite exit0 on d0b77effdafdcd7874fce0a8acb4384e6deb7edf, 8m57s, log sha256 prefix2a00c0f3d785; descendants docs-only. BR-1 corrected across README config path/key/value, default cap, main read, status, operator label and TL owner with atlas link; diff check passes. Operator label confirmed on PR469. TL owns real multi-car follow-up per bd560b95; main status follows merge. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Claimed and started from fresh main. TL dispatch e96c835f approves the one-file 60-minute pilot; #622 remains claimed pending a separate coordinated restart. Startup weave regenerated local-gates-guard.yml and merge-check.yml; reverted only those generated changes per dispatch (#699 pending). Baseline sdlc landing status reports trains off.

ARCH-DRY: use existing ariadne#449 config/parser, default car cap and train lifecycle without new runtime logic. ARCH-PURE/EVOLVE: this is read-only input to the existing typed parser, with no format change or rewrite path. ARCH-PURPOSE: the real ≥2-car conformance remains required after configuration lands. Existing train ownership, capacity, uncertainty, security and retention policies remain provider-owned; no new states, credentials, hot path or accumulating artifacts. ariadne#475 (departure slots) and ariadne#470 (grant hold) are related external edges, not implementation in this pilot.

### 2026-10-19 — candidate configuration

Added only train_interval_minutes=60; the provider retains its default max-car cap. JSON validation confirms exactly that field/value; git diff --check passes. Existing provider controls pass (2.067s): go test ./cmd/sdlc -run '^(TestParseLandingConfig|TestReadLandingScheduleAbsentIsOff|TestLandingScheduleSourceFailsToSerial|TestLandingPRLabelMarksLocalGates|TestLandingStatusShowsTrainsAndReadiness)$' -count=1 in the slot-private Ariadne. The config is read from main, so branch status remains off until landing; no live train acceptance claimed.

### 2026-10-19 21:01 PDT — suite and operator verification confirmed

Required recorded suite passed on d0b77effdafdcd7874fce0a8acb4384e6deb7edf: make PAIR_GO_TRIMPATH=-trimpath test, exit0, execution8m57s after424s queue; log .git/sdlc/local-checks/d0b77effdafdcd7874fce0a8acb4384e6deb7edf/suite.log, sha256 prefix2a00c0f3d785, recorded sdlc-local/suite. Later commits are documentation only. Operator label local-gates-verified is present on PR469; latest local-gates-guard and verdict pass, as does merge-check. Canonical TL dispatch e517c822 directs close then sdlc merge --yes. No self-label or CI bypass. First real multi-car train remains TL-owned per the recorded acceptance revision; after landing this slot checks main's 60-minute status.

### 2026-10-19 21:04 PDT — BR-1 documentation correction

Close round1 returned FIX-THEN-SHIP with Important BR-1 and did not finalize because the ledger remained blocking. Added root README coverage for the full configuration surface: path/key, 60-minute value, inherited car limit, main-ref read, status command, operator label and TL conformance owner, with an atlas link. Added the preventative lesson. No behavior/config/test change; suite evidence remains valid. Following the binary's explicit instruction to rerun close for disposition.

## Revisions

### 2026-10-19 20:17 PDT — TL owns post-landing conformance

TL disposition bd560b95-151f-4607-81e4-f27f96071a67 supersedes the original acceptance below: one suite, one close review, then merge after the operator label. Close covers the configuration; the TL watches pair landing status, records the first ≥2-car train in the test-economy project, files a bug if it misbehaves, and owns subsequent repo rollout. This removes the circular dependency between whole-issue close and post-landing acceptance without adding a milestone split. Spec, Done when and Plan now reflect that handoff.

Original acceptance preserved:


- `.sdlc/landing.json` is on pair main; `sdlc landing status` in pair shows trains on with a 60-minute interval.
- One real train with ≥2 cars has landed, with the evidence above logged; or any failure is filed against ariadne#449's
  follow-ups with that evidence.
- Then (operator, 10-19): once pair's first train proves it, the same `.sdlc/landing.json` goes to ariadne and
  parley.nvim, each as its own labeled PR (filed by the hub in those repos).

PR https://github.com/xianxu/pair/pull/469 contains configuration head d0b77effdafdcd7874fce0a8acb4384e6deb7edf. Required recorded suite is queued in sdlc-local-1133542116, log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/e5b9ea2dd1d90ef283cd8304af62865e/run.log; preserve and poll session 61967 rather than starting a duplicate. Hub requested the operator label; no label or gate waiver applied by this slot.

### 2026-10-19 — waiting for operator verification

PR469 local-gates-guard and local-gates-guard/verdict fail as expected: landing configuration requires the operator’s local-gates-verified label. The hub has requested it and will notify this slot. No guard bypass or self-applied label. The required suite remains next in the machine FIFO (session61967); the local check runs independently and must be polled before close. After suite PASS: finish Plan ticks, single sdlc close, then merge only after operator verification.

### 2026-10-19 21:04 PDT — review documentation refinement

BR-1 adds README coverage to the completed plan; acceptance and runtime scope are unchanged. The original plan named only the config and existing default car limit.
