---
id: 000561
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '12884d43813c0315d0e63fcd384da3ff6aab5fef' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T20:20:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch is crash-only: always write a crash record, and reconcile in-flight messages against transcripts after a restart

## Problem

On 2026-10-14 the laptop was force-slept on 1% battery (19:36–19:47 PDT). After the wake, Couch logged its last event
at 19:48:33 and exited **without a crash record** (the three earlier crash records, 10-09 and 10-10, are all "the
terminal stopped accepting output (a write waited up to 5 s)"; the operator saw an accumulated TTY log, consistent with
a repaint burst hitting a just-woken terminal). Messages in flight to the old Couch were lost; workers resent theirs,
but nothing told a sender which of its sends were undelivered: `couch --message-status` answers "no connected recipient
retains this message", because receipts live only in Couch's memory.

The operator's direction: **Couch is crash-only.** Don't make it survive every terminal or OS condition. Design the
rest of the system so a Couch exit is routine. Durable state lives in git (issues, branches, suite records) and in the
agents' transcripts. Couch rebuilds from them on restart, as #528's auto-attach already does for slots.

## Spec

1. **Every exit leaves a crash record**, including exits on a blocked terminal write: the cause, build, uptime and the
   last few supervision events, written before any terminal I/O that might block (or from a watchdog).
2. **Message reconcile after a restart:** for messages sent in the last N minutes (default: since the previous Couch
   generation's last event), match each sender-side ID against the receiver's transcript, where each delivery is
   printed as `[Couch peer from <slot>; delivery <ID>]`. Report delivered, undelivered and unknown (transcript
   unreadable) per message, and offer, or for the hub perform, a resend of the undelivered ones. This is a CLI verb
   (agent-first); the UI may show a summary.
3. `--message-status` after a restart falls back to the same transcript evidence instead of "no connected recipient
   retains this message".
4. Record the crash-only principle in the slot-recovery target (workshop/targets/slot-recovery.md).

## Done when

- A forced terminal-write stall produces a crash record (test with a fake terminal that blocks).
- After a simulated restart, the reconcile verb classifies sent IDs correctly against stateful fake transcripts
  (Codex and Claude shapes; Qoder if its transcript is readable), and resends only the undelivered ones.
- `--message-status` for a pre-restart ID answers from transcript evidence.

## Plan

- [ ]

## Log

### 2026-10-14
