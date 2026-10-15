---
id: 000568
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '4b8dcbe00b4b60d3f40f6bc7c8794bfacd896cf0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T07:54:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0c579d9d", done: "5c85287f"}
actual_hours: 0.58
---

# Review fresh-restore headless Neovim exits without failure diagnostics

## Problem

During pair#556 M1 recorded verification, the unchanged shell test tests/review-fresh-restore-test.sh failed at missing-history-does-not-adopt-old-receipt. The Python assertion saw a nonzero headless Neovim return with empty combined output; its assertion omits the return code, so the retained log cannot distinguish a signal from ordinary exit. The isolated full script immediately passed all five cases on the same code. This is not a confirmed root cause or a #556 recovery regression.

## Spec

Make every unsuccessful driver invocation explain its exit using bounded, case-local evidence: process return code/signal, separate stdout/stderr, Neovim messages, review-child log and fixture phase/artifact presence. Capture messages before shutdown so a silent nonzero qa exit remains diagnosable. Preserve evidence in the suite output before temporary cleanup; do not capture the caller's files or unrestricted environment.

Coordinate with pair:2's suite/environment measurements. Compare the recorded-suite wrapper, environment, TMPDIR/path shape and shared Neovim state against isolation. Reproduce the demonstrated cause deterministically, fix its owning boundary, and retain the real draft/opener/RPC behavior and exact child cleanup. No timeout increase, retry or skip. Root-cause implementation will be recorded as a revision after evidence identifies it.

## Done when

- The failing assertion records return code, bounded Neovim output and enough per-case state to identify a failure without retaining private/user state.
- Repeated portable stateful runs reproduce or bound the failure; fix its demonstrated cause rather than increasing a timeout.
- The test still exercises the real draft/Go opener/review RPC boundary, and temporary children are cleaned by exact identities.

## Plan

- [x] Add bounded failure diagnostics first; exercise a deliberately silent nonzero editor exit and a Lua/message failure so missing evidence is red before the harness repair.
- [x] Reproduce the suite-context failure using pair:2's measurements and one-variable probes; name the actual cause in Log/Revisions, then add a deterministic regression and minimal fix at that boundary.
- [x] Run the fresh-restore and affected review fixtures, a clean make -k build, and required recorded suite; fresh SHIP review, land, and notify pair:2 plus hub.

ARCH-DRY: reuse ReviewTestEnvironment and current stateful Zellij fake; do not create a parallel product launcher. ARCH-ORDER: evidence must be captured before Neovim exits and before fixture cleanup; diagnostic tests force silent/error exit orderings. ARCH-FUNERAL: diagnostic files remain inside each temporary case, are printed bounded on failure, then removed with their owning fixture; no persistent logs or caller state added.

## Log


- 2026-10-15: closed — Deterministic real-init regression fails before repair because both draft filesystem watches remain open at VimLeavePre; now both close before global teardown. Shared owner tests pass start refusal, close-once and queued callback invalidation. Real silent/message exit regressions prove rc, separate streams and pre-exit messages/phase diagnostics. make -k test-lua test-review passed; four concurrent real fresh-restore scripts passed all 20 cases. make -k -B build passed at clean f6493cbc0e041e32b560bc5302143fb4bdcda31e. Corrected required sdlc-local/suite passed on that head (8m33s; fresh-restore all five cases, artifactpath 3.960s, couchcore 248.004s; SHA 89215f27792bc167d8b8b52016fec6ec1ea763811c01448059de8a545af90bcb). Atlas maps owner; no retries, sleeps, timeout increases, skips or blanket cleanup. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
2026-10-15: recorded suite at db86a287f8c74bbda952dfe00356e3b8f11d4601 exited 2 after 3m39s, before the Go phase. Retained common-Git log .git/sdlc/test-stats/failed/eef944b68fc13088226c6a06a1a85b99.log; SHA256 prefix92b0a48d20e5. Earlier cases fresh-restores-history and wrong-conversation-ack-refused passed. Isolated command bash tests/review-fresh-restore-test.sh subsequently passed all five cases. No changes to review editor/opener files in the #556 correction window.

