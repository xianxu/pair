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
    function(id) calls[#calls + 1] = 'commit:' .. id; return true end,
    function(body, no_submit) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(no_submit); return true, no_submit and 'composed' or 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return table.remove(ids, 1) end)
  ok(submit.submit_operator_text('full', 'clean', false), 'authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full:id-a|send:clean:false|commit:id-a', 'prepared append precedes send and submitted commit follows it')
  calls = {}
  ok(submit.submit_operator_text('full', 'clean', false), 'next authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full:id-b|send:clean:false|commit:id-b', 'successful submit consumes append ID')
  calls = {}
  ok(submit.send_generated_prompt('control'), 'generated send succeeds')
  ok(table.concat(calls, '|') == 'send:control:nil', 'generated prompt bypasses Pair log')
end

do
  local calls = {}
  local submit = M.new(
    function() calls[#calls + 1] = 'append'; return true end,
    function() calls[#calls + 1] = 'commit'; return true end,
    function() calls[#calls + 1] = 'send'; return false, 'composed', 'focus-draft exited 17' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'id-composed' end)
  ok(submit.submit_operator_text('full', 'clean', true), 'post-compose refocus failure cannot duplicate staged text')
  ok(table.concat(calls, '|') == 'append|send|notify:Pair input composed with UI warning — focus-draft exited 17', 'post-compose UI failure remains non-evidence and reports warning')
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
    function() calls[#calls + 1] = 'commit'; return true end,
    function() calls[#calls + 1] = 'send'; return true, 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'retry-id' end)
  ok(not submit.submit_operator_text('full', 'clean'), 'append failure fails closed')
  ok(submit.submit_operator_text('full', 'clean'), 'append retry succeeds')
  ok(table.concat(calls, '|') == 'append:retry-id|notify:Pair log append failed — disk full|append:retry-id|send|commit', 'append failure retains stable retry ID')
end

do
  local calls = {}
  local commit_attempts = 0
  local ids = { 'id-a', 'id-b' }
  local submit = M.new(
    function() calls[#calls + 1] = 'append'; return true end,
    function()
      calls[#calls + 1] = 'commit'
      commit_attempts = commit_attempts + 1
      if commit_attempts == 1 then return false, 'marker failure' end
      return true
    end,
    function() calls[#calls + 1] = 'send'; return true, 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return table.remove(ids, 1) end)
  ok(submit.submit_operator_text('full', 'clean'), 'post-send marker failure cannot request a duplicate send')
  ok(table.concat(calls, '|') == 'append|send|commit|notify:Pair log submit marker failed — marker failure', 'post-send marker failure stays fail-closed for evidence')
  calls = {}
  ok(submit.submit_operator_text('next', 'next'), 'next submission performs commit-only recovery first')
  ok(table.concat(calls, '|') == 'commit|append|send|commit', 'commit-only recovery never retransmits the dispatched body')
end

do
  local calls = {}
  local ids = { 'id-a', 'id-b' }
  local attempts = 0
  local submit = M.new(
    function(body, id)
      calls[#calls + 1] = 'append:' .. body .. ':' .. id
      attempts = attempts + 1
      if attempts == 1 then return false, 'indeterminate publication' end
      return true
    end,
    function(id) calls[#calls + 1] = 'commit:' .. id; return true end,
    function(body, no_submit) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(no_submit); return true, no_submit and 'composed' or 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return table.remove(ids, 1) end)
  ok(not submit.submit_operator_text('old', 'old', false), 'indeterminate preparation suppresses send')
  ok(submit.submit_operator_text('edited', 'edited', false), 'edited retry uses a fresh prepared attempt')
  ok(table.concat(calls, '|') == 'append:old:id-a|notify:Pair log append failed — indeterminate publication|append:edited:id-b|send:edited:false|commit:id-b',
    'edited retry cannot submit the prior prepared entry')
end

do
  local calls = {}
  local submit = M.new(
    function(body, id) calls[#calls + 1] = 'append:' .. body .. ':' .. id; return true end,
    function(id) calls[#calls + 1] = 'commit:' .. id; return true end,
    function(body, no_submit) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(no_submit); return true, no_submit and 'composed' or 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'id-compose' end)
  ok(submit.submit_operator_text('compose only', 'compose only', true), 'no-submit still transfers text')
  ok(table.concat(calls, '|') == 'append:compose only:id-compose|send:compose only:true', 'no-submit entry is never committed as submitted evidence')
end

do
  local calls = {}
  local submit = M.new(
    function() calls[#calls + 1] = 'append'; return true end,
    function() calls[#calls + 1] = 'commit'; return true end,
    function() calls[#calls + 1] = 'send'; return false, 'indeterminate', 'write-body exited 17' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'id-failed' end)
  ok(not submit.submit_operator_text('full', 'clean'), 'failed delivery preserves authored draft')
  ok(table.concat(calls, '|') == 'append|send|notify:Pair input dispatch failed — write-body exited 17', 'failed delivery cannot commit submitted evidence')
end

do
  local calls = {}
  local submit = M.new(
    function() calls[#calls + 1] = 'append'; return true end,
    function() calls[#calls + 1] = 'commit'; return true end,
    function() calls[#calls + 1] = 'send'; return false, 'dispatched', 'focus-draft exited 17' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'id-sent' end)
  ok(submit.submit_operator_text('full', 'clean'), 'post-dispatch UI failure cannot request retransmission')
  ok(table.concat(calls, '|') == 'append|send|commit|notify:Pair input dispatched with UI warning — focus-draft exited 17', 'post-dispatch UI failure still commits true evidence')
end

do
  local calls = {}
  local sends = 0
  local submit = M.new(
    function(_, id) calls[#calls + 1] = 'append:' .. id; return true end,
    function(id) calls[#calls + 1] = 'commit:' .. id; return true end,
    function(_, _, resume_phase)
      calls[#calls + 1] = 'send:' .. tostring(resume_phase)
      sends = sends + 1
      if sends == 1 then return false, 'written', 'submit exited 17' end
      return true, 'dispatched'
    end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'id-staged' end)
  ok(not submit.submit_operator_text('full', 'clean'), 'failed submit retains confirmed written phase')
  ok(not submit.submit_operator_text('edited', 'edited'), 'edited body cannot overwrite staged composer text')
  ok(submit.submit_operator_text('full', 'clean'), 'unchanged retry resumes at submit')
  ok(table.concat(calls, '|') == 'append:id-staged|send:start|notify:Pair input dispatch failed — submit exited 17|notify:Pair input dispatch blocked — retry the already staged authored body before editing|append:id-staged|send:written|commit:id-staged',
    'written phase resumes without minting or rewriting')
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
