-- nvim/review/reconcile.lua — concurrent-edit reconciliation (issue #89 M2).
-- When an agent round lands on a buffer the human edited since the agent reviewed
-- it (v1 ≠ v0), reconcile PER-RECORD: records whose `old` still anchors in the
-- live buffer apply as normal (span-granular, no prose regression); records whose
-- span the human changed become 🤖<…>[reconcile — …] markers placed on the human's
-- changed hunk. The key move: a conflict is modeled as a SYNTHETIC replacement
-- record, so the whole reconcile is one apply.apply call (one snapshot, one undo
-- block). Pure core (classify / conflict_marker / plan_conflicts) + a thin glue
-- (reconcile_round) that calls vim.diff + apply.apply.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconstruct = dofile(here .. 'reconstruct.lua')
local marker_codec = dofile(here .. '../marker_codec.lua')

-- Split records against the live buffer v1. clean = `old` still anchors (the exact
-- reconstruct.nth_offset test + `or 1` fallback apply.apply uses, so classify
-- faithfully predicts what apply lands); conflicts = the human edited that span.
function M.classify(records, v1)
  local clean, conflicts = {}, {}
  for _, r in ipairs(records or {}) do
    local off = r.old and r.old ~= '' and reconstruct.nth_offset(v1, r.old, r.occurrence or 1)
    if off then clean[#clean + 1] = r else conflicts[#conflicts + 1] = r end
  end
  return { clean = clean, conflicts = conflicts }
end

-- One conflict marker: 🤖<human hunk>[reconcile — agent wanted: • old → new (why …)].
-- BOTH the <…> body and every variable field inside [...] (old/new/explain) run
-- through esc_quote (escapes <>[]{}\), so brackets in quoted code can never break
-- the marker's parse — the only unescaped `]` is the closer. Structural text
-- (the bullets, arrows, "why:") carries no marker delimiters. Pure.
function M.conflict_marker(hunk_text, intents)
  local esc = marker_codec.esc_quote
  local lines = { '🤖<' .. esc(hunk_text) .. '>[reconcile — agent wanted:' }
  for _, it in ipairs(intents or {}) do
    local why = (it.explain and it.explain ~= '') and (' (why: ' .. esc(it.explain) .. ')') or ''
    lines[#lines + 1] = '  • ' .. esc(it.old or '') .. ' → ' .. esc(it.new or '') .. why
  end
  return table.concat(lines, '\n') .. ']'
end

-- Glue (the only vim-touching function): reconcile an agent round onto the live
-- buffer. Clean records apply as-is; conflicts (located via vim.diff hunks) become
-- synthetic replacement records, and the WHOLE reconcile is ONE apply.apply call
-- (one snapshot, one undo block, apply.apply unchanged). Returns
-- (enriched, dropped, n_conflicts). apply is lazy-loaded so the pure module + its
-- test don't pull the buffer-manipulation module at load.
local _apply
local function apply_mod()
  _apply = _apply or dofile(here .. 'apply.lua')
  return _apply
end

function M.reconcile_round(buf, records, v0)
  local apply = apply_mod()
  local v1 = apply.buf_content(buf)
  local split = M.classify(records, v1)
  if #split.conflicts == 0 then
    local enriched, dropped = apply.apply(buf, split.clean)
    return enriched, dropped, 0
  end
  local ok, hunks = pcall(vim.diff, v0 or '', v1, { result_type = 'indices' })
  if not ok or type(hunks) ~= 'table' then
    local enriched, dropped = apply.apply(buf, records) -- vim.diff failure fallback (spec §8)
    return enriched, dropped, 0
  end
  local synth = M.plan_conflicts(split.conflicts, v0, v1, hunks)
  local combined = {}
  for _, r in ipairs(split.clean) do combined[#combined + 1] = r end
  for _, r in ipairs(synth) do combined[#combined + 1] = r end
  local enriched, dropped = apply.apply(buf, combined)
  return enriched, dropped, #synth
end

local function split_lines(s)
  local out, from = {}, 1
  while true do
    local nl = s:find('\n', from, true)
    if not nl then out[#out + 1] = s:sub(from); break end
    out[#out + 1] = s:sub(from, nl - 1); from = nl + 1
  end
  return out
end

-- Positional occurrence index of `needle` at 1-based byte offset `at` in `hay`:
-- non-overlapping count of matches strictly before `at`, +1. So a synthetic
-- record anchors the SPECIFIC v1 hunk region even when its text repeats elsewhere.
local function occurrence_at(hay, needle, at)
  local n, from = 0, 1
  while needle ~= '' do
    local s, e = hay:find(needle, from, true)
    if not s or s >= at then break end
    n = n + 1; from = e + 1
  end
  return n + 1
end

local MAX_HUNK_LINES = 200

-- Nearest NON-EMPTY line to `start_idx` (1-based), searching backward first (so a
-- deletion's marker lands on the preceding kept line, where the content was), then
-- forward. Returns the index, or nil when every line is empty (degenerate blank doc).
local function nearest_nonempty(lines, start_idx)
  local n = #lines
  start_idx = math.max(1, math.min(start_idx or 1, n))
  for d = 0, n do
    local b = start_idx - d
    if b >= 1 and lines[b] ~= '' then return b end
    local f = start_idx + d
    if f <= n and lines[f] ~= '' then return f end
  end
  return nil
end

-- Turn conflict records into SYNTHETIC replacement records (one per changed hunk),
-- ready to feed apply.apply alongside the clean records. `hunks` is the vim.diff
-- `result_type='indices'` output {start_a,count_a,start_b,count_b} (1-based),
-- passed as DATA so this stays pure/testable without invoking vim.diff.
function M.plan_conflicts(conflicts, v0, v1, hunks)
  local v1_lines = split_lines(v1)
  local v1_starts, off = {}, 1
  for i, line in ipairs(v1_lines) do v1_starts[i] = off; off = off + #line + 1 end

  -- group conflicts by the hunk their v0 line-span intersects (fallback: own group)
  local groups, order = {}, {}
  for _, r in ipairs(conflicts or {}) do
    local s0 = reconstruct.nth_offset(v0, r.old, r.occurrence or 1)
    local hi
    if s0 then
      local first = reconstruct.line_of(v0, s0)
      local last = reconstruct.line_of(v0, s0 + math.max(#r.old, 1) - 1)
      for i, h in ipairs(hunks or {}) do
        local a0 = h[1] - 1
        local a1 = a0 + math.max(h[2], 1) - 1 -- 0-based v0 line range of the hunk
        if not (last < a0 or first > a1) then hi = i; break end -- intersects
      end
    end
    local key = hi and ('h' .. hi) or ('f' .. (#order + 1)) -- no hunk → own fallback group
    if not groups[key] then
      groups[key] = { hunk = hi and hunks[hi] or nil, intents = {} }
      order[#order + 1] = key
    end
    table.insert(groups[key].intents, r)
  end

  local synth = {}
  for _, key in ipairs(order) do
    local g = groups[key]
    local h = g.hunk
    local sb = h and h[3] or 1     -- v1 start line (1-based)
    local cb = h and (h[4] or 0) or 0 -- v1 line count
    local anchor_old, anchor_at, new

    -- Primary: a non-empty changed hunk → REPLACE the human's hunk text with the
    -- wrapped marker (this is the in-place reconcile placement).
    if cb >= 1 and cb <= MAX_HUNK_LINES then
      local slice = {}
      for i = sb, sb + cb - 1 do slice[#slice + 1] = v1_lines[i] end
      local hunk_text = table.concat(slice, '\n')
      if hunk_text ~= '' then
        anchor_old, anchor_at = hunk_text, v1_starts[sb] or 1
        new = M.conflict_marker(hunk_text, g.intents)
      end
    end

    -- Fallback: deletion (cb==0), blank hunk, huge hunk, or no hunk. APPEND the
    -- marker onto the nearest NON-EMPTY v1 line so a conflict is NEVER silently
    -- dropped (the whole point of the issue). Huge hunks reference their size
    -- instead of quoting the whole region.
    if not new then
      local idx = nearest_nonempty(v1_lines, sb)
      local quoted = (cb > MAX_HUNK_LINES) and string.format('(region changed — %d lines)', cb) or ''
      if idx then
        anchor_old, anchor_at = v1_lines[idx], v1_starts[idx]
        new = anchor_old .. '\n' .. M.conflict_marker(quoted, g.intents)
      else
        -- degenerate: v1 is entirely blank. Emit with an empty `old` so apply.apply
        -- drops it as 'empty old' — COUNTED + WARNed, never silent.
        anchor_old, anchor_at = '', 1
        new = M.conflict_marker(quoted, g.intents)
      end
    end

    synth[#synth + 1] = {
      old = anchor_old,
      occurrence = occurrence_at(v1, anchor_old, anchor_at),
      new = new,
      explain = 'reconcile',
      reconcile = true,
    }
  end
  return synth
end

return M
