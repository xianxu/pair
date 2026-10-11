---
id: 000437
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'a911781addcc081e27a23122736ea85a9ddfe560' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --session-sync: the one mutation path for a session-sync file, with an owed-events ledger

## Problem

The TL's status file (`type: session-sync`, parley.nvim#313) has become the operator's control surface, but its rules
live only in the TL's prompt, and on 10-10 the TL broke both kinds (ops learnings kink 60): it wrote the file while the
operator held the lock (safety), and it dispatched a batch without folding the operator's submit into the file
(liveness). Prompt rules erode in long, busy sessions; gates don't. The operator (10-10): make session-sync a tool the
agent has to learn and can't sensibly bypass, as `sdlc` is, and keep it agent-neutral (no harness-specific hooks).

## Spec

`couch --session-sync <verb> FILE`, the only supported way for an agent to change a session-sync file:
- `diff`: the operator's edits since the agent's last write (against the shadow copy), plus the lock holder.
- `write --section HEADING BODYFILE` / `write --all BODYFILE`: refuses while the operator holds the lock; refuses
  unread operator edits until `--accept-edits`; finds sections by whole heading lines only; restamps "Updated";
  refreshes the shadow copy. `--fold ID[,ID]` marks owed events handled in the same step.
- `release`: deletes an agent-held lock (the agent never writes the lock).
- `status [--json]`: lock holder, last agent write, and **owed events**: the operator's submits and every peer
  message delivered to the file's `owner:` slot since its last write, by delivery ID, unfolded.
- **The ledger:** Couch is the broker, so it records each submit and each delivery to an owner slot as owed against
  that slot's session-sync file; `write --fold` and `release` clear them. Lives beside the file (gitignored) or in
  Couch's store; settle in the plan.
- Agent-neutral: nothing here assumes Claude Code. The prototype is ops' `skills-draft/xx-tl/scripts/tl-status.py`
  (diff / section / replace / release with the same guards).
- Consumers: parley.nvim's liveness ping (filed alongside) reads `status --json` and `couch --actors --json`
  (`Resting`, `LastActivity`).

## Done when

- [ ] A write under an operator lock refuses and changes nothing (test).
- [ ] A write over unread operator edits refuses until `--accept-edits` (test).
- [ ] A delivery to the owner slot appears in `status` as owed until a write folds it (test).
- [ ] A section write never matches a heading quoted in prose (test).
- [ ] The xx-tl skill (ops) switches from `tl-status.py` to this verb.

## Plan

- [ ]

## Log

### 2026-10-10
