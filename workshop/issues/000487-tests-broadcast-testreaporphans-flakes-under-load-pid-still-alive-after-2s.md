---
id: 000487
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '7b0833e7e2db315d65ab28e0086002f7f76cf888' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: broadcast TestReapOrphans flakes under load (pid still alive after 2s)

## Problem

pair:4 saw `broadcast` `TestReapOrphans` fail once in a full package run at load average ~125 (the pid still alive after 2 s), passing 3 of 3 alone. Same family as #473/#476/#480: a real-time window asserted under load.

## Spec

Same rules as #480: wait on the process's exit (or an injected clock) bounded only by the test deadline, not a 2 s window; fix the product if reaping itself has the race.

## Done when

- 200 runs under -race and load, green; the cause in the Log.

## Plan

- [ ]

## Log

### 2026-10-12
