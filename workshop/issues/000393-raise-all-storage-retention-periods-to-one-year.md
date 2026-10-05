---
id: 000393
status: open
deps: []
github_issue:
created: 2026-10-05
updated: 2026-10-05
estimate_hours:
card_mirror: '0cffc0eae01c0a5a67cac144caccc4553caf5078' # card fields mirrored from issue-cards; edit via sdlc
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
