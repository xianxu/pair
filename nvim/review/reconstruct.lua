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

function M.decorate(records, content, which)
  which = which or 'new'
  local highlights, diagnostics = {}, {}
  for _, r in ipairs(records) do
    local anchor, occ
    if which == 'old' then anchor, occ = r.old, r.occurrence
    else anchor, occ = r.new, r.new_occurrence end
    local off = anchor and anchor ~= '' and M.nth_offset(content, anchor, occ or 1)
    if off then
      local lnum, col = M.pos_at(content, off)
      local last, end_col = M.pos_at(content, off + #anchor)
      if not M.is_marker_proposal(anchor) then
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

return M
