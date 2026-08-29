local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local delivery = dofile(here .. 'draft_send.lua')

local function run_failure(no_submit, failed_kind)
  local calls = {}
  local settled = 0
  local function action(label)
    local kind = label:match('draft%.send%.(.+)$')
    calls[#calls + 1] = kind
    return { code = kind == failed_kind and 17 or 0 }
  end
  local ok, phase, err = delivery.send('body', no_submit, action, function() settled = settled + 1 end)
  return ok, phase, err, calls, settled
end

for _, kind in ipairs({ 'focus-agent', 'write-body', 'submit', 'focus-draft' }) do
  local ok, phase, err = run_failure(false, kind)
  assert(not ok and err:match(kind:gsub('%-', '%%-')), kind .. ' must report failure')
  local want = ({ ['focus-agent'] = 'start', ['write-body'] = 'indeterminate', submit = 'written', ['focus-draft'] = 'dispatched' })[kind]
  assert(phase == want, kind .. ' phase = ' .. tostring(phase) .. ', want ' .. want)
end

for _, kind in ipairs({ 'focus-agent', 'write-body', 'newline', 'focus-draft' }) do
  local ok, phase = run_failure(true, kind)
  local want = ({ ['focus-agent'] = 'start', ['write-body'] = 'indeterminate', newline = 'written', ['focus-draft'] = 'composed' })[kind]
  assert(not ok and phase == want, kind .. ' compose phase = ' .. tostring(phase) .. ', want ' .. want)
end

local ok, phase = delivery.send('body', false, function() return { code = 0 } end, function() end)
assert(ok and phase == 'dispatched', 'successful submit confirms dispatch')
ok, phase = delivery.send('body', true, function() return { code = 0 } end, function() end)
assert(ok and phase == 'composed', 'successful compose-only transfer is not dispatch')

local settled = 0
delivery.send(string.rep('x', 201), false, function() return { code = 0 } end, function() settled = settled + 1 end)
assert(settled == 1, 'large body settles exactly once between write and submit')

local function stateful_zellij(fail_once)
  local state = { focus = 'draft', composer = '', dispatches = {}, calls = {} }
  local failed = false
  local function action(label, argv)
    local kind = label:match('draft%.send%.(.+)$')
    state.calls[#state.calls + 1] = kind
    if kind == fail_once and not failed then failed = true; return { code = 17 } end
    if kind == 'focus-agent' then state.focus = 'agent'
    elseif kind == 'focus-draft' then state.focus = 'draft'
    elseif kind == 'write-body' then assert(state.focus == 'agent'); state.composer = state.composer .. argv[4]
    elseif kind == 'submit' then assert(state.focus == 'agent'); state.dispatches[#state.dispatches + 1] = state.composer; state.composer = ''
    elseif kind == 'newline' then assert(state.focus == 'agent'); state.composer = state.composer .. '\n'
    end
    return { code = 0 }
  end
  return state, action
end

do
  local state, action = stateful_zellij('submit')
  local first_ok, first_phase = delivery.send('body', false, action, function() end)
  assert(not first_ok and first_phase == 'written' and state.composer == 'body')
  local retry_ok, retry_phase = delivery.send('body', false, action, function() end, first_phase)
  assert(retry_ok and retry_phase == 'dispatched')
  assert(#state.dispatches == 1 and state.dispatches[1] == 'body' and state.composer == '', 'submit retry must not rewrite staged body')
end

do
  local state, action = stateful_zellij('newline')
  local first_ok, first_phase = delivery.send('body', true, action, function() end)
  assert(not first_ok and first_phase == 'written' and state.composer == 'body')
  local retry_ok, retry_phase = delivery.send('body', true, action, function() end, first_phase)
  assert(retry_ok and retry_phase == 'composed' and state.composer == 'body\n', 'compose retry must not rewrite staged body')
end

local blocked_ok, blocked_phase = delivery.send('body', false, function() error('must not act') end, function() end, 'indeterminate')
assert(not blocked_ok and blocked_phase == 'indeterminate', 'indeterminate body write blocks automatic retry')

print('draft_send_test ok')
