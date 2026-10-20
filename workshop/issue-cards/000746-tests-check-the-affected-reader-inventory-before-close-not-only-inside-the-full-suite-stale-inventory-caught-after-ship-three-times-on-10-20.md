---
id: 000746
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# tests: check the affected-reader inventory before close, not only inside the full suite (stale inventory caught after SHIP three times on 10-20)

## Problem

pair#725's affected-reader inventory (`tests/affected-go-readers.json`) is checked only inside the recorded full
suite, so slots discover a stale inventory after their close review is SHIP, at the most expensive point. On 10-20
it cost three items a suite slot each plus a re-close: pair#727 (sank train 190 for 1 h 45 m), pair#726 (one
fixture file) and pair#706 (eight test files).
