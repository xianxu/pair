---
id: '000008'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# drop <leader>cs and <leader>cp

## Problem

Two nvim-internal leader bindings shipped in v1:
- `<leader>cs` — send only the section between `---` markers
- `<leader>cp` — paste-and-reflow at cursor (raw, no quoting)

Both presumed a "draft as notebook" workflow with multiple in-flight prompts separated by `---`. That workflow didn't materialize — actual usage is "compose in nvim, Alt+Return clears, repeat." The leaders go unused.

`<leader>cp` is also redundant — vim's default `p` already pulls from system clipboard (we set `clipboard=unnamedplus`). The reflow part is the only delta, but raw paste is rarely what you want anyway; copy-on-select handles the common case (paste with quote + reflow).

User feedback: "I don't find those useful."
