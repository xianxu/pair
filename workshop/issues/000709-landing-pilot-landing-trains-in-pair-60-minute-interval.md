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
flow: {kind: quick, provenance: inferred, spec: "6d428b55", done: "a39fa4db"}
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
train PR's merge commit, and its `sdlc landing status` line, and append them to this issue's Log.

## Done when

- `.sdlc/landing.json` is on pair main; `sdlc landing status` in pair shows trains on with a 60-minute interval.
- One real train with ≥2 cars has landed, with the evidence above logged; or any failure is filed against ariadne#449's
  follow-ups with that evidence.
- Then (operator, 10-19): once pair's first train proves it, the same `.sdlc/landing.json` goes to ariadne and
  parley.nvim, each as its own labeled PR (filed by the hub in those repos).

## Plan

- [ ] Add the single-field 60-minute config; retain the existing max-car default.
- [ ] Verify JSON and existing provider config/status tests; publish the PR and record the required suite.
- [ ] Request operator local-gates verification through the hub; obtain review/partial-landing disposition before landing.
- [ ] After landing, record main status and the first ≥2-car train evidence; close only after conformance.

## Log

### 2026-10-19

Claimed and started from fresh main. TL dispatch e96c835f approves the one-file 60-minute pilot; #622 remains claimed pending a separate coordinated restart. Startup weave regenerated local-gates-guard.yml and merge-check.yml; reverted only those generated changes per dispatch (#699 pending). Baseline sdlc landing status reports trains off.

ARCH-DRY: use existing ariadne#449 config/parser, default car cap and train lifecycle without new runtime logic. ARCH-PURE/EVOLVE: this is read-only input to the existing typed parser, with no format change or rewrite path. ARCH-PURPOSE: the real ≥2-car conformance remains required after configuration lands. Existing train ownership, capacity, uncertainty, security and retention policies remain provider-owned; no new states, credentials, hot path or accumulating artifacts. ariadne#475 (departure slots) and ariadne#470 (grant hold) are related external edges, not implementation in this pilot.
