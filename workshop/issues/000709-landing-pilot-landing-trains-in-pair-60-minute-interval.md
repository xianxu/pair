---
id: 000709
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '740c2ee436c57a4c1b8ff73ff0b613aca2844d93' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-19
