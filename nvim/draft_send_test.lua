local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local delivery = dofile(here .. 'draft_send.lua')

local function run_failure(failed_kind)
  local calls = {}
  local settled = 0
  local function action(label)
    local kind = label:match('draft%.send%.(.+)$')
    calls[#calls + 1] = kind
    return { code = kind == failed_kind and 17 or 0 }
  end
  local ok, phase, err = delivery.send('body', action, function() settled = settled + 1 end)
  return ok, phase, err, calls, settled
end

for _, kind in ipairs({ 'focus-agent', 'write-body', 'submit', 'focus-draft' }) do
  local ok, phase, err = run_failure(kind)
  assert(not ok and err:match(kind:gsub('%-', '%%-')), kind .. ' must report failure')
  local want = ({ ['focus-agent'] = 'start', ['write-body'] = 'indeterminate', submit = 'written', ['focus-draft'] = 'dispatched' })[kind]
  assert(phase == want, kind .. ' phase = ' .. tostring(phase) .. ', want ' .. want)
end


local ok, phase = delivery.send('body', function() return { code = 0 } end, function() end)
assert(ok and phase == 'dispatched', 'successful submit confirms dispatch')

local settled = 0
delivery.send(string.rep('x', 201), function() return { code = 0 } end, function() settled = settled + 1 end)
assert(settled == 1, 'large body settles exactly once between write and submit')

settled = 0
delivery.send('short body', function() return { code = 0 } end, function() settled = settled + 1 end)
assert(settled == 1, 'short body settles exactly once between write and submit')

-- An agent pane that honors bracketed paste: it keeps what arrives between the
-- markers. Unframed input comes back tagged, so a test sees the difference
-- rather than the fake quietly accepting it. Whether a real agent reassembles a
-- paste across tty reads is not this fake's to claim; probes/claudedraftsend
-- measures that live.
local function paste_agent(wire)
  return delivery.unframe(wire) or ('<unframed>' .. wire)
end

local function stateful_zellij(fail_once)
  local state = { focus = 'draft', composer = '', dispatches = {}, calls = {} }
  local failed = false
  local function action(label, argv)
    local kind = label:match('draft%.send%.(.+)$')
    state.calls[#state.calls + 1] = kind
    if kind == fail_once and not failed then failed = true; return { code = 17 } end
    if kind == 'focus-agent' then state.focus = 'agent'
    elseif kind == 'focus-draft' then state.focus = 'draft'
    elseif kind == 'write-body' then assert(state.focus == 'agent'); state.composer = state.composer .. paste_agent(argv[4])
    elseif kind == 'submit' then assert(state.focus == 'agent'); state.dispatches[#state.dispatches + 1] = state.composer; state.composer = ''
    end
    return { code = 0 }
  end
  return state, action
end

do
  local state, action = stateful_zellij('submit')
  local first_ok, first_phase = delivery.send('body', action, function() end)
  assert(not first_ok and first_phase == 'written' and state.composer == 'body')
  local retry_ok, retry_phase = delivery.send('body', action, function() end, first_phase)
  assert(retry_ok and retry_phase == 'dispatched')
  assert(#state.dispatches == 1 and state.dispatches[1] == 'body' and state.composer == '', 'submit retry must not rewrite staged body')
end

-- pair#211: the body is written as exactly one bracketed paste.
assert(delivery.frame('hi') == '\27[200~hi\27[201~', 'body is wrapped in paste markers')
local cmds = delivery.commands('line one\nline two')
assert(cmds[2].argv[4] == '\27[200~line one\nline two\27[201~', 'the write-chars arg is the framed body')
assert(cmds[2].opts.redact[4] == cmds[2].argv[4], 'the trace redacts (and hashes) the bytes actually written')
-- Markers inside the body cannot end the paste early or nest one.
assert(delivery.frame('a\27[201~b\27[200~c') == '\27[200~abc\27[201~', 'embedded paste markers are stripped')
assert(delivery.unframe(delivery.frame('a\nb')) == 'a\nb', 'unframe inverts frame')
assert(delivery.unframe('plain') == nil and delivery.unframe('\27[200~open') == nil, 'unframe refuses an unframed wire')
-- Other escapes are the operator's text, not framing: left alone.
assert(delivery.frame('x\27[Ay') == '\27[200~x\27[Ay\27[201~', 'non-marker escapes pass through')

-- A body the size #211 lost (3+ tty reads of ~1 KiB) goes as one framed paste.
do
  local lines = {}
  for i = 1, 70 do lines[#lines + 1] = string.format('line %03d abcdefghijklmnopqrstuvwxyz', i) end
  local body = table.concat(lines, '\n')
  assert(#body > 2048, 'the body must span at least three 1 KiB reads')
  local state, action = stateful_zellij()
  local sent_ok = delivery.send(body, action, function() end)
  assert(sent_ok and state.dispatches[1] == body, 'a 3+-read-sized body is sent as one paste, byte-exact')
end

local blocked_ok, blocked_phase = delivery.send('body', function() error('must not act') end, function() end, 'indeterminate')
assert(not blocked_ok and blocked_phase == 'indeterminate', 'indeterminate body write blocks automatic retry')

print('draft_send_test ok')
