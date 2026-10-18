---
id: 000674
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9c05e2ad9ccb1f0a00816dea539ce410cd4d1710' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T12:30:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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

- [x] Root cause. Before the helper installs `cancel`, SIGINT runs Python's `default_int_handler`, which raises KeyboardInterrupt at the next eval check. Where Python swallows exceptions, the cancellation is lost:
  - the importlib module-lock weakref callback (pair:3, #650);
  - zipimport's argv[0] path-entry probe (pair:7, #659/#671), which runs before any helper code.

  Import order cannot close the window. SIGTERM is SIG_DFL then, so it is unaffected.
- [x] Fix:
  - The fixture launches the helper with SIGINT and SIGTERM blocked, inherited across exec, and restores its own mask right after spawning.
  - The helper unblocks after installing `cancel`, so a startup signal waits in the kernel and is then delivered.
  - An unblocked caller is unchanged.
  - No wider wait and no retry.
- [x] Deterministic regression: `test_startup_cancellation_is_never_lost` (SIGINT and SIGTERM), plus the #598 SIG_IGN case on the same gate. The signal is sent while the child is held before the helper; the gate is released; the test requires exit 128+signal with no descendant started.
- [x] Diagnostics add the caller's blocked mask beside `caller_SIGINT`, and the child reports its own disposition and mask.

## Log

2026-10-18: discovered while taking over #650 (PR 434). Recorded integration attempt 8f2fa5ea50afe6009e976e25261f2029; make exit 2 in 2m45s after 1565 s queue. Saved log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/8f2fa5ea50afe6009e976e25261f2029.log; SHA256 prefix ccfba26078cd. Six unittest methods ran in 5.714s, one failed subcase. Stderr: Exception ignored while calling weakref callback _get_module_lock.<locals>.cb, frozen importlib._bootstrap line 457, KeyboardInterrupt. helper_pid=21478; descendant_group=21486; caller_SIGINT=default_int_handler. #598 merge 555014c077da is an ancestor; it normalized inherited disposition in the fixture, leaving helper import order unchanged. The helper imports os/select/signal/subprocess/sys/time before registering its cancellation handlers. Swallowing in the callback is observed; an exact deterministic reproduction is not yet performed.

The merge runner automatically queued a diagnostic main baseline on db5ddb4e2bf5af6ec85c3c916e51b17523c12085, attempt 4135a602541e54d2c39ebb218696ab89, after the failed branch run. No unchanged integration rerun yet. Cross-reference #598 and #659.

Consolidated from #671 (duplicate, wontfix per hub): #659 conflict suite on f35faa470d4dd2e484b31150326b379635b0fd91, attempt e3644ec5a5ebc04b7b8fa2219b970aa7. Suite exit 2 in 2m57s; log `.git/sdlc/test-stats/failed/e3644ec5a5ebc04b7b8fa2219b970aa7.log`, SHA256 prefix 90d9d93cf898. SIGINT at 0.02 s; helper 49714 alive, descendant group 49715, returncode None. Stderr: "Failed checking if argv[0] is an import path entry" (frozen zipimport `__init__`/`_read_directory`), KeyboardInterrupt. pair:7's targeted repeat passed in 2.826s.

2026-10-18 (pair:2): measured.
- RED: with the helper's unblock removed, all three gate cases fail, each with the helper alive past the 3 s bound (returncode None, descendant started): the field symptom, deterministic.
- GREEN: with the fix, `tests/without-ancestors-test.py` runs 7 tests OK in 2.1 s, and `test_cancellation_during_startup` passed 25 of 25 repeated runs.
