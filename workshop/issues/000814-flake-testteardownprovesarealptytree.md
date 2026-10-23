---
id: 000814
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '57290b78bdad10a3e1a98761875902444f2d5c26' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestTeardownProvesARealPtyTree

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestTeardownProvesARealPtyTree: M1 round2 reviewer ran on exact pushed head32ababba112e3519b0009afb904b3df84acaf724. Full couchcore run failed497.705s with TestTeardownProvesARealPtyTree at teardown_live_test.go:106: Teardown: teardown not proven at sweep: list lease holders: context deadline exceeded (test2.66s). Same unchanged checkout immediately reran go test ./cmd/internal/couchcore -run ^TestTeardownProvesARealPtyTree$ -count=1 and passed exit0 in1.332s; git diff of teardown.go/teardown_live_test.go and git status were empty. Verified actual tool outputs in reviewer session /Users/xianxu/.codex/sessions/2026/10/22/rollout-2026-10-22T13-18-38-01a16544-02de-72e0-8b78-e0bf30018dc7.jsonl (failure chunk317f02, passing session12199/chunkc46dcd); committed M1 round2 sidecar records the failure and isolated pass. No recorded suite was rerun or marked passing based on this observation.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-22
