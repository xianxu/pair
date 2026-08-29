local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'submission.lua')

local fails = 0
local function ok(value, message)
  if not value then
    io.stderr:write('FAIL ' .. message .. '\n')
    fails = fails + 1
  end
end

do
  local calls = {}
  local ids = { 'id-a', 'id-b' }
  local submit = M.new(
    function(body, id) calls[#calls + 1] = 'append:' .. body .. ':' .. id; return true end,
    function(body, no_submit) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(no_submit) end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return table.remove(ids, 1) end)
  ok(submit.submit_operator_text('full', 'clean', false), 'authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full:id-a|send:clean:false', 'durable append precedes authored send')
  calls = {}
  ok(submit.submit_operator_text('full', 'clean', false), 'next authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full:id-b|send:clean:false', 'successful submit consumes append ID')
  calls = {}
  ok(submit.send_generated_prompt('control'), 'generated send succeeds')
  ok(table.concat(calls, '|') == 'send:control:nil', 'generated prompt bypasses Pair log')
end

do
  local calls = {}
  local attempts = 0
  local submit = M.new(
    function(_, id)
      calls[#calls + 1] = 'append:' .. id
      attempts = attempts + 1
      if attempts == 1 then return false, 'disk full' end
      return true
    end,
    function() calls[#calls + 1] = 'send' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'retry-id' end)
  ok(not submit.submit_operator_text('full', 'clean'), 'append failure fails closed')
  ok(submit.submit_operator_text('full', 'clean'), 'append retry succeeds')
  ok(table.concat(calls, '|') == 'append:retry-id|notify:Pair log append failed — disk full|append:retry-id|send', 'append failure retains stable retry ID')
end

local init = table.concat(vim.fn.readfile(here .. 'init.lua'), '\n')
local direct = 0
for _ in init:gmatch('send_to_agent%s*%(') do direct = direct + 1 end
ok(direct == 1, 'only the low-level function declaration names send_to_agent(')
for _, required in ipairs({
  'submit_operator_text%(body, stripped, no_submit%)',
  'submit_operator_text%(body, stripped%)',
  'send_generated_prompt%(out%)',
  'send_generated_prompt%(COMPACT_PROMPT%)',
  'send_generated_prompt%(body%)',
}) do
  ok(init:match(required) ~= nil, 'expected routed caller: ' .. required)
end

if fails > 0 then os.exit(1) end
print('all submission.lua tests passed')
