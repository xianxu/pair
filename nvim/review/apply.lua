-- nvim/review/apply.lua — apply records to a buffer as ONE undo-able block,
-- then decorate (issue #66 M1). Integration (needs the buffer API); the locate
-- math is delegated to the pure review.reconstruct.
--
-- Correctness rules (judge findings #1/#2 + milestone review I1/I2):
--   * resolve every old@occurrence against the BASE snapshot up front, then
--     apply BOTTOM-TO-TOP so earlier-in-file edits don't drift later offsets;
--   * decorate LIVE from the ACTUAL ranges edited (apply knows where each `new`
--     landed) — never re-find `new`; new_occurrence is computed only for the
--     commit-body / resume path, with the SAME non-overlapping counting that
--     reconstruct.nth_offset uses, so resume re-anchors consistently;
--   * the whole edit loop runs with `buf` current (nvim_buf_call) so the undo
--     break + undojoins target `buf` even when focus is elsewhere; single undo
--     block: first edit is a fresh change, undojoin only 2..N (E790-safe).
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

-- Index of the match of `needle` starting at `target_off`, counted
-- NON-OVERLAPPING (from = e + 1) to match reconstruct.nth_offset. nil when
-- target_off isn't a non-overlapping boundary (self-overlapping `new` adjacent
-- to identical bytes — rare; the resume anchor degrades, the LIVE decoration is
-- unaffected since it uses the actual edited range).
local function new_occurrence_of(content, needle, target_off)
  if not needle or needle == '' then return nil end
  local from, idx = 1, 0
  while true do
    local s, e = content:find(needle, from, true)
    if not s then return nil end
    idx = idx + 1
    if s == target_off then return idx end
    from = e + 1
  end
end

local function buf_content(buf)
  return table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
end

-- Place a list of {line, end_line, message} decorations (clears review ns first).
local function place(buf, decos)
  vim.api.nvim_buf_clear_namespace(buf, HL, 0, -1)
  local diags = {}
  for _, d in ipairs(decos) do
    vim.api.nvim_buf_set_extmark(buf, HL, d.line, 0, {
      end_row = d.end_line, end_col = 0, hl_eol = true, hl_group = 'DiffChange',
    })
    diags[#diags + 1] = {
      lnum = d.line, end_lnum = d.end_line, col = 0,
      message = d.message, severity = vim.diagnostic.severity.INFO, source = 'review',
    }
  end
  vim.diagnostic.set(DIAG, buf, diags, {})
end

-- Apply `records` to `buf`. Returns (enriched, dropped):
--   enriched — applied records carrying new_occurrence (for commit/resume);
--   dropped  — records that did NOT land, each { rec, reason }. The caller MUST
--              surface dropped (milestone review): a partial review presented as
--              complete is the bug. reason ∈ 'empty old' | 'not found' | 'overlap'.
function M.apply(buf, records)
  local dropped = {}
  if not records or #records == 0 then return {}, dropped end
  local base = buf_content(buf)

  -- resolve old@occurrence → base byte offset (document order, ascending)
  local resolved = {}
  for _, r in ipairs(records) do
    if not r.old or r.old == '' then
      dropped[#dropped + 1] = { rec = r, reason = 'empty old' }
    else
      local off = reconstruct.nth_offset(base, r.old, r.occurrence or 1)
      if off then resolved[#resolved + 1] = { rec = r, base_off = off }
      else dropped[#dropped + 1] = { rec = r, reason = 'not found' } end
    end
  end
  table.sort(resolved, function(a, b) return a.base_off < b.base_off end)

  -- bottom-to-top with base coordinates is correct only for NON-overlapping
  -- records; drop (and report) any whose [base_off, base_off+#old) intersects an
  -- already-accepted neighbor, else the higher edit clobbers bytes the lower one
  -- still addresses by base coords (silent corruption otherwise).
  local items, prev_end = {}, 0
  for _, it in ipairs(resolved) do
    if it.base_off < prev_end then
      dropped[#dropped + 1] = { rec = it.rec, reason = 'overlap' }
    else
      items[#items + 1] = it
      prev_end = it.base_off + #it.rec.old
    end
  end
  if #items == 0 then return {}, dropped end

  -- apply bottom-to-top as one undo block, all with `buf` current (I2)
  vim.api.nvim_buf_call(buf, function()
    for i = #items, 1, -1 do
      local it = items[i]
      local sr, sc = pos_at(base, it.base_off)
      local er, ec = pos_at(base, it.base_off + #it.rec.old)
      if i == #items then
        vim.cmd('silent! let &undolevels = &undolevels') -- fresh undo block
      else
        pcall(vim.cmd, 'undojoin')
      end
      vim.api.nvim_buf_set_text(buf, sr, sc, er, ec, vim.split(it.rec.new, '\n', { plain = true }))
    end
  end)

  -- decorate LIVE from the actual edited ranges (final_off = base_off shifted by
  -- the length deltas of all lower-offset edits); enrich new_occurrence for resume.
  local final = buf_content(buf)
  local shift, enriched, decos = 0, {}, {}
  for _, it in ipairs(items) do
    local final_off = it.base_off + shift
    shift = shift + (#it.rec.new - #it.rec.old)
    local nr = vim.tbl_extend('force', {}, it.rec)
    nr.new_occurrence = new_occurrence_of(final, it.rec.new, final_off)
    enriched[#enriched + 1] = nr
    decos[#decos + 1] = {
      line = reconstruct.line_of(final, final_off),
      end_line = reconstruct.line_of(final, final_off + #it.rec.new),
      message = it.rec.explain or '',
    }
  end
  place(buf, decos)
  return enriched, dropped
end

-- Resume render: locate `records` in `content` by new_occurrence and decorate.
-- Used when re-rendering a past round from its commit body (not the live path).
-- Named `render` (not `decorate`) to avoid colliding with reconstruct.decorate,
-- which is PURE and returns data — this one side-effects (places extmarks).
function M.render(buf, records, content)
  content = content or buf_content(buf)
  local d = reconstruct.decorate(records, content, 'new')
  local decos = {}
  for i, h in ipairs(d.highlights) do
    decos[#decos + 1] = {
      line = h.line, end_line = h.end_line,
      message = (d.diagnostics[i] and d.diagnostics[i].message) or '',
    }
  end
  place(buf, decos)
end

M.HL, M.DIAG = HL, DIAG
return M
