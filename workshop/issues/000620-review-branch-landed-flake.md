---
id: 000620
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '2516f42bd2a9c0d1da073d46c0d323070e0aa698' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:31:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ddfa5bf7", done: "e0800183"}
---

# Review branch restore intermittently misses landed acknowledgment

## Problem

The recorded full suite for #604 failed in tests/review-branch-restore-test.sh: its review-pane handoff did not publish data/landed.json within the fixture's five-second wait, then read_text raised FileNotFoundError. The fixture printed an empty draft Neovim log; it did not retain the review pane's state at that boundary.

Failed head: fe4eb6d6e83bd53b9d42fcf27811a383c348b222. First observed here: 2026-10-16, during the #604 recorded suite. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/07f7702637f31ade0443abc6b434b7f2.log. The local-check record reports exit 2 in 5m38s and SHA256 prefix eee92d770af5. The failure was before go test ./..., so #604's test executables and their private clipboard environment had not run.

One unchanged isolated repeat passed: bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh; log /tmp/pair604-review-branch-restore-repeat.log. The current head differed only by #604's atlas/issue prose; fixture and executable source were unchanged. This is one failure and one passing repeat, not a calibrated rate.

## Spec

Capture the review-pane handoff/landing state at the missed publication boundary and determine why acknowledgment is missing. Distinguish scheduling delay, lost watcher event, rejected context and editor failure. The existing five-second wait followed by an unconditional file read is where the failure is observed, not a proven cause. Do not merely increase that timeout.

The #604 production diff only adds the explicit Couch switcher clipboard action and its clipboard-runtime seam. This fixture runs review readiness/editor handoff operations, without entering that action. Keep diagnosis/fix independent of #604.

## Done when

- A reproducer and bounded diagnostics identify the missed handoff acknowledgment.
- A regression exercises the identified ordering through the real fixture seam.
- Repeated checks report before/after failure counts and conditions without weakening the assertion or hiding failed attempts.

## Plan

- [ ] Capture bounded review-pane admission/apply/publication evidence through the real fixture, preserving the existing five-second assertion.
- [ ] Reproduce the demonstrated cause deterministically and apply the narrow fix without weakening branch/context/CRLF/BOM/recovery checks.
- [ ] Record before/after conditions and counts, targeted controls and the required full suite.

## Log

- 2026-10-16: Filed unassigned after one recorded failure and a passing unchanged targeted repeat. #604 retained its failed suite and started make -k to complete all remaining checks; no fixture or timeout change is included there.

- 2026-10-17: claimed after landing #598. Fresh branch prepared from main; first start-plan refreshed Ariadne and refused the older executing binary, then the second start-plan succeeded. One unchanged baseline via bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh passed (/tmp/pair620-baseline.log). Historical failed log confirms only an empty draft log and missing landed.json; it contains no review-pane state. Cause remains unknown.
- 2026-10-17 investigation plan posted to hub as proceeding unless objected: capture actual admission/authorization/apply/publication results and pending editor state in bounded diagnostics, then use controlled orderings to distinguish slow scheduling/IO from rejected context or failed editor effects. The watcher polls every 100 ms; there is no filesystem event to lose on this path. Preserve the five-second publication assertion. ARCH-PURPOSE: diagnose before changing budgets. ARCH-DRY: reuse existing real fixture and review observation seams. No operator environment action or production fix selected yet.
