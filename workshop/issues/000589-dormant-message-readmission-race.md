---
id: 000589
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: medium
card_mirror: '6a48f46aac4ca1bff3218e0bd7ca3cea08a95b47' # card fields mirrored from issue-cards; edit via sdlc
---

# Dormant message readmission test races its last asynchronous attempt

## Problem

The recorded pair#584 full suite at code head 9979945fb failed TestMessageSendToDormantSlotReadmits in cmd/internal/couchcmd/message_service_test.go:449. The test expected Code=unavailable but received Code=accepted after restoring launch evidence. Both the test and messaging service are unchanged from origin/main in #584. The previous full suite passed couchcmd; an isolated 30-repeat rerun passed in 5.061 seconds.

The fixture fires the counted retry callbacks, sleeps 100 ms, then makes launch evidence healthy and immediately sends. That elapsed sleep does not prove the last asynchronous admission result has reached Dormant. A late attempt may validly admit after launch becomes healthy. This is the suspected race, not yet a deterministic baseline reproduction.

## Spec

Measure the admission ordering with a controllable stateful fixture. Establish the exact Dormant state before restoring evidence, and prove the intended unavailable-send followed by successful readmission without an elapsed sleep as the oracle. Preserve valid production sends; do not make healthy admission refuse to satisfy this test.

## Done when

- A deterministic ordering test reproduces the fixture race or identifies a different root cause.
- The dormant readmission test waits for the relevant state/event rather than a fixed 100 ms delay.
- Repeated isolated and full-suite verification passes under concurrent load.

## Plan

- [ ]

## Log

2026-10-15: Filed unassigned from pair:2 while #584 remains on its own branch. Failing recorded run log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/79185f3a4b7716015218ff34bc63d1c8/run.log. Isolated command: go test ./cmd/internal/couchcmd -run '^TestMessageSendToDormantSlotReadmits$' -count=30 (PASS 5.061s). No production slots were touched.