2026-10-15: pair:3 claimed and started from main 03f5ba2af12069d6d6d628a34e0195a3dd05a0e9; substrate refreshed to ariadne 1e2e9bd573e7b08366688d10001e95583dd35e97. Verified both retained suite failures differ in refusal case but have the same empty combined output assertion. Existing driver prints xpcall errors but has no shutdown messages/return-code report. Coordinating with pair:2 via Couch.

2026-10-15: Investigation without tracked harness edits reproduced the same failure in an isolated script on current main binaries: detached-identity case returned 1 with empty output. A temporary VimLeavePre probe later reproduced missing-history rc=1; v:exiting was 0, messages empty, shell_error 0. v:errmsg was E184 ShFoldIfDoFor, but identical in passing cases; minimal draft-init/system true/system false/async false probes all exited 0 with that errmsg, so it is not established as cause. Verbose tracing changes timing; do not confuse a diagnostic-only passing sample with a fix.

2026-10-15: Temporary existing-log capture reproduced wrong-conversation-ack-refused rc=1 on shutdown-log sample 7 (after six complete successes; diagnostic sampling, not acceptance retries). Case-local NVIM_LOG_FILE reported `uv_loop_close() hang?`, two live fs_event handles, their async handles, and two live timers. Driver stderr/stdout remained empty. A server-start warning also named a long per-case XDG_RUNTIME_DIR endpoint; investigate independently rather than treating it as cause. Neovim's [v0.11.7 os_exit](https://github.com/neovim/neovim/blob/v0.11.7/src/nvim/main.c#L632) changes a planned exit 0 to 1 if event_teardown fails; its [loop_close](https://github.com/neovim/neovim/blob/v0.11.7/src/nvim/event/loop.c#L156) logs this hang and remaining handles. This establishes teardown as the failure stage, not a failed review assertion. init.lua creates raw scrollback-pending and slug fs_event watchers plus timers; lifecycle ownership/disposal must be traced before selecting the fix.

2026-10-15: Read-only hub peek confirms approval and pair:2's independent nvim.log evidence are queued in this slot and cannot arrive until this active turn ends (known Couch in-turn delivery limitation). Settling to receive those messages; no tracked harness/production edits yet, only plan/Log committed. Current branch 000568-review-fresh-restore-headless-neovim-exits-without-failure-diagnostics; clean build completed at plan head. Next: verify queued receipts, change-code, durable diagnostics with red/green, trace exact watcher/timer ownership and deterministic teardown regression, then required suite and fresh SHIP.

2026-10-15: Verified hub approval 41a927ba and passed change-code. Diagnostics red: both intentional early editor exits (37 silent, 38 with distinct stdout/stderr/message) failed the old assertion because it omitted return code and startup phase. Green: real-fixture diagnostic self-test now passes; current unmodified restore behavior passes all five cases once. Added pre-init VimLeavePre message/phase capture, separate per-case output files, bounded tails from Neovim/child/result logs and artifact-presence state; timeout stays 15 seconds and failures retain nonzero status. No root-cause repair yet.

2026-10-15: One-variable watcher-start probe (temporary wrapper of the real init source) observed both fs_event:start calls returning 0 with active=true at their intended case-local directory. Three complete runs passed; this excludes failed-start-as-sufficient-cause, not the shutdown race. A separate two-worker, six-sample experiment deferred only the driver until VimEnter plus vim.schedule. Sample 4 still failed missing-history after result-written with planned exit 0, empty messages/output, no child spawned and the same two fs_event/async/timer teardown hang. Therefore merely waiting for VimEnter is not a fix; do not add a startup tick or delay. The committed diagnostics now expose the failure without temporary probes. Both experiments are bounded diagnostic samples, not retries used to pass acceptance.

## Revisions

### 2026-10-15 — demonstrated watcher ownership defect

