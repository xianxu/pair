---
id: 000108
status: codecomplete
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.25
started: 2026-07-07T17:32:26-07:00
actual_hours: 0.11
---

# pair review mode spinner should be seconds

After >60 seconds, review mode displays minutes only. It should keep
second-level precision below one hour, e.g. `5m 10s`.

## Problem

The review-pane statusline spinner uses a compact elapsed-time formatter while
waiting for the agent. Once elapsed time reaches 60 seconds, the formatter drops
the seconds component (`2m`), which makes it harder to tell whether the review is
still actively advancing.

## Spec

- Keep elapsed-time formatting in the existing pure spinner helper
  `nvim/review/spinner.lua` (ARCH-PURE, ARCH-DRY).
- For elapsed time below one hour, always include seconds:
  - `45s`
  - `2m 0s`
  - `5m 10s`
- For elapsed time at or above one hour, show hours and minutes only:
  - `1h 0m`
  - `1h 2m`
- Clamp negative input to `0s` as today.

## Done when

- `nvim/review/spinner_test.lua` covers sub-minute, minute+seconds, hour+minutes,
  and negative elapsed values.
- The review statusline uses the updated formatter without UI rewiring.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
the stale repo-local calibration source reported by `sdlc estimate-source`.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.1 impl=0.1
total: 0.25
```

## Plan

- [x] Update spinner elapsed tests first for the new minute/hour formatting.
- [x] Change `nvim/review/spinner.lua` to include seconds below one hour and
      minutes-only precision at one hour or more.
- [x] Run the focused Lua spinner test and the repo test target that covers Lua.

## Log

### 2026-07-07
- 2026-07-07: closed — nvim -l nvim/review/spinner_test.lua; make test-lua; git diff --check; review verdict: SHIP

- Claimed #108 and scoped the formatter rule: seconds below one hour; hours and
  minutes only at one hour or more.
- Implementation: updated `nvim/review/spinner.elapsed` and its pure Lua tests
  for `2m 0s`, `5m 10s`, `1h 0m`, and `1h 2m`.
- Verification: `nvim -l nvim/review/spinner_test.lua` passed.
- Verification: `make test-lua` passed.
