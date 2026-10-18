---
id: 000671
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a3df65094da790a030e0c2cad2821768f0abe00f' # card fields mirrored from issue-cards; edit via sdlc
---

# Startup SIGINT can leave without-ancestors helper alive in cancellation test

## Problem

A recorded #659 integration-conflict suite on f35faa470d4dd2e484b31150326b379635b0fd91 failed tests/without-ancestors-test.py LifetimeTests.test_cancellation_during_startup at SIGINT delay0.02s. First seen 2026-10-18 during attempt e3644ec5a5ebc04b7b8fa2219b970aa7; suite exit2 after2m57s, log SHA256 prefix90d9d93cf898. Retained common-Git log: .git/sdlc/test-stats/failed/e3644ec5a5ebc04b7b8fa2219b970aa7.log.

## Spec

Investigate cancellation while Python is starting the reparenting helper; replace the measured race with deterministic boundary coverage or fix helper cancellation ownership as appropriate, without simply extending the three-second assertion. Failure diagnostics: helper_pid49714, descendant_group49715, helper returncodeNone; output reports Failed checking if argv[0] is an import path entry, frozen zipimport __init__/_read_directory, KeyboardInterrupt. process.wait(timeout3) expired although the helper was launched with timeout20. The fixture sends SIGINT after a real20ms startup sleep; exact location of interruption is scheduling dependent. All six test methods ran; one subtest failed. Cleanup completed before the recorded suite returned.

This helper and its tests arrived from newer main and have no #659 diff. Root cause remains unproven; an early interrupted import may leave the cancellation path incomplete, which is a hypothesis. Do not weaken descendant cleanup or signal status assertions.

## Done when

- Reproduce the failing startup boundary with retained helper/descendant receipts.
- Add deterministic regression for the measured cause, without a larger constant or blind retry.
- Verify helper lifetime ownership and report repeated-run failures before/after.

## Plan

- [ ]

## Log

- 2026-10-18: Filed unclaimed per hub direction. #659 retains this failed attempt, makes one unchanged recorded-suite rerun on the same head, then stops/reports if it fails again. A separate targeted repeat is diagnostic only, not replacement evidence.
