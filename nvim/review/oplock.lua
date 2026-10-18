-- nvim/review/oplock.lua — the operator lock for the reviewed file (pair#661 M2).
-- S1's lock: `FILE.lock` (holder: operator, time: RFC3339). While it exists every
-- agent apply refuses with exit 3, so the operator's turn is never written under.
-- The pane takes it on the first edit of the operator's turn and drops it on send
-- and at exit; it removes only a lock it took itself.
local M = {}
local held = {}

function M.path(file) return file .. '.lock' end

function M.held(file) return held[file] == true end

function M.take(file, now)
  if held[file] then return true end
  local p = M.path(file)
  if vim.uv.fs_stat(p) then return false end -- someone else's lock: leave it
  local f = io.open(p, 'w')
  if not f then return false end
  f:write('holder: operator\ntime: ' .. os.date('!%Y-%m-%dT%H:%M:%SZ', now or os.time()) .. '\n')
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
