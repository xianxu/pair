---
id: 000473
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '3bab27cf59807bd364a166fa34ebe6739504a0aa' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: two flaky pair tests (couchtty collaboration timers; couchcore launch-helper post-ack retries)

## Problem

Two pair tests fail intermittently on main: `couchtty` `TestCollaborationTimersRearmAndTearDown` (about 1 in 8, reported by pair:1 on 10-12) and `couchcore` `TestPostAckHandleRetriesReuseOneWaiterUntilReap` (failed once in a full run, then passed 8/8, reported by pair:2 on 10-12). Flaky tests cost every slot's full-suite run and erode trust in failures.

## Spec

Reproduce each under load (`-count` and `-race`, and inside a full `make -k test`), find the race or timing assumption, and fix the root cause (an injectable clock, deterministic ordering, waiting on the event instead of time). No retries or longer sleeps as the fix. If the cause is in product code rather than the test, fix the product.

## Done when

- Each test passes 200 runs under `-race` and in repeated full-suite runs; the cause is recorded in the Log.

## Plan

- [ ]

## Log

### 2026-10-12
