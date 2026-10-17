---
id: 000634
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# every step of a couch send spends the one exchange budget; no fresh timers outside admission

## Problem

A `couch --send-to` exchange has one budget, `SendTransportTimeout` (`AdmissionTimeout` plus 1 s), on both ends
(pair#618). Two steps of the send spend time outside the broker's `AdmissionTimeout`:

- the sender's authority check (`messageAuthority.current`, `message_service.go` around line 851), which runs before
  `Broker.Send` starts its own fresh `AdmissionTimeout`;
- `refusalFamilies` (`couchmessage/refusal.go`), which, after a failed admission, starts its own 250 ms timer from the
  broker context and repeats the families read `admit` already did.

The margin holds today: 2.25 s of broker time against 3 s. But a slow sender check moves the broker's verdict past the
exchange deadline, and the CLI then drops it. That is pair#618's original symptom: a refusal reason that never
reaches the sender. Found by pair#618's close review (BR-7).

Optional, same files: `serve` decodes each request twice (`requestBudget` and `requestIsRecoveryPreview`), and `Call`
marshals twice (BR-8). Fold that in only if the budget change touches those lines.
