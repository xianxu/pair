---
id: 000598
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-17
estimate_hours:
card_mirror: '319ecc1462ca4fc4e7a92efde30acf93e679a7b5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T09:39:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "48bd9509", done: "338a36e1"}
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

- [x] Reproduce early SIGINT loss with an ignored-SIGINT parent and pin the pre-handler ordering with a pipe gate.
- [x] Normalize the fixture child's inherited SIGINT without changing its caller; retain failure output and startup/lease observations.
- [ ] Verify both signal dispositions, bounded descendant cancellation and successful sidecars; record the full suite, close review and landing.

## Log

2026-10-15: observed in required recorded suite501ea461dee91eaad8c9ae484cbfa18d, source a0b3fb0de67824b57d0f42cedd32acd02a7d1778. Queue780.079s, execution188.925s, makeexit2, not an overall timeout. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/501ea461dee91eaad8c9ae484cbfa18d.log, SHA256ddf72cf3b67ba81c2891b26c021302619c0d9120ac6b9d17f401e525646a337a. Python3.14.5. Five unittest methods ran4.850s, one subcase error. tests/without-ancestors-test.py:103 subprocess.TimeoutExpired from process.wait(timeout=3), helper --timeout20, modewait. Files unchanged from origin/main; independent of #586's review diagnostics, which were not reached. Prior #586 diagnostics-source suite f6512608053de4d07e29acf206d870b31d169b51 passed14m1. No unchanged full-suite retry yet.

Read-only audit: the startup test launches wait_ready=False, sleeps0/10/20/40ms, sends TERM/INT and requires nonzero within3s. The helper installs signal handlers after imports, creates ready/start pipe handshake, forks/setsid/doubleforks, waits ready, sends start only if not cancelled. Finally closes handshake, kills owned group on failure and waitpid(intermediate). Fixture discards stdout/stderr; cleanup kills only a group whose descendant lease was written and kills the helper if it remains alive. No retained process state proves which startup phase failed.

2026-10-17: claimed after #635 landed. Unchanged source launched through a Python parent that explicitly ignores SIGINT reproduces TimeoutExpired at delays 0, 0.01 and 0.02 (one test, 9.699 s, three errors). Python preserves an inherited ignored SIGINT across exec until the helper installs its handler. Sending before that point cannot cancel; the helper then legitimately waits for its 20 s command. The direct interactive fixture parent has default_int_handler instead. Historical logs cannot establish their discarded initial disposition. This supplies a demonstrated cause of the identical failure, rather than evidence of a product cancellation defect.

2026-10-17 design: fixture-only quick fix, posted to hub as proceeding unless objected. Normalize SIGINT around helper launch, restoring the parent immediately afterward. A real-process bootstrap gate before helper entry pins the vulnerable window under an ignored-SIGINT parent; assert cancellation before gate release and preservation of the parent's disposition. Keep existing real descendant and success controls. Capture streams in an owned temporary file; timeout diagnostics report signal/disposition, process and lease observations plus captured output, without longer waits or retries. ARCH-DRY/PURPOSE: reuse the owned-group helper unchanged and fix the fixture's launch precondition. ARCH-PURE/MOCK: the boundary is real OS process/signal IO, not mocked signal calls. ARCH-CONSTRAINTS: retain 3 s exit and 2 s descendant budgets. ARCH-SECURE: signal only owned children/groups. ARCH-STATE: explicitly order pre-handler gate, cancellation and gate release. ARCH-FUNERAL: close diagnostic files and collect owned helper/descendants before removing their private directory. Headless real-process verification needs no operator environment change; no pair:0 staging proposed.

2026-10-17 deterministic RED: test_ignored_parent_sigint_cancels_before_helper_entry starts a real child with a pipe gate before runpy enters the helper. The child reports SIGINT disposition 1 (SIG_IGN), then the parent sends SIGINT and waits without releasing the gate. Failure is the expected lost cancellation after 3 s, not a fixture setup error (3.049 s total first measurement). Retained second reproduction: /tmp/pair598-gated-red.log. Cleanup kills only that owned gated child; no helper command or descendant starts.

2026-10-17 GREEN: all six real-process lifetime methods pass under normal parent (2.156 s) and explicitly ignored-SIGINT parent (2.241 s), including exact post-ready TERM/INT exit codes, timeout cleanup, intentional-success-sidecar survival and command exit status. A real noninteractive Bash background launch reports inherited SIGINT=1 and passes all six in 2.201 s (/tmp/pair598-background-green.log). Running origin/main's original startup test in that same background form reproduces the old 3 s TimeoutExpired at delays 0 and 0.01 (6.732 s total; /tmp/pair598-background-original-red.log). An initial diagnostic bootstrap used the wrong __main__ namespace and failed unittest discovery; corrected before this behavioral measurement.

2026-10-17 final targeted verification: make test-without-ancestors passes six methods in 1.851 s (/tmp/pair598-final-make.log); git diff --check passes. The new gate also verifies captured stderr and caller-disposition preservation. Replacing only the normalization call with getsignal (in-memory source mutant; no tracked mutation) fails the gate in 3.031 s. Its failure names phase=before helper entry, child_SIGINT=1, helper PID, no published descendant group, caller_SIGINT=1, and the retained pre-handler stderr marker (/tmp/pair598-reverted-normalization.log). Diagnostic reads use positional IO, bounded to 64 KiB, so observing output cannot move the child's write offset. tests/without-ancestors.py is unchanged. atlas/index.md maps the fixture contract; lesson records the inherited-signal class.

## Revisions

2026-10-17: refined the initial hypotheses with the ignored-SIGINT reproduction. The implementation scope is the fixture's signal inheritance and failure evidence; preserve the helper's existing handshake, bounded cleanup and successful-sidecar contract.
