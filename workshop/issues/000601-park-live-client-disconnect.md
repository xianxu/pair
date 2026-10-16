---
id: 000601
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '70e32f4e7aa8c5799a9f76d0f7cdcfa2b5d3b723' # card fields mirrored from issue-cards; edit via sdlc
---

# Live park conformance client disconnects before helper attachment

## Problem

First seen2026-10-16T06:12:03Z (2026-10-15 local) in pair#571 PR351 conformance run38859545931, job116838463577, on b6fdf8c48b6fdba37ee5533c39c758e55077304a. TestParkLifecycleLiveIntentOnlyMutation failed at cmd/internal/couchcore/park_lifecycle_live_test.go:246: production handoff helper exited before attaching, exit status4. Its captured PTY says the Zellij client lost connection to the server and was disconnected as a safety measure; the message suggests server messages were not consumed quickly enough. No quit intent was present. This is setup/attachment, before the test's750ms completion-observation assertion.

CI: https://github.com/xianxu/pair/actions/runs/38859545931/job/116838463577
Saved failed log: /tmp/pair571-ci-conformance-38859545931.log

One isolated unchanged headless repeat passed: PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcore -run '^TestParkLifecycleLiveIntentOnlyMutation$' -count=1 -v (test1.57s, package1.805s). Log: /tmp/pair571-park-live-repeat.log. The fixture/helper path has no #571 diff. The full normal suite and all recovery/home focused normal/race checks also pass; those normal runs skip opt-in live tests.

#432 tracks different conformance failures (duplicate quit trigger and cold-launch ownership); its Problem explicitly distinguishes this mutation case as passing. This entry tracks the distinct intermittent Zellij client disconnect before helper attachment.

Hypothesis only: fixture client/server backpressure or startup coordination on the hosted macOS runner. The Zellij diagnostic identifies the disconnection class, not proven scheduling causality. Inspect PTY drain/startup boundaries and collect reader/server progress before choosing a fix. Do not enlarge timeouts or retry inside the fixture to hide the failure.

## Spec

## Done when

- Reproduce or measure the pre-attachment disconnect, including hosted-runner evidence.
- Fix the proven class with a regression that distinguishes attachment failure from completion observation.
- Record repeated live/conformance evidence and retain any remaining uncertainty.

## Plan

- [ ]

## Log

Filed independently from #571, unassigned. No fix folded into the Codex home-routing patch. After the targeted pass, #571 retries the unchanged failed CI job once; a repeated unrelated failure is reported rather than bypassed.
