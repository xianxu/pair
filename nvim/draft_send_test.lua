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
  local ok, dispatched, err = delivery.send('body', no_submit, action, function() settled = settled + 1 end)
  return ok, dispatched, err, calls, settled
end

for _, kind in ipairs({ 'focus-agent', 'write-body', 'submit', 'focus-draft' }) do
  local ok, dispatched, err = run_failure(false, kind)
  assert(not ok and err:match(kind:gsub('%-', '%%-')), kind .. ' must report failure')
  assert(dispatched == (kind == 'focus-draft'), kind .. ' dispatch classification')
end

for _, kind in ipairs({ 'focus-agent', 'write-body', 'newline', 'focus-draft' }) do
  local ok, dispatched = run_failure(true, kind)
  assert(not ok and not dispatched, kind .. ' compose-only failure must never claim dispatch')
end

local ok, dispatched = delivery.send('body', false, function() return { code = 0 } end, function() end)
assert(ok and dispatched, 'successful submit confirms dispatch')
ok, dispatched = delivery.send('body', true, function() return { code = 0 } end, function() end)
assert(ok and not dispatched, 'successful compose-only transfer is not dispatch')

local settled = 0
delivery.send(string.rep('x', 201), false, function() return { code = 0 } end, function() settled = settled + 1 end)
assert(settled == 1, 'large body settles exactly once between write and submit')

print('draft_send_test ok')
