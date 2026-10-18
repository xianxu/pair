-- nvim/review/oplock.lua — the operator lock for the reviewed file (pair#661 M2).
-- S1's lock: `FILE.lock` (holder: operator, time: RFC3339). While it exists every
-- agent apply refuses with exit 3, so the operator's turn is never written under.
-- The pane takes it on the first edit of the operator's turn and drops it on send
-- and at exit. It records its pid, so a lock left by a pane that died (a crash,
-- SIGKILL) is reclaimed instead of blocking the agent for good; a lock with no
-- pid or a live owner (parley, another pane) is someone else's and is left alone.
local M = {}
local held = {}

function M.path(file) return file .. '.lock' end

function M.held(file) return held[file] == true end

local function owner(p)
  local f = io.open(p, 'r')
  if not f then return nil end
  local text = f:read('*a'); f:close()
  return tonumber(text:match('\npid: (%d+)'))
end

local function alive(pid) return vim.uv.kill(pid, 0) == 0 end

-- reclaim removes a lock whose recorded owner process is dead. True if the
-- file is now free of a dead pane's lock (or had none).
function M.reclaim(file)
  local p = M.path(file)
  if not vim.uv.fs_stat(p) then return true end
  local pid = owner(p)
  if pid and pid ~= vim.fn.getpid() and not alive(pid) then
    os.remove(p)
    return true
  end
  return false
end

function M.take(file, now)
  if held[file] then return true end
  local p = M.path(file)
  if vim.uv.fs_stat(p) then
    if owner(p) == vim.fn.getpid() then held[file] = true; return true end
    if not M.reclaim(file) then return false end -- someone else's lock: leave it
  end
  local f = io.open(p, 'w')
  if not f then return false end
  f:write('holder: operator\ntime: ' .. os.date('!%Y-%m-%dT%H:%M:%SZ', now or os.time()) .. '\npid: ' .. vim.fn.getpid() .. '\n')
  f:close()
  held[file] = true
  return true
end

function M.drop(file)
  if not held[file] then return end
  os.remove(M.path(file))
  held[file] = nil
end

return M
