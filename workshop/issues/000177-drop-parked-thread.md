---
id: 000177
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Drop a parked thread from the switcher

## Problem

**Threads accumulate and nothing removes them.** The operation set is
`prepare-start, start, list, show, stop, name, describe, publish-description,
switch, attach, park, detach, leave, resume` — none of which deletes a durable
thread record. The two near-misses do something else:

- `stop` is `ExecuteLiveOwner` + "signal an actor's child and forget it": it
  needs a live child, and its `Forget` mutates `c.reg` (the transitional
  live-handle cache), not ThreadStore. It cannot remove a durable record.
- `park --mode=abandon` abandons an **in-flight park transaction**
  (`park.go:367-376`, keyed on `current.Park.Identity` and the record revision),
  not a completed park.

Measured on this host 2026-09-02: **17 thread records, 12 of them parked** — 5
on `pair`, 4 on `brain`, 2 on `tools`, 2 on `kbench/competition/arc-agi-3`, 1 on
`parley.nvim`. Every one of them is a switcher row the operator scrolls past.

This is the **drain** side of `pair#175`'s ratchet: that issue is about creating
duplicate threads on a tree; this one is about there being no way to remove the
duplicates once created. Fixing only one leaves the panel filling up either way.

Removing one today means hand-editing the store — delete the record file,
delete its `manifest.json` entry, bump `generation` — while the supervisor is
live. That is exactly the second-writer situation the store's lock, revision
checks and write-ahead journal exist to prevent.

## Spec

**A `drop` operation, offered on a parked row in the switcher.**

- **Declaration** (`couchcore/ops.go` + the table in `ops_declarations_test.go`):
  `ConfirmRequired`, on the same grounds as `park` and `stop` — it destroys
  durable state. `EffectProcess` is wrong; this is a membership change.
- **Route through the live owner** (`ExecuteLiveOwner`), not `ExecuteDirectStore`.
  A membership change edits `manifest.json`, and the supervisor holds its own
  view; going through it means the drop cannot race the supervisor's cached
  manifest bytes.
- **Use ThreadStore's journal-backed membership path** — the same machinery
  `CreateThread` uses, in reverse (record file + manifest entry + generation, one
  recoverable transaction). Do not add a second way to mutate membership
  (`ARCH-DRY`); manifest and records drifting apart is a store that fails to
  load.
- **Refuse anything not verified-parked.** A live, creating, or unknown
  incarnation means a running session whose record must not vanish underneath
  it. Fail closed on a contradictory or undecodable record rather than guessing.

**Two decisions for the plan, both with a recommendation:**

1. **Artifacts.** The record is couch's; the pair artifacts are pair's, keyed by
   tag — `scrollback-<tag>-<agent>.raw`, `parked-scrollback-<tag>-<ts>`,
   `draft-<tag>.md`, `log-<tag>.md`, `ledger-<tag>.jsonl`, the queue dir.
   Dropping the record alone orphans those files permanently; dropping them too
   destroys transcripts the operator may still want. **Recommend: drop the
   record, report the artifact paths, and put deletion behind an explicit
   `--purge`.** Silent transcript deletion is not recoverable.
2. **Bulk.** The standing backlog is 12, so a strictly one-ref-at-a-time drop
   with a confirmation each is 12 prompts. **Recommend: allow selecting several
   parked rows and confirming once**, with the confirmation naming the count and
   the paths.

## Done when

- A parked row in the switcher offers `drop`; confirming removes the record and
  its manifest entry in one journal-backed transaction, and the store loads
  cleanly afterward.
- `drop` refuses a thread with a live, creating, or unknown incarnation.
- A crash mid-drop leaves the store loadable — either both changes applied or
  neither, recovered through the existing journal.
- Artifacts survive by default; `--purge` removes them and says which.
- Dropping several selected rows takes one confirmation.
- After a drop, a concurrent `start` on the same tree still succeeds — the
  manifest CAS sees a consistent generation.

## Plan

- [ ] `DropThread` on ThreadStore via the existing journal-backed membership
      path; tests for crash-between-steps recovery.
- [ ] Declare `drop` in `ops.go` + the declaration table, `ConfirmRequired`,
      `ExecuteLiveOwner`.
- [ ] Refusal for non-parked incarnations, including the undecodable case.
- [ ] Switcher affordance on parked rows; multi-select with one confirmation.
- [ ] `--purge` for artifacts, off by default, reporting what it removed.

## Log

### 2026-09-02

Raised after asking to remove two parked threads and finding no operation for
it. They were removed out of band at the operator's instruction, which is worth
recording both as precedent and as the reason this issue exists:

- dropped `couch-21baa48c3a7f009b` (`/Users/xianxu/workspace/tools`) and
  `couch-6a800cdf24e346af` (`/Users/xianxu/workspace/parley.nvim`);
- procedure: back up `~/.local/share/pair/couch`, remove each
  `threadstore/records/<scope>/<tag>.json`, remove its `manifest.json` entry,
  bump `generation` (56 → 57), then assert manifest entries and record files
  match in **both** directions;
- store went 17 → 15 threads and stayed consistent.

That was done with the supervisor live, which is the unsafe part: nothing
serialized it against a concurrent `CreateThread`. It worked because no thread
was being created at that moment, not because it was safe.
