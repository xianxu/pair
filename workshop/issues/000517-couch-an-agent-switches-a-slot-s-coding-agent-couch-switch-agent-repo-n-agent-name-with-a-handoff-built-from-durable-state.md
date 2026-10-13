---
id: 000517
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '3f2027ec27910a8613d20896a8a7f21736ad8b91' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T20:05:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state

## Problem

The operator (2026-10-12, Claude quota at 80%) wants an agent-guided agent switcher the TL can use, to move slots to Codex as they free up. Today only the operator can switch a slot's coding agent (the thread's actions, **switch coding agent**), and its handoff feels robotic: Couch launches a fresh conversation and asks the new agent to read the outgoing context and summarize its orientation. Related: pair#414 (a Claude → Grok switch lost its orientation and the attach was refused).

## Spec

Agent-guided: the agents who know the work write the handoff; Couch does the mechanics.
1. **A verb for agents:** `couch --switch-agent repo:N --agent NAME [--handoff-file PATH] --confirm [--json]`, the same operation as the switcher's form, with its guards (refuses a busy or unknown slot). Cost-tagged `[heavy]` in help. The TL can call it for any slot; a slot can call it on itself (`--switch-agent self`).
2. **The handoff is written by an agent, not generated:** before the switch, the outgoing agent (or the TL, for a slot it supervises) writes a short brief the way a colleague would: what the issue is, where the work stands (branch, last commit, what's verified), what's next, open questions, and anything not in the issue file. Couch delivers it verbatim as the new agent's first message. With no handoff (an idle slot on its resting branch), the new agent starts fresh with no orientation prompt.
3. **Durable state stays the source:** the brief points at the issue's Spec, Plan, Log and Revisions rather than restating them; the outgoing agent commits its work first.
4. The switcher's form gains the same path: it asks the outgoing agent for the brief, then switches.
5. Fold in or reference pair#414's attach/orientation failure, so delivery is reliable across agents (Claude, Codex, Grok).

### Agreed design direction

Use the existing broker, park/start, transcript resolver, retention lease and
orientation delivery. Persist one exact previous-session descriptor before the
target starts; it survives successful launch and is selected by `--peek
repo:N --previous`. Full recording reads run in a short-lived CLI. A short
agent-authored brief passes unchanged; a resting idle slot starts with no prompt.
The form asks the outgoing agent to commit and call `--switch-agent self`;
Couch acknowledges before waiting for the same authenticated incarnation to
settle. The brief timeout selects pointers-only content; a longer busy timeout
refuses without interrupting work. The detailed plan proposes limits for review.

## Done when

- The TL switches an idle slot to Codex with one couch call, and a mid-issue slot to Codex with a brief written by its outgoing agent; the new agent resumes the issue from the brief and the issue file.
- The operator approves the handoff's feel on a live switch.
- Tests cover the guards, delivery of a brief verbatim, and the no-handoff fresh start.
- The incoming agent can identify and read the exact previous native transcript
  and preserved tty recording, including an explicit full plain-text read;
  missing evidence is reported rather than silently substituted.
- The form completes without a second click. Authenticated current-incarnation
  callbacks, pointers-only timeout fallback, and a longer busy refusal are tested.
- The TL reviews the switch procedure in ops/skills-draft/xx-tl/SKILL.md and
  coordinates first use on paused ariadne #230/#282/#305 with committed briefs.

## Plan

Durable plan: [agent switch plan](../plans/000517-agent-switch-plan.md).

- [ ] M1 — Preserve exact previous-session evidence and expose previous/full tty reads.
- [ ] M2 — Add authenticated asynchronous switch requests, verbatim briefs and fresh starts.
- [ ] M3 — Wire the form, TL procedure, live acceptance and rollout.

## Log

### 2026-10-12

- Claimed in pair:2; start-plan refreshed the local ariadne tools, then started
  the issue branch from main. No implementation changes yet.
- Read #414: its details contain only the title, no reproduction. Existing
  switch tests and live registration/delivery are the regression surfaces.
- Metadata audit: 436 parked raw files and 436 events files, 286 capture metadata
  files, 70 live raw files; latest parked raw approximately 9.4 MB. No transcript
  content read and no live session changed. Large peek requests already fall
  back to current recording replay; the missing capability is previous selection.
- ARCH-DRY/ORDER: reuse current native resolution before park, exact preserved
  capture after park, and existing delivery state; add one source-bound switch
  coordinator rather than duplicate CLI/form lifecycles. Full replay remains
  outside the supervisor because the existing emulator keeps a drain goroutine.
- Durable plan records three actual review boundaries, test commands, proposed
  operating limits and live acceptance. Waiting on TL's plan review, then
  change-code's plan-quality gate; estimate follows plan-quality.

- Plan-quality round 1 refused with PQ-1 (request correlation) and PQ-2
  (function-level testing strategy). Verified both: receipt keys are caller
  scoped and cannot correlate an outgoing callback alone. Plan now specifies
  `--request ID` plus exact source identity and names risk-focused test seams.

## Revisions

- 2026-10-12 — the operator: the switch is driven by the TL's skill. Done when also: a "switch a slot's agent" procedure in
  ops' `skills-draft/xx-tl/SKILL.md` (when to switch, how to write the brief, the exact couch call, how to check the new
  agent picked it up). The TL reviews that section; the first real use moves the paused ariadne items (#230, #282, #305)
  to Codex with briefs.

- 2026-10-12 — operator scope revision relayed by ops:0 (receipt
  `4f8b1134-cc92-4af2-b4e0-a1deecf71f1a`): the incoming agent must be able to
  find and read the previous session itself. The brief names the previous
  agent-native transcript and per-incarnation tty log, stays short, and points
  at durable issue state. Measure the existing recording and discovery paths
  before designing. Add previous-incarnation peeking and access to the whole
  tty log as readable plain text (all or paged), with cost tags. The current
  live-buffer peek is insufficient even with a large requested line count.

- 2026-10-12 — TL confirms the operator flow (receipt
  `4a85bea3-11b4-435c-b08d-663b6c18c851`): the form requests the outgoing
  agent's brief; that agent commits and calls `couch --switch-agent self`
  with its handoff file. No second operator click. Couch acknowledges before
  waiting for that same incarnation's turn to end. A self request must prove
  the current caller incarnation. A configurable brief deadline (a few
  minutes) selects a pointers-only fallback and reports it on the notice line
  if the agent does not call back. The TL uses the same verb with `repo:N`.
  Deadline behavior while the source remains busy is being clarified.

- 2026-10-12 — TL confirms deadline policy and audit architecture (receipt
  `0106a718-0d2b-46d2-9261-c035f81d798f`): brief timeout selects fallback
  content only; it never interrupts a busy source. A second, longer deadline
  refuses and reports the reason on the notice line. Exact previous descriptor
  plus previous/full peek is approved as the design direction. Added these
  acceptance criteria and the durable three-milestone plan for review.

- 2026-10-12 — implementation plan approved by TL, receipt
  `ebf27554-f0e7-4d4e-8b00-2b46b83c529f`; defaults and full-read limits
  accepted. TL owns live-check scheduling. Proceed to change-code plan-quality;
  no permission to skip any gates. Add dirty-source commit/self guidance to M3.
