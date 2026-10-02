---
id: 000380
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '722d63e763eb6e87a7f23df919a261a8ecfa4d1b' # card fields mirrored from issue-cards; edit via sdlc
---

# Draft completion menu sticks after submit

## Problem

Sometimes, after a submission (Alt+⏎), the draft pane's as-you-type
completion menu stays on screen over the now-empty draft. In the operator's
screenshot (2026-10-01) the draft is cleared to a single empty line, yet a menu
of `test`-prefixed items is still shown: `test-changelog`,
`TestBareCouchInstalledCommand`, `TestMessageWrapperExitDisconnects…`, … Those
items match the word that was being typed *before* the send, not anything in
the buffer now. Intermittent.

The completion is Pair's own code (`nvim/init.lua`, built without a plugin:
`path_complete` / `word_complete` / `spell_complete` via
`_G.PairCompleteProbe.sink` = `vim.fn.complete`). parley.nvim is not involved.

## Spec

Candidate mechanisms (unverified):

1. **Submit never closes the menu.** `send_and_clear` (`nvim/init.lua`,
   bound to `<M-CR>` in insert mode) replaces the buffer with
   `nvim_buf_set_lines` and calls `startinsert`, but never dismisses an open menu
   (no `<C-e>` / `complete()` close). Replacing the buffer from the API under an
   open menu can leave it on screen, and its completion state, until the next
   key. This would explain the menu matching the old text. It doesn't by itself
   explain why it only happens sometimes; that could depend on whether a menu
   happened to be open at the moment of the send.
2. **The 30ms burst-debounce timer.** The `TextChangedI/P` handler defers
   `run_completers` for 30ms when keystrokes arrive less than 20ms apart. If
   the operator types the last characters quickly and then sends inside that
   window, the deferred run fires *after* `send_and_clear`. It would run with
   insert mode active again (because of `startinsert`) and could call
   `complete()` with a start column computed from stale state. That would
   explain both why it's rare and the stale items.

Fix direction (decide after reproducing): the submit should dismiss any open
menu and cancel any pending completion timer before it mutates the buffer.

## Done when

- A headless regression test (in the style of `tests/draft-complete-mode-test.sh`)
  reproduces the stuck menu for the confirmed mechanism, and passes after the fix.
- After a send, the draft never shows a completion menu built from the
  pre-send text.
- The operator confirms in a live session that the menu no longer sticks.

## Plan

- [ ]

## Log

### 2026-10-01

- Filed from an operator screenshot. The operator asked for parley.nvim at first;
  it's filed in pair because the draft's completion is Pair's code.
