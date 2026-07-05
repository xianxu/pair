---
id: 000101
status: open
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-05
estimate_hours:
---

# review nvim search should be case-insensitive (smartcase)

## Problem

Search in the review nvim pane is case-sensitive. It should be case-insensitive
by default but case-sensitive when the query contains an uppercase letter — the
"smart" option. Split out of #89 to keep that issue focused on concurrent-edit
reconciliation.

## Spec

In `nvim/review.lua` (the standalone review-pane init), near the top `vim.opt`
block, add:

```lua
vim.opt.ignorecase = true
vim.opt.smartcase = true
```

`/foo` matches case-insensitively; `/Foo` (any uppercase) stays case-sensitive.
Pane-local — `review.lua` is a self-contained `nvim -u` init, so this does not
touch the draft nvim or the user's own config.

## Done when

- `ignorecase` and `smartcase` are set in the review pane; `/foo` matches `Foo`,
  `/Foo` does not match `foo`.
- Asserted in `tests/review-window-test.sh` (`make test-review`).

## Plan

- [ ] Set `ignorecase`+`smartcase` in `nvim/review.lua`; assert both in the window test

## Log

### 2026-07-05
