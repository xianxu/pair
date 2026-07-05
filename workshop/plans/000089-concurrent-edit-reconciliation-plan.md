# Concurrent-Edit Reconciliation Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the human keep editing a doc while the review agent produces a round; reconcile the agent's edits onto the live buffer per-record, surfacing only true overlaps as `🤖<…>[reconcile — …]` markers — replacing the rejected hard edit-lock.

**Architecture:** Three milestones. **M1** makes multi-line `🤖<…>` markers fully work (highlight + resolve), the prerequisite for conflict rendering. **M2** adds the reconcile engine: classify each record against the live buffer (span-granular, today's behavior), and model each non-anchoring record as a *synthetic replacement record* so the whole reconcile is one `apply.apply` call. **M3** adds the apply-gate: a pure `decide_apply` that defers the round only while the human is mid-edit, plus a "results ready" winbar and human-edit durability (save-on-defer + save-on-VimLeave). All hard logic is pure (`gate`/`reconcile`/`markers` functions), tested without IO; the buffer/`vim.diff`/poke touches are the thin glue in `init.lua`/`review.lua` (ARCH-PURE).

**Tech Stack:** Lua (Neovim `nvim -l` for pure tests, headless nvim shell tests for glue), `vim.diff` builtin, the existing `apply`/`reconstruct`/`markers`/`marker_codec` review core.

**Spec:** `workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md`

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `gate.decide_apply(v0, v1, focused, mode)` | `nvim/review/gate.lua` | new |
| `reconcile.classify(records, v1)` | `nvim/review/reconcile.lua` | new |
| `reconcile.conflict_marker(hunk_text, intents)` | `nvim/review/reconcile.lua` | new |
| `reconcile.plan_conflicts(conflicts, v0, v1, hunks)` | `nvim/review/reconcile.lua` | new |
| `markers.spans_multiline(lines)` | `nvim/review/markers.lua` | modified |
| `MULTILINE_LINE_BUDGET` (50 → 200) | `nvim/review/markers.lua` | modified |

- **`gate.decide_apply(v0, v1, focused, mode) → 'apply' | 'defer'`** — the pure apply-gate. `v1==v0 → 'apply'` (nothing changed); `not focused → 'apply'`; `mode == 'n' → 'apply'`; else `'defer'` (mid-edit on the focused pane). String in/out; no vim API.
  - **Relationships:** consumed once per landed handoff by `init.on_agent_round`.
  - **DRY rationale:** first occurrence of the gate decision; single source so the test and the glue agree on the five cases (mirrors how `readiness.lua` / `mode.lua` isolate a pure decision the glue acts on).
  - **Future extensions:** more "safe point" signals (e.g. a pending-macro flag) widen the argument list, not the callers.

- **`reconcile.classify(records, v1) → { clean, conflicts }`** — split agent records: `clean` = those whose `old` still anchors in the live buffer (`reconstruct.nth_offset(v1, r.old, r.occurrence or 1)` resolves — the *exact* test + `or 1` fallback `apply.apply` uses); `conflicts` = the rest (the human changed that span). Pure (`v1` is a string; `reconstruct` is pure).
  - **DRY rationale:** reuses `reconstruct.nth_offset` — the same anchor test `apply.apply` runs — so classify faithfully predicts what apply will land (ARCH-DRY). No re-implemented matching.

- **`reconcile.conflict_marker(hunk_text, intents) → string`** — pure builder for one conflict marker: `🤖<esc(hunk_text)>[reconcile — agent wanted:\n  • esc(old) → esc(new) (why: esc(explain))\n  • …]`. **Both** the `<…>` body and the `[…]` intents run through `marker_codec.esc_quote` so unbalanced brackets in quoted code can't break the parse.
  - **DRY rationale:** one source for the conflict-marker wire format; the protocol docs (M2 Task 2.6) and the reconcile glue reference this, not a restated template.

- **`reconcile.plan_conflicts(conflicts, v0, v1, hunks) → synthetic_records`** — pure: resolve each conflict record's `old`@occurrence against `v0` → its base line; find the `hunks` entry (a `vim.diff` indices tuple, passed as data) covering that base line → the hunk's `v1` line-range = the human's current text; coalesce conflicts sharing a hunk; emit one synthetic replacement record per hunk `{ old = «v1 hunk text», occurrence = «nth in v1», new = conflict_marker(...), explain = "reconcile" }`. Pure given `hunks` as data (the `vim.diff` call is the glue that produces `hunks`).
  - **Future extensions:** hunk-size cap (200 lines, mirrors `MULTILINE_LINE_BUDGET`) — a larger hunk emits a short "region changed" marker instead of quoting the whole thing.

- **`markers.spans_multiline(lines) → spans`** — pure highlight spans that may cross lines: `{ row, col, end_row, end_col, hl_group }`, derived from `parse_markers` (which already crosses lines) by converting its doc-offset spans (`quoted.byte_start/byte_end`, each `section.byte_start/byte_end`, `strike.*`) to (row,col) pairs. Replaces the per-line `highlight_spans` for rendering.
  - **DRY rationale:** `parse_markers` is the single multi-line parser; the highlighter now derives from it instead of re-scanning per line (removing the per-line `highlight_spans` duplication that couldn't see across lines).

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `reconcile.reconcile_round(buf, records, v0)` | `nvim/review/reconcile.lua` | new | buffer + `vim.diff` + `apply.apply` |
| `apply_round` / `on_agent_round` / session `base` | `nvim/review/init.lua` | modified | handoff→apply orchestration |
| pane gate wiring (winbar, defer, Alt+Return, save) | `nvim/review.lua` | modified | UI state + saves |
| `tests/review-reconcile-test.sh` (process test) | `tests/` | new | headless nvim |

- **`reconcile.reconcile_round(buf, records, v0)`** — glue: `v1 = apply.buf_content(buf)`; `{clean, conflicts} = classify(records, v1)`; if no conflicts, `apply.apply(buf, clean)`; else `hunks = vim.diff(v0, v1, {result_type='indices'})`, `synth = plan_conflicts(conflicts, v0, v1, hunks)`, `apply.apply(buf, clean ++ synth)`. Returns `(enriched_clean, dropped, n_conflicts)`.
  - **Injected into:** called by `init.apply_round` on the reconcile branch. The pure `classify`/`plan_conflicts`/`conflict_marker` do the reasoning; this seam only does the `vim.diff` + `apply.apply` IO.
  - **Future extensions:** `vim.diff` opts (e.g. `linematch`) tuned here without touching the pure planners.

- **`init.lua` (`apply_round`, `on_agent_round`, `set_base`, session `base`)** — `apply_round(buf, records)` = the fast/reconcile branch (renamed from today's `on_agent_round` body, reading `v0` from `sessions[buf].base`); `on_agent_round(buf, records)` = the watcher entry that consults `gate.decide_apply` (via the injected `pane_state`) and either calls `on_defer` or `apply_round`; `set_base(buf, content)` stores `v0`.
  - **Injected into:** `pane_state`/`on_defer` hooks are set by `review.lua` (the UI layer), keeping `init.lua` free of focus/mode/winbar concerns (ARCH-PURE thin seam).

- **`review.lua` pane gate wiring** — `pane_state(buf) → {focused, mode}`; `on_defer(buf, records)` = stash pending + save buffer + winbar + `clear_awaiting`; `mark_awaiting` → `set_base`; Alt+Return branches on the pending slot; winbar set/clear; `VimLeave` saves if modified.
  - **Future extensions:** the pending slot is a single value today; a queue would widen it (spec §8 keeps it single).

- **`tests/review-reconcile-test.sh`** — process-level headless-nvim test driving `reconcile.reconcile_round` on a real buffer (clean-only, conflict, coalesced-hunk, fast-path). Added to `make test-review`.

---

## Chunk 1: M1 — multi-line marker foundation

**Milestone review boundary.** Closes with `sdlc milestone-close --issue 89 --milestone M1`.

### Task 1.1: multi-line highlight spans (`markers.spans_multiline`)

**Files:**
- Modify: `nvim/review/markers.lua` (add `M.spans_multiline`)
- Test: `nvim/review/markers_test.lua` (add cases)
- Modify: `nvim/review.lua` (`render_markers` uses `spans_multiline` + multi-line extmarks)

- [ ] **Step 1: Write the failing test** in `nvim/review/markers_test.lua`

```lua
-- A 🤖<…> whose quoted body spans two lines yields a span crossing rows.
do
  local lines = { 'before 🤖<first', 'second>[note] after' }
  local spans = markers.spans_multiline(lines)
  local quoted
  for _, s in ipairs(spans) do
    if s.hl_group == 'ParleyReviewQuoted' then quoted = s end
  end
  assert(quoted, 'expected a quoted span')
  assert(quoted.row == 0, 'quoted starts on row 0, got ' .. tostring(quoted.row))
  assert(quoted.col == 7, 'quoted starts at the 🤖 col (after "before "), got ' .. tostring(quoted.col))
  assert(quoted.end_row == 1, 'quoted ends on row 1, got ' .. tostring(quoted.end_row))
end
```

(The `col == 7` assertion pins the start to the 🤖 itself — `highlight_spans` starts the quoted/strike span at the marker, not at the `<`.)

- [ ] **Step 2: Run to verify it fails.** `nvim -l nvim/review/markers_test.lua` → FAIL (`spans_multiline` nil).

- [ ] **Step 3: Implement `M.spans_multiline`** in `markers.lua`. Reuse `parse_markers` (already multi-line) and convert its doc-offset spans to (row,col). Add near `highlight_spans`:

```lua
-- Multi-line highlight spans from the multi-line parser. Unlike highlight_spans
-- (per-line), a 🤖<…>/section span may cross rows (end_row > row).
function M.spans_multiline(lines)
  local line_starts, off = {}, 1
  for i, line in ipairs(lines) do line_starts[i] = off; off = off + #line + 1 end
  -- 1-based doc offset → (row0, col0); same binary search parse_markers uses.
  local function pos_of(offset)
    local lo, hi = 1, #line_starts
    while lo < hi do
      local mid = math.floor((lo + hi) / 2) + 1
      if line_starts[mid] <= offset then lo = mid else hi = mid - 1 end
    end
    return lo - 1, offset - line_starts[lo]
  end
  local spans = {}
  for _, m in ipairs(M.parse_markers(lines)) do
    -- push(start_row, start_col, end_offset_1based, hl)
    local function push(sr, sc, endoff, hl)
      local er, ec = pos_of(endoff)
      spans[#spans + 1] = { row = sr, col = sc, end_row = er, end_col = ec, hl_group = hl }
    end
    -- quoted/strike start at the 🤖 itself (m.line,m.col) — NOT at `<`/`~` —
    -- matching highlight_spans (col_start = pos-1 = the marker). end = the closer.
    if m.quoted then push(m.line, m.col, m.quoted.byte_end, 'ParleyReviewQuoted') end
    if m.strike and m.strike.text ~= '' then push(m.line, m.col, m.strike.byte_end, 'ParleyReviewStrike') end
    for _, s in ipairs(m.sections) do
      local sr, sc = pos_of(s.byte_start)  -- start at the [ / { bracket
      push(sr, sc, s.byte_end, s.type == 'agent' and 'ParleyReviewAgent' or 'ParleyReviewUser')
    end
  end
  return spans
end
```

Note: `m.line`/`m.col` are the 🤖's 0-based row/col (from `parse_markers`); `quoted.byte_end`/section `byte_end` are the closer's 1-based doc offset. This mirrors `highlight_spans` exactly (quoted/strike from the 🤖, sections from their bracket) — just allowing `end_row > row`. Verify against `highlight_spans` for a single-line marker when writing (same cols).

- [ ] **Step 4: Run to verify it passes.** `nvim -l nvim/review/markers_test.lua` → PASS.

- [ ] **Step 5: Point `render_markers` at the multi-line spans** in `nvim/review.lua` (currently uses `markers.highlight_spans` + single-line extmark). Change to `markers.spans_multiline` and place `end_row`/`end_col`:

```lua
for _, s in ipairs(markers.spans_multiline(lines)) do
  pcall(vim.api.nvim_buf_set_extmark, buf, MARK_NS, s.row, s.col, {
    end_row = s.end_row, end_col = s.end_col, hl_group = s.hl_group,
  })
end
```

- [ ] **Step 6: Assert render in the window test.** In `tests/review-window-test.sh` (`wdriver.lua`), after setting a two-line `🤖<a\nb>[x]` buffer + `render_markers`, assert an extmark exists with `end_row > row` in `review_markers` NS. Add a `grep -q` line.

- [ ] **Step 7: Run** `make test-lua && bash tests/review-window-test.sh` → PASS.

- [ ] **Step 8: Commit** `#89 M1: multi-line 🤖<…> highlight spans (spans_multiline)`.

### Task 1.2: within-range accept/reject (`resolve_at_cursor`)

**Files:**
- Modify: `nvim/review.lua` (`resolve_at_cursor`, `marker_end_pos` already multi-line)
- Test: `tests/review-window-test.sh`

- [ ] **Step 1: Failing test** — in `wdriver.lua`, set buffer `{ 'x 🤖<a', 'b>{c}', 'y' }`, cursor on **row 2** (inside the marker's second line), call `resolve_at_cursor(buf, 'accept')`, assert the two lines collapse to the accepted `c` (i.e. the marker resolved though the cursor wasn't on its first line). Add a `grep -q '^ml-resolve-in-range$'`.

- [ ] **Step 2: Run** `bash tests/review-window-test.sh` → FAIL (no marker matched on row 2).

- [ ] **Step 3: Implement.** In `resolve_at_cursor` (`nvim/review.lua`), the match loop currently keys off `m.line == row0`. Compute each marker's end row from `marker_end_pos(m)` and match when `row0 ∈ [m.line, end_row]`:

```lua
for _, m in ipairs(markers.parse_markers(lines)) do
  local end_row = select(1, marker_end_pos(m))
  if row0 >= m.line and row0 <= end_row then
    if not target then target = m end
    -- byte-range check only meaningful on the start line; keep the col check there
    if row0 == m.line and cur[2] >= m.col and cur[2] < (select(2, marker_end_pos(m)) or math.huge) then target = m; break end
  end
end
```

(Keep the existing `clear_decoration_at_line` fallback for the no-marker case.)

- [ ] **Step 4: Run** → PASS.

- [ ] **Step 5: Commit** `#89 M1: resolve 🤖<…> with cursor anywhere in its line range`.

### Task 1.3: audit `jump_marker` + `resolve_paragraph_to_cursor` for multi-line

**Files:**
- Modify (if needed): `nvim/review.lua`
- Test: `tests/review-window-test.sh`

- [ ] **Step 1: Add a characterization test** — a paragraph containing a multi-line marker; `resolve_paragraph_to_cursor(buf, 'accept')` with the cursor after it; assert the marker resolves and no sibling marker outside the paragraph is touched. `jump_marker`: assert `]m` lands on the marker's start line.

- [ ] **Step 2: Run** → observe. `jump_marker` keys off the marker start line (`pick.line`) — expected to already pass (jumping to a marker's start is correct). `resolve_paragraph_to_cursor` bounds by blank lines and `m.line`; a marker that *starts* in the paragraph is in-range. Confirm; only patch if the test fails.

- [ ] **Step 3: Fix only if red** — extend the paragraph-membership check to `m.line`..end_row if a multi-line marker straddling the paragraph's last line is mishandled. Otherwise leave unchanged (YAGNI) and note "audited, no change" in the issue `## Log`.

- [ ] **Step 4: Commit** `#89 M1: audit multi-line marker navigation/paragraph-resolve` (code or log-only).

### Task 1.4: raise the section budget (50 → 200)

**Files:**
- Modify: `nvim/review/markers.lua:18` (`MULTILINE_LINE_BUDGET`)
- Test: `nvim/review/markers_test.lua`

- [ ] **Step 1: Failing test** — a `🤖<…>` whose quoted body has 60 newlines currently fails to close (budget 50). Assert `parse_markers` returns a marker with a non-nil `quoted` for a 60-line body.

- [ ] **Step 2: Run** `nvim -l nvim/review/markers_test.lua` → FAIL.

- [ ] **Step 3: Implement** — `local MULTILINE_LINE_BUDGET = 200`. (The reconciler's hunk cap in M2 keeps conflict bodies ≤ this.)

- [ ] **Step 4: Run** → PASS.

- [ ] **Step 5: Commit** `#89 M1: raise multi-line marker section budget to 200`.

### Task 1.5: M1 milestone close

- [ ] **Step 1:** `make test-lua && make test-review` → all green. If `PAIR_SESSION_ID`/`PAIR_TAG` leak into the shell, prefix with `env -u PAIR_SESSION_ID -u PAIR_TAG` (see MEMORY: make-test env leak).
- [ ] **Step 2:** Update `atlas/review-workbench.md` — note multi-line `🤖<…>` support (highlight + resolve) under Modules.
- [ ] **Step 3:** `sdlc milestone-close --issue 89 --milestone M1 --verified '<evidence>'` (fix any Critical/Important from the auto-review before crossing).

---

## Chunk 2: M2 — reconcile engine

**Milestone review boundary.** Closes with `sdlc milestone-close --issue 89 --milestone M2`.

### Task 2.1: `reconcile.classify` (pure)

**Files:**
- Create: `nvim/review/reconcile.lua`
- Test: `nvim/review/reconcile_test.lua`
- Modify: `Makefile.local` (add `nvim -l nvim/review/reconcile_test.lua` to `test-lua`)

- [ ] **Step 1: Failing test** in `nvim/review/reconcile_test.lua`:

```lua
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconcile = dofile(here .. 'reconcile.lua')

-- clean vs conflict: 'kept' still exists in v1 → clean; 'gone' was edited → conflict.
do
  local v1 = 'alpha kept gamma'
  local recs = {
    { old = 'kept', occurrence = 1, new = 'KEPT', explain = 'a' },
    { old = 'gone', occurrence = 1, new = 'GONE', explain = 'b' },
  }
  local r = reconcile.classify(recs, v1)
  assert(#r.clean == 1 and r.clean[1].old == 'kept', 'kept is clean')
  assert(#r.conflicts == 1 and r.conflicts[1].old == 'gone', 'gone is a conflict')
end
```

- [ ] **Step 2: Run** `nvim -l nvim/review/reconcile_test.lua` → FAIL (file missing).

- [ ] **Step 3: Implement** `classify` in `reconcile.lua`:

```lua
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconstruct = dofile(here .. 'reconstruct.lua')
local marker_codec = dofile(here .. '../marker_codec.lua')

-- Split records: clean = old still anchors in the live buffer (same test + `or 1`
-- fallback as apply.apply); conflicts = the human changed that span.
function M.classify(records, v1)
  local clean, conflicts = {}, {}
  for _, r in ipairs(records or {}) do
    local off = r.old and r.old ~= '' and reconstruct.nth_offset(v1, r.old, r.occurrence or 1)
    if off then clean[#clean + 1] = r else conflicts[#conflicts + 1] = r end
  end
  return { clean = clean, conflicts = conflicts }
end

return M
```

- [ ] **Step 4: Run** → PASS.

- [ ] **Step 5: Register** the test: add `\tnvim -l nvim/review/reconcile_test.lua` to `test-lua:` in `Makefile.local`.

- [ ] **Step 6: Commit** `#89 M2: reconcile.classify (clean vs conflict, pure)`.

### Task 2.2: `reconcile.conflict_marker` (pure)

**Files:** `nvim/review/reconcile.lua`, `nvim/review/reconcile_test.lua`

- [ ] **Step 1: Failing test** — the marker escapes brackets in both sections and parses back to one marker:

```lua
do
  local s = reconcile.conflict_marker('human [text]', {
    { old = 'a[0]', new = 'b', explain = 'why' },
  })
  -- both sections escaped: no raw unescaped ] before the intended closer
  assert(s:match('^🤖<'), 'starts with quoted marker')
  local markers = dofile(here .. 'markers.lua')
  local parsed = markers.parse_markers(vim.split(s, '\n', { plain = true }))  -- feed as buffer lines
  assert(#parsed == 1, 'exactly one marker parses; got ' .. #parsed)
  assert(parsed[1].quoted.text == 'human [text]', 'quoted round-trips unescaped')
end
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement**:

```lua
-- One conflict marker. Both the <…> body and the [...] intents are esc_quote'd so
-- unbalanced brackets in quoted code can't break the parse (spec §3).
function M.conflict_marker(hunk_text, intents)
  local lines = { '🤖<' .. marker_codec.esc_quote(hunk_text) .. '>[reconcile — agent wanted:' }
  for _, it in ipairs(intents) do
    local why = (it.explain and it.explain ~= '') and (' (why: ' .. it.explain .. ')') or ''
    lines[#lines + 1] = '  • ' .. (it.old or '') .. ' → ' .. (it.new or '') .. why
  end
  local body = table.concat(lines, '\n') .. ']'
  -- escape the [...] section content (everything after the first '[') so brackets
  -- in code intents can't unbalance the section; the <…> body is already escaped.
  return M._escape_user_section(body)
end
```

Implement `_escape_user_section` to `esc_quote` only the text between the first unescaped `[` and the final `]` (leaving the `🤖<…>` prefix and the outer brackets intact). Simpler alternative accepted if it passes the round-trip test: build the intents string first, `esc_quote` it, then wrap: `'🤖<' .. esc(hunk) .. '>[' .. esc(intents_text) .. ']'` — `esc_quote` on the whole inner text keeps the outer `<`/`>`/`[`/`]` structural. Pick whichever the test proves; keep it one function.

- [ ] **Step 4: Run** → PASS (marker round-trips; brackets in intents don't break parse).

- [ ] **Step 5: Commit** `#89 M2: reconcile.conflict_marker (escaped both sections, pure)`.

### Task 2.3: `reconcile.plan_conflicts` (pure, hunks as data)

**Files:** `nvim/review/reconcile.lua`, `nvim/review/reconcile_test.lua`

- [ ] **Step 1: Failing test** — given `v0`, `v1`, and a `vim.diff`-shaped `hunks` tuple list, two conflict records in the same changed hunk coalesce to ONE synthetic record whose `old` is the hunk's v1 text and whose `new` is a reconcile marker mentioning both intents:

```lua
do
  local v0 = 'title\nold para here\ntail'
  local v1 = 'title\nHUMAN para now\ntail'
  local hunks = { { 2, 1, 2, 1 } }  -- v0 line 2 (1 line) ↔ v1 line 2 (1 line)
  local conflicts = {
    { old = 'old', occurrence = 1, new = 'OLD', explain = 'x' },
    { old = 'para here', occurrence = 1, new = 'PARA', explain = 'y' },
  }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  assert(#synth == 1, 'two conflicts in one hunk coalesce; got ' .. #synth)
  assert(synth[1].old == 'HUMAN para now', 'old = v1 hunk text')
  assert(synth[1].new:match('OLD') and synth[1].new:match('PARA'), 'both intents present')
  assert(synth[1].reconcile == true, 'synthetic record is tagged for the body filter')
end

-- Repeated hunk text: the changed hunk's v1 text also appears earlier verbatim →
-- the synthetic record's occurrence must point at the SECOND (changed) copy.
do
  local v0 = 'dup line\nZ\ndup line\ntail'   -- line 2 changes
  local v1 = 'dup line\ndup line\ndup line\ntail'
  local hunks = { { 2, 1, 2, 1 } }           -- v0 line 2 ↔ v1 line 2
  local conflicts = { { old = 'Z', occurrence = 1, new = 'ZED', explain = 'z' } }
  local synth = reconcile.plan_conflicts(conflicts, v0, v1, hunks)
  assert(#synth == 1 and synth[1].old == 'dup line', 'hunk text is the changed line')
  assert(synth[1].occurrence == 2, 'anchors the 2nd occurrence (the changed one), got ' .. tostring(synth[1].occurrence))
end
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** `plan_conflicts`:
  - Build `v0_lines`, `v1_lines` (`vim.split(..., '\n', {plain=true})`). `reconstruct.line_of(v0, off)` gives the 0-based line of an offset (verified export, `reconstruct.lua:26`).
  - For each conflict record, resolve its **v0 line-span**: `s = nth_offset(v0, r.old, r.occurrence or 1)`; `first = line_of(v0, s)`, `last = line_of(v0, s + #r.old - 1)` — a record whose `old` spans lines must match a hunk it touches, not only its first line.
  - Find the hunk whose **v0 range** `[start_a-1, start_a-1+max(count_a,1))` (0-based) **intersects** `[first, last]`. Group conflicts by that hunk index. (A conflict is classified as such because its `old` no longer exists in `v1` — so the human changed those bytes — so its v0 span always intersects some hunk. If, defensively, none intersects, emit a fallback single-line marker at the record's `first` line so the intent is never silently dropped — the exact bug this issue exists to kill.)
  - For each group: `hunk_text = table.concat({v1_lines[start_b .. start_b+max(count_b,1)-1]}, '\n')` (cap at 200 lines; if larger, `hunk_text = string.format('(region changed — %d lines)', count_b)`). Compute the synthetic `old`'s **true positional occurrence** in `v1`: count non-overlapping matches of `hunk_text` in `v1` strictly before this hunk's v1 byte offset, `+1` (do **not** hardcode `1` — an identical hunk text elsewhere in `v1`, or two hunks sharing text, would otherwise anchor the marker at the wrong region). Emit `{ old = hunk_text, occurrence = <computed>, new = M.conflict_marker(hunk_text, intents_of_group), explain = 'reconcile', reconcile = true }`.
  - Return the synthetic records in document order (by hunk v1 offset). The `reconcile = true` tag is what Task 2.5 filters on for the landed-artifact body.

- [ ] **Step 4: Run** → PASS.

- [ ] **Step 5: Commit** `#89 M2: reconcile.plan_conflicts (coalesce by hunk → synthetic records, pure)`.

### Task 2.4: `reconcile.reconcile_round` (glue) + process test

**Files:**
- Modify: `nvim/review/reconcile.lua` (add `reconcile_round`)
- Create: `tests/review-reconcile-test.sh`
- Modify: `Makefile.local` (add to `test-review`)

- [ ] **Step 1: Failing process test** `tests/review-reconcile-test.sh` (model on `tests/review-apply-test.sh`): headless nvim, `dofile reconcile.lua` + `apply.lua`. Cases:
  1. **clean-only**: `v0='a b c'`, buffer `v1='a b c'` but edited elsewhere (`'a b c EXTRA'`), records target `b`→`B`; assert `b` became `B`, no conflict marker.
  2. **conflict**: `v0='a b c'`, buffer edited to `'a X c'` (human changed `b`'s span), record targets `b`→`B`; assert a `🤖<…>[reconcile` marker replaced the `a X c` region and `b`/`B` are mentioned.
  3. **fast path**: `v1==v0`; assert `reconcile_round` == `apply.apply` result (delegate).

- [ ] **Step 2: Run** `bash tests/review-reconcile-test.sh` → FAIL.

- [ ] **Step 3: Implement** `reconcile_round(buf, records, v0)`:

```lua
local apply = dofile(here .. 'apply.lua')

function M.reconcile_round(buf, records, v0)
  local v1 = apply.buf_content(buf)
  local split = M.classify(records, v1)
  if #split.conflicts == 0 then
    return apply.apply(buf, split.clean)  -- all clean (incl. the fast path when v1==v0)
  end
  local ok, hunks = pcall(vim.diff, v0, v1, { result_type = 'indices' })
  if not ok or type(hunks) ~= 'table' then
    return apply.apply(buf, records)  -- vim.diff failure fallback (spec §8): best-effort
  end
  local synth = M.plan_conflicts(split.conflicts, v0, v1, hunks)
  local combined = {}
  for _, r in ipairs(split.clean) do combined[#combined + 1] = r end
  for _, r in ipairs(synth) do combined[#combined + 1] = r end
  local enriched, dropped = apply.apply(buf, combined)
  return enriched, dropped, #synth
end
```

- [ ] **Step 4: Run** → PASS (clean-only, conflict, fast-path all green).

- [ ] **Step 5: Register** in `Makefile.local`: add `\tbash tests/review-reconcile-test.sh` to `test-review:`.

- [ ] **Step 6: Commit** `#89 M2: reconcile.reconcile_round glue (vim.diff + single apply.apply)`.

### Task 2.5: wire `init.lua` — fast/reconcile branch + `v0` base + landed accounting

**Files:**
- Modify: `nvim/review/init.lua` (`on_agent_round` → `apply_round`; add `set_base`, session `base`; reconcile branch; landed accounting)
- Modify: `nvim/review/poke_bodies.lua` (`agent_applied` mentions conflicts) + `nvim/review/poke_bodies_test.lua`
- Test: `tests/review-loop-test.sh` (reconcile end-to-end via the fake agent)

- [ ] **Step 1: Failing test** — extend `tests/review-loop-test.sh`: seed a review, snapshot `v0` (send), edit the buffer concurrently to overlap one record, drive a fake-agent handoff, assert the buffer shows the clean edits applied AND a `🤖<…>[reconcile` marker for the overlap; assert the landed-artifact / poke summary reports `N edit(s), 1 conflict(s)`.

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** in `init.lua`:
  - Add `reconcile = dofile(here .. 'reconcile.lua')`.
  - `function M.set_base(buf, content) if sessions[buf] then sessions[buf].base = content end end`.
  - Rename the current `on_agent_round` body to `M.apply_round(buf, records)`; at its top read `local v0 = (sessions[buf] or {}).base`; branch: `if v0 == nil or apply.buf_content(buf) == v0 then` → today's `apply.apply(buf, records)` path; `else` → `reconcile.reconcile_round(buf, records, v0)`. Keep the existing `before_agent_round`/save/`write_landed`/poke/`after_agent_round` scaffolding; capture `n_conflicts` from the reconcile branch (0 on fast path) for the landed accounting.
  - Landed accounting — **filter by the `reconcile` tag, not by count or by `new` prefix.** `apply.apply` sorts records by offset and drops some (`apply.lua:274`), so `enriched` is neither input-ordered nor 1:1 with input; and a *clean* exact-replacement's enriched `new` is itself a `🤖<old>{new}` marker (`display_new_for_record`, `apply.lua:106`), so a `🤖<`-prefix filter would wrongly drop clean marker-replacements. Because `plan_conflicts` stamps each synthetic record with `reconcile = true` and `apply.apply` copies extra record fields into `enriched` via `vim.tbl_extend('force', {}, it.rec)` (`apply.lua:313`), partition `enriched`:
    ```lua
    local clean_enriched, conflict_enriched = {}, {}
    for _, nr in ipairs(enriched) do
      if nr.reconcile then conflict_enriched[#conflict_enriched+1] = nr
      else clean_enriched[#clean_enriched+1] = nr end
    end
    local n_applied, n_conf = #clean_enriched, #conflict_enriched
    ```
    `summary = string.format('%d edit(s)%s', n_applied, n_conf>0 and string.format(', %d conflict(s)', n_conf) or '')`; `applied = n_applied`; `conflicts = n_conf`; embed only `clean_enriched` in the body (`record.embed_in_body(summary, clean_enriched)`). This makes the counts and the committed body correct even when apply drops a record.
  - `M.on_agent_round(buf, records)` stays the watcher callback but now delegates: for M2 it simply calls `apply_round` (the gate arrives in M3). Keep it a thin wrapper so M3 only edits the wrapper.

- [ ] **Step 4: Implement** `poke_bodies.agent_applied` conflict segment (+ its pure test): append ` (M to reconcile)` when conflicts > 0. Update `poke_bodies_test.lua`.

- [ ] **Step 5: Run** `make test-lua && bash tests/review-loop-test.sh` → PASS.

- [ ] **Step 6: Commit** `#89 M2: init.lua reconcile branch + v0 base + landed accounting`.

### Task 2.6: protocol docs (pair target + ariadne skill note)

**Files:**
- Modify: `workshop/targets/review-protocol.md` (pair)
- Modify: `../ariadne/.claude/skills/xx-fix/SKILL.md` (the "Pair review workbench" / "Shipping"-adjacent section)

- [ ] **Step 1:** In `review-protocol.md`, document the reconcile state: `v0` snapshot at send; on a landed round, if the buffer changed, per-record reconcile — clean records apply, conflicts become `🤖<human hunk>[reconcile — …]` markers; the agent commits the reconciled doc as its round (Option A); landed-artifact gains `conflicts`.
- [ ] **Step 2:** In ariadne `xx-fix` SKILL, add a note under the Pair review-workbench section: a `🤖<…>[reconcile — …]` marker carries the human's current text plus the agent's blocked intents; treat it as an ordinary `🤖[…]` request on the next round — read both, produce a record that replaces the marker with the reconciled text. **Cross-repo:** check ariadne is on `main` and clean-ish before committing; commit only this file (ariadne has unrelated in-flight work — see this session's history).
- [ ] **Step 3: Commit** each in its own repo (pair via normal flow; ariadne with the single-file discipline used for #164). No code test; the docs are the contract restatement (they *derive* from `reconcile.conflict_marker`'s format — ARCH-PURPOSE: the agent consumer must actually know the semantics, not just pair).

### Task 2.7: M2 milestone close

- [ ] **Step 1:** `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (full suite; parley's `parley_harness_golden` is a known pre-existing failure — see MEMORY).
- [ ] **Step 2:** Update `atlas/review-workbench.md` — add `reconcile.lua` to Modules (pure `classify`/`conflict_marker`/`plan_conflicts` + `reconcile_round` glue) and describe the reconcile flow in "The loop".
- [ ] **Step 3:** `sdlc milestone-close --issue 89 --milestone M2 --verified '<evidence>'`.

---

## Chunk 3: M3 — apply-gate UX + durability

**Milestone review boundary.** Closes with `sdlc milestone-close --issue 89 --milestone M3`, then `sdlc close`.

### Task 3.1: `gate.decide_apply` (pure)

**Files:**
- Create: `nvim/review/gate.lua`
- Test: `nvim/review/gate_test.lua`
- Modify: `Makefile.local` (`test-lua`)

- [ ] **Step 1: Failing test** `nvim/review/gate_test.lua` — the five cases:

```lua
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local gate = dofile(here .. 'gate.lua')
assert(gate.decide_apply('x', 'x', true, 'i') == 'apply', 'case1 unchanged → apply')
assert(gate.decide_apply('x', 'y', false, 'i') == 'apply', 'case2 not focused → apply')
assert(gate.decide_apply('x', 'y', true, 'n') == 'apply', 'case3 normal mode → apply')
assert(gate.decide_apply('x', 'y', true, 'i') == 'defer', 'case4 mid-edit → defer')
assert(gate.decide_apply('x', 'y', true, 'V') == 'defer', 'case4 visual → defer')
```

- [ ] **Step 2: Run** `nvim -l nvim/review/gate_test.lua` → FAIL.

- [ ] **Step 3: Implement**:

```lua
local M = {}
-- Pure apply-gate: decide whether a landed agent round applies now or defers.
function M.decide_apply(v0, v1, focused, mode)
  if v1 == v0 then return 'apply' end        -- nothing changed
  if not focused then return 'apply' end      -- human is in another pane
  if mode == 'n' then return 'apply' end      -- on the pane, not editing
  return 'defer'                              -- mid-edit (i/R/v/V/^V/s/…)
end
return M
```

- [ ] **Step 4: Run** → PASS. **Step 5:** add to `test-lua`. **Step 6: Commit** `#89 M3: gate.decide_apply (pure five-case gate)`.

### Task 3.2: gate dispatch in `init.lua`

**Files:** `nvim/review/init.lua`

- [ ] **Step 1: Failing test** — in `tests/review-loop-test.sh` (or window test), set an injected `M.pane_state = function() return {focused=true, mode='i'} end` and a recorder `M.on_defer`, drive a handoff with a changed buffer, assert `apply_round` did NOT run (buffer unchanged) and `on_defer` fired with the records.

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** — make `on_agent_round` the gate:

```lua
M.gate = dofile(here .. 'gate.lua')
function M.on_agent_round(buf, records)
  local v0 = (sessions[buf] or {}).base
  local v1 = apply.buf_content(buf)
  local st = (M.pane_state and M.pane_state(buf)) or { focused = false, mode = 'n' }
  if M.gate.decide_apply(v0, v1, st.focused, st.mode) == 'defer' and M.on_defer then
    M.on_defer(buf, records)
    return
  end
  return M.apply_round(buf, records)
end
```

`pane_state`/`on_defer` are nil in headless apply tests → default `{focused=false}` → always applies (preserves M2 tests). Injected by `review.lua` (Task 3.3).

Note the nil-base edge: `decide_apply(nil, v1, true, 'i')` returns `'defer'` (case-1 `v1==v0` can't fire for a nil base). Benign — a stray first-round handoff mid-edit just defers, and Alt+Return → `apply_round` → nil base → fast path applies it. Acceptable; no special-case needed.

- [ ] **Step 4: Run** → PASS. **Step 5: Commit** `#89 M3: init.lua on_agent_round consults the gate`.

### Task 3.3: pane wiring in `review.lua` — pane_state, defer, save, Alt+Return, winbar

**Files:** `nvim/review.lua`; Test: `tests/review-window-test.sh`

- [ ] **Step 1: Failing tests** in `wdriver.lua`:
  - `pane_state` returns `{focused=<bool>, mode=<string>}` (drive `FocusGained`/`FocusLost` autocmds; assert `focused` flips).
  - After a defer (simulate `on_defer(buf, records)`), the **file on disk holds the buffer's current edits** (save-on-defer) and the winbar shows the ready cue and `awaiting_since` is cleared.
  - With a pending round set, `Alt+Return` calls `apply_round(buf, pending)` (not `finish_human_turn`) and clears the pending slot + winbar; with no pending, `Alt+Return` still calls `finish_human_turn`.
  - `mark_awaiting` calls `set_base` with the saved content (assert `review.set_base` invoked — inject a recorder, or assert the session base equals the saved buffer).

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** in `review.lua` (the pane already owns `mark_awaiting`/`clear_awaiting`/`finish_human_turn`; wire the hooks):
  - Track focus: `local focused = true` + autocmds `FocusGained → focused=true`, `FocusLost → focused=false` (the pane is created focused).
  - `review.pane_state = function(_) return { focused = focused, mode = vim.fn.mode() } end`.
  - Pending slot: `local pending_records = nil`.
  - `review.on_defer = function(buf, records) pending_records = records; review.human_round(buf, 'defer') --[[saves]] ; clear_awaiting(); show_winbar(true) end`. (Uses `review.human_round`'s existing save so save-on-defer reuses one save path — ARCH-DRY.)
  - **Snapshot `v0` in `finish_human_turn`, not `mark_awaiting`.** After `review.human_round(buf, 'updated')` saves (existing line, `review.lua:452`), call `review.set_base(buf, apply.buf_content(buf))` — the just-saved submitted content. Do NOT put `set_base` in `mark_awaiting`: `request_ship` (`review.lua:466`) also calls `mark_awaiting` but does *not* save first, so a base captured there would be the unsaved buffer. (A ship produces no reconcile round, so it needs no base.) `finish_human_turn`'s save-then-`set_base` ordering matches the spec §8 v0 contract.
  - `finish_human_turn` (Alt+Return): at the very top (before the save), `if pending_records then local r = pending_records; pending_records = nil; show_winbar(false); return review.apply_round(buf, r) end` — consume the pending round instead of submitting. (Wire the same guard into the `<M-S-CR>` menu handler: pending → apply, else open menu.)
  - `show_winbar(on)`: `vim.wo.winbar = on and '%#WarningMsg#✨ agent results ready · ⌥⏎ to apply' or ''`. Clear it in `after_agent_round` and on consume.
  - `VimLeave` (extend the existing teardown autocmd, `review.lua:554`): `if vim.api.nvim_buf_is_valid(buf) and vim.bo[buf].modified then pcall(review.human_round, buf, 'exit') end` — reuse the exposed `human_round` save; **do not** reference `init.lua`'s file-local `save` (out of scope here).
  - Expose `review.apply_round`, `review.set_base` from `init.lua` (add them to the returned `M`); `review.human_round` is already exposed.

- [ ] **Step 4: Run** `make test-lua && bash tests/review-window-test.sh && bash tests/review-loop-test.sh` → PASS.

- [ ] **Step 5: Commit** `#89 M3: apply-gate pane wiring — pane_state, defer+save, winbar, Alt+Return dispatch, save-on-exit`.

### Task 3.4: statusline flip + winbar polish

**Files:** `nvim/review.lua`; Test: `tests/review-window-test.sh`

- [ ] **Step 1:** While pending (winbar shown), assert the statusline is NOT the awaiting spinner (defer cleared `awaiting_since`) and the winbar carries the cue. Add `grep -q '^pending-winbar$'` / `'^pending-no-spinner$'`.
- [ ] **Step 2: Run** → adjust `refresh_statusline`/`show_winbar` so the two states are coherent (spinner only while `awaiting_since`; winbar only while pending). **Step 3: Commit** `#89 M3: winbar/statusline coherence for the pending state`.

### Task 3.5: M3 close + issue close

- [ ] **Step 1:** `env -u PAIR_SESSION_ID -u PAIR_TAG make test` — full suite green (except the known parley golden).
- [ ] **Step 2:** **Live smoke** (the reconcile UX can't be fully proven headless): in a real pair session, open a review, edit while the agent produces a round, confirm (a) non-overlapping edits apply without a conflict, (b) an overlap surfaces a `🤖<…>[reconcile]` marker, (c) mid-insert defers with the winbar and Alt+Return applies, (d) your edits survive a defer→quit→reopen (durability). Record in `## Log`.
- [ ] **Step 3:** Update `atlas/review-workbench.md` — the apply-gate + winbar + durability under "The review window".
- [ ] **Step 4:** `sdlc milestone-close --issue 89 --milestone M3 --verified '<evidence>'`.
- [ ] **Step 5:** `sdlc actual --issue 89` then `sdlc close --issue 89 --verified '<evidence incl. live smoke>' --actual <measured>`.

---

## Notes on ARCH principles

- **ARCH-PURE:** the reasoning is pure and unit-tested without IO — `gate.decide_apply`, `reconcile.classify`/`conflict_marker`/`plan_conflicts`, `markers.spans_multiline`. The IO seams (`reconcile_round`'s `vim.diff` + `apply.apply`; `init.lua` orchestration; `review.lua` focus/mode/winbar/save) are thin and inject the pure results. `plan_conflicts` deliberately takes `hunks` as data so the hard mapping logic is testable without `vim.diff`.
- **ARCH-DRY:** the whole reconcile is one `apply.apply` call (no second apply path); `classify` reuses `reconstruct.nth_offset` (the same anchor test apply runs); `spans_multiline` derives from `parse_markers` (the one multi-line parser), retiring the per-line `highlight_spans` for rendering; save-on-defer reuses `human_round`'s save.
- **ARCH-PURPOSE:** the protocol docs (Task 2.6) make the *agent* consumer derive the reconcile-marker semantics — the point isn't just that pair renders the marker, it's that the agent reconciles it; leaving the agent side as un-updated docs would be under-delivering the purpose.
