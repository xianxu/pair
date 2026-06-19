-- nvim/review/handoff.lua — the ephemeral records handoff file (issue #66 M1).
-- The agent writes it (atomic temp+rename); nvim watches, consumes, unlinks.
-- Its appearance IS the "round ready" signal — both data and signal in one file.
--
-- Detection is a TIMER POLL, not fs_event: macOS FSEvents is flaky/laggy and
-- init.lua's scrollback watcher already polls for that reason. The atomic write
-- guarantees the poll never reads a half-written file.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local record = dofile(here .. 'record.lua')

local function data_dir()
  local xdg = vim.env.XDG_DATA_HOME
  local base = (xdg and xdg ~= '') and xdg or (assert(vim.env.HOME) .. '/.local/share')
  return base .. '/pair'
end

function M.path(tag)
  return data_dir() .. '/review-handoff-' .. tag .. '.json'
end

-- Write records atomically (temp + rename). Used by the agent / fake / tests.
function M.write(tag, records)
  vim.fn.mkdir(data_dir(), 'p')
  local p = M.path(tag)
  local tmp = p .. '.tmp'
  local f = assert(io.open(tmp, 'w'))
  f:write(record.encode(records))
  f:close()
  assert(os.rename(tmp, p))
  return p
end

-- Poll for the handoff; on appearance decode → unlink → cb(records).
-- Returns a stop() function. opts.interval ms (default 100).
function M.watch(tag, cb, opts)
  opts = opts or {}
  local p = M.path(tag)
  local timer = vim.uv.new_timer()
  timer:start(0, opts.interval or 100, vim.schedule_wrap(function()
    if not vim.uv.fs_stat(p) then return end
    local fh = io.open(p, 'r')
    if not fh then return end
    local data = fh:read('*a'); fh:close()
    os.remove(p) -- consume regardless, so a bad handoff never loops forever
    local ok, recs = pcall(record.decode, data)
    if ok and recs then
      cb(recs)
    else
      -- never silent (milestone review): a malformed handoff drops the round
      vim.notify('review: handoff decode failed — round dropped', vim.log.levels.WARN)
    end
  end))
  return function()
    if timer and not timer:is_closing() then timer:stop(); timer:close() end
  end
end

return M
