---
id: 000437
status: codecomplete
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-11
estimate_hours:
card_mirror: '011ef74036916ca53f183f6d3adb951ecf1308aa' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T23:41:16-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.90
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

- [x] A write under an operator lock refuses and changes nothing (test).
- [x] A write over unread operator edits refuses until `--accept-edits` (test).
- [x] A delivery to the owner slot appears in `status` as owed until a write folds it (test).
- [x] A section write never matches a heading quoted in prose (test).
- [ ] The xx-tl skill (ops) switches from `tl-status.py` to this verb.

## Plan

Durable plan: [`workshop/plans/000437-…-plan.md`](../plans/000437-couch-session-sync-the-one-mutation-path-for-a-session-sync-file-with-an-owed-events-ledger-plan.md)
(decisions on ledger location, submit recognition, owed semantics, JSON shape).

- [x] File rules: sections by whole heading, restamp, owner/holder parse (`sessionsync/file.go`)
- [x] Per-slot delivery ledger in the Couch store, bounded trim (`sessionsync/ledger.go`)
- [x] Owed computation + per-file sidecar state (`sessionsync/owed.go`, `state.go`)
- [x] `Sync` diff/write/release/status with the guards (`sessionsync/sync.go`)
- [x] `couch --session-sync` CLI verb + JSON envelope (`couchcmd/session_sync.go`)
- [x] Daemon appends a ledger row per landed delivery (`couchcmd/message_service.go`)
- [x] Couch skill text, atlas, hand-off to ops:0 (gitignore line, tl-status.py mapping)

## Revisions

### 2026-10-11 — plan review (fresh-eyes subagent) and implementation
- **Owed is not "since its last write"** (the Spec's wording): a plain write never clears a delivery; only
  `write --fold` / `release --fold` do, and `release` folds the file's submits. An implicit clear on every write is
  the liveness hole this issue closes. The horizon is the sidecar's `Since` (shadow mtime on first use).
- **What counts as delivered:** submitted, indeterminate, an endpoint error or a non-terminal answer; only explicit
  cancel/expiry is skipped. Uncertain rows say so.
- **`status` refuses `not-recording`** when Couch predates the ledger or an append failed (`recording.json`), so an
  old or broken Couch never reads as "nothing owed". `release` keeps the unread-edits guard (exit 4).
- **`owner:` must be an exact `repo:N`**; anything else refuses rather than matching nothing.
- **Ledger bound:** by size (256 KiB → ≤512 rows younger than 14 days), one stat per append.

### 2026-10-11 — close review round 1 (FIX-THEN-SHIP)
- `release` takes `--accept-edits` (records read edits without a write) and both verbs take `--accept-gap`, so every
  refusal's remedy is reachable (BR-1).
- Slot addresses use couch's grammar (`couchcore.ParseWorkspaceReference`), escaped into the filename (BR-2).
- A failed append is a per-slot, restart-proof gap record; files acknowledge it with `--accept-gap` (BR-2, BR-3).
- README lists the verb (BR-4).
- **Done-when 5 moves to ops:** the xx-tl skill switch is the TL's edit in ops (the TL owns ops; this slot never edits
  it). The hand-off (gitignore line, command mapping) went to ops:0 (delivery 3c7ff078); the row is tracked there.

## Log




- 2026-10-11: closed — Merge of origin/main (#435 Deliver progress param) resolved by passing progress through and recording after; couchcmd/sessionsync/couchmessage/cmd/couch green except the pre-existing continuation test. Post-close fix: unreadable gap record dated by mtime (test + mutation-checked); writeAtomic delegates to durablefile.WriteAtomic. (fixed-then-ship; full suite: make -k test (5-var scrub): only test-changelog (passes under scratchpad TMPDIR) and test-pair-embedded-runtime fail; go test ./... -count=1: artifactpath (no #437 file listed), continuation writer, couchcore spawn, gcruntime, launcher x5 -- the same set that fails identically on the merge base in this env); review verdict: SHIP
- 2026-10-11: closed — Round-1 fixes (BR-1..4) each pinned by a test and mutation-checked: release --accept-edits/--accept-gap reachable remedies, couch ParseWorkspaceReference slot grammar with escaped filenames (non-ASCII family test), per-slot restart-proof gap record acknowledged per file, README line. go test sessionsync/couchcmd/cmd/couch: only TestContinuationWriterPublishesExactCheckpointAcrossWorktrees fails (pre-existing: fails identically on the merge base in this env); artifactpath lists none of the #437 files (its remaining violations are identical to the merge base). Earlier full suite: every other failure reproduced on a merge-base worktree. Live read-only: status --json on ops tl-status refuses not-recording against the pre-#437 Couch; diff reports lock free. Done-when 5 moved to ops by Revision (TL owns ops; hand-off delivery 3c7ff078).; review verdict: SHIP
- 2026-10-11: flow upgraded quick → full — 1289 added lines in code files (limit 100); an earlier round of this close already ran the full review
### 2026-10-10
- Claimed from the TL's dispatch (ops:0). Confirmed parley.nvim#314's `status --json` shape via ops:0 (delivery 4ce3cdf0).
- Couch persists no deliveries today (atlas: memory-only receipts); the ledger hooks the daemon's `messageEndpoint.Deliver` in couchcmd, not `couchmessage`, to stay out of #433/#435.
- parley#313's submit is a plain `couch --send-to <owner>` with body `submitted: <abs path>`; recognized by body, no wire change.
