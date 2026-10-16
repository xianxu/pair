---
id: 000596
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'c27bb33172de774f7ecea5e888441e5457c0f88b' # card fields mirrored from issue-cards; edit via sdlc
---

# Session-sync telemetry budget assertion flakes under suite load

## Problem

First seen2026-10-15 during pair#571's recorded full suite at03495ef700652896ff63f7e93e0ea3b239587b0a. TestSessionSyncWriteTelemetryBestEffort/timeout failed at cmd/internal/couchcmd/session_sync_test.go:335: telemetry exceeded budget:531.972583ms (assertion500ms). The enclosing subtest took1.13s. No session-sync code changes in #571.

Recorded command: make PAIR_GO_TRIMPATH=-trimpath test; its Go phase continues all packages after this failure. Inflight log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/6664352ee3e1ee93acd32ffd5f9c3766/run.log. Final recorded log will be /Users/xianxu/workspace/pair/.git/sdlc/local-checks/03495ef700652896ff63f7e93e0ea3b239587b0a/suite.log.

The fixture sleeps600ms in its server and times the whole session-sync write/fold against500ms, rather than only the bounded telemetry leg. Hypothesis: scheduling or filesystem work under concurrent suite load pushes the outer measurement over500ms; not yet calibrated or proven. Investigate the root cause and use deterministic deadline/admission evidence where possible; do not merely enlarge the timeout.

## Spec

## Done when

- Establish whether the telemetry operation exceeded its intended deadline or the wall-clock test measures unrelated work.
- Fix the proven class with a regression that remains meaningful under machine load.
- Record targeted and full-suite evidence, including the before/after failure rate if load-dependent.

## Plan

- [ ]

## Log

Filed from #571 as independent unassigned test-economy work. No fix is folded into the Codex adapter reduction. One targeted unchanged repeat is being recorded by #571.
