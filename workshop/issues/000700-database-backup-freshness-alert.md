---
id: 000700
status: open
deps: [pair#688]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3f4ac00f3be6cbd801dc1a728d2cdcc161e8ccb9' # card fields mirrored from issue-cards; edit via sdlc
---

# Warn when a database backup is older than its freshness threshold

## Problem

#688 found thread_history_1.sqlite had no verified bundle for roughly 55 hours while continuous passes could finish with database skips. A pass-completion marker did not expose individual database freshness.

## Spec

Warn the operator when a configured database's last verified backup bundle exceeds an agreed freshness threshold, initially proposed as two days. Derive timestamps from verified catalog records, distinguish never-backed-up and failed/deferred acquisition, and avoid repeated alert spam. Decide how a currently checked-unchanged database should be treated so an old but current bundle is not mistaken for missing data. Coordinate thresholds and recovery wording with the operator before code. Reuse #688's per-database projection; do not add a second snapshot path or hot retries.

## Done when

- A stale or never-verified database produces a clear bounded warning with the database name and one recovery action.
- Tests cover threshold boundaries, unchanged verified content, current failure, no catalog entry and deduplication across repeated checks.
- The warning uses existing operator attention surfaces and has a documented lifetime/reset policy.

## Plan

- [ ]

## Log

- Follow-up requested by hub in approval f477a6cb-0f84-4d98-9557-481147bd3338; filed unclaimed, independent of #688's bounded acquisition and report delivery.
