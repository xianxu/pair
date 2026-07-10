---
id: 000114
status: done
deps: []
github_issue:
created: 2026-07-10
updated: 2026-07-10
estimate_hours: 0.45
started: 2026-07-10T07:54:48-07:00
actual_hours: 0.15
---

# draft completion should skip outside insert mode

## Problem

Draft-pane typeahead completion is scheduled from `TextChangedI/P`. If the user
leaves Insert mode before a debounced callback runs, the callback can still call
`vim.fn.complete()`. Neovim raises `E785: complete() can only be used in Insert
mode`, currently observed when selecting a word around spell-check/completion
flows.

## Spec

- The draft completion runner must no-op unless the current mode is Insert or
  completion-select mode.
- Delayed/debounced callbacks must re-check mode at execution time, not only at
  scheduling time.
- Existing path, word, and spell completion behavior in Insert mode is
  preserved.

ARCH-PURPOSE: the fix must prevent every `run_completers()` path from reaching
`complete()` outside Insert mode, not only one completer.
ARCH-DRY: put the guard at the shared completion runner, not separately in
`path_complete`, `word_complete`, and `spell_complete`.

## Done when

- A regression simulates the draft completer running in Normal/Visual mode and
  proves it does not call `complete()` or raise `E785`.
- Existing nvim headless tests pass for the draft completion area.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 0.9
item: lua-neovim design=0.10 impl=0.20
item: milestone-review design=0.00 impl=0.15
total: 0.45
```

## Plan

- [x] Expose or isolate the draft completer mode guard enough for a headless
      regression.
- [x] Add a failing regression for a scheduled completer execution outside
      Insert mode.
- [x] Guard the shared completion runner before any completer can call
      `vim.fn.complete()`.
- [x] Run focused nvim tests and close #114.

## Log

### 2026-07-10
- 2026-07-10: close gate — Addressed the FIX-THEN-SHIP close-review findings: production guard is local to run_completers(), and the regression now covers Normal mode, Visual mode, and a scheduled post-Insert callback. Verified bash tests/draft-complete-mode-test.sh; bash tests/autopair-test.sh; bash tests/cr-newline-test.sh; make test-lua; git diff --check; make test. No atlas update: no new user-facing surface or architecture, only a guard and regression test for existing draft completion behavior.; review verdict: FIX-THEN-SHIP
- 2026-07-10: close gate — Fixed draft typeahead E785 by guarding the shared run_completers() entry point to only run in Insert-mode variants; reproduced pre-fix failure with bash tests/draft-complete-mode-test.sh. Verified bash tests/draft-complete-mode-test.sh; bash tests/autopair-test.sh; bash tests/cr-newline-test.sh; make test-lua; make test-draft-complete; git diff --check; make test. No atlas update: no new user-facing surface or architecture, only a guard and regression test for existing draft completion behavior.; review verdict: FIX-THEN-SHIP
- Created from reported stack trace:
  `Vim:E785: complete() can only be used in Insert mode` from
  `word_complete()` via scheduled `run_completers()`.
- Added `tests/draft-complete-mode-test.sh`; before the guard it failed with
  `Vim:E785: complete() can only be used in Insert mode`, reproducing the
  reported error through the real `nvim/init.lua` runner.
- Fixed the shared draft completer runner to no-op unless Neovim reports an
  Insert-mode variant, covering path, word, and spell typeahead from the single
  `run_completers()` entry point (`ARCH-DRY`, `ARCH-PURPOSE`).
- Verification passed: `bash tests/draft-complete-mode-test.sh`,
  `bash tests/autopair-test.sh`, `bash tests/cr-newline-test.sh`,
  `make test-lua`, `make test-draft-complete`, `git diff --check`, and
  `make test`.
- Addressed close-review findings by keeping the production guard local to
  `run_completers()` instead of globally mutable, and extending the regression
  across Normal mode, Visual mode, and a scheduled post-Insert callback.
- Condensed the generated close-review sidecar to a bounded durable summary so
  it records the verdict and resolutions without embedding full transcripts.
