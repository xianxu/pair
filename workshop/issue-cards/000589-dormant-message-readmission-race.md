---
id: 000589
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: medium
---

# Dormant message readmission test races its last asynchronous attempt

## Problem

The recorded pair#584 full suite at code head 9979945fb failed TestMessageSendToDormantSlotReadmits in cmd/internal/couchcmd/message_service_test.go:449. The test expected Code=unavailable but received Code=accepted after restoring launch evidence. Both the test and messaging service are unchanged from origin/main in #584. The previous full suite passed couchcmd; an isolated 30-repeat rerun passed in 5.061 seconds.

The fixture fires the counted retry callbacks, sleeps 100 ms, then makes launch evidence healthy and immediately sends. That elapsed sleep does not prove the last asynchronous admission result has reached Dormant. A late attempt may validly admit after launch becomes healthy. This is the suspected race, not yet a deterministic baseline reproduction.
