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
  local submit = M.new(
    function(body) calls[#calls + 1] = 'append:' .. body; return true end,
    function(body, no_submit) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(no_submit) end,
    function(message) calls[#calls + 1] = 'notify:' .. message end)
  ok(submit.submit_operator_text('full', 'clean', false), 'authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full|send:clean:false', 'durable append precedes authored send')
  calls = {}
  ok(submit.send_generated_prompt('control'), 'generated send succeeds')
  ok(table.concat(calls, '|') == 'send:control:nil', 'generated prompt bypasses Pair log')
end

do
  local calls = {}
  local submit = M.new(
    function() calls[#calls + 1] = 'append'; return false, 'disk full' end,
    function() calls[#calls + 1] = 'send' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end)
  ok(not submit.submit_operator_text('full', 'clean'), 'append failure fails closed')
  ok(table.concat(calls, '|') == 'append|notify:Pair log append failed — disk full', 'append failure notifies without send')
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
