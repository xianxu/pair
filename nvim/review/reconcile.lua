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
    local anchor_old, anchor_at, new
    if h and h[4] and h[4] >= 1 then
      local sb, cb = h[3], h[4]
      local capped = math.min(cb, MAX_HUNK_LINES)
      local slice = {}
      for i = sb, sb + capped - 1 do slice[#slice + 1] = v1_lines[i] end
      local hunk_text = table.concat(slice, '\n')
      anchor_at = v1_starts[sb] or 1
      if cb > MAX_HUNK_LINES then
        -- huge hunk: don't quote (or discard) it — prepend a marker to the first
        -- line and keep the human's text intact.
        anchor_old = v1_lines[sb] or ''
        new = M.conflict_marker(string.format('(region changed — %d lines)', cb), g.intents)
          .. '\n' .. anchor_old
      else
        anchor_old = hunk_text
        new = M.conflict_marker(hunk_text, g.intents) -- replace the hunk with the wrapped marker
      end
    else
      -- deletion (cb==0) or no hunk: anchor on a nearby kept v1 line, append a
      -- marker (empty quoted body) so the intent is never silently dropped.
      local idx = h and math.max(math.min(h[3] - 1, #v1_lines), 1) or 1
      anchor_old = v1_lines[idx] or ''
      anchor_at = v1_starts[idx] or 1
      new = anchor_old .. '\n' .. M.conflict_marker('', g.intents)
    end
    if anchor_old ~= '' then
      synth[#synth + 1] = {
        old = anchor_old,
        occurrence = occurrence_at(v1, anchor_old, anchor_at),
        new = new,
        explain = 'reconcile',
        reconcile = true,
      }
    end
  end
  return synth
end

return M
