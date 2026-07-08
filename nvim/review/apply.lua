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
local markers = dofile(here .. 'markers.lua')
local wrap = dofile(here .. 'wrap.lua')

local HL = vim.api.nvim_create_namespace('review')
local DEF_HL = vim.api.nvim_create_namespace('review_define')
local DIAG = vim.api.nvim_create_namespace('review_diag')

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

local function exact_replacement_marker(rec)
  if type(rec.new) ~= 'string' or rec.new:find('\n', 1, true) then return nil end
  local parsed = markers.parse_markers({ rec.new })
  if #parsed ~= 1 then return nil end
  local marker = parsed[1]
  if marker.line ~= 0 or marker.col ~= 0 or marker.raw ~= rec.new then return nil end
  if not marker.quoted or marker.quoted.text ~= rec.old then return nil end
  if marker.strike or #marker.sections ~= 1 then return nil end
  local section = marker.sections[1]
  if section.type ~= 'agent' then return nil end
  return section.text
end

local function is_word_byte(s, idx)
  if idx < 1 or idx > #s then return false end
  return s:sub(idx, idx):match('[%w_]') ~= nil
end

local function shrink_replacement_span(old, new)
  if old == new or old == '' then return nil end

  local start = 1
  local old_len, new_len = #old, #new
  while start <= old_len and start <= new_len and old:sub(start, start) == new:sub(start, start) do
    start = start + 1
  end

  local old_end, new_end = old_len, new_len
  while old_end >= start and new_end >= start and old:sub(old_end, old_end) == new:sub(new_end, new_end) do
    old_end = old_end - 1
    new_end = new_end - 1
  end

  -- A one-character case fix is technically the minimal byte delta, but it is a
  -- poor review unit. Expand changes inside words to whole-token markers.
  while start > 1 and (is_word_byte(old, start - 1) or is_word_byte(new, start - 1)) do
    start = start - 1
  end
  while old_end < old_len and new_end < new_len
      and (is_word_byte(old, old_end + 1) or is_word_byte(new, new_end + 1)) do
    old_end = old_end + 1
    new_end = new_end + 1
  end

  local display_old = old:sub(start, old_end)
  local display_new = new:sub(start, new_end)
  if display_old == '' or display_new == '' then return nil end
  return {
    prefix = new:sub(1, start - 1),
    old = display_old,
    new = display_new,
    suffix = new:sub(new_end + 1),
  }
end

local function display_new_for_record(rec)
  local replacement = exact_replacement_marker(rec)
  if not replacement then return rec.new end
  local span = shrink_replacement_span(rec.old, replacement)
  if not span then return rec.new end
  return span.prefix
    .. '🤖<' .. markers.esc_quote(span.old) .. '>{' .. markers.esc_quote(span.new) .. '}'
    .. span.suffix
end

-- Clear both review decoration layers (shared by place + apply_snapshot, so
-- "cleared" means the same everywhere). The HL extmark ns is cleared; the
-- diagnostic layer is replaced via vim.diagnostic.set (empty here).
local function clear(buf)
  vim.api.nvim_buf_clear_namespace(buf, HL, 0, -1)
  vim.api.nvim_buf_clear_namespace(buf, DEF_HL, 0, -1)
  vim.diagnostic.set(DIAG, buf, {}, {})
end

