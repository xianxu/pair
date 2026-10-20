---
id: 000746
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '567416ca73910f4c38dd5344570a86664af0edee' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: check the affected-reader inventory before close, not only inside the full suite (stale inventory caught after SHIP three times on 10-20)

## Problem

pair#725's affected-reader inventory (`tests/affected-go-readers.json`) is checked only inside the recorded full
suite, so slots discover a stale inventory after their close review is SHIP, at the most expensive point. On 10-20
it cost three items a suite slot each plus a re-close: pair#727 (sank train 190 for 1 h 45 m), pair#726 (one
fixture file) and pair#706 (eight test files).

## Spec

- A fast `make test-affected-inventory` (seconds) that runs only the reader-inventory guard.
- Run it where it's cheap: in the close gate's preflight (or the affected check's first step) and listed in
  tl-worker's "before close" steps, so a stale inventory is caught before review, not in the suite.
- The failure message names each file and the exact command that updates its disposition and hash.

## Done when

- The target exists and fails fast on a stale entry; the close preflight (or tl-worker's checklist) runs it; a test
  pins the message.

## Plan

- [ ]

## Log

### 2026-10-20
