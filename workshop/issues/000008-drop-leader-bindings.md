---
id: 000008
status: working
deps: [000001]
created: 2026-05-02
updated: 2026-05-02
---

# drop <leader>cs and <leader>cp

## Problem

Two nvim-internal leader bindings shipped in v1:
- `<leader>cs` — send only the section between `---` markers
- `<leader>cp` — paste-and-reflow at cursor (raw, no quoting)

Both presumed a "draft as notebook" workflow with multiple in-flight prompts separated by `---`. That workflow didn't materialize — actual usage is "compose in nvim, Alt+Return clears, repeat." The leaders go unused.

`<leader>cp` is also redundant — vim's default `p` already pulls from system clipboard (we set `clipboard=unnamedplus`). The reflow part is the only delta, but raw paste is rarely what you want anyway; copy-on-select handles the common case (paste with quote + reflow).

User feedback: "I don't find those useful."

## Spec

- Remove `send_section` function and its `<leader>cs` binding from `nvim/init.lua`.
- Remove `paste_and_reflow` function and its `<leader>cp` binding from `nvim/init.lua`.
- Remove the two `<leader>cs` / `<leader>cp` lines from `bin/pair --help` KEYBINDINGS.
- Remove from README Keybindings table.
- Remove from atlas/architecture.md nvim section.

Keeps Alt+Return, Alt+u, Alt+d, Alt+x, Alt+i. That's the actual surface.

## Plan

- [ ] Remove `send_section` function + keymap from `nvim/init.lua`.
- [ ] Remove `paste_and_reflow` function + keymap from `nvim/init.lua`.
- [ ] Remove from `bin/pair --help`.
- [ ] Remove from README.
- [ ] Update atlas/architecture.md.
- [ ] `nvim --headless -u init.lua -c qa` clean.

## Log

### 2026-05-02

Filed after user surfaced as unused.
