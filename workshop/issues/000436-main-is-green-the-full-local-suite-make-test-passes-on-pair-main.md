---
id: 000436
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'cfa553d8fa9f9ce2fc384bff4a84e7fb2b1d880d' # card fields mirrored from issue-cards; edit via sdlc
---

# Main is green: the full local suite (make test) passes on pair main

## Problem

pair's CI on main runs only the zellij conformance job (green since pair#324). The full local suite is not green on
main: on 10-10, pair:1 found that couchcore takes over 15 minutes and `TestSpawnComposesProductionPairRegistrationBoundary`
fails, both reproducing on main. While main is red, "it also fails on main" is the evidence every worker cites, and it
hides new breakage. The operator (batch-2 recap, 10-10): "let's ensure ariadne and pair are green on main".

## Spec

- Run `make test` (and `go test ./...`) on a clean main; list every failure and every package over 5 minutes.
- Each failure: fix it at the root, or quarantine it by name with an owning issue (as pair#324 did for pair#432).
  Quarantine is the exception, and every quarantined test is named in the issue's Log.
- couchcore's run time: find what makes it take over 15 minutes (live tests, timeouts, serial waits) and bring it
  down, or move the slow live tests behind an explicit lane that `make test` doesn't run by default, stated in TOOLING.
- Record the suite's wall time before and after in the Log.

## Done when

- [ ] `make test` passes on main from a clean slot, twice in a row, with the wall time recorded.
- [ ] No package takes over 5 minutes in the default lane.
- [ ] Every quarantined test has an owning issue.

## Plan

- [ ]

## Log

### 2026-10-10
