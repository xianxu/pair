---
id: 000528
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '74f6a4c0e5527e6a03f93bb0e551bc0a5d4cef28' # card fields mirrored from issue-cards; edit via sdlc
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

- [x] Measure native metadata, exact pair:6 evidence and current recovery owners.
- [ ] Approve durable design: workshop/plans/000528-slot-recovery-plan.md.
- [ ] M1 — Prevent split terminal incarnations and show attachable-agent liveness.
- [ ] M2 — Calibrated native recovery with tty context and safe fresh fallback.


## Revisions

- 2026-10-13 09:03 PDT — TL receipt a9307d75-b068-482d-8738-e8d6a344640a
  approves two measured corrections: the cutoff is the launch producing the
  retained tty evidence, not a later empty/recovery helper start; classify
  subagent/tool execution by source/parent metadata, not originator alone.
  The original Spec above retains the dispatched wording as provenance; these
  corrections govern implementation.

- 2026-10-13 09:12 PDT — TL receipts 08e69ac9 and f0c32c35 supersede the
  initial ladder priority and unconditional-confirmation policy. The original
  pair:6 agent remains alive; a second same-name Zellij server claimed the
  operator-facing socket. Prevent split incarnations first, with shared
  ownership observations and a Pair process lock; measure serialization-off
  configuration on isolated scratch sessions before considering a Zellij fork.
  Liveness means the agent the operator can attach to. After safe ownership is
  established, calibrated strong inference auto-selects, close calls ask, and
  resumed native conversation receives tty-tail context. Pair:6 is read-only;
  its live recovery stays with the operator. See the authoritative revision in
  workshop/plans/000528-slot-recovery-plan.md.

- 2026-10-13 09:21 PDT — TL receipt baf2f67b approves no-fork safety:
  serialization disabled for new servers, exact launch-policy attestation,
  Pair-level effect lock, and unsafe legacy/contested attach refused into one
  visible Relaunch action. Optional legacy cache cleanup is a separate machine
  operation with exact paths and operator approval; no deletion is authorized.
  Full implementation design and calibration are in the plan's final revisions.

## Log

- Measurement: pair:6 scope 50f39cf60172c338, tag 1-pair-42. Root
  01a1333f-ffcb-7760-a1b1-50f52544102d is source=cli, originator=codex-tui,
  parent absent, exact worktree cwd. The 20:15:55 rollout is its subagent,
  not a second interactive root. Ledger rows 17/18 identify the original
  Oct 12 20:13:11.331 launch (ordinal 18, Pair-log offset 0); root begins
  3.426 seconds later. Last activity Oct 13 07:54:11.154 precedes the latest
  recovery helper by 93.180 seconds, but differs from tty mtime by only 15ms.
- Existing RenderBoundedLines with 100 history rows returned 139 rows including
  viewport. Whitespace/Markdown-decoration normalization gives a full final
  assistant-message match: 28 words, 21/21 eight-word windows (100%). Overlay
  measurement passes in 0.902s. No live slot state changed.
- Claude 2.1.296: cwd/entrypoint/isSidechain/message timestamp exist; root cli
  versus sdk-cli and subagents are distinguishable, but cwd may change within
  a conversation. Grok 1.0.50: 42/42 summaries have info.cwd; only 4/42 declare
  session_kind, all headless. Missing kind and relationship=primary do not
  prove interactive. Scanner chronology is first activity, not last; existing
  causal event normalization omits assistant text and cannot be used unchanged
  as a display-tail renderer.
- Liveness audit: inventory trusts hosted/recorded Pair helpers, detached
  projection trusts server/client count, and reattach overlay promotes attach
  completion to live. All three consumers need the same agent-usability fact.

### 2026-10-13
- Verified dispatch receipt 65b09489-edf7-445a-8c0d-32e0c934eb62.
  Filled the skeleton from the operator/TL requirements; claim already belongs
  to pair:2. Measurement and design only at this stage.

- Scratch Zellij 0.45.1 measurement: all tested disabling configs still read
  preexisting serialized layouts; live attachment preserves the original
  server. New nonserializing sessions refuse dead attachment. Cache env
  overrides tested did not redirect the macOS cache. All scratch servers
  cleaned. No live Pair process, record, capture or cache was changed.
- Advisory spec review found and the revised design addresses the second
  Pair binding gate, pre-ledger durable target intent, absence-backed capture
  provenance (not a forged park), and the missing historical launch/capture
  join. One advisory pass only; mandatory plan review remains change-code.