Root cause is draft filesystem watcher lifetime, not an environment-only failure or review assertion. Two valid watchers started by init.lua have no explicit shutdown owner; failures retain those fs_event/async resources during Neovim event-loop teardown and turn a planned exit 0 into 1. A one-variable temporary intervention closes exactly the two created watchers at VimLeavePre; eight full scripts across two workers (40 cases) all pass. Deferring until VimEnter did not fix it. Deterministic real-init regression is red: both real watchers remain open after VimLeavePre, before Neovim's global resource sweep.

Delta: add one small fs_watch helper owning start/refusal, stop/close at VimLeavePre, and queued-callback invalidation; migrate both draft watcher sites. Preserve their paths, callbacks and FocusGained fallback. Unit lifecycle tests cover successful delivery, close exactly once, callbacks queued before/after close, allocation failure and returned/thrown start refusal. No raw blanket uv.walk cleanup, sleeps, retries, budget increases or skipped initialization. Document the new owner in atlas. Required suite and fresh SHIP still follow. ARCH-DRY/ORDER/FUNERAL shape this common owner.

2026-10-15: Root repair green: real-init deterministic regression now observes both watchers closing at VimLeavePre; diagnostics self-test passes (2 tests, 1.838s), fs_watch lifecycle test passes close-once/late-callback/start-refusal cases, and all five real fresh-restore cases pass. Both watcher sites now derive from fs_watch; no background handle sweep, sleeps or timeout changes. Atlas maps the new owner; full review/Lua suites and recorded suite remain.

2026-10-15: Verification at implementation 4f81e8dfeca5aafb186204ef82409736ce891827: make -k -B build passed; Pair/Couch embed that exact revision with vcs.modified=false. make -k test-lua test-review passed, including the diagnostics/real-init lifecycle regression (1.789s) and all five fresh-restore cases. Four additional full fresh-restore scripts across two workers passed all 20 cases while the targeted and recorded suites were running. Required recorded suite is in progress; no failure was suppressed or retried.

2026-10-15: First recorded suite passed the new lifecycle/diagnostic tests (3.331s) and all five fresh-restore cases, then artifactpath’s exhaustive inventory correctly failed the undeclared new helper and embedded mirror. Interrupted that known-red suite before wasting the remaining broad Go phase; retained its failure evidence. Added fs_watch.lua as non-artifact logic (receives an already-bound directory; constructs no artifact filename) and the embedded copy as GeneratedMirror. No gate waiver or known-violations entry. Targeted artifactpath verification and a new recorded suite follow on the corrected code.

2026-10-15: Corrected artifactpath package passes (8.983s), including the exact exhaustive inventory test that failed in the recorded run.

2026-10-15: Rebuilt corrected implementation f6493cbc0e041e32b560bc5302143fb4bdcda31e with make -k -B build successfully. The new required suite runs on that pushed head; Lua watcher/review behavior is unchanged from the prior verified implementation, with only exhaustive inventory declarations corrected.

2026-10-15: Required sdlc-local/suite passed on corrected implementation f6493cbc0e041e32b560bc5302143fb4bdcda31e (exit 0, 8m33s). Real fresh-restore all five cases passed inside the recorded run; diagnostics/lifecycle tests passed (1.575s), artifactpath passed (3.960s), couchcore passed (248.004s). Full log SHA-256: 89215f27792bc167d8b8b52016fec6ec1ea763811c01448059de8a545af90bcb. Implementation verification is complete; fresh close review and guarded landing follow this checkpoint. Subsequent changes are evidence only.

2026-10-15: Fresh close review returned SHIP, high confidence, no findings. Reviewer independently verified diagnostics and watcher lifecycle regressions go red against base, pass at head, and verified all five real restore cases plus artifactpath. Its longer review TMPDIR exposed a separate overlong restore socket endpoint; rerunning with a shorter TMPDIR established that different cause. Filed independent unclaimed #570 for the path envelope; no timeout, retry or skip was used to repair or accept the original watcher failure. This known TMPDIR limit remains outside #568 and is not hidden by the passing recorded suite.
