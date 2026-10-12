---
id: 000500
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '9a4d9e87146e8b8e838253530b9747a5bf1459ef' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T14:32:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.01
---

# roll out ariadne#309 workflows (local-gates-guard, merge-check on push) to pair

## Problem

ariadne#309's rollout reaches pair through the weave catch-up: it seeds `.github/workflows/local-gates-guard.yml`
(new) and an updated `merge-check.yml` (also runs on push to main) into each slot's checkout. Until pair's main
carries them, every pair slot's tree is dirty after a catch-up and `sdlc start-plan` refuses (TL, 10-12).

## Spec

Commit the two files exactly as weave seeded them (byte-identical to ariadne's `.github/workflows/` copies). No
edits: the guard's content is ariadne's, and a local-gates change is its own operator-verified issue there.

## Done when

- pair's main carries both files byte-identical to ariadne's; a slot's catch-up leaves its tree clean.

## Plan

- [x] Stash the seeded files from pair:6's resting branch (tagged), restore onto this branch, verify identical, commit.

## Log

### 2026-10-12
- 2026-10-12: closed — Both workflow files cmp byte-identical to ariadne's .github/workflows copies (the weave seed source) and parse as YAML; committed exactly as seeded. No Go/Lua/shell touched, so make test is unaffected. --no-atlas: CI workflow rollout from the base layer, no pair surface or flow changed.; review verdict: SHIP
- 2026-10-12: flow upgraded quick → full — 152 added lines in code files (limit 100)
- Seeded at 14:31 by sdlc's catch-up of pair-slot6/ariadne; `cmp` against ../ariadne/.github/workflows: both
  identical. The guard isn't on pair's main yet, so this PR isn't blocked by it.
