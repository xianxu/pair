---
id: 000009
status: done
deps: [000001]
created: 2026-05-02
updated: 2026-05-02
---

# migrate drafts and logs to XDG data dir

## Problem

Drafts and per-session prompt logs live under `~/scratch/`. That was lazy v1 placement. The right home per XDG Base Directory spec is `${XDG_DATA_HOME:-~/.local/share}/pair/`.

## Spec

**Path migration:**

| Old | New |
|---|---|
| `~/scratch/pair-draft-<tag>.md` | `${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md` |
| `~/scratch/pair-log-<tag>.md` | `${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md` |
| `~/.cache/pair/quit-<session>` | unchanged (already correct XDG cache) |
| `~/.local/share/pair/undo/` | unchanged (already correct XDG data) |

Note the file-name simplification: drop the `pair-` prefix on draft/log filenames since they're already inside the `pair/` dir.

**Data-dir resolver in `bin/pair`:**

```bash
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
mkdir -p "$DATA_DIR"
DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
```

Same convention in `nvim/init.lua` for the log path (used by `append_log`).

Layout file's draft path also updated to use `$XDG_DATA_HOME` (with default fallback) inline.

**Migration of existing files:** add a one-time auto-migration in `bin/pair` — if `~/scratch/pair-draft-*.md` or `~/scratch/pair-log-*.md` files exist, move them to the new location with name fix-up (strip the `pair-` prefix). Print a one-line note to stderr explaining what moved. Idempotent: if no source files exist, no-op.

**Help text and README updates** to reflect the new paths.

## Plan

- [x] Add `DATA_DIR` resolver to `bin/pair` (XDG_DATA_HOME with fallback).
- [x] Update `bin/pair` to use `$DATA_DIR/draft-<tag>.md` instead of `~/scratch/pair-draft-<tag>.md`.
- [x] Update `nvim/init.lua` `append_log()` to use `$PAIR_DATA_DIR/log-<tag>.md` (exported by bin/pair, with same fallback chain).
- [x] Update `zellij/layouts/main.kdl` draft pane args to point nvim at `${XDG_DATA_HOME:-$HOME/.local/share}/pair/draft-<tag>.md`.
- [x] Add one-time auto-migration in `bin/pair`: scan `~/scratch/pair-{draft,log}-*`, move to new location with name fix-up, log count to stderr.
- [x] Update `--help` FILES section.
- [x] Update README FILES section.
- [x] Update atlas/architecture.md Data layout section.
- [x] `bash -n`, `zellij setup --dump-layout`, `nvim --headless` all clean.
- [ ] Manual smoke test: with old `~/scratch/pair-draft-claude.md` present, run `pair`, verify the file moved and the session loads it.

## Log

### 2026-05-02

Filed after user pointed out non-XDG placement.
