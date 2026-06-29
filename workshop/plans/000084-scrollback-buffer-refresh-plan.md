# Scrollback Buffer Refresh Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Alt+/ scrollback viewer refresh its backing render on demand, with `G` refreshing first and then landing at the newest end.

**Architecture:** Keep the existing `pair-scrollback-open` launch path and `.ansi` viewer model. Add a thin refresh seam inside `nvim/scrollback.lua`: resolve the current `.ansi` path to sibling `.raw` and `.events.jsonl`, run `pair-scrollback-render`, reload the buffer, redecorate ANSI, and restore read-only viewer invariants. ARCH-DRY: reuse the existing renderer and `decorate_buffer`; do not duplicate rendering in Lua. ARCH-PURE: keep path derivation and position choice small/testable, with shelling out isolated to one helper. ARCH-PURPOSE: `G` must satisfy the issue's purpose by refreshing before jumping to end, not just moving to the stale file end.

**Tech Stack:** Lua in Neovim, existing Go `pair-scrollback-render`, headless `nvim -l nvim/scrollback_test.lua`.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `scrollback_paths` | `nvim/scrollback.lua` | new |

- **scrollback_paths** — derives `.ansi`, `.raw`, and `.events.jsonl` sibling paths for the current viewer buffer.
  - **Relationships:** 1:1 with the current scrollback viewer buffer.
  - **DRY rationale:** Centralizes filename derivation so the refresh path and tests do not restate string substitutions.
  - **Future extensions:** If a future live-refresh mode needs the same paths, it should reuse this helper.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `refresh_scrollback_buffer` | `nvim/scrollback.lua` | new | `pair-scrollback-render`, buffer IO |
| `G` keymap | `nvim/scrollback.lua` | modified | user key input |

- **refresh_scrollback_buffer** — runs the renderer and reloads the current `.ansi` buffer in-place.
  - **Injected into:** Exposed through `_G.PairScrollbackTest` so tests can provide a fake renderer command.
  - **Future extensions:** Add auto-refresh polling later without changing render/reload internals.

- **G keymap** — replaces default `G` in the scrollback viewer with refresh-then-end.
  - **Injected into:** Normal-mode buffer keymap only for scrollback viewer buffers.
  - **Future extensions:** A separate `r` manual refresh can be added if needed, but `G` is the required path now.

---

## Chunk 1: Refresh Helper And G Binding

### Task 1: Write failing headless tests

**Files:**
- Modify: `nvim/scrollback_test.lua`
- Modify: `nvim/scrollback.lua`

- [x] **Step 1: Add test for refresh reloading changed `.ansi` content**

Add a headless test that:
- Creates temp `.raw`, `.events.jsonl`, and `.ansi` files with the scrollback filename shape.
- Opens the `.ansi` in a scratch buffer.
- Stubs a renderer function that overwrites `.ansi` with new ANSI-styled content.
- Calls `_G.PairScrollbackTest.refresh_buffer(bufnr, { renderer = fake })`.
- Asserts the buffer now contains stripped new text, ANSI decoration was reapplied, and `modifiable=false` / `readonly=true`.

- [x] **Step 2: Add test for `G` behavior**

Add a headless test that:
- Starts with a two-line `.ansi`.
- Stubs refresh to rewrite it to four lines.
- Calls `_G.PairScrollbackTest.refresh_then_end(bufnr, { renderer = fake })`.
- Asserts the cursor is on line 4 after refresh.

- [x] **Step 3: Run the focused test to verify RED**

Run:

```bash
nvim -l nvim/scrollback_test.lua
```

Expected: FAIL because `refresh_buffer` / `refresh_then_end` are not implemented.

### Task 2: Implement minimal refresh support

**Files:**
- Modify: `nvim/scrollback.lua`

- [x] **Step 1: Add path derivation helper**

Implement a local helper that derives:
- `ansi` from the current buffer name.
- `raw` by replacing `.ansi` with `.raw`.
- `events` by replacing `.ansi` with `.events.jsonl`.

