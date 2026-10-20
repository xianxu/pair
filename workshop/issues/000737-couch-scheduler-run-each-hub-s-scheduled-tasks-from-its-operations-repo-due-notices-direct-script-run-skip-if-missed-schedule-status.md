---
id: 000737
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '018f049e3fd3b434abe6d8f9ac1be9f16eb1712a' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status)

## Problem

Recurring operational work has no scheduler: it runs when someone remembers, or on private timers inside pair
(backup freshness, the Drive report). Project couch-scheduling (ops, `teams/devinfra/projects/couch-scheduling/`)
settled the contour with the operator on 10-20; its `design.md` is the reference.

## Spec

- At start and on change, the supervisor reads each hub repo's `scripts/scheduled/<task>/` folders: a Go program
  (`main`) plus a schedule declaration (cron-like time, timeout, scope `machine` default or `team` with an owner,
  optional run-once-if-missed).
- Persisted next-due per task, per machine. Missed runs **skip** by default and wait for the next slot; opt-in
  run-once on start.
- When due: if the task has a `script-run` subcommand, run it directly in the hub repo with its timeout, under the
  sandbox and machine test budget (exit 0 quiet, logged; 10 wake the hub with its output; anything else a failure
  notice). Otherwise deliver a notice to the team's hub: "scheduled: <task> is due; see `<path>/main --help`".
- Team-scoped tasks run only on the hub of their owner operator; an owner not listed in `team.md`'s Operators
  section refuses with a clear reason.
- `couch --schedule --status`: tasks, scope, last run and result, next due. Run history stays local and bounded.
- Each hub only ever runs its own repo's tasks.

## Done when

- Fake-clock tests pin: due notice delivery, `script-run` exit codes 0/10/other, timeout, skip-if-missed and the
  opt-in run-once, team-owner refusal, a hub never running another hub's tasks.
- Live: a trivial ops task (the landing-plan refresh) runs on schedule for a day and shows in `--schedule --status`.

## Plan

- [ ]

## Log

### 2026-10-20
