---
id: 000655
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7870221fc248b3666a8402da5ca79fbe063507f0' # card fields mirrored from issue-cards; edit via sdlc
---

# Flake: hoprttcmd TestSpawnTimerMeasuresTheCommandNotTheHarness fixed 15 ms band fails under machine load

## Problem

Flake in the recorded pre-merge suite, 2026-10-17: `cmd/internal/hoprttcmd` `TestSpawnTimerMeasuresTheCommandNotTheHarness` failed on #620's integrated head 7c96d30f (PR #407):

    hoprtt_test.go:69: fork+exec median 16.935 ms is outside the plausible band; the harness is measuring itself, not the command

- The test gives a fork+exec of /usr/bin/true a fixed 15 ms ceiling. Its comment budgets for the suite's own spawn load (about 4.5 ms), not for whole-machine load. At the time the machine's load average was about 10 now, 11.5 over 5 min and 18 over 15 min: two full suites plus fleet slots running.
- #620 doesn't touch hoprttcmd, and the main baseline passed, so this is the band failing under machine load, not a code change.
- Log retained: /Volumes/AgentScratch/tmp/pair-618-share/suite-7c96d30f-hoprtt-flake.log (sha256 prefix f1d958641097; original at .git/sdlc/test-stats/failed/e9a569fffa42f6450a9b09fb08569c4b.log).

## Spec

Make the positive control independent of machine load while still catching the bug it pins (a harness that times a clock subprocess, about 18.7 ms against about 1.9 ms). Options:
- Compare against a same-run baseline: time a trivial in-process operation, or measure the harness overhead directly, and assert the ratio rather than an absolute ceiling.
- Or assert on the harness structure (no subprocess clock read) with a fake clock, and keep the live timing only as a soft log.

## Done when

- The test can't fail from machine load alone. Checked with a load generator running alongside it, for example 20 parallel `yes > /dev/null` processes.
- Reverting to a subprocess-clock harness still fails it.

## Plan

- [ ]

## Log

### 2026-10-17
