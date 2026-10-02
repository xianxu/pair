---
id: 000381
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'eabe8962cfc55e35d12f25fe6cd5b133f17fb836' # card fields mirrored from issue-cards; edit via sdlc
---

# codex couch threads never bind their native conversation

## Problem

The `brain` couch thread (`couch-6b111ea230c149dc`, scope `2e51fcf9799b1d8f`,
agent codex) ran from 2026-09-29 to 2026-10-01 without its launch ledger ever
recording a binding to the native codex conversation. The ledger held one
`launch` row and nothing after it. When couch parked the thread, it had no
conversation to resume into, and the row turned `unusable: binding lost —
repairable`, so the operator could not resume it.

The conversation was intact and unambiguous. The root rollout was
`01a0eda0-b38c-7992-988a-74ad95e6925e`, started 3s after the launch. Every
other brain rollout in the window was a subagent whose `parent_thread_id`
pointed at it. `pair session-repair codex couch-6b111ea230c149dc --scope-key
2e51fcf9799b1d8f` (preview) found it as a "unique current-launch prompt and
progress correlation", which means the evidence for binding was there all
along.

Repair diagnostics: 42 `parent_conflict` errors, "Codex metadata ID disagrees
with path ID; retaining filename identity", all from forked subagent rollouts
(`forked_from_id` = the root), plus about 14k `turn_unusable` entries. Suspect:
forked subagent transcripts carry the parent's session metadata, and the live
binder treats that as a conflict or ambiguity and never commits a binding. The
offline correlation in the repair tool tolerates it. This is not yet
confirmed.

## Spec

- Find why the live binder (the sessionwatch incremental inventory) never
  confirmed a binding for this launch, when the repair-time correlation found
  a unique root. Reproduce with a fake codex store containing a root plus
  forked subagent rollouts that share the root's metadata id.
- The binder and repair should share one correlation rule (ARCH-DRY). If repair
  can prove the root, the live path should too.

## Done when

- A test: a codex launch whose store holds the root plus forked subagent
  rollouts (metadata id = root) binds to the root during the live session.
- Parking such a thread leaves it `parked (resumable)`, not `binding lost`.

## Plan

- [ ]

## Log

### 2026-10-01
