-- nvim/review/apply.lua — the review decoration layer (issue #66 M1): highlight
-- and diagnose a round's spans, snapshot and restore those decorations for
-- projection, render a past round on resume, and place definition footnotes.
-- It never edits buffer text: an agent round reaches the document through
-- `pair review apply`, and the pane only reloads it (pair#664). The locate math
-- is delegated to the pure review.reconstruct.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local reconstruct = dofile(here .. 'reconstruct.lua')
local wrap = dofile(here .. 'wrap.lua')

local HL = vim.api.nvim_create_namespace('review')
local DEF_HL = vim.api.nvim_create_namespace('review_define')
local DIAG = vim.api.nvim_create_namespace('review_diag')

local function buf_content(buf)
  return table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
end

-- Clear both review decoration layers (shared by render + apply_snapshot, so
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
