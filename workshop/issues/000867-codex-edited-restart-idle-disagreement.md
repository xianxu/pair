---
id: 000867
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
severity: medium
card_mirror: '7d9fc2e3eaceff84568a4cb5b694cfc80d847307' # card fields mirrored from issue-cards; edit via sdlc
---

# Diagnose Codex edited-restart idle precondition disagreement

## Problem

One #863 native EditedRestart/Codex run failed the first restart with 'switch-agent: source must be known idle; retry after its turn finishes' despite startupArgumentsWaitIdle having returned. The precise successful resource sample was not logged in that failing run. A pinned-main control and one instrumented branch run then passed. Cause is not established; neither branch introduction nor an unrelated flake is claimed.

Failure: production source7a71f335350082c048aca4c84058cd350c6d0eb8, TestDisposableEditedRestart/codex FAIL22.08s (whole test80.69s; Claude passed51.53s). Receipt dda5ab2f-ae24-4aca-9841-7b19f75d54a8, failure operation-608d8669b7e335214bb6373ade2b8ef1, request2026-10-24T02:51:54.437009-07:00 ->02:51:54.518428. It was refused before an effect. All owned processes/IO cleaned.
Capture workshop/evidence/000863/branch-required-native-7a71f335.log, SHA256217ecb09534658cc33ca98bdc08f71ff87141e1305d0ca28b8ba42af424f0f11.

Main control: main4053f3da995746efcc1c6296cffbde4a67a7f343, native consumer c9dbf637d18c869d1ecae545201b2e5d1ab755b2. Main used the operator-approved explicit hub startup gesture because main lacks #863's automatic-fresh path; later restart steps unchanged. EditedRestart/Codex PASS48.85s (case39.22s); resume/fallback, transcript preservation and exact cleanup. Successful resource samples logged.
Capture workshop/evidence/000863/main-edited-restart-codex-4053f3da.log, SHA256f13c712825482e0ea0216f679b2d2fdf41ce4cf13833c56be0e93c00750eddae.

Instrumented branch: sourceeb2922b2206443b0fee6dd379ea4d084630fe9d8, test-only failure capture and success-sample diagnostics; production idle/settle policy unchanged. ONE authorized EditedRestart/Codex run PASS99.20s (case82.31s; package99.417s), exact cleanup. NOT REPRODUCED; no causation claim.
Capture workshop/evidence/000863/branch-codex-idle-instrumented-eb2922b2.log, SHA256dc6088073ec8841046290735b0969c9b76a8424822b830b02423a680c2f1753c.

All three captures are pushed on #863 branch at43819597fb1cfbd6d9200e33b64124ece8dcd358, not yet main; use git show that branch without stacking code. No dependency on #863 landing. The separate Zellij metadata panic is #866, not this failure.

## Spec

Root-cause the discrepancy with retained state transitions. startupArgumentsWaitIdle in cmd/couch/startup_arguments_fleet_test.go accepts a fresh resource sample for the exact source scope/tag where IdleKnown && Idle, while checking native PID/generation. resources_inventory.go derives this from liveness.Settled plus broker/picker state at sample time. agent_switch_service.go:90 independently checks the current liveness.Settled at admission; nil/false refuses. A resource observation is not a reservation, so later activity can legitimately invalidate it.

Read-only diff audit found no changed settling, registry, resource projection, switch admission, fixture painting or cue producer in #863. Hypotheses: cached idle precedes delayed worker-start cue delivery, or asynchronous session-watch consumption of the fixture transcript's task_started/task_complete records causes a later unsettle; chrome resize/progress output was also considered but not established. Missing failing evidence includes exact successful sample, wrapper session token/settle transitions, accepted automatic-input timing and lifecycle-journal watermark. Existing test-only failure capture records fresh resources/peek, accepted input metadata and bounded private wrapper-event tails.

Do not weaken production idle admission, add a delay/timeout constant, introduce automatic switch retries, or call a single pass proof of causation. Use a deterministic stateful interleaving to qualify the actual transition and then fix its owner (test completion barrier or production bug, whichever evidence shows).

## Done when

- The mismatch has a measured cause, with the source transition retained and classified independently of #866.
- A deterministic test drives an idle observation followed by the relevant source/lifecycle transition and asserts correct admission plus no unintended effects.
- Any fixture fix awaits an explicit completion/watermark rather than sleeps; production remains conservative on unknown/unsettled input.
- Bounded native qualification uses the existing shared owner and exact cleanup, with no blind rerun or calibrated flake-rate claim from these three observations.

## Plan

- [ ]

## Log

2026-10-24: Filed unclaimed at TL24a27ad0's disposition. #863 may proceed to recorded pre-close and close with this unresolved one-off retained here; the passing instrumented run is not described as a fix.
