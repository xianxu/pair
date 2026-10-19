---
id: 000700
status: working
deps: [pair#688]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5ebb5d3c44bc164afd11b8abea4d01082f3227b3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T06:12:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
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

- [ ] Implement and qualify the single-pass design in [the durable plan](../plans/000700-database-backup-freshness-alert-plan.md): pure freshness and daily admission, typed bounded command report, scheduler/operator delivery, deterministic and fake-backed tests.
- [ ] Document the threshold, quiet-database exception, once/day lifetime and recovery; submit the recorded full suite, close, and land on SHIP.

## Log

- Follow-up requested by hub in approval f477a6cb-0f84-4d98-9557-481147bd3338; filed unclaimed, independent of #688's bounded acquisition and report delivery.

## Revisions

- 2026-10-19, hub dispatch 212fe964: use #688's per-database report, default configurable 48h threshold and once-per-source/day Couch notification; catalog owns backup age, with no new store.
- 2026-10-19, hub 44e2a280: suppression is once per source per UTC day within one Couch lifetime; a fresh warning after restart is explicitly allowed. A current validated unchanged snapshot is healthy even with an old catalog timestamp. Warn only failed/deferred checks past the threshold, or immediately when never verified. No catalog timestamp refresh or recompression to make quiet databases look newer.

## Design log

- Claimed and started from fresh main in pair:7. Existing scheduler runs a subprocess daily, retries failed passes after six hours and only exposes a generic status notice. Its failure suppression is volatile. Existing dismissible/evictable attention records and lossy diagnostic logs cannot prove daily delivery, so they are not used as delivery authority (ARCH-PURPOSE).
- Existing LatestVerified already derives catalog evidence by exact source identity/path. Reuse it and carry that source identity into runtime summaries; do not scan or snapshot again (ARCH-DRY).
- Proposed configuration is the inherited PAIR_ARCHIVE_DATABASE_MAX_AGE duration environment variable, unset = 48h; invalid or nonpositive values fail before work. This avoids changing continuous.json or any existing persisted format for one optional knob (ARCH-EVOLVE). No machine environment is changed by this task.
- Read-only seam exploration confirmed the existing scheduler/report and notification boundaries. No production edits or tests/builds for #700 have started. Full plan goes to hub before change-code.
