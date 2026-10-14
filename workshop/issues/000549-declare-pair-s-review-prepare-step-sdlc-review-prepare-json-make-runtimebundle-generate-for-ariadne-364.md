---
id: 000549
status: open
deps: [ariadne#364]
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'e5e3244aea682fa658ddffcd83b771c4a85870fd' # card fields mirrored from issue-cards; edit via sdlc
---

# Declare pair's review prepare step (.sdlc/review-prepare.json: make runtimebundle-generate) for ariadne#364

## Problem

ariadne#364 adds a per-repo review preparation step, so the boundary reviewer's disposable checkout can build generated, untracked files before testing (pair#528 M1's review failed without pair's runtime bundle assets). ariadne:3 measured it on a disposable checkout of pair main 8d8f0e13: runtimebundle checks failed without assets and passed after fenced preparation (553 ms), with the source unchanged.

## Spec

After ariadne#364 lands, add `.sdlc/review-prepare.json` to pair: `{"version": 1, "command": ["make", "runtimebundle-generate"], "timeout_minutes": 10}` (exact schema per ariadne#364's landed contract). `.sdlc/` is a protected path, so the PR needs the operator's `local-gates-verified` label.

## Done when

- The declaration is on pair main; a boundary review of a pair change touching runtimebundle builds and tests it in the reviewer's checkout.

## Plan

- [ ]

## Log

### 2026-10-14
