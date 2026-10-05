---
id: 000393
status: working
deps: []
github_issue:
created: 2026-10-05
updated: 2026-10-05
estimate_hours:
card_mirror: '0178dba3a2dd86af8fd0a3157ede0b952c4aed8b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-05T12:06:46-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "4be4cfd8", done: "c54c1b42"}
---

# Raise all storage retention periods to one year

## Problem

The operator wants pair/couch storage retention effectively off. Three periods
currently delete data:

- `storagegc.RetentionPeriod`, 60 days: session data (drafts, prompts, ledger,
  terminal data, recovery state).
- `storagegc.CaptureRetentionPeriod`, 7 days: raw/events captures of parked threads.
- `diagnosticlog.RetentionPeriod`, 7 days: diagnostic logs.

## Spec

Set all three periods to 365 days (operator decision 2026-10-05: all three, not just
session data). Collection keeps running and still never removes data whose owner is
alive; nothing becomes eligible until a year has passed.

User-facing text that hard-codes "60 days" or "7 days" (decision reasons,
`pair gc` output, README, atlas) is derived from the constants or updated, so it
can't drift from the values again (ARCH-DRY).

Risk accepted: #376 (diagnostic logs don't rotate while Couch runs) can now grow
logs for a year instead of a week. Watch the data directory size.

## Done when

- The three constants are 365 days, and the existing boundary tests (which already
  use the constants) pass.
- No user-facing "60 days"/"7 days" retention text remains.
- `go test` passes for storagegc, diagnosticlog, gcruntime and gccmd.

## Plan

- [ ] Change the constants and derive the reason/output text from them
- [ ] Update README and atlas/storage-retention.md

## Log

### 2026-10-05

- Operator request: "turn it off by setting retention period to 1 year", for all
  three periods.
