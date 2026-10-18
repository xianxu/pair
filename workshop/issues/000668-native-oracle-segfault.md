---
id: 000668
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '16aed5795c7e8c03b3abb93ff7ea9de355b95e7f' # card fields mirrored from issue-cards; edit via sdlc
---

# Native terminal oracle subprocess crashes with SIGSEGV in macOS conformance

## Problem

While landing #659, macOS CI run 39111981159/job 117629869893 failed TestNativeConsoleWrapperZellij/wrapped-zellij at terminal_native_test.go:345: independent xterm oracle: signal: segmentation fault (empty subprocess output). First observed 2026-10-18 08:36:46 UTC, branch head 46e0bd0379ccf8ccabb972fa3b62aea2a9a36871; tested PR merge fb2be7a7ff1a4779bdf61884229748d808e2ae1d. Full failed-job log retained at /tmp/pair659-ci-39111981159.log; public run is https://github.com/xianxu/pair/actions/runs/39111981159/job/117629869893 .

## Spec

Identify why the independent Node xterm oracle process crashes and add a regression for the measured cause, preserving native terminal frame assertions. assertNativeOracle invokes node tests/terminal-oracle/driver.cjs with a JSON frame request via stdin; CombinedOutput returned a segmentation fault and no output. Do not retry inside the test or weaken the oracle assertion.

This is distinct from #660's missing session during reattach and #603's shell-actor discovery timeout. Direct baseline passed in 3.11s; wrapped receipts retained one BOOT=7195:e18042497f7e9a471cb2cadf and STATE with that same identity advancing 1 to 2 before the oracle crash. The wrapped subtest failed in 2.34s. The CI run predates #660 diagnostics (PR429 landed 15:19 UTC); no new socket/phase facts were available. Fixture and oracle files have no #659 diff against its integrated main. Other heads passed this conformance family, but this specific crash has one observation, not a measured flake rate.

## Done when

- Retained diagnostics identify the crashing subprocess/runtime and causal input or environment.
- A deterministic regression protects the measured cause without relaxing terminal checks or increasing deadlines.
- Native conformance passes with before/after failure counts reported.

## Plan

- [ ]

## Log

- 2026-10-18: Filed unassigned under the fleet flake rule after distinguishing it from #660; #659 will rerun only this failed CI job once on the unchanged head and stop/report on a second failure. Its full integration suite passed exit 0 in 11m3s before this CI failure. Historical root cause remains unknown.
