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

- [x] Add the single-field 60-minute config; retain the existing max-car default.
- [ ] Verify JSON and existing provider config/status tests; publish the PR and record the required suite.
- [ ] Obtain suite and single close review; request operator local-gates verification through the hub before landing.
- [x] Record the TL-owned post-landing status and real-train conformance handoff (landing follows close and operator verification).

## Log

### 2026-10-19

Claimed and started from fresh main. TL dispatch e96c835f approves the one-file 60-minute pilot; #622 remains claimed pending a separate coordinated restart. Startup weave regenerated local-gates-guard.yml and merge-check.yml; reverted only those generated changes per dispatch (#699 pending). Baseline sdlc landing status reports trains off.

ARCH-DRY: use existing ariadne#449 config/parser, default car cap and train lifecycle without new runtime logic. ARCH-PURE/EVOLVE: this is read-only input to the existing typed parser, with no format change or rewrite path. ARCH-PURPOSE: the real ≥2-car conformance remains required after configuration lands. Existing train ownership, capacity, uncertainty, security and retention policies remain provider-owned; no new states, credentials, hot path or accumulating artifacts. ariadne#475 (departure slots) and ariadne#470 (grant hold) are related external edges, not implementation in this pilot.

### 2026-10-19 — candidate configuration

Added only train_interval_minutes=60; the provider retains its default max-car cap. JSON validation confirms exactly that field/value; git diff --check passes. Existing provider controls pass (2.067s): go test ./cmd/sdlc -run '^(TestParseLandingConfig|TestReadLandingScheduleAbsentIsOff|TestLandingScheduleSourceFailsToSerial|TestLandingPRLabelMarksLocalGates|TestLandingStatusShowsTrainsAndReadiness)$' -count=1 in the slot-private Ariadne. The config is read from main, so branch status remains off until landing; no live train acceptance claimed.

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
