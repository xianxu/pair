-- nvim -l nvim/interrupt_test.lua
local dir = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':h')
local M = dofile(dir .. '/interrupt.lua')

local function eq(got, want, what)
  if got ~= want then error(string.format('%s: got %s, want %s', what, vim.inspect(got), vim.inspect(want))) end
end

-- Grok cancels a turn on Ctrl+C; every other agent on ESC.
eq(M.byte_for('grok'), 3, 'grok interrupt byte')
for _, agent in ipairs({ 'claude', 'codex', 'agy', 'muse', 'qoder', 'unknown' }) do
  eq(M.byte_for(agent), 27, agent .. ' interrupt byte')
end

-- The current agent comes from the agent file, so a mid-session switch is
-- followed; a missing or empty file falls back to the start-up agent.
local tmp = vim.fn.tempname()
vim.fn.writefile({ 'grok' }, tmp)
eq(M.current_agent(tmp, 'claude'), 'grok', 'agent file wins')
vim.fn.writefile({ '' }, tmp)
eq(M.current_agent(tmp, 'codex'), 'codex', 'empty file falls back')
eq(M.current_agent(tmp .. '.missing', ''), 'claude', 'missing file and no fallback')
os.remove(tmp)

print('interrupt_test ok')
