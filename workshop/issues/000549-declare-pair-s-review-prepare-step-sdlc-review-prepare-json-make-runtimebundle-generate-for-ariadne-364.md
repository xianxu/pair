---
id: 000549
status: working
deps: [ariadne#364]
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'ed6aeb4017d6f2dd14a3266492243d49b29cf396' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T01:47:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Declare pair's review prepare step (.sdlc/review-prepare.json: make runtimebundle-generate) for ariadne#364

## Problem

ariadne#364 adds a per-repo review preparation step, so the boundary reviewer's disposable checkout can build generated, untracked files before testing (pair#528 M1's review failed without pair's runtime bundle assets). ariadne:3 measured it on a disposable checkout of pair main 8d8f0e13: runtimebundle checks failed without assets and passed after fenced preparation (553 ms), with the source unchanged.

## Spec

After ariadne#364 lands, add `.sdlc/review-prepare.json` to pair: `{"version": 1, "command": ["make", "runtimebundle-generate"], "timeout_minutes": 10}` (exact schema per ariadne#364's landed contract). `.sdlc/` is a protected path, so the PR needs the operator's `local-gates-verified` label.

## Done when

- The declaration is on pair main; a boundary review of a pair change touching runtimebundle builds and tests it in the reviewer's checkout.

## Plan

- [ ] Declare the approved version-1 preparation command, verify clean-checkout generation and runtimebundle tests, then open a PR for the operator label.

## Log

### 2026-10-14

### 2026-10-14 — authorized delivery scope

The TL approved the exact version-1 declaration and a PR stopping at the protected-path label. Reuse the existing runtimebundle generator (ARCH-DRY); no new executable logic or schema. Verify a fresh source snapshot lacks embedded assets, then execute the declared argv and runtimebundle tests without tracked-source changes. The boundary review itself consumes the pinned declaration. The operator owns local-gates-verified; this slot will not label or merge the PR.
