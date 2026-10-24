---
id: 000872
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
---

# Admitted worker reboot loses steward positive-list authentication during receipt polling

## Problem

In the approved #865 disposable Claude scale collection, N10 failed while starting worker5. The CLI first safely retried a catalog-warming refusal, then admitted reboot request c7f11c5f-10b2-472f-822d-ced172133631. Subsequent operation-status polling failed with unavailable: caller steward thread9223742121301770243-fixture-1 was not positively listed. The CLI correctly reported outcome uncertain, and no reboot was resent. This is a captured symptom; actual session disappearance versus discovery/projection error is unresolved. Do not assert a confirmed scale threshold or merge this with #866's earlier startup-probe panic.
