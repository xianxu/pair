---
id: 000589
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: '949f2fa61004c81422d3f3e2de4d5d15371783bd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:45:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "92f53f2f", done: "94047817"}
actual_hours: 4.94
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

- [x] Reproduce the late final retry with a controlled authority barrier; record the expected unavailable assertion failing.
- [x] Synchronize retries and send-triggered readmission with channels at the existing authority/retry seams; keep launch unhealthy until the refusal is observed. No sleeps or longer deadlines (ARCH-ORDER, ARCH-DRY).
- [x] Run repeated race verification and the recorded full suite; close through the SDLC review, then await SHIP for landing.

## Log


- 2026-10-16: closed — Controlled late-final-admission reproduction failed the original unavailable assertion; fixed fixture passed 50 race repeats, TestMessage* -race and couchmessage -race. Recorded full suite passed unchanged ff788e65 after the sole unrelated launcher flake was filed as #609 under TL-approved one-rerun policy. Fixture-only change creates no architectural surface (--no-atlas); no production admission changes, sleeps, or extended deadlines.; review verdict: SHIP
2026-10-15: Filed unassigned from pair:2 while #584 remains on its own branch. Failing recorded run log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/79185f3a4b7716015218ff34bc63d1c8/run.log. Isolated command: go test ./cmd/internal/couchcmd -run '^TestMessageSendToDormantSlotReadmits$' -count=30 (PASS 5.061s). No production slots were touched.

2026-10-15: Controlled ordering proof: a temporary overlay held the target's sixth (last) admission at messageAuthority.launch across the test's original 100 ms sleep. It then applied the original launch=true mutation, released that admission, waited for its valid connection, and sent. Send returned accepted with exactly six target admission calls: no send-triggered readmission occurred. This deterministically proves a fixture ordering consistent with the full-suite failure, without claiming that run's exact schedule. Ten repetitions passed under the race detector in 3.279s. Messaging source and test remain unchanged from main; no production slot effects.

Local measurement artifacts (not required to understand the issue): /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-589-measure.ljLPET/. proof.json, proof_test.go and proof.log retain the controlled case. Command: go test -race -overlay=<artifact-dir>/proof.json ./cmd/internal/couchcmd -run '^TestDormantLateFinalRetryMeasurement$' -count=10 -v.

A fixture-only candidate overlay passed fifty race-detector repetitions in 5.649s (candidate.json, candidate_test.go, candidate.patch, candidate.log). It keeps launch=false, holds the seventh admission, and sends until that admission begins, requiring unavailable while held. The seventh check proves a send awakened Dormant; only then does it set launch=true, release, and await connection. This removes the sleep oracle and also prevents a fast successful readmission from racing the first-send refusal assertion. This is measurement, not a landed fix; the issue's owner should choose the smallest actual-state synchronization. TL approved keeping the fix separate from #584 and using the unchanged-head recorded rerun under the flake rule (8e95bb66).

2026-10-16: pair:1 claimed the released item. Small fixture-only plan posted to ops:0; authorized to proceed after posting. Production admission behavior stays unchanged. Final landing waits behind #544, #605 and #607.

2026-10-16: Red: go test -race -overlay=/tmp/pair-589-red-qvv91upo/overlay.json ./cmd/internal/couchcmd -run '^TestDormantLateFinalRetryMeasurement$' -count=1 -v failed at the original unavailable assertion with Code=accepted, after explicitly holding the final admission across launch restoration. The overlay reuses pair:2's controlled proof with the original expectation; it does not change production code. Green: the fixture-only replacement passed 50 race repetitions (2.758s); TestMessage* passed with -race (8.394s), and couchmessage passed with -race (9.828s). Retry callbacks now arrive on a channel; launch remains unhealthy until the post-ladder admission starts, and that admission is held until the unavailable response is observed. The existing 3s rig bound is retained; no sleeps added. No production behavior or architectural surface changed.

2026-10-16: Negative control: disabling the production Registry SendTargeted→readmit transition via /tmp/pair-589-control-t3ey19h0/overlay.json made the fixed test fail within its existing 3s bound; the healthy production transition passed 50 race repetitions. This confirms the test reaches the production readmission path.

Recorded full suite on ff788e65bb02408429d7e974389a2511d8492db2 completed all tests and packages, failing only TestCancelledHandoffReportsAChildsCleanExit in launcher (handoff -1,nil vs expected 0,nil). No launcher diff from main. Its 30 isolated -trimpath repetitions passed (4.425s). At ops:0's direction this separate fixture failure was filed as unassigned #609, preserving failure log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/515ad3a2617b9f0f7d1aa19c39691236.log. The one authorized unchanged-head rerun passed every check: sdlc local-check run --name suite --rev ff788e65bb02408429d7e974389a2511d8492db2, exit 0 in 15m17s, log sha256 prefix 66c7dd9900bb, recorded sdlc-local/suite on that head. Its full log is /Users/xianxu/workspace/pair/.git/sdlc/local-checks/ff788e65bb02408429d7e974389a2511d8492db2/suite.log. No sleeps or longer deadlines in the fixture fix; no new architectural surface, so close acknowledges no-atlas. Mandatory close review follows; landing remains held behind #605, #590 and #607 pending SHIP from ops:0.
