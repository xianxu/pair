---
id: 000588
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '3f69a2b89b3cddd062f79f5c7e76af5047ba0f4a' # card fields mirrored from issue-cards; edit via sdlc
---

# Cost model: replay recorded TTY logs and defend per-byte coefficients (disk, syscalls, allocations) with counted tests

## Problem

pair and Couch costs are found by symptom. Part of ops project resource-observability (design notes in
`ops/teams/devinfra/projects/resource-observability/project.md`): state costs as drivers × coefficients and defend the
coefficients with counted tests.

## Spec

- A **replay harness** that drives recorded TTY logs (parked captures: raw + events sidecar) through pair wrap's real
  write and parse paths, with counting fakes for the filesystem and clock.
- **Counted coefficients** per replay: disk bytes written per TTY byte, write and fsync and rename syscalls per MB,
  event-log records per event, ledger bytes per launch (separate fixture), allocations per MB. Recorded as a small
  checked-in baseline per fixture.
- A test fails when a coefficient exceeds its baseline by a stated margin (default 25%), naming the coefficient. Updating
  a baseline is an explicit, reviewed change (an env var regenerates it).
- Two or three fixtures: an idle-heavy agent session, a burst-heavy one (a test run's output), and a resize-heavy one.
  Fixtures are small (≤ 10 MB) and scrubbed of secrets.

## Done when

- The harness runs in the normal suite in under 30 s; the coefficients for #579's write path match the measured 1.19×
  within margin; a deliberate regression (per-event sync) fails the test with the coefficient's name.

## Plan

- [ ]

## Log

### 2026-10-15
