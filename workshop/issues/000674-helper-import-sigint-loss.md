---
id: 000674
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd35db9c396bb4af8a56c805841677c00ba92e01c' # card fields mirrored from issue-cards; edit via sdlc
---

# Startup SIGINT swallowed by importlib weakref callback before helper handler

## Problem

The pair#650 integration suite on aa8176700b0e50b230c72eeaf190c9b284342810 failed in LifetimeTests.test_cancellation_during_startup (SIGINT, delay 0.02 s) despite the pair#598 fixture normalization being present. The helper did not exit within the existing 3 s bound; captured stderr reports KeyboardInterrupt ignored in importlib._bootstrap module-lock weakref callback. The caller SIGINT was default_int_handler. This differs from the inherited SIG_IGN cause fixed in #598. No causal link to slug gating is shown: the helper and lifetime test files are unchanged by #650.

## Spec

Classify the pre-handler startup signal window using retained diagnostics and a deterministic regression. Determine whether default SIGINT is swallowed by an importlib weakref callback before the helper installs its handler. Keep owned descendant cleanup, intentional successful sidecars, and the existing bounds. No blind retries or larger timeouts.

## Done when

- A deterministic regression covers the demonstrated startup signal-loss seam.
- Cancellation remains bounded with owned descendants cleaned and successful sidecars preserved.
- Failure diagnostics distinguish the inherited-SIG_IGN class in #598 from swallowed pre-handler exceptions.

## Plan

- [ ]

## Log

2026-10-18: discovered while taking over #650 (PR 434). Recorded integration attempt 8f2fa5ea50afe6009e976e25261f2029; make exit 2 in 2m45s after 1565 s queue. Saved log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/8f2fa5ea50afe6009e976e25261f2029.log; SHA256 prefix ccfba26078cd. Six unittest methods ran in 5.714s, one failed subcase. Stderr: Exception ignored while calling weakref callback _get_module_lock.<locals>.cb, frozen importlib._bootstrap line 457, KeyboardInterrupt. helper_pid=21478; descendant_group=21486; caller_SIGINT=default_int_handler. #598 merge 555014c077da is an ancestor; it normalized inherited disposition in the fixture, leaving helper import order unchanged. The helper imports os/select/signal/subprocess/sys/time before registering its cancellation handlers. Swallowing in the callback is observed; an exact deterministic reproduction is not yet performed.

The merge runner automatically queued a diagnostic main baseline on db5ddb4e2bf5af6ec85c3c916e51b17523c12085, attempt 4135a602541e54d2c39ebb218696ab89, after the failed branch run. No unchanged integration rerun yet. Cross-reference #598 and #659.
