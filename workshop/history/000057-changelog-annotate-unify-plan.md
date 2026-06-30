# Alt+q annotation in change-log viewer + shared `nvim/annotate.lua` — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the scrollback viewer's 🤖-marker machinery into a shared `nvim/annotate.lua`, refactor `scrollback.lua` onto it with no behavior change, then wire the same Alt+q annotate flow into the change-log viewer (`changelog.lua`).

**Architecture:** `annotate.lua` is a single module loaded by both read-only viewers via `dofile` (same dir-relative pattern as `adapt.lua`, because each viewer launches with `nvim -u <viewer>.lua` and may not have `nvim/` on its runtimepath). Its **pure core** (marker parse / escape / extract — no IO) is exposed for direct unit testing; its **thin IO/UI seam** (`attach`, the floating prompt, the read-only rewrite dance, the `VimLeavePre` sidecar emit, the quit-confirm gate) is parameterized per buffer by `{ bufnr, pending_path, footer, source_label }`. Each viewer keeps what genuinely differs: scrollback owns SGR rendering + `Alt+b` + the footer affordance config; changelog owns markdown colorize + the async distill refresh + the reload-vs-marker guard.

**Tech Stack:** Lua (Neovim API), headless `nvim -l` test harness (`make test-lua`), no third-party plugins.

---

## Core concepts

