-- A confirmed lifecycle command (quit, detach, restart, agent restart) either
-- ends or changes this session, or it failed, and a failure must be seen
-- (#284). Pair's Alt+n under Couch confirmed "Reload pair?", ran a
-- `pair restart` that refused, and dropped the refusal: the key read as dead.
local M = {}

-- run executes argv through deps.system and, on a non-zero exit, reports the
-- command's own output -- or, when it printed nothing, the command and its exit
-- status -- through deps.notify at ERROR. deps.status reads the exit status of
-- the last deps.system call. Returns whether the command succeeded.
function M.run(argv, deps)
  local out = deps.system(argv)
  local status = deps.status()
  if status == 0 then return true end
  local text = (out or ''):gsub('%s+$', '')
  if text == '' then
    text = table.concat(argv, ' ') .. ': exit ' .. tostring(status)
  end
  deps.notify(text, vim.log.levels.ERROR)
  return false
end

return M
