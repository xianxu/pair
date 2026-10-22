---
id: 000817
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9c115c033f820a3414f9134cd7f64d943d6025cf' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: expose stale running wrapper revisions after binary updates

## Problem

Couch restarts and rebuilding the on-disk Pair binary do not replace existing wrappers. The operator can therefore run a current supervisor while admission still uses older code, without a clear per-wrapper indication.

During the 2026-10-22 live stall investigation for #791/#810, the wrapper mapped a different executable inode from the current disk binary. The frozen stream admitted correctly on current code and reproduced the visible-peek/hidden-admission discrepancy with pre-#735 resize behavior. The running wrapper revision was unproved. TL reports #735 landed at 13:25 on October 20, after ops:0 and roughly fifteen other wrappers started that morning; the chronic stalls are very likely stale pre-fix wrappers.

## Spec

Plan a bounded way to expose the actual running wrapper revision versus the available disk revision, including unknown evidence. Flag wrappers missing a fix-carrying landing in an operator-facing inspection such as resources or the restart summary. Do not infer running revision from the current disk file, process age, agent version or a supervisor restart. Select the actual identity/reporting seam and version/ancestry comparison during design; this filing does not authorize automatic relaunch or new admission policy.

## Done when

- Operator inspection distinguishes matching, outdated and unknown running-wrapper revisions with their evidence and an appropriate refresh action.
- A rebuilt disk binary with an unchanged old wrapper is reported correctly; refreshing only the supervisor does not clear that warning.
- Tests cover current, old, unknown and incompatible identity responses, including a wrapper that predates revision reporting.

## Plan

- [ ]

## Log

2026-10-22: Filed unclaimed at authenticated ops:0 request 6e54f49d-cdcf-44db-b1f4-53714a10c64b. Related #735, #791, #810. No implementation or rollout authorized by this filing.
