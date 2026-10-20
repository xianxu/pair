---
id: 000741
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# Couch: live concurrent resume conformance with an owned hub caller

## Problem

Follow-up to pair#726. Tier A covers the real broker socket, CLI and console recovery/launch/adoption with three parked slots and stateful fake native dependencies. It does not qualify the same concurrent batch against a running native Couch supervisor with a fixture-owned live hub caller identity. The operator's recovery batch exposed client exchange timeouts that narrower tests missed.
