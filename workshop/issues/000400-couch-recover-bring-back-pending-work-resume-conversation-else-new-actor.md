---
id: 000400
status: open
deps: [pair#399]
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: '716d97629c9796af1becf7452b374db27907c4b7' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --recover: bring back pending work (resume conversation, else new actor)

## Problem

After a crash, getting the fleet back took manual work, and the recovery
report got in the way. On 2026-10-06 (see #399) the operator had to:

1. run `--recover-plan-from-sdlc`,
2. run its `resume` steps by hand (`ariadne:2`, `pair:1`, `pair:2`, `pair:4`),
3. and then work out the rest themselves. The report **held** 11 slots as
   `unidentified-work`.

The hold is the wrong call. `classifyRecover` (`recoverplan.go` ~610)
answers "which issue owns this work?" (claim / issue branch) **before** it
looks at the agent. So a slot with dirty, unclaimed work and a perfectly
resumable conversation gets no step. Resuming that conversation in its
worktree doesn't touch the work. It brings back the one actor that knows
what the work is. The conversation is the identification. The same holds
when no conversation exists: a new actor plus the operator can read
`git status` as well as anyone, and holding protects nothing.

The verb is also missing. `ActorActions` (`actor_actions.go:51`) offers
`resume` only when a thread record exists, so slots with no thread
(`diary:0`, `metis:0`, `robots:0`) have no action at all.

The Console also wouldn't switch to `pair:2` (already resumed and live) while
the next remote `--resume` jobs ran back to back (~12 s each). Switching to a
live thread waited on other slots' lifecycle work, with no feedback.

## Spec

### Vocabulary (operator-facing, single-sourced)

- **Startup** attaches only live agents plus the actor explicitly requested.
  This is unchanged; behavior after the crash was correct.
- **Pending work**: a slot that is not at rest, i.e. the complement of
  `hostAtRest`, given readable evidence. Either:
  1. an sdlc claim, or
  2. other work: dirty tree, unlanded commits, a non-resting branch, an
     in-progress operation.
  Both have the same remedy.
- **Recover** (the remedy): reap if orphaned (#399), then resume the previous
  conversation if one exists, otherwise start a new actor on the slot. No
  "ask the agent to claim it" note; the agent and operator sort that out.
- **Not recovered**: idle slots (at rest), and slots whose evidence can't be
  read (`git-unknown`, e.g. `xianxu:0`), which need inspection. An unusable
  or never-started thread record (`parli:1`, `tools:1`) is not an
  exception: if the slot has pending work, recover starts a new actor.

### Surface

- `couch --recover repo:N [--json]`: recover one slot through the live
  Couch, like `--resume`.
- `couch --recover --pending [--json]`: recover every slot with pending
  work, as one batch. The report's rows for those slots carry `recover` as
  their step, replacing the `unidentified-work` hold.
- The `couch` skill's "Recovering slots after a restart" section uses this
  vocabulary. "Recover pending work" from the operator means
  `--recover --pending`.
- Remote slot operations must not block the Console from switching to an
  already-live thread. When something does wait, the switcher shows it
  (e.g. "busy: resuming pair:2").

## Done when

- [ ] `classifyRecover`: pending work with a resumable conversation →
      `recover` (resume). Pending work with no thread or conversation →
      `recover` (new actor). Unreadable evidence → hold. At rest → idle.
      Table-driven tests cover each row, including the 11 cases from
      2026-10-06.
- [ ] `--recover` falls back to a new actor when a thread exists but its
      conversation can't be resumed (test pins which signal decides this).
- [ ] `--recover --pending` on a fake fleet recovers exactly the pending
      slots. A rerun of the report afterwards shows no `recover` steps.
- [ ] An orphaned slot recovers through #399's reap.
- [ ] Switching to a live thread while a remote `--recover` job runs
      succeeds, or shows busy feedback; verified with `COUCH_TRACE` (no
      `no-destination` drop).
- [ ] Atlas (`couch.md` recover-plan section) and the skill updated.

## Plan

- [ ]

## Log

### 2026-10-06

- Shaped in a brain session with the operator after the 16:47 `$TMPDIR` wipe
  (#399). Operator decisions: startup stays live-only; claim and other
  pending work share one remedy; no claim-nudge note; agent-less dirty slots
  get a new actor rather than a hold.
