---
id: 000603
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '300dd3337b8800704ccb1057d17191a2f1dcb8d3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T15:35:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Native wrapped Zellij conformance times out finding shell actor

## Problem

GitHub macOS conformance intermittently stalls waiting for the real shell actor in TestNativeConsoleWrapperZellij/wrapped-zellij. This blocks unrelated PRs after their local full suite and review pass.

First observed here: 2026-10-16 06:27:17 UTC (2026-10-15 23:27:17 PDT), run 38860695250, attempt 1, head 550c434561059eeda72ed209041b5468916081f1, while landing #571. The subtest failed after 284.48 seconds at terminal_native_test.go:227: timed out waiting for real shell actor.

Run: https://github.com/xianxu/pair/actions/runs/38860695250/job/116842060496

Local diagnostic log: /tmp/pair571-ci-conformance-38860695250.log (ephemeral; the issue does not depend on its retention).

## Spec

Find why the wrapped Zellij fixture does not observe its real shell actor. Keep the assertion meaningful; do not address this by raising its timeout. In the failing run the direct-zellij-baseline passed in 1.73 seconds. Wrapped receipts recorded BOOT, CPR and submitted input bytes; wrapper.log recorded canonical OSC 777 and native-fixture-complete notifications, but shell-actor observation did not complete. These facts do not yet establish the root cause. Runner scheduling, process discovery and terminal lifecycle remain hypotheses.

This is distinct from #601's park helper client-disconnect failure. No terminal fixture, shell actor or Zellij lifecycle code is changed by #571's reduced Codex exact-ID home routing. An intervening conformance run on the same reviewed code passed before a documentation-only main integration created this head. One observed failure is not a calibrated flake rate.

## Done when

- Root cause is reproduced with retained, bounded diagnostics that distinguish actor launch from discovery failure.
- A regression verifies the identified race without a larger timeout.
- Repeated runner/headless checks demonstrate the correction, reporting failure counts and conditions.

## Plan

- [ ] Trace startup, attachment, selection and publication; reproduce the lost shell presentation with controlled ordering and bounded diagnostics.
- [ ] Add a failing regression for the confirmed ordering and fix its owner without timeout increases or blanket retries.
- [ ] Run targeted race tests and repeated private native conformance, then the recorded full suite and close review.

## Log

- 2026-10-15: Filed unassigned per TL direction. #571 will retry only this failed CI job once on its unchanged head; a repeat failure is escalated rather than retried indefinitely.
- 2026-10-16: Fresh dispatch from ops:0 verified. Fast-forwarded resting branch, claimed and started #603. The wait is presentation admission plus visible shell text, not OS process discovery. Original CI log remains available locally. Investigating startup/attach/selection order under ARCH-ORDER; reuse the existing fake Child and real private Zellij fixture (ARCH-MOCK), with no host session changes (ARCH-SECURE). Diagnostics and subprocesses remain test-owned (ARCH-FUNERAL); timeouts remain unchanged (ARCH-CONSTRAINTS). Keep the fix at the existing presentation seam (ARCH-DRY/ARCH-PURE), covering the actual lost-event class (ARCH-PURPOSE).
