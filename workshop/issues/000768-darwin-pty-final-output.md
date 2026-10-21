---
id: 000768
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd1b258c6b57be47406e1427b9f54535b2cea7b70' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T05:20:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Preserve short-lived command output across Darwin PTY slave close

## Problem

Public pair scribe -log PATH -- CMD [ARGS...] accepts arbitrary commands (docs/pair-scribe.md; cmd/pair-go/main.go dispatches directly to scribecmd.Run), including short-lived commands that never open their controlling terminal. On macOS these can exit and lose unread PTY output before the first master read. #766 reproduced 10 empty-stdout results in 10000 runs; an instrumented wrapper copy saw 41/5000 first reads return EOF with zero bytes while the child exited successfully. Scribe already drains before cmd.Wait; this is not fixed by waiting longer or retrying reads after EOF.

XNU's last-slave-close path can discard unread output without a controlling-terminal reference. Opening /dev/tty in the child establishes S_CTTYREF and causes exit to drain. Ruby's primary investigation documents the same mechanism and links the kernel paths: https://bugs.ruby-lang.org/issues/20682 ; Apple source: https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/tty_tty.c . #766 fixes the test producer only, as approved by the TL; this production exposure remains.

## Spec

Preserve final output for admitted short-lived non-terminal-aware commands under scribe. Audit other generic Pair PTY entrypoints before generalizing. Consider an exec helper that establishes the child's /dev/tty reference before exec, versus an explicitly owned retained-slave/drain lifecycle. Preserve argv, child exit status, signal delivery and terminal behavior; no timeout increase or blind retry.

## Done when

- A deterministic native regression forces completed writes and closure of all child-owned slave descriptors before the first master read, then verifies exact stdout/log bytes, EOF and status.
- A descendant retaining the slave, including ignored SIGHUP, cannot prevent deterministic scribe completion after direct-child exit.
- Short-lived non-terminal-aware children and normal interactive shell/agent behavior pass, including child exit status, signals and cleanup ownership.
- The design records which Pair PTY entrypoints are covered and any remaining exposure.

## Plan

Proposed design: [durable plan](../plans/000768-darwin-pty-final-output-plan.md). Await TL approval before change-code. Recommend full flow for launch/lifetime risk, with one atomic review boundary.

- [ ] Approve Darwin-only retained slave for scribe, scope, native acceptance clarification and full flow.
- [ ] Pin delayed-first-read loss with an out-of-band child stdio-close barrier; implement scribe-local start/lease ownership.
- [ ] Qualify normal drain, abort cleanup, interactive behavior, exit/signals and resource lifetimes.
- [ ] Document audited remaining exposures, qualify the public built command, run recorded suite, close/review and land.

## Log

Discovered while investigating #766. Public scribe command is direct evidence of production admission; the shipped documented shell hook normally wraps a long-lived /bin/zsh, whose slave reference protects its ordinary subprocesses. Explicit scribe printf and final shell exec remain exposed. File unclaimed per TL instruction; do not expand #766's approved fixture-only scope.


### 2026-10-21 — planning checkpoint (pending approval)

Audited all three production launch owners: scribe, wrap's runProxy/startNativePTY, and ptychild.Start (Couch/term). Recommend retaining one parent slave on Darwin through scribe's existing output drain, releasing before the sole child wait. Preserve original exec.Cmd/direct startup and synchronous executable failures; no helper protocol, retries or deadlines.

Native planning probe forced child stdio closure before first master read: unprotected baseline lost final-marker; child /dev/tty and parent retention each preserved it. The retained variant reached EOF while retained and exited7. This is feasibility evidence, not automated acceptance.

Do not generalize this patch: wrap uses EOF to stop a still-live process group, so retention changes a target that closes stdio then keeps running. ptychild exposes EOF as input-ended before reap/publication. Both remain exposed and need separate lifetime designs; file unclaimed follow-ups only after TL approves scope. ARCH-PURPOSE/DRY shaped the scribe-local choice; ARCH-STATE/FUNERAL require explicit lease release and one wait owner. Abort must close the master before joining wait, because exit may be blocked draining unread output.

## Revisions

- 2026-10-21 (proposed, pending TL approval): choose scribe-only Darwin retained-slave ownership after auditing other production owners. Original Spec remains the scope contract. Clarify the first Done when, originally “write and exit before the first master read”: require finished writes and closure of all child-owned slave descriptors before the first read, then exact stdout/log, EOF and correct reap status. A protected child may be in kernel exit-drain, so requiring waitpid completion before reading would deadlock. The durable plan records both alternatives, remaining exposures and qualification. No production/test edits yet.

### 2026-10-21 — TL approval

TL approved full flow, scribe-only retention and acceptance clarification (receipt a711dfa3-61f8-4c22-847e-25caf4a69a63). Added explicit descendant-held-slave contract to the plan: Darwin session-leader exit revokes the terminal, scribe drains to EOF and reaps its direct child without waiting for descendants. Native planning probe observed exact final-marker and EOF while an ignored-SIGHUP descendant still held the slave; direct child exited7. Qualification must prove this through production Run; failure is a stop/re-plan, not a new timeout. Separate unclaimed wrap/ptychild issues requested.

Filed unclaimed follow-ups #769 (wrap) and #770 (ptychild), each preserving probe provenance and the owner-specific termination-semantics risk.

### 2026-10-21 — plan review round1

PQ-1/PQ-2 addressed: initialization abort explicitly kills the owned direct child, closes master/slave before the sole wait, treats already-exited as benign and preserves the initialization error. The plan names startPTY, releaseSlave, run and abortPTY with adversarial test strategies rather than case inventories. Normal completion still waits for the direct child without stopping live closed-stdio targets.
