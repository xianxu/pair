---
id: 000549
status: done
deps: [ariadne#364]
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '64e8cdece65d08cb0b8230fb4dd283b19350a8c7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T01:47:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "f89d6add", done: "cbf818c8"}
actual_hours: 0.08
---

# Declare pair's review prepare step (.sdlc/review-prepare.json: make runtimebundle-generate) for ariadne#364

## Problem

ariadne#364 adds a per-repo review preparation step, so the boundary reviewer's disposable checkout can build generated, untracked files before testing (pair#528 M1's review failed without pair's runtime bundle assets). ariadne:3 measured it on a disposable checkout of pair main 8d8f0e13: runtimebundle checks failed without assets and passed after fenced preparation (553 ms), with the source unchanged.

## Spec

After ariadne#364 lands, add `.sdlc/review-prepare.json` to pair: `{"version": 1, "command": ["make", "runtimebundle-generate"], "timeout_minutes": 10}` (exact schema per ariadne#364's landed contract). `.sdlc/` is a protected path, so the PR needs the operator's `local-gates-verified` label.

## Done when

- The declaration is on pair main; a boundary review of a pair change touching runtimebundle builds and tests it in the reviewer's checkout.

## Plan

- [x] Declare the approved version-1 preparation command, verify clean-checkout generation and runtimebundle tests, then open a PR for the operator label.

## Log

### 2026-10-14
- 2026-10-14: closed — Clean snapshot runtimebundle tests fail without assets and pass after declared argv (0.181s), tracked source hashes unchanged. Round 1 independently reproduced RED/GREEN and passed runtimebundle/runtimebundlegen tests. BR-1 README now documents version, command, timeout and atlas link; git diff --check clean. Config-only change, focused generation/tests sufficient; full suite not run. Pinned preparation exercised by close. PR stops for operator local-gates-verified label; no agent label/merge.; review verdict: SHIP

### 2026-10-14 — authorized delivery scope

The TL approved the exact version-1 declaration and a PR stopping at the protected-path label. Reuse the existing runtimebundle generator (ARCH-DRY); no new executable logic or schema. Verify a fresh source snapshot lacks embedded assets, then execute the declared argv and runtimebundle tests without tracked-source changes. The boundary review itself consumes the pinned declaration. The operator owns local-gates-verified; this slot will not label or merge the PR.

- Clean source snapshot verification: runtimebundle Go tests first failed because assets/runtime/files was absent. Executing the declaration argv generated the ignored assets; runtimebundle tests then passed (0.181s). SHA-256 hashes of all present tracked source files were unchanged. Configuration matches ariadne#364 version 1 and explicit ten-minute timeout. Boundary close will exercise its pinned preparation path; no new production code, so a full repository suite is unnecessary for this declaration-only change.

- Close round 1 independently reproduced missing-asset RED and preparation GREEN, and passed runtimebundle/runtimebundlegen tests. Its sole finding BR-1 requested README documentation; added the declaration version, argv, timeout and atlas link. No executable behavior changed.
