---
id: 000651
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0f170cae153ccc82320122e4dcb5b5f139d4573b' # card fields mirrored from issue-cards; edit via sdlc
---

# Relaunch can strand a slot mid-park: resume refuses 'park transaction unfinished' with no named recovery verb (pair:6 had to start fresh)

## Problem

## Spec

2026-10-17 ~11:50: `couch --relaunch pair:6 --confirm`, a Codex slot on tag 1-pair-55 whose binding had just been confirmed, didn't finish. Its output ended with a recovery menu ("abandon — give up on it and leave the thread occupied"), and the slot had no live wrapper afterwards. `couch --resume pair:6` then refused: "resume-parking: a park transaction is unfinished; recover that park or Start fresh". `couch --help` names no verb for recovering a park, and the resume candidates listed only "Resume from disk (confirmed)", which refuses, and "Start fresh". `scrollback-1-pair-55-codex.raw` was missing, apparently moved by the half-done park. The hub had to discard the conversation with a fresh reboot. That was cheap here because pair:6 was idle between items, but it would lose real work mid-item.

Earlier the same day, pair:4's reboot failed similarly at zellij session ownership ("ownership unresolved"), which may be pair#640's reply misrouting.

Wanted:
- A relaunch either completes or rolls back its park, so the slot stays resumable from disk.
- An interrupted park is recoverable by a named verb, shown in the refusal: finish it, or restore the scrollback and binding, then resume.
- The relaunch's failure output names the step that failed (park, kill, spawn) and the recovery command.

## Done when

- A test interrupts a relaunch at each park step; afterwards `couch --resume` succeeds, or the refusal names a recovery verb that, once run, makes it succeed.
- The relaunch failure text names the failed step and the recovery command.

## Plan

- [ ]

## Log

### 2026-10-17
