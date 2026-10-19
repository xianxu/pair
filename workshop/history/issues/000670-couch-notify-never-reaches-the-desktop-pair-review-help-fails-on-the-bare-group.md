---
id: 000670
status: done
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '82be7573820d3c459aa100dd1c048bce96d77751' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T11:33:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "222d3bbc", done: "69353fb4"}
actual_hours: 0.07
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

- Successful `couch --notify` reports "Notification written; desktop delivery not
  confirmed." README and atlas name Ghostty desktop-notifications, macOS
  notification permission/alert style and Focus requirements. Per hub approval,
  desktop presentation stays unconfirmed and no live test or transport change
  occurs pending the operator's answer.
- `pair review --help` lists the review subcommands; a test covers it.

## Plan

- [x] Add pure group help derived from existing Families, handling bare groups and
  --help/-h before leaf resolution; preserve leaf routing and reject unknown
  children. Cover all implemented groups so this grammar defect cannot recur
  in siblings (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
- [x] Regress review bare/--help/-h with all six current leaves, unknown children,
  and sibling group help; run dispatcher tests and public CLI smoke from a
  private build. No native sessions or operator terminal writes.
- [x] Propose notification wording: successful terminal write reports "Notification
  written; desktop delivery not confirmed." Document Ghostty desktop-notifications
  (default true), macOS Notifications > Ghostty permissions/banner settings and
  Focus policy. Do not change transport or settings without further evidence.
  Preserve one-shot/no-retry behavior and existing uncertain-write handling
  (ARCH-ORDER, ARCH-SECURE). Wait for operator evidence before larger changes.
- [x] Run targeted packages/catalog once notification scope is agreed and acceptance
  evidence is recorded.

Landing gates: recorded suite PASS on final code, normal close review and merge.
The worker contract permits close review to overlap a queued suite; this does
not assert a suite pass before its result.

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
- 2026-10-18: closed — Dispatcher/main CLI and repolint package tests pass; notify CLI/service regressions pass, including exact unconfirmed-delivery wording and no retries. Private executable lists six review leaves for bare/--help; leaf usage preserved. Recorded suite queued on final code and must pass before merge per tl-worker overlap rule. Hub 7339982c approves reporting/docs-only notify scope and forbids live test/transport/restart; desktop presentation remains unconfirmed. README/atlas document Ghostty and macOS requirements.; review verdict: SHIP


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

### 2026-10-18 — approved bounded delivery

Hub 7339982c-2ff2-41d6-8652-38d2d10d4334 verified submitted: approved reporting
and docs with review help, no transport change or live test pending operator.
Group-help regressions first failed with unknown-command exit 2. Pure help now
derives every visible implemented group's leaves from Families; Resolve and
streaming/leaf routes remain unchanged. Dispatcher and main package tests passed
(0.233s / 10.652s). Private executable's bare review and review --help list all
six leaves. Existing apply/round --help both still emit their existing usage;
round retains its existing exit 2 (not part of the group defect).
Initial catalog passed 1.922s. Notification exact-wording regression failed on
the old claimed-write text, before changing the output. No live notify sent.

## Revisions

- 2026-10-18: after read-only trace and hub approval, bound delivery to group
  help, truthful notification-write reporting and terminal/OS requirements.
  Operator display remains unconfirmed; transport changes and live tests held.

Targeted final runs: notify CLI/service tests PASS 0.449s; dispatcher PASS
0.163s; main CLI PASS 9.845s; full repolint package PASS 6.448s. Notification
existing refusal/lost-reply/no-duplicate cases remain green. Diff check clean.
Done-when revised explicitly to the hub-approved bounded acceptance; desktop
presentation remains unproved and is not claimed by these tests.

Recorded suite queued on pushed code bcc7af1de (Git push confirmed). No-upstream
push was corrected with explicit upstream; initial local-check refused the
unpushed head and did not run tests. Source unchanged between the refusal and
recorded queue admission. Hub approved no live notification or restart for this
landing; operator desktop evidence is still pending.

### 2026-10-18 — byte comparison and close review

Hub d15ac52a-30a4-4e4c-8923-b5f77ab8ef52 verified submitted: operator recalls
many prior forwarded popups, weakening permission/focus as the leading hypothesis.
Read-only comparison of suppression commit dc428db5c's parent shows old
TestSplitNotificationAcrossTakeover and TestHiddenNotificationDoesNotWaitForActivePartialSequence
asserting host bytes `\x1b]777;notify;pair;BODY\x1b\\`. Canonical child ingress
was OSC777/BEL, while Zellij-mapped ingress was OSC9 `pair: BODY` with BEL/ST;
Endpoint converted both to effects and Presenter.EmitEffects normalized host
output to OSC777/ST. Current --notify writes the same protocol/terminator with
default title Couch instead of pair. Same title/body yields identical envelopes.

NotifyDesktop -> Presenter.Notify -> writeOneShot -> write/ttyio bypasses child
policy, endpoint parser, effect deduplication and frame diffing. Successful return
requires complete write; stalls/partial failure return an error. Existing
exact-wire/serialization/no-replay/partial-write tests PASS terminal 0.174s and
supervisor-versus-child-policy test PASS couchtty 0.291s. No live output observed.
Static code/test evidence cannot prove bytes emitted during past missed popups.
Report submitted to hub as fc87949f-155b-4dae-aaf7-f3dcecc59eb9; transport unchanged.

Close review round 1 SHIP with no findings; reviewer independently reverted
both behavior fixes via overlays and observed the new regressions fail. Its
catalog host-metadata gap was qualified with process-only Git configuration.
No new lesson from findings (none). PR 436 opened; suite remains queued, no pass
claimed. Live presentation remains unconfirmed as approved.

### 2026-10-18 — recorded suite failure and one authorized rerun

Attempt 9989aa4b1c7d8bcec0eddeb488549f8c on bcc7af1de155c07a8bf07066863c4d9964490e6a
waited 2877s, then failed after 2m57s, log SHA256 b8462fdea8a6. Retained log:
.git/sdlc/test-stats/failed/9989aa4b1c7d8bcec0eddeb488549f8c.log.
Only failure: tests/without-ancestors-test.py LifetimeTests.test_cancellation_during_startup,
SIGINT delay0.02s; helper33008 stayed live (returncodeNone), descendant_group33031,
zipimport KeyboardInterrupt. All six methods completed, one subtest failed;
make stopped before Go, so no full suite pass is claimed.

Hub 35472371-02ab-4c10-8748-6899b5edc48b verified submitted: #671 was its
wontfix duplicate of #674; #674 backs ONE unchanged rerun. Same failure again
means stop/report, with no qualification around it. Exact source --rev was
refused before execution because bcc7af1d is no longer a remote branch tip;
current head e56956f06dee2ea66466ce0474b7699c8db55b5d differs only in issue
Log/close records. The rerun uses that current pushed head with identical code,
tests, configuration and suite command. No retry result yet.

One authorized unchanged-code rerun b44358d2252bc8799463939dd441ddfe PASSED on
e56956f06dee2ea66466ce0474b7699c8db55b5d: queue2173s, runtime11m35s,
log SHA256 ab0545e940a5. Retained log:
.git/sdlc/local-checks/e56956f06dee2ea66466ce0474b7699c8db55b5d/suite.log.
All six without-ancestors tests passed in1.841s, shell/editor phases and final
Go packages passed; no second failure. This is the only rerun, backed by #674
and hub approval. Source/tests/config are identical between failed and passed
attempts; intervening changes are issue/close logs only. Entering normal merge.
