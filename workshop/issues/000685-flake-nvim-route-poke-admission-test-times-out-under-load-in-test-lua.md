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


### 2026-10-22 — #773 integrated suite occurrence

Recorded attempt1d8fae6745226f7d80414d02eccb7c87 on6d654228d39eb7157597a2a7dab85f1a30cab026 completed exit2 in2m58s after the Lua test target reached tests/nvim-route-poke-admission-test.sh. run_headless reported TIMEOUT after30s with no nvim output. Complete failed log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/1d8fae6745226f7d80414d02eccb7c87.log; SHA256bb89a55b7ae5c3f2727a8bed8b76873685c9c553fb6ec4de2a9a992f0e5a0573. This is another occurrence of #685, not a new issue. The route/poke Lua, shell driver, fake-zellij and run-headless sources are unchanged by #773 compared with integrated main. One unchanged focused command bash tests/nvim-route-poke-admission-test.sh returned exit0 and nvim-route-poke-admission-test ok. Observed load1 at that repeat end34.61 (not the original failure's load). No timeout increase, test skip or production fix. The recorded classification is unattributable-failure because the shell timeout has no Go test identity; it is not eligible for exact-package flake resume.
