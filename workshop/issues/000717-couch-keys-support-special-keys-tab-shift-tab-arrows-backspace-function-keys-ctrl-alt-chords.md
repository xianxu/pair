---
id: 000717
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 1.24
collaboration-mode: collaboration
card_mirror: '9477153666c87eb534d681466a9cf6f9c41928c7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T01:12:01-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch --keys: support special keys (tab, shift-tab, arrows, backspace, function keys, ctrl/alt chords)

## Problem

`couch --keys` accepts only `text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line` and `wait:MS`. The hub
regularly needs keys outside that set to drive a slot's agent UI without the operator: Tab and Shift+Tab (Claude's
mode cycling, completion), Left/Right, Backspace, Ctrl+C / Ctrl+O style chords, Page Up/Down, and function keys
(Codex's `f2 to view` warnings). Today each such moment becomes a "please press X" ask to the operator (10-19: the
operator pressed Esc on a `/usage` panel for the TL).

## Spec

1. A named-key vocabulary for `--keys`: `tab`, `shift-tab`, `left`, `right`, `backspace`, `delete`, `home`, `end`,
   `pageup`, `pagedown`, `f1`–`f12`, and chords `ctrl-<letter>` / `alt-<letter>` (one table: name → bytes, the single
   source the parser, `couch --keys --help` and the skill read; see pair#716).
2. Each key keeps today's admission rules: driving keys need an empty composer and a finished turn; only the
   recovery-only set (`clear-line`, `esc`, and pair#691's panel-esc) may start on an occupied state. Interrupting
   chords (`ctrl-c`) are a deliberate, separately named action with their own refusal rules, never a plain step: an
   interrupt kills a turn, so it must not ride along in a key sequence by accident.
3. Unknown key names refuse before anything is sent, listing the accepted names.

## Done when

- The new names send the right bytes (table test against the terminal sequences Pair's wrapper emits).
- A sequence containing an unknown name refuses with the list and sends nothing.
- `ctrl-c` refuses as a plain step and works only through its named interrupt form, with a test.
- `couch --keys --help` lists every name from the same table (shared with pair#716's test).

## Plan

Durable plan: `workshop/plans/000717-couch-keys-support-special-keys-tab-shift-tab-arrows-backspace-function-keys-ctrl-alt-chords-plan.md` (two milestones; state design table there).

- [x] M1 — named keys in the one step table (bytes, parser, help, skill)
- [ ] M2 — `couch --interrupt repo:N --confirm`: one Ctrl+C, only on an active turn

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.1 impl=0.16
item: greenfield-go-module   design=0.3 impl=0.24
item: milestone-review       design=0.0 impl=0.14
item: milestone-review       design=0.0 impl=0.14
item: atlas-docs             design=0.05 impl=0.04
design-buffer: 0.15
total: 1.24
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* M1 extends #716's table (smaller module, mirror). M2 is a new request shape through the reducer, wrapper, service and CLI (greenfield, single concern). Impl values are 40% of the v2 midpoints (0.4, 0.6, 0.35, 0.1). Design is low because the plan is settled and reviewed (thorough plan doc, so a 15% buffer). The calibration doc is flagged stale (#127), so the numbers are provisional.

## Log


- 2026-10-20: closed M1 — agentdrive TestStepBytes (every named key bytes), TestRefusedStepsStateTheirReason, TestUnknownStepListsTheNames, TestRecipeDirectionsAreCallerArrows, TestStepsTableIsTheParsersAcceptedSet; couchcmd TestKeysHelpIsTheVerbsOwnHelp (families once, refused chords with reasons), TestSkillKeyStepsMatchTheTable; wrapcmd TestAgentDriveSpecialKeysReachTheAgentAsTableBytes (every new key through the real writer and Pair input translation; a writer-path mutation goes red). Targeted packages green.; review verdict: SHIP
- Edges: `~ pair#716` (keys help; one shared table) and `~ pair#691` (esc on an open panel). Land after #691 or
  account for its admissibility function.
- If "special characters" meant arbitrary text in `text:` steps (quotes, `$`, backticks, newlines) rather than named
  keys, fold that in as a Spec item: today the skill advises a file for messages with shell syntax, but `--keys` has no
  file form.
