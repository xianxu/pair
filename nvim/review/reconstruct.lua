-- nvim/review/reconstruct.lua — pure: records + content → decoration inputs
-- (issue #66 M1). Returns 0-based line ranges + explains; no vim API.
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

-- 0-based line containing the given 1-based byte offset.
function M.line_of(content, byte_offset)
  local n = 0
  for i = 1, byte_offset - 1 do
    if content:sub(i, i) == '\n' then n = n + 1 end
  end
  return n
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
      local lnum = M.line_of(content, off)
      local last = M.line_of(content, off + #anchor)
      highlights[#highlights + 1] = { line = lnum, end_line = last }
      diagnostics[#diagnostics + 1] = { lnum = lnum, end_lnum = last, message = r.explain or '' }
    end
  end
  return { highlights = highlights, diagnostics = diagnostics }
end

return M
