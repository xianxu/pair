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
    function(body, phase) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(phase); return true, 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return table.remove(ids, 1) end)
  ok(submit.submit_operator_text('full', 'clean'), 'authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full:id-a|send:clean:start|commit:id-a', 'prepared append precedes send and submitted commit follows it')
  calls = {}
  ok(submit.submit_operator_text('full', 'clean'), 'next authored submit succeeds')
  ok(table.concat(calls, '|') == 'append:full:id-b|send:clean:start|commit:id-b', 'successful submit consumes append ID')
  calls = {}
  ok(submit.send_generated_prompt('control'), 'generated send succeeds')
  ok(table.concat(calls, '|') == 'send:control:nil', 'generated prompt bypasses Pair log')
end

-- send_generated_prompt must report a FAILED send, not swallow it.
--
-- It used to discard send_low_level's result and return an unconditional true,
-- which made a failed send indistinguishable from a successful one --
-- :PairDoctor consumed the operator's note on the strength of that true, so a
-- failing `zellij action` (the degraded machine the capture exists for) left a
-- cleared draft and no notification. Nothing here covered it: reverting the fix
-- left this file AND tests/pair-doctor-test.sh green.
do
  local calls = {}
  local submit = M.new(
    function() return true end,
    function() return true end,
    function(body) calls[#calls + 1] = 'send:' .. body; return false, 'start', 'no attached UI' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return 'id-x' end)
  ok(submit.send_generated_prompt('doomed') == false,
    'a failed generated send returns false rather than an unconditional true')
  ok(table.concat(calls, '|'):find('notify:', 1, true) ~= nil,
    'a failed generated send notifies the operator')
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
    function(body, phase) calls[#calls + 1] = 'send:' .. body .. ':' .. tostring(phase); return true, 'dispatched' end,
    function(message) calls[#calls + 1] = 'notify:' .. message end,
    function() return table.remove(ids, 1) end)
  ok(not submit.submit_operator_text('old', 'old'), 'indeterminate preparation suppresses send')
  ok(submit.submit_operator_text('edited', 'edited'), 'edited retry uses a fresh prepared attempt')
  ok(table.concat(calls, '|') == 'append:old:id-a|notify:Pair log append failed — indeterminate publication|append:edited:id-b|send:edited:start|commit:id-b',
    'edited retry cannot submit the prior prepared entry')
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
    function(_, resume_phase)
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

-- Each DOM chord is a fresh transaction, even after an uncertain prior send.
for _, failure in ipairs({ 'append', 'start', 'written', 'indeterminate', 'marker' }) do
  local appends, sends, commits, notifications = {}, {}, {}, {}
  local sequence = 0
  local submit = M.new(
    function(body, id)
      appends[#appends + 1] = { body = body, id = id }
      if failure == 'append' and #appends == 1 then return false, 'disk full' end
      return true
    end,
    function(id)
      commits[#commits + 1] = id
      if failure == 'marker' then return false, 'marker unavailable' end
      return true
    end,
    function(body, phase, target)
      sends[#sends + 1] = { body = body, phase = phase, target = target }
      if #sends == 1 and failure ~= 'append' and failure ~= 'marker' then
        return false, failure, 'delivery failed'
      end
      return true, 'dispatched'
    end,
    function(message) notifications[#notifications + 1] = message end,
    function() sequence = sequence + 1; return 'log-' .. sequence end)
  local first = submit.submit_operator_text('old', 'old manifest', { id = 'bundle-old', target = { pane_id = 7 } })
  ok(first == (failure == 'marker'), failure .. ': only confirmed dispatch reports success')
  ok(submit.submit_operator_text('edited', 'new manifest', { id = 'bundle-new', target = { pane_id = 8 } }),
    failure .. ': fresh attachment sends edited draft')
  ok(#appends == 2 and appends[2].body == 'edited' and appends[2].id ~= appends[1].id,
    failure .. ': fresh append ID contains current authored text')
  local latest = sends[#sends] or {}
  ok(latest.body == 'new manifest' and latest.phase == 'start' and latest.target.pane_id == 8,
    failure .. ': current generated text and target start independent delivery')
  ok(appends[2] and #commits == (failure == 'marker' and 2 or 1) and commits[#commits] == appends[2].id,
    failure .. ': only confirmed sends attempt submitted markers')
  ok(#notifications > 0, failure .. ': failure remains visible')
end

-- A DOM chord with no captured notes still has fresh-send semantics.
for _, failure in ipairs({ 'append', 'start', 'written', 'indeterminate', 'marker' }) do
  local appends, sends, commits = {}, {}, {}
  local submit = M.new(
    function(body, id)
      appends[#appends + 1] = { body = body, id = id }
      if failure == 'append' and #appends == 1 then return false, 'disk full' end
      return true
    end,
    function(id)
      commits[#commits + 1] = id
      if failure == 'marker' then return false, 'marker unavailable' end
      return true
    end,
    function(body, phase, target)
      sends[#sends + 1] = { body = body, phase = phase, target = target }
      if #sends == 1 and failure ~= 'append' and failure ~= 'marker' then
        return false, failure, 'delivery failed'
      end
      return true, 'dispatched'
    end, function() end, function() return 'log-' .. (#appends + 1) end)
  local send = submit.submit_fresh_operator_text
  ok(send('old', 'old prompt') == (failure == 'marker'), failure .. ': fallback reports only confirmed dispatch')
  ok(send('edited', 'new prompt'), failure .. ': fallback sends edited current draft')
  ok(#appends == 2 and appends[2].body == 'edited' and appends[2].id ~= appends[1].id,
    failure .. ': fallback appends fresh current draft')
  local latest = sends[#sends] or {}
  ok(latest.body == 'new prompt' and latest.phase == 'start' and latest.target == nil,
    failure .. ': fallback starts fresh without an attachment target')
  ok(appends[2] and #commits == (failure == 'marker' and 2 or 1) and commits[#commits] == appends[2].id,
    failure .. ': fallback never marks an unconfirmed send')
end

-- An unchanged draft still starts a new send when the chord supplies a new ID.
do
  local sends, ids = 0, {}
  local submit = M.new(function(_, id) ids[#ids + 1] = id; return true end,
    function() return true end,
    function(_, phase) sends = sends + 1; ok(phase == 'start', 'fresh identical body starts delivery'); return true, 'dispatched' end,
    function() end, function() return 'log-' .. (#ids + 1) end)
  ok(submit.submit_operator_text('body', 'manifest', { id = 'a', target = {} }), 'first unchanged draft sends')
  ok(submit.submit_operator_text('body', 'manifest', { id = 'b', target = {} }), 'second unchanged draft sends')
  ok(sends == 2 and ids[1] ~= ids[2], 'unchanged draft is two independent sends')
end

-- Ordinary retry protection and fresh DOM delivery cannot lock each other.
do
  local sends = 0
  local submit = M.new(function() return true end, function() return true end,
    function(_, _, target)
      sends = sends + 1
      if target then return true, 'dispatched' end
      return false, 'indeterminate', 'write uncertain'
    end, function() end, function() return 'log-' .. sends end)
  ok(not submit.submit_operator_text('ordinary', 'ordinary'), 'ordinary send can remain uncertain')
  ok(submit.submit_operator_text('DOM', 'manifest', { id = 'fresh', target = {} }), 'ordinary uncertainty cannot block fresh DOM')
  ok(not submit.submit_operator_text('edited ordinary', 'edited ordinary'), 'DOM send preserves ordinary uncertainty protection')
  ok(sends == 2, 'ordinary uncertainty still prevents another ordinary send')
end

do
  local sends = 0
  local submit = M.new(function() return true end, function() return true end,
    function(_, _, target)
      sends = sends + 1
      if target then return false, 'written', 'submit failed' end
      return true, 'dispatched'
    end, function() end, function() return 'log-' .. sends end)
  ok(not submit.submit_operator_text('DOM', 'manifest', { id = 'failed', target = {} }), 'DOM send can fail after write')
  ok(submit.submit_operator_text('ordinary', 'ordinary'), 'DOM failure cannot freeze ordinary draft')
  ok(sends == 2, 'ordinary send follows failed DOM transaction')
end

do
  local sends = 0
  local submit = M.new(function() return true end, function() return true end,
    function() sends = sends + 1; return true, 'dispatched' end,
    function() end, function() return 'invalid-attachment' end)
  ok(not submit.submit_operator_text('body', 'manifest', { id = 'bundle' }), 'attachment without target fails closed')
  ok(not submit.submit_operator_text('body', 'manifest', { target = {} }), 'attachment without ID fails closed')
  ok(sends == 0, 'malformed attachments cannot fall back to ordinary delivery')
end

-- A DOM-only budget reaches each effect; confirmed delivery survives marker errors.
do
  local operation={check=function()end}
  local order={}
  local submit=M.new(function(_,_,op)
    ok(op==operation,'fresh append receives operation');order[#order+1]='append';return true
  end,function(_,op)
    ok(op==operation,'fresh commit receives operation');order[#order+1]='commit';error('marker deadline')
  end,function(_,_,_,op)
    ok(op==operation,'fresh send receives operation');order[#order+1]='send';return true,'dispatched'
  end,function()end,function()return 'bounded' end)
  local called,result=pcall(submit.submit_fresh_operator_text,'body','text',nil,operation)
  ok(called and result,'marker exception cannot erase confirmed delivery')
  ok(table.concat(order,',')=='append,send,commit','bounded fresh effects keep order')
  operation.check=function()error('submission timed out')end
  local failed,reason=submit.submit_fresh_operator_text('body','text',nil,operation)
  ok(not failed and tostring(reason):find('timed out',1,true),'expired fresh send returns reason')
  ok(#order==3,'expired send performs no effects')
end

local init = table.concat(vim.fn.readfile(here .. 'init.lua'), '\n')
local direct = 0
for _ in init:gmatch('send_to_agent%s*%(') do direct = direct + 1 end
ok(direct == 1, 'only the low-level function declaration names send_to_agent(')
for _, required in ipairs({
  'submit_operator_text%(body, stripped%)',
  'send_generated_prompt%(out%)',
  'send_generated_prompt%(COMPACT_PROMPT%)',
  'send_generated_prompt%(body%)',
}) do
  ok(init:match(required) ~= nil, 'expected routed caller: ' .. required)
end

if fails > 0 then os.exit(1) end
print('all submission.lua tests passed')
