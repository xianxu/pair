---
id: 000731
status: working
deps: [pair#727]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'dc70636b23b8d0aa523d38d8d715cbd0b2233a91' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T20:32:00-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a4f91cf6", done: "e7484f0b"}
---

# A pane that fell back to a native shell still classifies as a Pair terminal

## Problem

pair#727 improves wrapper failures by restoring the terminal and execing a native interactive shell, so the operator keeps a usable pane. The fallback shell can still be classified as a live Pair terminal.

Source findings in pair#727 (2026-10-20):

- termcmd.OSRuntime.RegisterTerminalPane records the wrapper PID; syscall.Exec preserves that PID.
- workbenchshortcut.TerminalPaneRegistry.LiveIDs checks only PID liveness, so the wrapper registration survives the handoff.
- RoleForPaneWith first calls RoleForPane, which accepts a startup TerminalCommand containing pair term or a title equal to terminal / prefixed terminal. Removing a registry record alone therefore does not prove the pane has lost its Pair role.
- TerminalPaneRegistry currently has no serialization lock: Register is plain append. A lock on the data inode would not serialize an atomic rename. FullscreenReturnStore has a separate stable lock, but borrowing it would couple registration to native fullscreen operations.

No native before/after capture of template, title and pane_command across exec has been made. These are source findings; pair#727's disposable native placement/focus conformance does not establish exec metadata behavior.

## Spec

Measure the native metadata across a wrapper-to-shell exec in a disposable session, then choose a bounded lifetime/role fix. A plain native fallback shell must not be treated as a Pair-aware terminal for global shortcut delivery. Keep the working shell even when bookkeeping fails. Preserve other panes and unknown registry records; avoid a new native settings resolver or heuristic expansion.

## Done when

- A disposable native regression captures the PID, initial command, title and actual pane_command before/after the handoff and proves the selected classification behavior.
- After fallback, LiveIDs and RoleForPaneWith no longer grant that pane a Pair-terminal role; ordinary active Pair terminals still work.
- Another pane's registration survives, including concurrent registration/retirement if the chosen design mutates that registry; unknown records remain intact.
- A bookkeeping failure is visible and leaves the native shell usable.

## Plan

- [x] Confirm the measured owner/role design with the hub; durable plan: [terminal-owner-role-plan](../plans/000731-terminal-owner-role-plan.md).
- [ ] Implement and qualify the shared owner observation and role projection under the approved plan, then close and land through SDLC.

## Log

Deferred from pair#727 by the hub on 2026-10-20 (delivery 7225aa16-eefc-46c0-97b6-49c250d1ca29): land the reviewed resilience improvement, which leaves a working shell instead of a dead pane. No registry/role change belongs in #727. The earlier retirement-only proposal was withdrawn before implementation when the role and lock assumptions proved incomplete.

### 2026-10-20 — claim, native measurement and proposed plan

- Hub dispatched #731 after #706 landed; verified unowned, claimed, and ran start-plan from main. No #706 work remains in flight. Bare start-plan caught up the sibling Ariadne build, as required by the fleet tool rule.
- Hub d78ab148-d0d9-49e4-9255-f5983c3ee9a6 approved bounded disposable native diagnosis. Installed Zellij0.45.1 private session p731-31448 used private HOME, config/data/cache/runtime and ZELLIJ_SOCKET_DIR. A controlled registered pair term executable gated syscall.Exec into /bin/sh. PID31490 and birth remained unchanged; startup command and terminal title survived; pane_command changed to sh -i. LiveIDs and both existing role projections still granted RightTerminal. A command executed by the interactive shell acknowledged the same PID. Cleanup closed the exact session/client; private list-sessions reported none, and the exact owned PIDs were gone. Capture: [metadata-capture.json](000731-a-pane-that-fell-back-to-a-native-shell-still-classifies-as-a-pair-terminal.side/metadata-capture.json). This is native metadata evidence, not a production fault-recovery test. Temporary probe source was removed from cmd; retained owned diagnostic source/binary are under /Volumes/TestScratch/tmp/pair731.p79ksE.
- Found a registry-only cached-draft fast path in termcmd and a direct registry reader in Couch focus admission, alongside layout/clipboard consumers. A pane-command-only fix misses that path. Proposed plan uses existing identity-fenced native per-PID argv evidence plus mandatory registered ownership in RoleForPaneWith. It leaves the append-only registry format and all unknown rows intact, avoiding retirement rewrites/new locks and new lifetime artifacts (ARCH-DRY, ARCH-EVOLVE, ARCH-FUNERAL). No production/test implementation has begun; awaiting hub plan approval before change-code.
- Hub d495d55d-e94e-4033-8b07-6095ee2c618a approved the plan with one compatibility requirement: confirmed shell argv denies; unavailable/unreadable native inspection retains today's liveness-based registry result and records why. Revised the current design/state/test tables and appended its history under Revisions; both policies and diagnostic failure are mandatory tests. No implementation yet; entering change-code next.

## Revisions

### 2026-10-20 — approved inspection-unavailable compatibility

The original native owner proposal denied unavailable inspection. The hub's approved contract distinguishes it from confirmed different argv: retain today's registry answer and bounded diagnostic reason when native inspection is unavailable. This accepts possible stale ownership in restricted contexts to preserve working terminal panes. Confirmed wrapper-to-shell exec still loses the role; registry bytes/schema remain unchanged.
