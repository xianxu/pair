---
id: 000669
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Native continuation fixture times out during registration

## Problem

TestContinuationZellijSeedTransportLive fails its original 25-second registration context on unchanged main as well as #665 M2. This blocks native acceptance independently of the M2 readiness-sleep repairs. Retain the distinction between fresh ready-record admission and later ownership admission; the two retained failures do not prove identical internal causes.
