---
id: 000685
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'abaca49cb013d0c1bce0dbd6d10d11017992e4de' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: nvim-route-poke-admission-test times out under load in test-lua

## Problem

## Spec

tests/nvim-route-poke-admission-test.sh timed out in `make -f Makefile.local test-lua` (run_headless: TIMEOUT after 30s, nvim did not exit) on the pair#664 branch, 2026-10-18, under a loaded machine (two ariadne full suites running). It passed alone right after (`bash tests/nvim-route-poke-admission-test.sh` ok). The test (nvim/route_poke_admission_test.lua with the fake zellij) does not touch the review pane that #664 changed. run_headless printed no nvim output, so the hang point is unknown.

Evidence: log kept in the pair:2 scratchpad as flake-route-poke-664.log.

## Done when

- The hang point is found (run_headless captures the driver's last step, or the test logs progress), and the test either cannot hang or fails with a named step inside its own deadline.
- 50 consecutive runs pass under load (a concurrent full suite).

## Plan

- [ ]

## Log

### 2026-10-18