-- Highlight the exact inserted/new span. Marker-rendered proposals already show
-- the delta inline, and empty deletions have no new bytes to highlight.
local function hl_extmark(buf, h, ns)
  if h.no_highlight then return end
  ns = ns or HL
  local last = math.max(0, vim.api.nvim_buf_line_count(buf) - 1)
  local sl = math.max(0, math.min(h.line or 0, last))
  local el = math.max(sl, math.min(h.end_line or sl, last))
  local sc = h.col or 0
  local ec = h.end_col
  local sl_text = vim.api.nvim_buf_get_lines(buf, sl, sl + 1, false)[1] or ''
  sc = math.max(0, math.min(sc, #sl_text))
  if ec == nil then
    local el_text = vim.api.nvim_buf_get_lines(buf, el, el + 1, false)[1] or ''
    ec = #el_text
  end
  local el_text = vim.api.nvim_buf_get_lines(buf, el, el + 1, false)[1] or ''
  ec = math.max(0, math.min(ec, #el_text))
  if sl == el and sc == ec then return end
  pcall(vim.api.nvim_buf_set_extmark, buf, ns, sl, sc, {
    end_row = el, end_col = ec, hl_group = 'DiffChange',
    right_gravity = true,
    end_right_gravity = false,
  })
end

-- Usable wrap width for the virtual_lines "why": window text columns minus the
-- gutter (textoff) minus a margin for the indent/connector nvim renders. Mirrors
-- parley's diag_wrap_width; falls back to 76 with no window.
local function diag_wrap_width()
  local ok, info = pcall(function()
    return vim.fn.getwininfo(vim.api.nvim_get_current_win())[1]
  end)
  if not ok or type(info) ~= 'table' then return 76 end
  return math.max(30, (info.width or 80) - (info.textoff or 0) - 6)
end

local function diag_of(d)
  return {
    lnum = d.line or d.lnum, end_lnum = d.end_line or d.end_lnum,
    col = d.col or 0, end_col = d.end_col,
    -- hard-wrap the "why" (virtual_lines doesn't soft-wrap — M4a issue 2.1).
    message = wrap.wrap(d.message or '', diag_wrap_width()),
    -- short source: it surfaces as the inline "header" (was 'review' = 6 cols →
    -- 🤖 ≈ 2 cols, M4a issue 2.2).
    severity = vim.diagnostic.severity.INFO, source = d.source or '🤖',
  }
end

-- Place a list of decoration spans + diagnostics (clears first).
local function place(buf, decos)
  clear(buf)
  local diags = {}
  for _, d in ipairs(decos) do
    hl_extmark(buf, d)
    diags[#diags + 1] = diag_of(d)
  end
  vim.diagnostic.set(DIAG, buf, diags, {})
end

-- Snapshot the current decorations for projection: ranged extmarks + the
-- diagnostics (which carry the message). The two layers are independent (after
-- riding, an extmark moves but its diagnostic doesn't), so they're stored —
-- and restored — separately, never paired.
function M.snapshot(buf)
  local hl = {}
  for _, m in ipairs(vim.api.nvim_buf_get_extmarks(buf, HL, 0, -1, { details = true })) do
    local details = m[4] or {}
    hl[#hl + 1] = {
      line = m[2], col = m[3],
      end_line = details.end_row or m[2], end_col = details.end_col or m[3],
    }
  end
  local def_hl = {}
  for _, m in ipairs(vim.api.nvim_buf_get_extmarks(buf, DEF_HL, 0, -1, { details = true })) do
    local details = m[4] or {}
    def_hl[#def_hl + 1] = {
      line = m[2], col = m[3],
      end_line = details.end_row or m[2], end_col = details.end_col or m[3],
    }
  end
  local diags = {}
  for _, d in ipairs(vim.diagnostic.get(buf, { namespace = DIAG })) do
    diags[#diags + 1] = {
      lnum = d.lnum, col = d.col,
      end_lnum = d.end_lnum, end_col = d.end_col,
      message = d.message, source = d.source,
    }
  end
  return { hl = hl, def_hl = def_hl, diags = diags }
end

-- Restore a snapshot (undo/redo coherence). Independent layers; shares clear().
function M.apply_snapshot(buf, snap)
  clear(buf)
  snap = snap or {}
  for _, h in ipairs(snap.hl or {}) do hl_extmark(buf, h) end
  for _, h in ipairs(snap.def_hl or {}) do hl_extmark(buf, h, DEF_HL) end
  local diags = {}
  for _, d in ipairs(snap.diags or {}) do diags[#diags + 1] = diag_of(d) end
  vim.diagnostic.set(DIAG, buf, diags, {})
end

local function definition_message(d)
  local term = d.term or d.id or 'definition'
  local definition = d.definition or '(no definition)'
  return term .. ' — ' .. definition
end

local function is_definition_diag(d)
  return d.source == '📖'
end

function M.place_definitions(buf, definitions)
  vim.api.nvim_buf_clear_namespace(buf, DEF_HL, 0, -1)
  local kept = {}
  for _, d in ipairs(vim.diagnostic.get(buf, { namespace = DIAG })) do
    if not is_definition_diag(d) then
      kept[#kept + 1] = {
        lnum = d.lnum, col = d.col,
        end_lnum = d.end_lnum, end_col = d.end_col,
        message = d.message, source = d.source,
        severity = d.severity,
      }
    end
  end
  for _, d in ipairs(definitions or {}) do
    local deco = {
      line = d.line or d.lnum,
      col = d.col or 0,
      end_line = d.end_line or d.end_lnum or d.line or d.lnum,
      end_col = d.end_col,
      message = definition_message(d),
      source = '📖',
    }
    hl_extmark(buf, deco, DEF_HL)
    kept[#kept + 1] = diag_of(deco)
  end
  vim.diagnostic.set(DIAG, buf, kept, {})
end

local function overlaps_line(first, last, row)
  last = last or first
  return first <= row and row <= last
end

-- Clear the review decoration that covers `row` (0-based), preserving unrelated
-- highlights/diagnostics. Used by Alt+a as "I accept this styled suggestion"
-- when the agent changed text directly and there is no inline 🤖 marker to resolve.
function M.clear_at_line(buf, row)
  local cleared = false
  for _, m in ipairs(vim.api.nvim_buf_get_extmarks(buf, HL, 0, -1, { details = true })) do
    local id, line, details = m[1], m[2], m[4] or {}
    if overlaps_line(line, details.end_row, row) then
      pcall(vim.api.nvim_buf_del_extmark, buf, HL, id)
      cleared = true
    end
  end

  local kept = {}
  for _, d in ipairs(vim.diagnostic.get(buf, { namespace = DIAG })) do
    if overlaps_line(d.lnum, d.end_lnum, row) then
      cleared = true
    else
      kept[#kept + 1] = {
        lnum = d.lnum,
        end_lnum = d.end_lnum,
        col = d.col,
        end_col = d.end_col,
        message = d.message,
        severity = d.severity,
        source = d.source,
      }
    end
  end
  vim.diagnostic.set(DIAG, buf, kept, {})
  return cleared
end

function M.clear_all(buf)
  clear(buf)
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
      it.new_text = display_new_for_record(it.rec)
      items[#items + 1] = it
      prev_end = it.base_off + #it.rec.old
    end
  end
  if #items == 0 then return {}, dropped end

  -- apply bottom-to-top as one undo block, all with `buf` current (I2)
  vim.api.nvim_buf_call(buf, function()
    for i = #items, 1, -1 do
      local it = items[i]
      local sr, sc = reconstruct.pos_at(base, it.base_off)
      local er, ec = reconstruct.pos_at(base, it.base_off + #it.rec.old)
      if i == #items then
        vim.cmd('silent! let &undolevels = &undolevels') -- fresh undo block
      else
        pcall(vim.cmd, 'undojoin')
      end
      vim.api.nvim_buf_set_text(buf, sr, sc, er, ec, vim.split(it.new_text, '\n', { plain = true }))
    end
  end)

  -- decorate LIVE from the actual edited ranges (final_off = base_off shifted by
  -- the length deltas of all lower-offset edits); enrich new_occurrence for resume.
  local final = buf_content(buf)
  local shift, enriched, decos = 0, {}, {}
  for _, it in ipairs(items) do
    local final_off = it.base_off + shift
    shift = shift + (#it.new_text - #it.rec.old)
    local nr = vim.tbl_extend('force', {}, it.rec)
    nr.new = it.new_text
    nr.new_occurrence = new_occurrence_of(final, it.new_text, final_off)
    enriched[#enriched + 1] = nr
    local sr, sc = reconstruct.pos_at(final, final_off)
    local er, ec = reconstruct.pos_at(final, final_off + #it.new_text)
    decos[#decos + 1] = {
      line = sr, col = sc, end_line = er, end_col = ec,
      no_highlight = it.new_text == '' or reconstruct.is_marker_proposal(it.new_text),
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
  clear(buf)
  for _, h in ipairs(d.highlights) do
    hl_extmark(buf, h)
  end
  local diags = {}
  for _, diag in ipairs(d.diagnostics) do
    diags[#diags + 1] = diag_of(diag)
  end
  vim.diagnostic.set(DIAG, buf, diags, {})
end

M.HL, M.DEF_HL, M.DIAG = HL, DEF_HL, DIAG
M.buf_content = buf_content -- shared (init/projection) so the join lives in one place
return M