This work moves a ~400-line marker subsystem out of `scrollback.lua` (1240 lines) into `annotate.lua` and points both viewers at it. The conceptual split is the load-bearing decision: the **pure marker functions** (string→value, no buffer/IO) test without mocks, satisfying **ARCH-PURE**; consolidating one copy of them defended by both viewers satisfies **ARCH-DRY** (the issue's whole reason to exist).

### Pure entities (the conceptual core)

These are lifted verbatim from `scrollback.lua` (no logic change) into `annotate.lua` and exposed on the module table `M` for direct testing. They take strings / line-lists / marker-tables and return values — no buffer reads, no file IO, no `vim.api` mutation (a couple call read-only `vim.fn.strdisplaywidth`, which is deterministic display-width math, not IO).

| Name | Lives in | Status |
|------|----------|--------|
| `MARKER_BOT` (🤖 byte constant) | `nvim/annotate.lua` | new (moved) |
| `esc_x` / `esc_y` / `unescape` / `find_unescaped` | `nvim/annotate.lua` | new (moved) |
| `find_markers_in_line` | `nvim/annotate.lua` | new (moved) |
| `strip_markers` | `nvim/annotate.lua` | new (moved) |
| `marker_key` | `nvim/annotate.lua` | new (moved) |
| `collect_markers_by_line` | `nvim/annotate.lua` | new (moved) |
| `format_extraction` | `nvim/annotate.lua` | modified (moved + `source_label` opt) |
| `new_marker_count` | `nvim/annotate.lua` | new |
| `truncate_to_width` / `wrap_to_width` | `nvim/annotate.lua` | new (moved) |

- **`find_markers_in_line`** — the byte-walk parser returning every `🤖[Y]` / `🤖<X>[Y]` marker as `{kind, X?, Y, range, parts}`. The keystone; everything else consumes its output.
  - **Relationships:** 1:N — one line yields N markers. Consumed by `strip_markers`, `collect_markers_by_line`, `format_extraction`, `highlight_markers`, `marker_under_cursor`.
  - **DRY rationale:** Currently the sole copy lives in scrollback; changelog would otherwise need its own parser. This is the single largest reason the issue exists (**ARCH-DRY**).

- **`format_extraction`** — walks buffer lines, subtracts the load-time baseline (so only user-added markers ship), returns the markdown `> quote\nY` block. **Gains a `source_label` option** (the one behavior change in the moved core): when set, each quote line is prefixed `> [<label>] <quote>` instead of `> <quote>`.
  - **Relationships:** consumes `find_markers_in_line` + `strip_markers` + `marker_key`; baseline from `collect_markers_by_line`.
  - **DRY rationale:** one extraction format for both viewers; source differentiation is a parameter, not a fork.
  - **Why per-quote, not a header line:** the draft pickup (`init.lua` `pair_pickup_scrollback_pending`) counts comments via `content:gmatch('\n> ')`. A standalone `> [change log]` header line would inflate that count by one (wrong "picked up N" toast). Prefixing each quote keeps exactly one `> ` per marker, so the count stays correct **and** every shipped question carries its source. (Verified against `init.lua:2585-2586`.)

- **`new_marker_count`** — pure helper: given `(lines, baseline)`, returns the number of non-empty user-added markers (current minus baseline, same subtraction `format_extraction` does). Replaces scrollback's current `block:gmatch('\n> ')` recount in the Esc-confirm message with a label-independent count (the source_label prefix must not change the number shown).
  - **DRY rationale:** the confirm gate and `format_extraction` agree on "what counts as a new marker" from one definition rather than two.

**Test surface:** `nvim/annotate_test.lua` (new, headless `nvim -l`) exercises every row above directly, no mocks — this *is* the ARCH-PURE boundary made visible. Cases: bare + scoped parse; escaped delimiters (`\>`, `\]`, `\\>`) round-trip through `esc_*`/`unescape`; multiple markers per line; malformed (unclosed) marker ignored; `strip_markers` whitespace trim; `marker_key` collision avoidance; `format_extraction` baseline subtraction + empty-Y drop + scoped-uses-X + bare-uses-stripped-line + `source_label` prefix; `new_marker_count` matches the block's marker count with and without a label.

### Integration points (where pure meets the world)

All new/moved IO+UI lives in `annotate.lua` behind `M.attach` and a few helpers. These are the "thin shell" of ARCH-PURE — they wrap buffer mutation, the floating prompt window, the sidecar file write, and the quit autocmd.

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `M.attach{bufnr,pending_path,footer,source_label}` | `nvim/annotate.lua` | new | buffer state + keymaps + autocmd registration |
| `highlight_markers` | `nvim/annotate.lua` | new (moved) | extmark namespace |
| `open_marker_prompt` | `nvim/annotate.lua` | new (moved) | floating window + keymaps |
| `add_marker_normal` / `add_marker_visual` / `edit_marker` / `rewrite_line` / `marker_under_cursor` | `nvim/annotate.lua` | new (moved) | read-only buffer unlock→insert→relock |
| footer flow (`add_footer_comment` / `update_footer_line` + `FOOTER_HINT`/`FOOTER_PREFIX`) | `nvim/annotate.lua` | new (moved, gated) | footer affordance row, only when `footer=true` |
| `M.emit` (was `emit_pending`) + `VimLeavePre` autocmd | `nvim/annotate.lua` | new (moved) | sidecar file (`os.rename` atomic) |
| `M.confirm_quit(bufnr)` | `nvim/annotate.lua` | new | `vim.fn.confirm` + `qa` |
| `M.has_new_markers(bufnr)` / `M.on_reloaded(bufnr)` | `nvim/annotate.lua` | new | per-buffer baseline state (for changelog's reload guard) |
| scrollback wiring | `nvim/scrollback.lua` | modified | `dofile` annotate, call `attach`, keep SGR + Alt+b + arrow-nops |
| changelog wiring | `nvim/changelog.lua` | modified | `dofile` annotate, call `attach`, guard async reloads |

- **`M.attach{...}`** — the one entry point a viewer calls after it has loaded + (read-only-)set-up its buffer. It: snapshots the load-time baseline (`collect_markers_by_line`), runs `highlight_markers`, appends the footer affordance line **iff `footer=true`**, records per-buffer config in module-level `state[bufnr]`, and sets the `Alt+q` normal+visual keymaps. It does **not** bind quit keys (each viewer binds its own chosen keys to `M.confirm_quit` — scrollback deliberately omits `q`, changelog keeps `q`).
  - **Injected into:** nothing — it's the seam. The pure core is injected into *it*. Per-buffer state lives in `state[bufnr] = {pending_path, footer, source_label, baseline, footer_row, footer_text}`; footer fields stay nil when `footer=false`, which makes every `if row == footer_row` check naturally a no-op (no separate gating branch needed).
  - **Future extensions:** a third read-only viewer (e.g. a log/diff viewer) attaches with its own `source_label`; nothing else changes.

- **`emit_pending` + `VimLeavePre`** — registered once per process at module load; iterates `state` and writes each attached buffer's extraction to its `pending_path` via tmp+`os.rename`. (Each viewer is its own `nvim` process, so `state` holds exactly one buffer in practice — keyed-by-bufnr for robustness, matching the existing `initial_markers_by_buf` shape.)
  - **Injected into:** uses `format_extraction(lines, baseline, {source_label})`. Footer text appended only when `footer=true`.

- **`M.has_new_markers` / `M.on_reloaded`** — the changelog-specific wrinkle (Spec §"Async-reload conflict"). Markers are buffer *text*; the distiller's `M.reload` replaces all lines and would wipe a marker added during the spinner. `has_new_markers(bufnr)` = `new_marker_count(current_lines, baseline) > 0`. changelog wraps every reload site: skip the reload when markers are present (annotations win; the fresh log is on disk for the next `Alt+l`). `on_reloaded(bufnr)` re-snapshots the baseline + re-highlights after a reload that *did* run, so the baseline stays aligned with reloaded content.
  - **Injected into:** changelog's `start_refresh` reload paths.

**Test surface for integration points:** scrollback's existing `scrollback_test.lua` (prompt-pattern) must stay green — it's the regression net proving the extraction didn't break the load path. `changelog_test.lua` gains a headless smoke check: `attach` a changelog buffer with `footer=false, source_label='change log'`, insert a marker as buffer text (simulating the Alt+q result without driving the floating prompt), run `emit_pending`, assert the sidecar file contains the `> [change log] …` tagged block; and assert `has_new_markers` is true and a guarded reload is skipped. The floating-prompt UI itself is not unit-testable headlessly (window + insert-mode keymaps) — its logic is covered by the pure-core tests on what it produces; the wiring smoke test covers the rest. This limit is logged, not hidden.

---

## Chunk 1: M1 — Extract `annotate.lua`, refactor `scrollback.lua` (behavior-preserving)

**Review boundary M1.** Goal: the marker machinery lives once in `annotate.lua`; `scrollback.lua` is refactored onto it; `scrollback_test.lua` stays green; new `annotate_test.lua` covers the pure core. **No scrollback behavior change** (Spec "Out of scope"). This boundary's review question: *is the extraction behavior-preserving?*

### Task 1.1: Create `annotate.lua` with the pure marker core + tests

**Files:**
- Create: `nvim/annotate.lua`
- Create: `nvim/annotate_test.lua`
- Modify: `Makefile.local:74-79` (`test-lua` target — it lists test files **explicitly**, no glob; the new test won't run unless registered)

- [ ] **Step 1: Create `nvim/annotate.lua` skeleton with the pure core moved verbatim**

Move these from `scrollback.lua` (lines cited for the source), unchanged except `local fn` → `M.fn` exposure:
- `MARKER_BOT` (`scrollback.lua:280`)
- `esc_x`, `esc_y`, `unescape`, `find_unescaped` (`:354-400`)
- `find_markers_in_line` (`:410-461`)
- `strip_markers` (`:514-521`)
- `marker_key` (`:532-537`)
- `collect_markers_by_line` (`:542-556`)
- `truncate_to_width`, `wrap_to_width` (`:622-659`)

`format_extraction` (`:563-600`) moves with **one** change — add the `opts` param:

```lua
function M.format_extraction(buf_lines, baseline_by_line, opts)
  opts = opts or {}
  local qprefix = opts.source_label and ('> [' .. opts.source_label .. '] ') or '> '
  baseline_by_line = baseline_by_line or {}
  local pieces = {}
  -- ... identical walk ...
      table.insert(pieces, qprefix .. quote .. '\n' .. m.Y)
  -- ...
  return table.concat(pieces, '\n\n')
end
```

Add the new pure helper:

```lua
-- Count user-added (beyond-baseline), non-empty markers across buf_lines.
-- Same subtraction format_extraction does; used by the confirm gate so the
-- count is independent of any source_label prefix.
function M.new_marker_count(buf_lines, baseline_by_line)
  baseline_by_line = baseline_by_line or {}
  local n = 0
  for i, line in ipairs(buf_lines) do
    local markers = M.find_markers_in_line(line)
    local skip = {}
    for k, v in pairs(baseline_by_line[i] or {}) do skip[k] = v end
    for _, m in ipairs(markers) do
      if not m.Y:match('^%s*$') then
        local k = M.marker_key(m)
        if (skip[k] or 0) > 0 then skip[k] = skip[k] - 1
        else n = n + 1 end
      end
    end
  end
  return n
end
```

Expose every pure fn on `M` (e.g. `M.find_markers_in_line = find_markers_in_line`). End file with `return M`.

Then register the new test in the `make test-lua` target — `Makefile.local:74-79` lists each test file by hand (no glob), so add a line under `test-lua:`:

```make
	nvim -l nvim/annotate_test.lua
```

Without this the new test silently never runs and M1's "annotate green" verification would be false.

- [ ] **Step 2: Write the failing test `nvim/annotate_test.lua`**

Mirror `changelog_test.lua`'s harness (it loads its module by dir-relative `dofile` and counts failures). Cover the pure core:

```lua
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'annotate.lua')
local MARKER = '\240\159\164\150'  -- 🤖 = U+1F916, 4 UTF-8 bytes
local fails = 0
local function check(c, m) if not c then io.stderr:write('FAIL '..m..'\n'); fails = fails + 1 end end

-- bare + scoped parse
local bare = M.find_markers_in_line(MARKER..'[hello]')
check(#bare == 1 and bare[1].kind == 'bare' and bare[1].Y == 'hello', 'bare parse')
local scoped = M.find_markers_in_line(MARKER..'<sel>[c]')
check(scoped[1].kind == 'scoped' and scoped[1].X == 'sel' and scoped[1].Y == 'c', 'scoped parse')

-- escaped-delimiter round-trip
check(M.unescape(M.esc_y('a]b')) == 'a]b', 'esc_y/unescape round-trip on ]')
check(M.unescape(M.esc_x('x>y]z')) == 'x>y]z', 'esc_x/unescape round-trip on >]')
check(M.unescape(M.esc_x('a\\>b')) == 'a\\>b', 'backslash-then-delim round-trip')

-- baseline subtraction: a load-time marker is NOT re-emitted; a fresh one is.
local lines = { MARKER..'[old]', 'text '..MARKER..'[new]' }
local baseline = M.collect_markers_by_line({ MARKER..'[old]' })  -- only line 1 pre-existing
local block = M.format_extraction(lines, { [1] = baseline[1] })
check(block:match('new') and not block:match('old'), 'baseline subtracts load-time marker')

-- empty-Y dropped
check(M.format_extraction({ MARKER..'[]' }, {}) == '', 'empty-Y marker dropped')

-- source_label prefixes each quote, count unchanged
local labelled = M.format_extraction({ 'q '..MARKER..'[why]' }, {}, { source_label = 'change log' })
check(labelled:match('^> %[change log%] '), 'source_label prefixes the quote')
check(M.new_marker_count({ 'q '..MARKER..'[why]' }, {}) == 1, 'new_marker_count = 1')
check(M.new_marker_count(lines, { [1] = baseline[1] }) == 1, 'new_marker_count subtracts baseline')
```

- [ ] **Step 3: Run the test, expect PASS (pure functions moved unchanged)**

Run: `make test-lua` (or `nvim -l nvim/annotate_test.lua`)
Expected: `ok annotate_test` — these are moved-verbatim functions so they pass immediately; the test's job is to lock the contract before scrollback depends on it.

- [ ] **Step 4: Commit**

```bash
git add nvim/annotate.lua nvim/annotate_test.lua
git commit -m "#57 M1: extract pure marker core into nvim/annotate.lua + tests"
```

### Task 1.2: Move the IO/UI seam into `annotate.lua` and add `attach`

**Files:**
- Modify: `nvim/annotate.lua`

- [ ] **Step 1: Move the UI/IO functions** from `scrollback.lua` into `annotate.lua`, rewritten to read per-buffer config from a module-level `state[bufnr]` instead of the scattered `*_by_buf` tables:
  - highlight groups + `highlight_markers` (`:468-508`)
  - `open_marker_prompt` (`:697-843`), `marker_under_cursor` (`:848-858`), `rewrite_line` (`:862-869`), `edit_marker` (`:877-909`)
  - footer flow: `FOOTER_HINT`/`FOOTER_PREFIX`, `update_footer_line` (`:914-924`), `add_footer_comment` (`:929-936`) — these run only when `state[bufnr].footer`
  - `add_marker_normal` (`:938-962`), `add_marker_visual` (`:964-1000`) — footer-row checks read `state[bufnr].footer_row` (nil ⇒ no-op when `footer=false`)
  - `emit_pending` (`:1011-1034`) → exposed as the public `M.emit(bufnr)` (one entry point for both `VimLeavePre` and the changelog smoke test) — reads `state[bufnr].{pending_path, baseline, footer, footer_row, footer_text, source_label}`; footer removal + append only when `footer`

- [ ] **Step 2: Write `M.attach`, `M.confirm_quit`, `M.has_new_markers`, `M.on_reloaded`**

```lua
local state = {}  -- bufnr -> { pending_path, footer, source_label, baseline, footer_row, footer_text }

function M.attach(opts)
  local b = opts.bufnr
  local lines = vim.api.nvim_buf_get_lines(b, 0, -1, false)
  state[b] = {
    pending_path = opts.pending_path,
    footer = opts.footer or false,
    source_label = opts.source_label,
    baseline = M.collect_markers_by_line(lines),  -- snapshot BEFORE footer line
    footer_text = nil,
  }
  highlight_markers(b)
  if state[b].footer then
    local row0 = vim.api.nvim_buf_line_count(b)
    set_modifiable(b, true)
    vim.api.nvim_buf_set_lines(b, row0, row0, false, { FOOTER_HINT })
    set_modifiable(b, false)
    state[b].footer_row = row0 + 1  -- 1-based
  end
  vim.keymap.set('n', '<M-q>', function() add_marker_normal(b) end, { buffer = b, silent = true })
  vim.keymap.set('x', '<M-q>', function() add_marker_visual(b) end, { buffer = b, silent = true })
  vim.b[b].pair_annotate = true  -- VimLeavePre sentinel
end

function M.has_new_markers(b)
  if not state[b] then return false end
  local lines = vim.api.nvim_buf_get_lines(b, 0, -1, false)
  return M.new_marker_count(lines, state[b].baseline) > 0
end

function M.on_reloaded(b)  -- after a guarded reload that DID run: realign baseline
  if not state[b] then return end
  state[b].baseline = M.collect_markers_by_line(vim.api.nvim_buf_get_lines(b, 0, -1, false))
  highlight_markers(b)
end

function M.confirm_quit(b)  -- bound by each viewer to its quit key(s)
  local st = state[b]
  local lines = vim.api.nvim_buf_get_lines(b, 0, -1, false)
  if st and st.footer_row then table.remove(lines, st.footer_row) end
  local n = st and M.new_marker_count(lines, st.baseline) or 0
  local has_footer = st and st.footer and st.footer_text and st.footer_text ~= ''
  if n == 0 and not has_footer then vim.cmd('qa'); return end
  local parts = {}
  if n > 0 then table.insert(parts, string.format('%d 🤖[] marker%s', n, n == 1 and '' or 's')) end
  if has_footer then table.insert(parts, 'overall comment') end
  local prompt = 'Exit viewer? ' .. table.concat(parts, ' + ') .. ' will be sent.'
  if vim.fn.confirm(prompt, '&Yes\n&No', 1, 'Question') == 1 then vim.cmd('qa') end
end
```

Expose `emit_pending` as the public `M.emit(bufnr)`. Register the `VimLeavePre` autocmd once at module load, iterating buffers with `vim.b[b].pair_annotate` and calling `M.emit(b)`.

- [ ] **Step 3: Run the pure-core test to confirm no regression**

Run: `make test-lua`
Expected: `ok annotate_test` still passes (UI moves don't touch the pure fns).

- [ ] **Step 4: Commit**

```bash
git add nvim/annotate.lua
git commit -m "#57 M1: move marker UI/IO seam into annotate.lua behind attach()"
```

### Task 1.3: Refactor `scrollback.lua` onto `annotate.lua`

**Files:**
- Modify: `nvim/scrollback.lua`

- [ ] **Step 1: Load annotate + delete the moved code.** After the `adapt` load block, add:

```lua
local annotate
do
  local src = debug.getinfo(1, 'S').source:sub(2)
  local dir = src:match('(.*/)') or './'
  annotate = dofile(dir .. 'annotate.lua')
end
```

Delete from `scrollback.lua` every function/constant now in `annotate.lua` (the marker core + the UI seam + the footer flow + `emit_pending`/`sidecar_path` logic — keep `sidecar_path` *call* but pass its result as `pending_path`). **Keep** in scrollback: PALETTE/`resolve_256`/`apply_sgr`/`hl_for`/`process_line`/`decorate_buffer` (SGR), `prompt_pattern`/`jump_to_prompt` (Alt+b), the pair-launcher stubs, the Alt-arrow/`ZZ`/`ZQ` nops + `ttimeoutlen` + eob fillchar (scrollback-specific zellij-chord safety — Spec keeps these per-viewer), viewport positioning.

- [ ] **Step 2: Rewrite the `BufReadPost` autocmd** to call `decorate_buffer` then set read-only then `annotate.attach`:

```lua
decorate_buffer(bufnr)
vim.bo[bufnr].modifiable = false
vim.bo[bufnr].readonly = true
vim.bo[bufnr].buftype = 'nofile'
vim.bo[bufnr].swapfile = false
annotate.attach({ bufnr = bufnr, pending_path = sidecar_path(), footer = true })  -- no source_label: byte-identical scrollback emit
-- keep: <Esc> -> annotate.confirm_quit(bufnr); Alt+b/B -> jump_to_prompt; arrow-nops; viewport positioning
vim.keymap.set('n', '<Esc>', function() annotate.confirm_quit(bufnr) end, { buffer = bufnr, silent = true })
```

Keep `_G.PairScrollbackTest = { prompt_pattern = prompt_pattern }` (trimmed to what `scrollback_test.lua` actually uses — the marker fns are now tested in `annotate_test.lua`).

- [ ] **Step 3: Run scrollback's regression test**

Run: `make test-lua`
Expected: `nvim/scrollback.lua: prompt pattern tests passed` AND `ok annotate_test`. (prompt-pattern test is the load-path regression net — if the `dofile` or trimmed `_G` table broke, it fails here.)

- [ ] **Step 4: Manual scrollback behavior check (no-regression proof for the review)**

Run a live pair session, `Alt+/`, then verify against `main`'s behavior: Alt+q normal drops a bare marker; Alt+q visual wraps a selection; edit-in-place works; the footer line is present + accepts an overall comment; Esc with markers confirms; quit ships the (un-prefixed, byte-identical) block to the draft. Record the steps + result in the issue `## Log`.

- [ ] **Step 5: Commit**

```bash
git add nvim/scrollback.lua
git commit -m "#57 M1: refactor scrollback.lua onto shared annotate.lua (no behavior change)"
```

### Task 1.4: Close milestone M1

- [ ] **Step 1:** `sdlc milestone-close --issue 57 --milestone M1 --verified '<evidence: make test-lua green (scrollback + annotate); manual scrollback parity vs main>'`. Fix any Critical/Important the auto-dispatched review raises before crossing. Log the `Review-Verdict:` outcome in `## Log`.

---

## Chunk 2: M2 — Wire `changelog.lua` through `annotate.lua`

**Review boundary M2.** Goal: Alt+q works in the change-log viewer (normal + visual), questions ship to the draft tagged `> [change log] …`, and a marker added during the async spinner survives the background reload. This boundary's review question: *does the changelog wiring work without regressing the async refresh?*

### Task 2.1: Attach annotate in the changelog viewer

**Files:**
- Modify: `nvim/changelog.lua`

- [ ] **Step 1: Load annotate** with the same dir-relative `dofile` block as scrollback (add near the top of `changelog.lua`).

- [ ] **Step 2: Call `attach` in the interactive `BufReadPost`/`BufWinEnter` autocmd**, after `M.setup` (which sets read-only), before `M.start_refresh`:

```lua
M.setup(args.buf)
vim.cmd('normal! G')
annotate.attach({
  bufnr = args.buf,
  pending_path = pending_path(),  -- data_dir .. '/scrollback-pending-' .. tag .. '.md', tag = PAIR_TAG or PAIR_AGENT or 'claude'
  footer = false,
  source_label = 'change log',
})
M.start_refresh(args.buf)
```

Add a `pending_path()` local mirroring scrollback's `sidecar_path` (same path the draft picks up — **do not** invent a new file; Spec §"Pending file"). Bind quit keys to the shared gate (replacing the current `qa!`):

```lua
for _, key in ipairs({ '<Esc>', 'q' }) do
  vim.keymap.set('n', key, function() annotate.confirm_quit(args.buf) end, { buffer = args.buf, silent = true })
end
```

- [ ] **Step 3: Guard `M.reload` against marker loss** in `start_refresh`. Route every reload through a guard:

```lua
local function safe_reload()
  if annotate.has_new_markers(bufnr) then return end  -- annotations win; disk has the fresh log for next Alt+l
  M.reload(bufnr, log)
  annotate.on_reloaded(bufnr)  -- realign baseline + re-highlight
end
```

There are exactly **two** `M.reload` call sites in `start_refresh`: inside `reload_if_changed` (`changelog.lua:125`) and the final exit reload (`:154`). Change both to `safe_reload()`. (`reload_if_changed`'s mtime-key check stays; only the `M.reload` call inside it is guarded — which automatically makes the `:153` `reload_if_changed()` call on the exit path safe too.) **Ordering:** declare `safe_reload` *before* `reload_if_changed`, since `reload_if_changed` (a `local function`) must capture it — a forward `local function` reference would be nil at call time.

- [ ] **Step 4: Manual smoke (interactive)** — log steps + result in `## Log`:
  - `Alt+l`, wait for distill, `Alt+q` on an entry → type a question → quit → focus draft → assert the draft shows `> [change log] <entry>` + the question, and the "picked up 1 comment" toast (count correct).
  - `Alt+l` again, `Alt+q` *during* the spinner → confirm the marker is NOT wiped when the distiller's reload fires.

- [ ] **Step 5: Commit**

```bash
git add nvim/changelog.lua
git commit -m "#57 M2: wire Alt+q annotate + source tag + reload guard into changelog viewer"
```

### Task 2.2: Changelog wiring smoke test

**Files:**
- Modify: `nvim/changelog_test.lua`

- [ ] **Step 1: Write the failing smoke test.** With `_G.PAIR_CHANGELOG_TEST=true` (interactive wiring skipped), load `annotate.lua`, attach a scratch buffer with `footer=false, source_label='change log'`, point `pending_path` at a temp file:

```lua
local annotate = dofile(here .. 'annotate.lua')
local MARKER = '\240\159\164\150'  -- 🤖
local buf = vim.api.nvim_create_buf(false, true)
vim.api.nvim_buf_set_lines(buf, 0, -1, false, { '## 2026-06-12', '', '- M1 done for #53' })
local pend = (os.getenv('TMPDIR') or '/tmp') .. '/pair-cl-annotate-test.md'
annotate.attach({ bufnr = buf, pending_path = pend, footer = false, source_label = 'change log' })
-- no footer affordance line appended:
check(vim.api.nvim_buf_line_count(buf) == 3, 'footer=false adds no affordance line')
-- simulate Alt+q result by inserting a marker as buffer text, then has_new_markers + emit
vim.bo[buf].modifiable = true
vim.api.nvim_buf_set_lines(buf, 2, 3, false, { '- M1 done for #53 '..MARKER..'[why M1 first?]' })
vim.bo[buf].modifiable = false
check(annotate.has_new_markers(buf) == true, 'has_new_markers true after add')
annotate.emit(buf)  -- the single public emit; VimLeavePre calls this too
local got = table.concat(vim.fn.readfile(pend), '\n')
check(got:match('> %[change log%] .-why M1 first%?'), 'sidecar tagged with change-log source')
os.remove(pend)
```

`emit_pending` is exposed as the public `M.emit(bufnr)` (Task 1.2) — the `VimLeavePre` autocmd and this test share that one entry point, no test-only alias.

- [ ] **Step 2: Run, expect PASS**

Run: `make test-lua`
Expected: `ok changelog_test` (+ scrollback + annotate green).

- [ ] **Step 3: Commit**

```bash
git add nvim/changelog_test.lua nvim/annotate.lua
git commit -m "#57 M2: changelog annotate wiring smoke test"
```

### Task 2.3: Atlas + close

- [ ] **Step 1: Update `atlas/`** for the new shared surface: add/point an entry for `nvim/annotate.lua` (the shared 🤖-marker subsystem both viewers consume), note the scrollback↔changelog↔annotate relationship, and ensure `atlas/index.md` links it. (Per AGENTS.md §8 — new surface at milestone close.)

- [ ] **Step 2:** `sdlc close --issue 57 --milestone M2 --verified '<evidence: make test-lua green across all 3 nvim tests; manual changelog Alt+q ships tagged block; marker survives spinner reload; scrollback parity vs main>'`. Let it compute `--actual`. Fix any Critical/Important from the auto-dispatched review first; log the verdict.

---

## Notes / risks

- **Process isolation:** scrollback and changelog are separate `nvim -u` processes, so `annotate.lua`'s module-level `state`/`VimLeavePre` never cross-contaminate. Keyed-by-bufnr anyway for robustness + to match the existing `*_by_buf` idiom.
- **`qa!` vs `qa`:** changelog's quit becomes `qa` via `confirm_quit` (so the confirm gate can run); `VimLeavePre` → `emit_pending` fires on any quit including the confirmed `qa`, so questions ship regardless of quit path.
- **Out-of-scope (do not fold in):** the #53 forward-notes in the issue `## Log` (force-refresh, anchor cost, silent model-failure, env renames, agy over-match, untested distiller branches). Track separately.
- **Headless test limit (logged, not hidden):** the floating `open_marker_prompt` window + insert-mode keymaps aren't driven in the headless tests; coverage is the pure-core parse/extract tests + the wiring smoke test on what the prompt *produces*. Manual interactive checks (Tasks 1.3/2.1) close the gap and are recorded in `## Log`.

## Revisions

### 2026-06-17 — plan-quality judge refinements (VERDICT: INFO, all non-blocking)

The `sdlc change-code` plan-quality judge verified every load-bearing claim and
gave 3 minor refinements; folded in:

1. **[ARCH-DRY] sidecar resolver in annotate, not a 3rd copy.** Add
   `annotate.default_pending_path()` (the `data_dir .. '/scrollback-pending-' .. tag .. '.md'`
   logic, tag = `PAIR_TAG or PAIR_AGENT or 'claude'`). Both viewers call it
   instead of each keeping a local copy (`scrollback.lua:1004-1009` had one;
   changelog would have added a second). `init.lua:2560-2565`'s draft-side copy
   stays — it doesn't load annotate. (Done in M1 for scrollback, M2 for changelog.)
2. **[behavior-preservation] quit-confirm noun.** `M.confirm_quit` takes the
   viewer noun from `state[bufnr].quit_noun` so scrollback keeps emitting
   `"Exit scrollback? …"` byte-identically (was at `scrollback.lua:1100`).
   `attach` opt `quit_noun` — scrollback: `'scrollback'`, changelog: `'change log'`,
   default `'viewer'`.
3. **[test coverage] assert the reload SKIP, not just the predicate.** M2's
   changelog smoke test adds a headless assertion: with a marker present,
   simulate the guarded reload (`if annotate.has_new_markers(buf) then`-skip) and
   assert the buffer text (with the marker) is left intact — so the highest-risk
   M2 behavior isn't verified by hand alone.
