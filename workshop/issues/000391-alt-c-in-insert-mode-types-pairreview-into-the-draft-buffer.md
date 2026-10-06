---
id: 000391
status: open
deps: []
github_issue:
created: 2026-10-04
updated: 2026-10-04
estimate_hours:
card_mirror: 'c8bff485eed5cf3fefd356d326db5337997abd34' # card fields mirrored from issue-cards; edit via sdlc
---

# Alt+c in insert mode types :PairReview into the draft buffer

## Problem

When no review pane is live and there is no review target, Alt+c takes the
`prompt` branch of the review toggle. That branch runs
`vim.api.nvim_feedkeys(':PairReview ', 'n', false)` (`nvim/init.lua:1010`), and the
keys go into whatever mode the draft buffer is in. In insert mode the text
`:PairReview ` is typed into the buffer, so the file-select command line never opens.
Replace, visual and select modes misbehave in the same way.

## Spec

Before feeding `:PairReview `, the prompt path brings the buffer back to normal mode.
For example, it feeds `<C-\><C-n>` (from any mode, including insert, replace,
visual, select and operator-pending) and then the command text, or it calls
`vim.cmd.stopinsert()` and leaves visual mode before opening the command line. In
normal mode nothing changes. Only the `prompt` action changes; `hide`, `show`,
`open` and `wait` already avoid typing into the buffer.

Look at the other `nvim_feedkeys(':…')` sites, such as `nvim/init.lua:3636`, for the
same insert-mode leak, and fix that whole class rather than one site.

## Done when

- A headless nvim test: with the draft in insert mode and the toggle in `prompt`, the
  buffer text is unchanged and the cmdline holds `PairReview `. The same holds from
  visual mode.
- From normal mode, the behaviour is the same as today.
- Any other `:`-feeding site found in the sweep is fixed or noted as unaffected.

## Plan

- [ ] Write a failing headless test (insert mode → prompt)
- [ ] Normalize to normal mode before feeding the command (the shared helper if the
      sweep finds more sites)

## Log

### 2026-10-04

- Reported by the operator. `install_global_maps` binds every workbench chord with
  `vim.keymap.set({ 'n', 'i' }, …)` (`nvim/workbench_route.lua:137`), so `<M-c>` →
  `PairReviewToggle` (`workbench_actions.lua:11`) runs from insert mode, and only the
  `prompt` branch types into the buffer. Visual mode reaches it only if the mapping
  gains `x`/`s` modes. The fix should still feed `<C-\><C-n>` so it holds whatever
  mode it is called from.
