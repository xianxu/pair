-- Headless integration driver loaded after the real nvim/init.lua.
local case = assert(os.getenv('PAIR_TEST_TX_CASE'))
local pair = assert(os.getenv('PAIR_TEST_BIN'))
local log_path = assert(os.getenv('PAIR_LOG_PATH'))
local fail_kind = case
local failed = false
local state = { focus = 'draft', composer = '', dispatches = {} }

_G.PairTestSessionLogAppend = function(body, id)
  local out = vim.fn.system({ pair, 'session-log', 'append', '--append-id', id }, body)
  return vim.v.shell_error == 0, out
end

_G.PairTestSessionLogCommit = function(id)
  if case == 'commit' and not failed then failed = true; return false, 'commit failure' end
  local out = vim.fn.system({ pair, 'session-log', 'commit', '--append-id', id })
  return vim.v.shell_error == 0, out
end

_G.PairTestZellijExecutor = function(label, argv)
  local kind = assert(label:match('draft%.send%.(.+)$'))
  if kind == fail_kind and not failed then failed = true; return { code = 17 } end
  if kind == 'focus-agent' then
    state.focus = 'agent'
  elseif kind == 'focus-draft' then
    state.focus = 'draft'
  elseif kind == 'write-body' then
    assert(state.focus == 'agent')
    state.composer = state.composer .. argv[4]
  elseif kind == 'submit' then
    assert(state.focus == 'agent')
    state.dispatches[#state.dispatches + 1] = state.composer
    state.composer = ''
  end
  return { code = 0 }
end

local notifications = {}
local original_notify = vim.notify
vim.notify = function(message) notifications[#notifications + 1] = tostring(message) end

local first = _G.submit_operator_text('BODY', 'BODY')
if case == 'focus-agent' or case == 'submit' then
  assert(not first)
  assert(_G.submit_operator_text('BODY', 'BODY'), table.concat(notifications, '\n'))
elseif case == 'write-body' then
  assert(not first)
  assert(not _G.submit_operator_text('BODY', 'BODY'))
elseif case == 'focus-draft' or case == 'commit' then
  assert(first)
else
  error('unknown case ' .. case)
end

if case == 'commit' then
  assert(_G.submit_operator_text('NEXT', 'NEXT'))
end

local file = assert(io.open(log_path, 'rb'))
local raw = file:read('*a')
file:close()
local pairlog = dofile((debug.getinfo(1, 'S').source:match('@?(.*/)') or './') .. 'pairlog.lua')
local entries = assert(pairlog.parse(raw))
local facts = {}
for _, entry in ipairs(entries) do
  if entry.state == 'submitted' then facts[#facts + 1] = entry.body end
end
table.sort(facts)

if case == 'write-body' then
  assert(#state.dispatches == 0 and #facts == 0 and state.composer == '')
elseif case == 'commit' then
  assert(#state.dispatches == 2 and state.dispatches[1] == 'BODY' and state.dispatches[2] == 'NEXT')
  assert(#facts == 2 and facts[1] == 'BODY' and facts[2] == 'NEXT')
else
  assert(#state.dispatches == 1 and state.dispatches[1] == 'BODY')
  assert(#facts == 1 and facts[1] == 'BODY')
end

vim.notify = original_notify
vim.cmd('qall!')
