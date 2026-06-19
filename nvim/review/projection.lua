-- nvim/review/projection.lua — decoration coherence across undo/redo (#66 M2).
-- Ported from parley's review/projection.lua; draws via review.apply's
-- snapshot/apply_snapshot (M2) instead of parley's skill_render.
--
-- nvim's undo reverts TEXT only; review decorations are drawn once per round and
-- otherwise ride/persist, so after an undo the style goes stale. This records the
-- decoration set per content-state (keyed by a content hash) and, on any text
-- change, PROJECTS the right style onto the current state:
--   - content matches a recorded state (undo/redo landing) → re-render it;
--   - novel forward state (manual edit) → the live decorations keep riding, and
--     we snapshot them so a later undo restores them.
-- `M.decide` is PURE; the watcher/hashing/snapshot-apply are the thin IO seam.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local apply = dofile(here .. 'apply.lua')

local _state = {}    -- buf → { records = {[hash]=snapshot}, order, watching, autocmd }
local _applying = {} -- buffers whose own round-apply is in flight

local MAX_RECORDS = 200

local function bufstate(buf)
  _state[buf] = _state[buf] or { records = {}, order = {} }
  return _state[buf]
end

-- Insert/replace a record, maintaining FIFO order + the cap.
local function put(s, h, snap)
  if s.records[h] == nil then
    table.insert(s.order, h)
    if #s.order > MAX_RECORDS then
      local oldest = table.remove(s.order, 1)
      s.records[oldest] = nil
    end
  end
  s.records[h] = snap
end

local function content(buf)
  return table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), "\n")
end

local function hash(buf)
  return vim.fn.sha256(content(buf))
end

-- PURE: "restore" if we've recorded this exact content before (undo/redo),
-- else "capture" (a novel forward state).
function M.decide(records, h)
  return records[h] ~= nil and "restore" or "capture"
end

function M.set_applying(buf, v)
  _applying[buf] = v or nil
end

-- Record the CURRENT decorations under the current content hash (round end).
function M.record(buf)
  put(bufstate(buf), hash(buf), apply.snapshot(buf))
end

-- Record an EMPTY decoration set for the pre-round base — so undoing back across
-- the round clears the stale style. GUARD: only the TRUE base (never-seen
-- content) is emptied; a later round's pre-content is a prior round's OUTPUT,
-- already recorded WITH its decorations, so don't clobber it (else undoing one
-- round back would wrongly clear that round's style).
function M.record_empty_for(buf, base_content)
  local s = bufstate(buf)
  local h = vim.fn.sha256(base_content or "")
  if s.records[h] == nil then
    put(s, h, { hl = {}, diags = {} })
  end
end

-- Project the recorded style onto the current content state (the watcher body).
function M.project(buf)
  if _applying[buf] or not vim.api.nvim_buf_is_valid(buf) then
    return
  end
  local s = bufstate(buf)
  local h = hash(buf)
  if M.decide(s.records, h) == "restore" then
    apply.apply_snapshot(buf, s.records[h]) -- undo/redo → re-render the record
  else
    put(s, h, apply.snapshot(buf))          -- novel forward state → snapshot riding decos
  end
end

-- Attach the TextChanged/InsertLeave watcher once per buffer (lazy — only after
-- a round). NOT TextChangedI (would sha256 the buffer per keystroke).
function M.ensure_watch(buf)
  local s = bufstate(buf)
  if s.watching then return end
  s.watching = true
  s.autocmd = vim.api.nvim_create_autocmd({ "TextChanged", "InsertLeave" }, {
    buffer = buf,
    callback = function() M.project(buf) end,
  })
end

-- Forget a buffer's projection state (tests / teardown); removes the watcher so
-- a surviving buffer doesn't double-attach next round.
function M.reset(buf)
  local s = _state[buf]
  if s and s.autocmd then
    pcall(vim.api.nvim_del_autocmd, s.autocmd)
  end
  _state[buf] = nil
  _applying[buf] = nil
end

return M
