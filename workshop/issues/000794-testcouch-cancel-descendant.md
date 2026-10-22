---
id: 000794
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '36f8bd59eb8c2a050ed203d4ae318952882a8f37' # card fields mirrored from issue-cards; edit via sdlc
---

# testcouch cancellation sometimes returns with owned descendant alive

## Problem

First observed 2026-10-21 during #738 integration on d97b0b24f9327b853a0391da8e94c9de75121e04: TestCommandWrappersJoinCanceledDescendants/provision failed in 2.03s at binaries_test.go:145: provision returned with owned descendant still running: pid=24448 generation=1792639775.37310. Testcouch package failed23.590s; all other suite packages completed. Source is newly landed shared disposable-fleet harness; #738 does not change it.

Retained full suite log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/d6f342c66af6368129df356ee2f9d326.log (attempt d6f342c66af6368129df356ee2f9d326). The existing merge runner automatically queued a main baseline; its result is pending.

One unchanged targeted repeat on the exact integrated head passed: go test ./cmd/internal/testcouch -run ^TestCommandWrappersJoinCanceledDescendants$ -count=1 -v. Both build and provision subtests passed0.04s each; package PASS0.297s. This establishes an intermittent observation, not a root cause or a measured failure rate; no full-suite pass on that same head is claimed yet.

## Spec

Determine whether cancellation returns before exact owned descendants exit, or whether fixture/process observation misclassifies completed cleanup. Preserve generation fencing and joining guarantees for both build and provision wrappers. Add a deterministic failing regression at the actual cause; no timeout bump, assertion weakening or cleanup waiver.

## Done when

- Retained failure is explained by measured ordering or ownership evidence.
- A deterministic regression fails before the fix and passes after it for the affected wrapper path.
- Build and provision cancellation still join exact owned descendants; unrelated process generations are untouched.

## Plan

- [ ]

## Log

Filed unclaimed from pair:7 while #738 main-baseline diagnosis remains queued. No production/test changes or blind full-suite rerun made; targeted repeat only. Failure is not yet registered as a known flake because current register help requires saved same-head suite fail-to-pass evidence (or external evidence), beyond this targeted repeat.
