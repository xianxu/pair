---
id: 000528
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '8839eac90d5d212164a34973d2f16bc711687428' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T08:57:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log

## Problem

On 2026-10-13 pair:6 could not resume: relaunch reported
`resume-binding-provisional`, claiming no completed turn proved the conversation.
The operator could identify its Codex rollout (2026-10-12T20-13-14,
01a1333f-…) by worktree cwd, codex-tui originator and 07:54 last activity,
matching the tty's “Worked for 19m 19s • 7:54 AM”. Another rollout at 20:15:55
(last active 07:37) shared the cwd but belonged to a boundary review. After
Couch restart, Zellij resurrected empty “Waiting to run” panes while Couch
still reported the slot live. The operator called this a weak spot in the
Couch/Pair stack and requested robust recovery with a few visible states,
each offering one primary action.

## Spec

Measure Claude Code, Codex and Grok transcript metadata first: worktree cwd,
interactive provenance, last activity and extractable tail text. Recovery ladder:

1. Proven binding resumes through the existing path.
2. Unproven binding gathers interactive transcript candidates with exact slot
   worktree cwd and modification after the slot's last launch. Rank by activity
   alignment with the tty and transcript-tail/tty-tail text overlap. Exactly one
   strong candidate offers an explicit confirmation: resume the session with
   cwd, last-active time and tail-match percentage shown. Never resume inferred
   identity silently or treat an unconfirmed heuristic as proven authority.
3. Ambiguous or absent candidates offer fresh startup oriented from preserved
   tty evidence, branch and issue. Orientation names the exact previous screen
   path and `couch --peek repo:N --previous --all` as the read route.
4. Dead recorded processes or resurrected panes with no running commands show
   “needs relaunch”, never “live”. Unknown observations retain their uncertainty.

Design before code; report the durable design to the TL for approval. Observe
pair:6 read-only while measuring; no live recovery/reboot is authorized here.

## Done when

- pair:6's exact case offers the correct inferred Codex session with evidence
  and resumes it only after operator confirmation.
- Ambiguous/no candidates select a fresh tty-oriented session as the fallback.
- Resurrected/dead slots show “needs relaunch”.
- Tests cover every rung using measured native transcript shapes and stateful
  lifecycle doubles, including reviewer exclusion and uncertain observations.

## Plan

- [ ] Measure native metadata, exact pair:6 evidence and current recovery owners.
- [ ] Write durable design and obtain TL/operator approval before code.


## Log

### 2026-10-13
- Verified dispatch receipt 65b09489-edf7-445a-8c0d-32e0c934eb62.
  Filled the skeleton from the operator/TL requirements; claim already belongs
  to pair:2. Measurement and design only at this stage.
