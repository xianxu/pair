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

return M
