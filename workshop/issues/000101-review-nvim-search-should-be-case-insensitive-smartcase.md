---
id: 000101
status: codecomplete
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-06
estimate_hours: 0.25
started: 2026-07-06T14:46:52-07:00
actual_hours: 0.20
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

The test asserts the actual search *behavior*, not just the flag state, since the
outcome is what matters: with buffer `{ 'Foo' }` a `/foo` search finds the line
(ignorecase), and with buffer `{ 'foo' }` a `/Foo` search finds nothing
(smartcase, uppercase → case-sensitive). Uses `vim.fn.search(pat, 'nW')` — which
honors `ignorecase`/`smartcase` — mirroring the existing `review-smoothscroll`
option-assertion pattern (lua `check()` writes a token, shell greps for it). A
plain flag assertion is added too, but the behavioral pair is the real guard
against a future regression that flips only one of the two options.

## Done when

- `ignorecase` and `smartcase` are set in the review pane; `/foo` matches `Foo`,
  `/Foo` does not match `foo`.
- Asserted in `tests/review-window-test.sh` (`make test-review`).

## Estimate

Trivial, fully-specced config change (design pre-resolved in the Spec). Method A.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.0 impl=0.1
item: milestone-review design=0.0 impl=0.15
total: 0.25
```

- item 1: two `vim.opt` lines in `review.lua` + option assertion + behavioral
  search assertion in `tests/review-window-test.sh`.
- item 2: mandatory boundary review at `sdlc close`.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Plan

Atomic single-pass (2 files, ~6 lines) — plain checkboxes, one `sdlc close`.

- [x] Set `vim.opt.ignorecase = true` + `vim.opt.smartcase = true` in
      `nvim/review.lua`, in the top `vim.opt` block (pane-local; self-contained
      `nvim -u` init, so it doesn't touch the draft nvim or the user's config).
- [x] Assert in `tests/review-window-test.sh` (mirroring the `review-smoothscroll`
      pattern): (a) both options set (`review-smartcase-opt`), and (b) **behavioral**
      (`review-smartcase-search`) — buffer `{aaa,Foo}` + `/foo` finds line 2
      (ignorecase); buffer `{aaa,foo}` + `/Foo` finds nothing (smartcase).
- [x] `make test-review` green; negative test confirms both assertions FAIL when
      the options are stripped (exit 2), pass when restored (exit 0).

## Log

### 2026-07-05

- Split out of #89 (concurrent-edit reconciliation) to stay focused.

### 2026-07-06
- 2026-07-06: closed — full make test green (exit 0); review pane search is smartcase — behaviorally asserted in review-window-test.sh: buffer {aaa,Foo}+/foo finds line 2 (ignorecase), {aaa,foo}+/Foo finds nothing (smartcase); negative test confirms both assertions FAIL when the two options are stripped (exit 2) and pass when restored (exit 0).; review verdict: SHIP

- Implemented: `ignorecase`+`smartcase` in `nvim/review.lua` (top `vim.opt` block,
  after the gutter opts); option + behavioral assertions in
  `tests/review-window-test.sh` (mirrors the `review-smoothscroll` token pattern).
- Verified: `make test-review` green (both `review-smartcase-{opt,search}` pass);
  negative test — stripping the two options fails both assertions (exit 2),
  restoring passes (exit 0). Full `make test` before close.