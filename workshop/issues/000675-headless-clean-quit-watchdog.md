---
id: 000675
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cabb7b81aa7c846b08b2100a6cbf6327f9a1fa9d' # card fields mirrored from issue-cards; edit via sdlc
---

# Headless clean-quit watchdog expires under load

## Problem

The clean-exit positive control in tests/run-headless-test.sh timed out under machine load during #672's integrated suite. The test runs nvim --headless -c 'qall!' under a 10-second watchdog and expected exit 0, but received 124. The test does not retain the captured positive-control output on failure, so this evidence proves watchdog expiry, not whether Neovim startup or shutdown consumed the bound.

## Spec

Diagnose the clean-exit control's dependence on the host Neovim startup environment and machine load. Keep the deliberate hang controls meaningful and retain enough diagnostic output to identify the failing phase. Do not solve this by silently increasing timeouts.

## Done when

- A deterministic regression covers the identified cause, with the clean-exit and deliberate-hang cases still distinguished.
- The control uses an appropriate owned environment and reports useful diagnostics on failure.

## Plan

- [ ]

## Log

2026-10-18: Observed during #672 integrated head e3fdf09d57ce6e512e9fadd68d7e5d5691497873, suite attempt 8bd605240566e399dbb5e25560d14664. Suite failed after 1m5s, before Go tests. Log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/8bd605240566e399dbb5e25560d14664.log; reported SHA256 prefix 44334a2035eb. Exact failure: clean-quitting nvim should return 0, got rc=124. Deliberately modified-buffer hang returned 124 within about 4 seconds against its 3-second bound; generic nonterminating child returned 124 within about 2 seconds. Other early build, retention, host-preflight and pidfile-lifecycle checks passed. #672 does not modify this script or watchdog. Its preceding branch suite passed in 14m26s on 5e299f51362df4c72591502cca46971ec7df5a80, attempt 4cc707f458a5841456ca2f13eeb56a10. Hub classified this as independent and authorized one issue-backed unchanged integration rerun, canceling the queued diagnostic main baseline. No timeout or fixture changes made in #672.
