---
id: 000683
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
severity: high
---

# Successful park cleanup remains unresolved and lifecycle failures lose diagnostics

## Problem

Operator UI relaunches of ariadne:0 and pair:0 failed on 2026-10-18 around 17:49–17:51 PDT. The reported Couch build included #651, #676 and #670. Each cleanup helper published a matching success completion and preserved scrollback, but Couch left the park awaiting_completion and did not publish VerifiedPark. Subsequent resume refused resume-parking. The original UI failure message was not retained with an inspectable operation, and the hub reported no new launch-diagnostics/failures.json entry; the initiating cause could not be established from the surviving evidence.

The operator later recovered through the menu; both outcomes were reported as recovered (Fresh). Read-only inspection found new live tags, while the old records had been archived with their parks still unresolved. Whether the operator selected Fresh or Couch selected it is not established; do not infer an automatic Fresh transition from the label alone.
