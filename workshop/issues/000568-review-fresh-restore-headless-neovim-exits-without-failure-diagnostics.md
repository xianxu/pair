---
id: 000568
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'f0159ce3f453ee80ce2f9f66904b5ef1a7cf1287' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T07:54:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
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

- [ ] Add bounded failure diagnostics first; exercise a deliberately silent nonzero editor exit and a Lua/message failure so missing evidence is red before the harness repair.
- [ ] Reproduce the suite-context failure using pair:2's measurements and one-variable probes; name the actual cause in Log/Revisions, then add a deterministic regression and minimal fix at that boundary.
- [ ] Run the fresh-restore and affected review fixtures, a clean make -k build, and required recorded suite; fresh SHIP review, land, and notify pair:2 plus hub.

ARCH-DRY: reuse ReviewTestEnvironment and current stateful Zellij fake; do not create a parallel product launcher. ARCH-ORDER: evidence must be captured before Neovim exits and before fixture cleanup; diagnostic tests force silent/error exit orderings. ARCH-FUNERAL: diagnostic files remain inside each temporary case, are printed bounded on failure, then removed with their owning fixture; no persistent logs or caller state added.

## Log

2026-10-15: recorded suite at db86a287f8c74bbda952dfe00356e3b8f11d4601 exited 2 after 3m39s, before the Go phase. Retained common-Git log .git/sdlc/test-stats/failed/eef944b68fc13088226c6a06a1a85b99.log; SHA256 prefix92b0a48d20e5. Earlier cases fresh-restores-history and wrong-conversation-ack-refused passed. Isolated command bash tests/review-fresh-restore-test.sh subsequently passed all five cases. No changes to review editor/opener files in the #556 correction window.

2026-10-15: pair:3 claimed and started from main 03f5ba2af12069d6d6d628a34e0195a3dd05a0e9; substrate refreshed to ariadne 1e2e9bd573e7b08366688d10001e95583dd35e97. Verified both retained suite failures differ in refusal case but have the same empty combined output assertion. Existing driver prints xpcall errors but has no shutdown messages/return-code report. Coordinating with pair:2 via Couch.

2026-10-15: Investigation without tracked harness edits reproduced the same failure in an isolated script on current main binaries: detached-identity case returned 1 with empty output. A temporary VimLeavePre probe later reproduced missing-history rc=1; v:exiting was 0, messages empty, shell_error 0. v:errmsg was E184 ShFoldIfDoFor, but identical in passing cases; minimal draft-init/system true/system false/async false probes all exited 0 with that errmsg, so it is not established as cause. Verbose tracing changes timing; do not confuse a diagnostic-only passing sample with a fix.

2026-10-15: Temporary existing-log capture reproduced wrong-conversation-ack-refused rc=1 on shutdown-log sample 7 (after six complete successes; diagnostic sampling, not acceptance retries). Case-local NVIM_LOG_FILE reported `uv_loop_close() hang?`, two live fs_event handles, their async handles, and two live timers. Driver stderr/stdout remained empty. A server-start warning also named a long per-case XDG_RUNTIME_DIR endpoint; investigate independently rather than treating it as cause. Neovim's [v0.11.7 os_exit](https://github.com/neovim/neovim/blob/v0.11.7/src/nvim/main.c#L632) changes a planned exit 0 to 1 if event_teardown fails; its [loop_close](https://github.com/neovim/neovim/blob/v0.11.7/src/nvim/event/loop.c#L156) logs this hang and remaining handles. This establishes teardown as the failure stage, not a failed review assertion. init.lua creates raw scrollback-pending and slug fs_event watchers plus timers; lifecycle ownership/disposal must be traced before selecting the fix.

2026-10-15: Read-only hub peek confirms approval and pair:2's independent nvim.log evidence are queued in this slot and cannot arrive until this active turn ends (known Couch in-turn delivery limitation). Settling to receive those messages; no tracked harness/production edits yet, only plan/Log committed. Current branch 000568-review-fresh-restore-headless-neovim-exits-without-failure-diagnostics; clean build completed at plan head. Next: verify queued receipts, change-code, durable diagnostics with red/green, trace exact watcher/timer ownership and deterministic teardown regression, then required suite and fresh SHIP.
