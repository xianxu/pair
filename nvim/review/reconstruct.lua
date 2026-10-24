-- nvim/review/reconstruct.lua — pure: records + content → decoration inputs
-- (issue #66 M1). Returns 0-based spans + explains; no vim API.
--
-- This is the RESUME / from-commit path: review.apply produces *live*
-- decorations from the exact ranges it just edited, while this module rebuilds
-- them from records extracted out of a frozen commit body.
--
--   which='old' → locate each record by `occurrence`-th match of `old`
--   which='new' → locate by `new_occurrence`-th match of `new`
-- Never cross them (judge finding #1).
local M = {}

-- byte offset of the occurrence-th plain (non-pattern) match of `needle`.
-- Exported so review.apply reuses the same locate logic.
function M.nth_offset(haystack, needle, occurrence)
  local from, found = 1, nil
  for _ = 1, (occurrence or 1) do
    local s, e = haystack:find(needle, from, true)
    if not s then return nil end
    found, from = s, e + 1
  end
  return found
end

-- Build 1-based byte offsets for each line's first byte. Accepts either the
-- string content shape used by reconstruct/reconcile or the line-array shape
-- used by markers.
function M.line_starts(content_or_lines)
  local starts = {}
  if type(content_or_lines) == 'table' then
    local off = 1
    for i, line in ipairs(content_or_lines) do
      starts[i] = off
      off = off + #line + 1
    end
    return starts
  end

  local content = content_or_lines or ''
  starts[1] = 1
  for i = 1, #content do
    if content:sub(i, i) == '\n' then starts[#starts + 1] = i + 1 end
  end
  return starts
end

-- 1-based byte offset → (row, col) both 0-based.
function M.pos_of(line_starts, byte_offset)
  if #line_starts == 0 then return 0, (byte_offset or 1) - 1 end
  local lo, hi = 1, #line_starts
  while lo < hi do
    local mid = math.floor((lo + hi) / 2) + 1
    if line_starts[mid] <= byte_offset then lo = mid else hi = mid - 1 end
  end
  return lo - 1, byte_offset - line_starts[lo]
end

-- Positional occurrence index of `needle` at 1-based byte offset `at` in `hay`:
-- non-overlapping count of matches strictly before `at`, +1.
function M.occurrence_at(haystack, needle, at)
  local n, from = 0, 1
  while needle ~= '' do
    local s, e = haystack:find(needle, from, true)
    if not s or s >= at then break end
    n = n + 1; from = e + 1
  end
  return n + 1
end

-- 0-based line containing the given 1-based byte offset.
function M.line_of(content, byte_offset)
  local row = M.pos_of(M.line_starts(content), byte_offset)
  return row
end

-- 1-based byte offset → (row, col) both 0-based.
function M.pos_at(content, byte_offset)
  return M.pos_of(M.line_starts(content), byte_offset)
end

function M.is_marker_proposal(text)
  return type(text) == 'string' and text:match('🤖[<{~]') ~= nil
end

-- One presentation fallback for live rounds and resume records.
function M.presentation(span, round_presentation)
  return span.presentation or round_presentation or 'highlight'
end

function M.decorate(records, content, which, round_presentation)
  which = which or 'new'
  local highlights, diagnostics = {}, {}
  for _, r in ipairs(records) do
    local anchor, occ
    if which == 'old' then anchor, occ = r.old, r.occurrence
    else anchor, occ = r.new, r.new_occurrence end
    local off = anchor and anchor ~= '' and M.nth_offset(content, anchor, occ or 1)
    local presentation = M.presentation(r, round_presentation)
    if off and presentation ~= 'direct' then
      local lnum, col = M.pos_at(content, off)
      local last, end_col = M.pos_at(content, off + #anchor)
      if presentation == 'highlight' and not M.is_marker_proposal(anchor) then
        highlights[#highlights + 1] = { line = lnum, col = col, end_line = last, end_col = end_col }
      end
      diagnostics[#diagnostics + 1] = {
        lnum = lnum, col = col,
        end_lnum = last, end_col = end_col,
        message = r.explain or '',
      }
    end
  end
  return { highlights = highlights, diagnostics = diagnostics }
end

-- Resolve review records against `base` (pair#661). Every record resolves
-- against the base text: occurrence is 1-based and counts plain,
-- non-overlapping byte matches; records sort by offset (ties by input order),
-- and one starting before the previous accepted span ends is an overlap.
-- Returns (kept, bad): kept = { {rec, index, base_off} } in base order;
-- bad = { {index, reason, rec} } by index, reason ∈ 'empty old' | 'not found' |
-- 'overlap'. Couch's Go resolver runs the same testdata/edit-records fixtures.
function M.resolve(base, records)
  local resolved, bad = {}, {}
  for i, r in ipairs(records or {}) do
    if not r.old or r.old == '' then
      bad[#bad + 1] = { index = i, reason = 'empty old', rec = r }
    else
      local off = M.nth_offset(base, r.old, r.occurrence or 1)
      if off then resolved[#resolved + 1] = { rec = r, index = i, base_off = off }
      else bad[#bad + 1] = { index = i, reason = 'not found', rec = r } end
    end
  end
  table.sort(resolved, function(a, b)
    if a.base_off ~= b.base_off then return a.base_off < b.base_off end
    return a.index < b.index
  end)
  local kept, prev_end = {}, 0
  for _, it in ipairs(resolved) do
    if it.base_off < prev_end then
      bad[#bad + 1] = { index = it.index, reason = 'overlap', rec = it.rec }
    else
      kept[#kept + 1] = it
      prev_end = it.base_off + #it.rec.old
    end
  end
  table.sort(bad, function(a, b) return a.index < b.index end)
  return kept, bad
end

-- resolve_text applies a fully resolving batch: (text, {}) or (nil, bad).
function M.resolve_text(base, records)
  local kept, bad = M.resolve(base, records)
  if #bad > 0 then return nil, bad end
  local out, at = {}, 1
  for _, it in ipairs(kept) do
    out[#out + 1] = base:sub(at, it.base_off - 1)
    out[#out + 1] = it.rec.new or ''
    at = it.base_off + #it.rec.old
  end
  out[#out + 1] = base:sub(at)
  return table.concat(out), bad
end

return M
