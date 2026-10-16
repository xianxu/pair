---
id: 000608
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '894687be37ec617450dca1233f92b3a2751b3752' # card fields mirrored from issue-cards; edit via sdlc
---

# Isolate fleet conformance from shared installed SDLC replacement

## Problem

TestFleetInventoryLiveConformance executes the machine's shared installed sdlc from PATH. Another slot's dev alias can replace that binary while the suite is running. The conformance test can therefore fail due to unrelated tool publication rather than the pinned pair source.

## Spec

Remove the dependency on a mutable shared installed binary at the test boundary. Use a verified invocation-owned producer build (or another isolated producer contract) while retaining real decoder/live-producer conformance. Record producer path/build identity and process outcome on failure. Deterministically exercise replacement of the shared installation and prove the owned test producer remains stable. No sleeps, larger budgets, retries or skips as a fix. This is separate from #586's fixture diagnostics.

## Done when

- A deterministic regression exposes dependence on the shared producer and passes with an invocation-owned producer.
- Conformance still exercises the real producer schema, verdicts and reasons.
- Concurrent replacement of the installed sdlc cannot change the binary used by an in-flight conformance fixture; failure evidence identifies producer provenance and outcome.

## Plan

## Log

2026-10-15: #586 integrated source e53bd02e6bbc8b367580379f8a370e6b93cedeff, recorded suite c5ff925188ea9588598fd5d74acac647 failed13m51 (831.222s), exit2, only parsed failure github.com/xianxu/pair/cmd/internal/couchcore.TestFleetInventoryLiveConformance. Test duration0.05s, error recoverplan_fleet_test.go:175 “sdlc fleet inventory: signal: killed”. Command has90s timeout under a2min context, so the observed50ms failure is not evidence those budgets expired. Other Go packages completed, review editor checks passed. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/c5ff925188ea9588598fd5d74acac647.log, SHA256624cbcaaca6326f5fe59c57ef626ddd17b967764f89a8469b468e0a4b347eadd.

Hub evidence (receipt49ab53a3-9d50-4e4e-9d8f-3a5c71359676): ariadne installed bin/sdlc was replaced at23:45 by a slot's dev alias, near the conformance invocation. This correlates the immediate SIGKILL with shared-binary replacement; the exact kernel kill mechanism is not independently established. The demonstrated root of the class is the test's externally mutable PATH producer dependency: exec.LookPath("sdlc") followed by OSProvisionIO.Run Program:"sdlc". The suite's pinned commit does not own that executable.

One isolated diagnostic run on the unchanged integrated source passed: go test ./cmd/internal/couchcore -run '^TestFleetInventoryLiveConformance$' -count=1 -v, test46.22s/package46.462s. This does not convert the failed full suite to a pass. Merge automatically queued its diagnostic main-base comparison on e7b676abb2083791028bed45eda22515361cca4c (run6e6e954f55eb8e7ba8dacebde0bf0e07); no #586 code/budget changes. Hub authorized this separate unassigned filing and one tool-owned rerun; no fix inside #586.
