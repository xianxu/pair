---
id: 000662
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Review observation diagnostic positive control times out in real BufWritePre resolver

## Problem

The normal make test shell/review phase can fail in ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The fixture calls a real readiness resolver during BufWritePre; a late synchronous resolver was killed at its existing 2500 ms deadline, producing an empty result and a write refusal. It blocked pair#658’s whole-device after measurement before the final Go recipe.
