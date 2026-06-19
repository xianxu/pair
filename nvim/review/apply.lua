-- nvim/review/apply.lua — apply records to a buffer as ONE undo-able block,
-- then decorate (issue #66 M1). Integration (needs the buffer API); the locate
-- math is delegated to the pure review.reconstruct.
--
-- Correctness rules (judge findings #1/#2):
--   * resolve every old@occurrence against the BASE snapshot up front, then
--     apply BOTTOM-TO-TOP so earlier-in-file edits don't drift later offsets;
--   * decorate from the ACTUAL ranges edited (compute each edit's
--     new_occurrence) — never re-find `new` by the old index;
--   * single undo block: first edit is a fresh change, undojoin only 2..N
--     (undojoin before a change / after an undo throws E790).
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconstruct = dofile(here .. 'reconstruct.lua')

local HL = vim.api.nvim_create_namespace('review')
local DIAG = vim.api.nvim_create_namespace('review_diag')

-- 1-based byte offset → (row, col) both 0-based, for the buffer API.
local function pos_at(content, offset)
  local row, last_nl = 0, 0
  for i = 1, offset - 1 do
    if content:sub(i, i) == '\n' then row = row + 1; last_nl = i end
  end
  return row, (offset - 1) - last_nl
end

-- 1-based index of the match of `needle` that starts exactly at `start_off`.
local function occurrence_at(content, needle, start_off)
  local idx, from = 0, 1
  while true do
    local s = content:find(needle, from, true)
    if not s then return idx end
    idx = idx + 1
    if s == start_off then return idx end
    from = s + 1
  end
end

local function buf_content(buf)
  return table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
end

-- Apply `records` to `buf`. Returns the records enriched with new_occurrence
-- (for the commit body / resume path).
function M.apply(buf, records)
  if not records or #records == 0 then return {} end
  local base = buf_content(buf)

  -- resolve old@occurrence → base byte offset; keep document order (ascending)
  local items = {}
  for _, r in ipairs(records) do
    local off = reconstruct.nth_offset(base, r.old or '', r.occurrence or 1)
    if off then items[#items + 1] = { rec = r, base_off = off } end
  end
  if #items == 0 then return {} end
  table.sort(items, function(a, b) return a.base_off < b.base_off end)

  -- apply bottom-to-top (descending offset) as one undo block. The first
  -- applied edit breaks the undo sequence (`:let &ul=&ul`) so the whole round
  -- is a SINGLE block, separate from the buffer's prior history — undoing the
  -- agent round must not also undo the user's earlier keystrokes. Subsequent
  -- edits undojoin into that block (undojoin before the first / after an undo
  -- throws E790, hence the pcall + first-edit guard).
  for i = #items, 1, -1 do
    local it = items[i]
    local sr, sc = pos_at(base, it.base_off)
    local er, ec = pos_at(base, it.base_off + #it.rec.old)
    if i == #items then
      vim.api.nvim_buf_call(buf, function() vim.cmd('silent! let &undolevels = &undolevels') end)
    else
      pcall(vim.cmd, 'undojoin')
    end
    vim.api.nvim_buf_set_text(buf, sr, sc, er, ec, vim.split(it.rec.new, '\n', { plain = true }))
  end

  -- enrich with new_occurrence against the FINAL content (each edit's final
  -- start = base_off shifted by the length deltas of all lower-offset edits)
  local final = buf_content(buf)
  local shift, enriched = 0, {}
  for _, it in ipairs(items) do
    local final_off = it.base_off + shift
    shift = shift + (#it.rec.new - #it.rec.old)
    local nr = vim.tbl_extend('force', {}, it.rec)
    nr.new_occurrence = occurrence_at(final, it.rec.new, final_off)
    enriched[#enriched + 1] = nr
  end

  M.decorate(buf, enriched, final)
  return enriched
end

-- Place extmark highlights + INFO diagnostics for `records` over `content`.
-- Clears the review namespaces first. Exposed so resume can re-render.
function M.decorate(buf, records, content)
  content = content or buf_content(buf)
  local deco = reconstruct.decorate(records, content, 'new')
  vim.api.nvim_buf_clear_namespace(buf, HL, 0, -1)
  for _, h in ipairs(deco.highlights) do
    vim.api.nvim_buf_set_extmark(buf, HL, h.line, 0, {
      end_row = h.end_line, end_col = 0, hl_eol = true, hl_group = 'DiffChange',
    })
  end
  local diags = {}
  for _, d in ipairs(deco.diagnostics) do
    diags[#diags + 1] = {
      lnum = d.lnum, end_lnum = d.end_lnum, col = 0,
      message = d.message, severity = vim.diagnostic.severity.INFO, source = 'review',
    }
  end
  vim.diagnostic.set(DIAG, buf, diags, {})
end

M.HL, M.DIAG = HL, DIAG
return M
