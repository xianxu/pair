---
id: 000013
status: working
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# bringing in some command nvim setting in

some minimal nvim setting (without installing plugins I guess):

1. file path completion, e.g. ./ ~/ etc. should start completion from the directory where pair is started.

Other suggestions of customization for nvim as a small input buffer, not relying on plugins.

## Done when

- Typing a path token (e.g. `./`, `~/`, `src/`) in the draft buffer pops a completion menu as you type — no manual `<C-x><C-f>` trigger.
- `<Tab>`/`<S-Tab>` cycle items, `<CR>` accepts a selected item (else inserts newline), `<Esc>`/typing dismisses.
- No regression in existing keybinds (`<M-CR>` send, `<M-i>` image, paste flow).
- Zero new dependencies; pure built-in nvim.

## Spec

Built-in nvim already has filename completion (`<C-x><C-f>`) but it's a manual typeahead. We want as-you-type behavior with fzf-style fuzzy matching on the filename portion, plugin-free.

Approach: `TextChangedI` + `TextChangedP` autocmd inspects the path token at cursor. Split it into a directory prefix (everything up to the last `/`) and a filter (everything after). List entries in the directory via `vim.fn.getcompletion(dir, 'file')`, then fuzzy-filter with `vim.fn.matchfuzzy(entries, filter)` (built-in since nvim 0.6). Push the result to the popup with `vim.fn.complete()`.

Example: user types `./`. dir = `./`, filter = ``, no fuzzy step → popup lists every entry in cwd. User then types `md` → token becomes `./md`, dir = `./`, filter = `md`, matchfuzzy filters cwd entries by `md` → `AGENTS.md` rises to the top.

`completeopt = 'menu,menuone,noinsert,noselect'` — popup shows even on a single match, nothing auto-inserts or auto-selects (so a stray newline doesn't accidentally confirm). `<Tab>`/`<S-Tab>`/`<CR>` get expr-mappings that delegate to completion when the popup is visible and fall through to their normal behavior otherwise.

Path detection: trigger only when the token at end-of-cursor contains `/` or starts with `~`. Plain words (no slash) don't trigger, so non-path typing stays quiet.

cwd is already correct: zellij spawns nvim from pair's start directory, and we never `:cd` away or set `autochdir`. So `./` resolves where the user expects.

Augroup: existing autocmds in `init.lua` aren't grouped — they duplicate on `:luafile` reload. Wrap the new autocmd (and existing ones, while we're touching this) in a named augroup with `clear = true` so iteration via `:luafile $PAIR_HOME/nvim/init.lua` is clean.

Limitations (acceptable):
- `getcompletion(dir, 'file')` runs synchronously per keystroke. Bounded by entries in *one directory*, not the whole tree, so cheap.
- Doesn't respect `.gitignore`. Acceptable — only listing the directory the user is typing into.
- No file-vs-directory icon hints in the popup. Trailing `/` on directory entries already distinguishes them.

## Plan

- [ ] Wrap existing autocmds in a `pair` augroup with `clear = true` so reloads don't duplicate handlers.
- [ ] Add `completeopt = 'menu,menuone,noinsert,noselect'`.
- [ ] Add `path_complete()`: split token on last `/` into dir + filter, call `getcompletion(dir, 'file')`, pass through `matchfuzzy(entries, filter)` if filter non-empty, hand results to `vim.fn.complete()`.
- [ ] Wire `path_complete` to `TextChangedI` and `TextChangedP` (both — popup-visible and not).
- [ ] Add expr keymaps in insert mode: `<Tab>` → `<C-n>` if pum visible else `<Tab>`; `<S-Tab>` → `<C-p>` else `<S-Tab>`; `<CR>` → `<C-y>` if pum has a selected item else `<CR>`.
- [ ] Manual verification:
  - `./` pops menu of cwd entries.
  - `./md` (after fresh `./`) fuzzy-filters to `AGENTS.md`, `README.md` etc.
  - `~/` pops home dir.
  - `src/foo` works mid-line, not just at start of line.
  - `<Tab>`/`<S-Tab>` cycle. `<CR>` accepts when item selected; inserts newline otherwise.
  - `<M-CR>` send still works. `<M-i>` image flow still works. Quote/inline paste still works.
- [ ] Update `atlas/architecture.md` with one line about the fuzzy as-you-type completion in the draft pane.

## Log

### 2026-05-03

