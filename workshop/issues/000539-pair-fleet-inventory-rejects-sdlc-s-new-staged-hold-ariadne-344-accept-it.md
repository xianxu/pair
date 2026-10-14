---
id: 000539
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'd1b69dd4a77990a54447a6264b4f7b604f9f3d94' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T17:07:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# pair: fleet inventory rejects sdlc's new 'staged' hold (ariadne#344); accept it

## Problem

ariadne#344 emits the fleet hold reason `staged` for an intentionally detached smoke checkout. Pair rejects that reason and loses the measured git facts. Its recovery explanation also mistakes that intentional hold for unidentified work.

## Spec

Accept `staged` through the shared fleet reason grammar. Preserve the producer’s held verdict and measured facts; describe a confirmed healthy stage as held with no recovery action. Unknown reasons, ordinary detached HEADs, dirt, operations, and missing evidence retain existing conservative handling. Model stage facts in the stateful fleet fake, including return to an ordinary checkout.

## Done when

- Staged inventory decodes and passes reason validation without losing git facts.
- Couch reports a healthy staged checkout as a normal hold with no recovery steps.
- Regression tests cover stage, unstaged detach, return to branch, and dirty/operation/probe precedence.
- Consumer reason-list sweep and targeted/full verification are recorded.

## Plan

- [ ] Update the shared grammar and fleet fake (ARCH-DRY); keep classification pure (ARCH-PURE).
- [ ] Add stage evidence and a held explanation after existing safety checks, with regressions.
- [ ] Run targeted tests, live read-only conformance, and make -k test; close and land.

## Log

### 2026-10-13

- Read producer JudgeCheckout and TreeRow: detached + staged is holds-work; dirty/operation take precedence, unread base remains unknown. Swept Go consumers: one reason grammar, the shared memberGit reader, recovery classifier, and stateful fleet fake. No other hard-coded hold lists found.
- ARCH-PURPOSE: acceptance also needs a normal-stage recovery explanation. ARCH-DRY/PURE: reuse grammar and existing evidence/classifier seams. No new IO, background work, durable state, or resource budget; ARCH-BOUNDARY/CONSTRAINTS/FUNERAL do not introduce additional mechanisms. Preserve existing safety ordering (ARCH-STATE).
