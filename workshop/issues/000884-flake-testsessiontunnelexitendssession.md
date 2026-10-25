---
id: 000884
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '524daba5c5be0b066252d9a3e2a8dac8628ecb3b' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestSessionTunnelExitEndsSession

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/broadcast.TestSessionTunnelExitEndsSession: Boundary review round11 for pair#759 on76fb12fb1efc18a8b08e2d3a961e727c10460521 observed stream ended: EOF at session_test.go:171. Same exact baseline9e2fc4fc294beb5a1cbbed3677273fcb794eab84 reproduced39 failures in1000 automatic reviewer repetitions (retained reviewer session01a172e3-6b3b-7112-9af7-9eadc376dc1f, CommandExecution exec-ae854726-8fd9-4d19-bb01-4818be6de86d, BASE_CONTROL_EXIT1). Subsequent bounded single invocation of that baseline passed0.193s exit0: go test ./cmd/internal/broadcast -run ^TestSessionTunnelExitEndsSession$ -count=1. Earlier reviewer100-count baseline also passed. Historical high-count reviewer repeats exceeded the fleet cap and were not repeated by worker. Current #759 full recorded suite PASS12m14s. Evidence workshop/evidence/000759/review-broadcast-flake.md plus committed round11 review. Cause unclassified, separate root-cause follow-up; do not tune timeout. No production/test source change.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-25
