---
id: 000670
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '49303845ed66b0b93f560c9e2b7fa3780eecb0ca' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T11:33:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# couch --notify never reaches the desktop; pair review --help fails on the bare group

## Problem

Two operator-facing gaps, both seen 2026-10-18:

1. `couch --notify` never reaches the operator as a desktop notification. The hub has sent about ten today (each new Needs-you item); the operator saw none, including a test titled "Couch test". Couch reports "Notification written to the operator terminal."
2. `pair review --help` fails with `pair-go: unknown command "review"`, though `pair review apply --help` and `pair review round --help` work: review is registered only as two-word subcommands.

## Spec

1. Find where the notification is lost: the escape Couch writes (OSC 9, 777 or 99), whether zellij or another layer between Couch and the terminal swallows it, and whether the operator's terminal app turns it into a macOS notification. Fix it so a hub `--notify` reliably raises a desktop notification on this machine, or, if that depends on a terminal setting, say exactly which and have `couch --notify` report when delivery is unconfirmed rather than claiming success.
2. `pair review` with no subcommand (and `pair review --help`) lists its subcommands.

## Done when

- A hub `couch --notify` raises a desktop notification on the operator's machine (operator-confirmed), or the command states plainly what the operator must enable.
- `pair review --help` lists the review subcommands; a test covers it.

## Plan

- [ ] Add pure group help derived from existing Families, handling bare groups and
  --help/-h before leaf resolution; preserve leaf routing and reject unknown
  children. Cover all implemented groups so this grammar defect cannot recur
  in siblings (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
- [ ] Regress review bare/--help/-h with all six current leaves, unknown children,
  and sibling group help; run dispatcher tests and public CLI smoke from a
  private build. No native sessions or operator terminal writes.
- [ ] Propose notification wording: successful terminal write reports "Notification
  written; desktop delivery not confirmed." Document Ghostty desktop-notifications
  (default true), macOS Notifications > Ghostty permissions/banner settings and
  Focus policy. Do not change transport or settings without further evidence.
  Preserve one-shot/no-retry behavior and existing uncertain-write handling
  (ARCH-ORDER, ARCH-SECURE). Wait for operator evidence before larger changes.
- [ ] Run targeted packages/catalog, recorded suite and normal close review/landing
  once notification scope is agreed and acceptance evidence is recorded.

The recommended approach reuses Families and existing terminal output. A separate
review-only registry would duplicate command knowledge; changing OSC protocols
or adding an OS notifier would guess at an unproved boundary. This is quick-flow
scope (under 100 production lines), with no milestones or new dependencies.
ARCH-MOCK: existing fake terminal/broker seams qualify reporting; no new external
service. ARCH-CONSTRAINTS: pure help over the finite registry; existing notification
256/4096-byte budgets and admission deadline retained. ARCH-FUNERAL: no new durable
runtime artifact, background process or cache.

## Log

### 2026-10-18


### 2026-10-18 — read-only diagnosis and hub decision

Claimed unowned #670 and started planning. start-plan caught up ariadne and
required one rerun with its refreshed binary; the rerun created this branch.
Installed `pair review --help` exits 2 with unknown command review: Families
registers six review leaves, Resolve deliberately does not recognize bare groups,
and Dispatch has no group help path. No code changed during diagnosis.

Couch PID 41916 writes OSC `ESC ] 777 ; notify ; TITLE ; BODY ESC \` via
handleNotify -> Console.NotifyDesktop -> Presenter.Notify -> writeOneShot ->
OSHost/ttyio. consoleRunnerFor uses stdout as the output file. lsof shows fd 0,
1 and 2 all /dev/ttys000. Read-only ancestry is Couch 41916 -> zsh 35159 -> login
35158 -> Ghostty 80193. Ghostty is 1.3.1 build 15212; the Couch environment has
TERM_PROGRAM=ghostty and no ZELLIJ/TMUX. Thus neither an outer Zellij nor Couch's
child notification suppression lies on the supervisor-owned notification path.
This establishes routing, not historical output bytes or desktop presentation.

Installed Ghostty ghostty.5.md documents desktop-notifications=true by default;
its existing Application Support config has no notification/config-file override.
Official reference: https://ghostty.org/docs/config/reference#desktop-notifications.
A config file does not prove the running app's effective settings or macOS
permission/Focus state. Exact loss remains unproved; no notification emitted,
UI inspected, slot changed, restart or relaunch performed.

Hub dispatch 1739c615-c17f-4f4f-9395-247de67cf6b4 and follow-up
31fa9896-1a09-4ff4-a970-aab22e14eb0b verified submitted for this incarnation.
The latter authorizes building review help now, asks for delivery-honest notify
wording, and holds bigger notification changes pending the operator's permission
and focused-window observations. Findings delivered to hub in submitted receipt
29132b1e-437e-48c5-a65c-399fb4b19a00. Previous TTY capture is unavailable;
retained branch/card and canonical current dispatch are the authority here.
