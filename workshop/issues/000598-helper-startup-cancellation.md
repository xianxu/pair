---
id: 000598
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '247f8b92c5c9cdfae2da29279f0fdc355d5ace0b' # card fields mirrored from issue-cards; edit via sdlc
---

# Diagnose helper startup cancellation timeout

## Problem

Recorded #586 integrated suite fails before the review fixtures in LifetimeTests.test_cancellation_during_startup for SIGINT with delay0.02. Helper process does not exit within the existing3s wait. No output retained because the fixture redirects both streams to DEVNULL.

## Spec

Diagnose the helper startup/cancellation boundary and retain actionable evidence. Determine whether a signal is sent before its handler is installed, cancellation is lost across startup/fork, or owned group cleanup/waitpid stalls. These are hypotheses, not a demonstrated cause. Replace scheduling assumptions with actual event/state evidence; no longer sleeps, retries, skips or larger budgets. Keep real descendant process-group cleanup and successful-sidecar behavior.

## Done when

- A deterministic regression reproduces the actual cause at the owning startup/cancellation seam.
- Cancellation remains bounded and cleans owned descendants, including startup races; successful intentional sidecars remain allowed.
- Failure diagnostics identify observed startup/handler/group/wait state instead of discarding all stderr.

## Plan

## Log

2026-10-15: observed in required recorded suite501ea461dee91eaad8c9ae484cbfa18d, source a0b3fb0de67824b57d0f42cedd32acd02a7d1778. Queue780.079s, execution188.925s, makeexit2, not an overall timeout. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/501ea461dee91eaad8c9ae484cbfa18d.log, SHA256ddf72cf3b67ba81c2891b26c021302619c0d9120ac6b9d17f401e525646a337a. Python3.14.5. Five unittest methods ran4.850s, one subcase error. tests/without-ancestors-test.py:103 subprocess.TimeoutExpired from process.wait(timeout=3), helper --timeout20, modewait. Files unchanged from origin/main; independent of #586's review diagnostics, which were not reached. Prior #586 diagnostics-source suite f6512608053de4d07e29acf206d870b31d169b51 passed14m1. No unchanged full-suite retry yet.

Read-only audit: the startup test launches wait_ready=False, sleeps0/10/20/40ms, sends TERM/INT and requires nonzero within3s. The helper installs signal handlers after imports, creates ready/start pipe handshake, forks/setsid/doubleforks, waits ready, sends start only if not cancelled. Finally closes handshake, kills owned group on failure and waitpid(intermediate). Fixture discards stdout/stderr; cleanup kills only a group whose descendant lease was written and kills the helper if it remains alive. No retained process state proves which startup phase failed.