Return `nil, message` if the buffer name is not an `.ansi` path.

- [x] **Step 2: Add renderer invocation seam**

Implement a local `run_renderer(paths, opts)` helper:
- Use `opts.renderer` when supplied by tests.
- Otherwise run `{ pair-scrollback-render, paths.raw, paths.events, paths.ansi }`.
- Resolve the binary from `$PAIR_HOME/bin/pair-scrollback-render` when `PAIR_HOME` is set; otherwise fall back to `pair-scrollback-render` on PATH.

- [x] **Step 3: Add buffer reload helper**

Implement `refresh_scrollback_buffer(bufnr, opts)`:
- Save old lines so failure can leave the buffer intact.
- Temporarily unlock the buffer.
- Run renderer.
- Read the updated `.ansi` with `vim.fn.readfile`.
- Replace buffer lines.
- Call existing `decorate_buffer(bufnr)`.
- Relock `modifiable=false`, `readonly=true`, `buftype=nofile`, `swapfile=false`.
- On error, restore old lines if needed, relock, notify warning, and return `false`.

- [x] **Step 4: Add refresh-then-end helper**

Implement `refresh_then_end(bufnr, opts)`:
- Calls `refresh_scrollback_buffer`.
- If refresh succeeds, moves cursor to last line and runs `normal! zb`.
- If refresh fails, leaves cursor and existing content alone.

- [x] **Step 5: Expose helpers for tests**

Extend `_G.PairScrollbackTest` with:
- `refresh_buffer`
- `refresh_then_end`
- `scrollback_paths`

### Task 3: Wire the user interaction

**Files:**
- Modify: `nvim/scrollback.lua`

- [x] **Step 1: Bind `G` in the BufReadPost callback**

Add a buffer-local normal-mode mapping:

```lua
vim.keymap.set('n', 'G', function() refresh_then_end(bufnr) end,
               { buffer = bufnr, silent = true })
```

- [x] **Step 2: Update statusline help**

Add `G refresh/end` to the statusline string so the feature is discoverable.

### Task 4: Verify and update issue

**Files:**
- Modify: `workshop/issues/000084-scrollback-buffer-refresh.md`

- [x] **Step 1: Run focused Lua test**

Run:

```bash
nvim -l nvim/scrollback_test.lua
```

Expected: PASS.

- [x] **Step 2: Run broader Lua target**

Run:

```bash
make test-lua
```

Expected: PASS.

- [x] **Step 3: Validate issue**

Run:

```bash
/Users/xianxu/workspace/ariadne/bin/sdlc issue validate workshop/issues/000084-scrollback-buffer-refresh.md
```

Expected: conforms.

- [x] **Step 4: Commit implementation**

Commit paths:

```bash
git add nvim/scrollback.lua nvim/scrollback_test.lua workshop/issues/000084-scrollback-buffer-refresh.md workshop/plans/000084-scrollback-buffer-refresh-plan.md
git commit -m "#84: refresh scrollback viewer on G"
```

## Revisions

### 2026-06-29 — boundary-review marker safety

Reason: the close boundary review found that refreshing an annotate-attached
scrollback buffer replaced all lines without consulting the marker reload guard,
which could erase pending `Alt+q` annotations and lose the footer affordance.

Delta:
- `nvim/annotate.lua` owns a footer-aware reload hook:
  `has_pending_annotations` guards destructive reloads, and `on_reloaded`
  rebaselines markers while recreating the scrollback footer row.
- `nvim/scrollback.lua` renders the backing `.ansi` on `G`, but skips replacing
  the visible buffer when pending inline markers or footer comments exist.
- `nvim/scrollback_test.lua` now covers both marker-protected refresh and clean
  annotate-attached refresh with footer restoration.
- A follow-up `FIX-THEN-SHIP` review requested explicit coverage for the
  renderer-failure branch; `nvim/scrollback_test.lua` now asserts renderer
  failure preserves visible lines and read-only viewer state.
