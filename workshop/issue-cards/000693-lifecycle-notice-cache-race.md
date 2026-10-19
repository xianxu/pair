---
id: 000693
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
severity: medium
---

# flake: lifecycle primary-slot notice can race verified workspace publication

## Problem

The #681 integrated suite on 807115355d853aa5ef041dd212fd90f0142d4994 failed TestLifecyclePublisherRegisteredPrimarySlotIsNotOrdinaryContent after1139.55s with "no lifecycle message" (lifecycle_notices_test.go:556). The package reached its20m deadline as the next, unrelated TestMessageAdmissionRejectsMismatchesThenGoesDormant/root began (0s). All other Go packages and prior script/editor checks completed successfully.
