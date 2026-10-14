---
id: 000555
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '118bb1a73d39ffe392ad0a77ffbbcd06a2a5aa46' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:02:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "bf551f03", done: "c12c8e15"}
---

# Ignore generated TL skills

## Problem

ariadne#350 exports TL skills; refresh generates four unignored paths and dirties every pair slot, blocking planning.

## Spec

Commit exactly weave-generated ignores for xx-tl-lead and xx-tl-worker under .agents/skills and .claude/skills. Keep weave as their source (ARCH-DRY); no runtime change.

## Done when

- The only product diff is those four generated .gitignore additions.
- All four generated skill paths are ignored and a second weave apply leaves tracked files unchanged.
- Quick-flow review and required landing checks pass.


## Plan



## Log

### 2026-10-14
- 2026-10-14: closed — Four exact generated ignores, all confirmed by git check-ignore; repeat weave compile passed at pair 678c62ddc0d9709f6009d10677c937924040eb93 and ariadne 7a7223628f1ec78a213ad4dd6ea4f0f9761c9f31, unchanged ignore blob fbbe4444974662bbc3fa32e5bff00738ae6c0583 and no tracked product changes. BR-1 measured host evidence in issue Log. No new architectural surface. Full suite required and running; landing stays gated.; review verdict: SHIP

TL approved this exact generated maintenance change and quick flow. Initial refresh completed; diff is exactly four ignore additions. No durable plan needed. Full suite will run because the landing gate requires it.

Verified the branch product diff is exactly four .gitignore additions; git check-ignore names all four matching rules. A second weave compile succeeded and git status --short was empty. git diff --check passed. PR #335 is open; required sdlc local-check suite runs on baef02f4534c168b8b195663463deb0d8d2c6844.

BR-1 host counter-measurement: the disposable review checkout lacks its sibling substrate, but this worker checkout has the declared ../ariadne. On pair HEAD 678c62ddc0d9709f6009d10677c937924040eb93 with Ariadne HEAD 7a7223628f1ec78a213ad4dd6ea4f0f9761c9f31, repeated `weave compile` exited 0 (115 actions). `.gitignore` blob before and after was fbbe4444974662bbc3fa32e5bff00738ae6c0583. `git diff --exit-code HEAD -- . ':!workshop/'` exited 0; `git status --short` showed only the review-generated untracked gate/review sidecars. The previous compile on product commit baef02f4534c168b8b195663463deb0d8d2c6844 also exited 0 and left the tracked tree completely clean. Both compiles used this same pinned Ariadne revision; no dependency refresh occurred between them. Local repeat output: /tmp/pair555-weave-repeat.log. This supplies the review's requested measured host evidence without installing substrate into its read-only disposable environment.
