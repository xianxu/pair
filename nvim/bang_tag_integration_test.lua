-- Headless driver loaded after the real nvim/init.lua (#337). A stub `couch`
-- on PATH records each invocation, so the production publish command runs
-- unmodified; the zellij executor and Pair log seams are faked as in
-- submission_integration_test.lua.
local case = assert(os.getenv('PAIR_TEST_BANG_CASE'))
local calls_path = assert(os.getenv('PAIR_TEST_COUCH_CALLS'))

local appended = {}
_G.PairTestSessionLogAppend = function(body)
  appended[#appended + 1] = body
  return true
end
_G.PairTestSessionLogCommit = function() return true end

local dispatches = {}
local composer = ''
_G.PairTestZellijExecutor = function(label, argv)
  local kind = assert(label:match('draft%.send%.(.+)$'))
  if kind == 'write-body' then composer = composer .. argv[4] end
  if kind == 'submit' then
    dispatches[#dispatches + 1] = composer
    composer = ''
  end
  return { code = 0 }
end

local function publishes()
  local f = io.open(calls_path, 'r')
  if not f then return {} end
  local out = {}
  for line in f:lines() do out[#out + 1] = line end
  f:close()
  return out
end

local function send(authored, stripped)
  return _G.submit_operator_text(authored, stripped or authored)
end

assert(send('! start working on #337'), 'bang line sends')
assert(dispatches[1] == 'start working on #337', 'agent gets the text after !: ' .. vim.inspect(dispatches))
assert(appended[1] == '! start working on #337', 'Pair log keeps the authored body')

assert(not send('!'), 'bare ! sends nothing')
assert(#dispatches == 1 and #appended == 1, 'bare ! neither dispatches nor logs')

assert(send('! first\nsecond'), 'multi-line bang draft sends')
assert(dispatches[2] == '! first\nsecond', 'multi-line draft is sent unchanged')

assert(send('=== sticky\n! tagged under a sticky block', '! tagged under a sticky block'), 'sticky + bang sends')
assert(dispatches[3] == 'tagged under a sticky block', 'sticky block is stripped before the bang test')

assert(send('plain prompt'), 'plain prompt sends')
assert(dispatches[4] == 'plain prompt', 'plain prompt unchanged')

local want = {
  'scope=S1 tag=T1 --internal publish-description --description=start working on #337',
  'scope=S1 tag=T1 --internal publish-description --description=tagged under a sticky block',
}
if case == 'couch' then
  assert(vim.wait(5000, function() return #publishes() >= #want end, 20),
    'publishes never arrived: ' .. vim.inspect(publishes()))
  vim.wait(300, function() return false end, 20)
  local got = publishes()
  table.sort(got)
  table.sort(want)
  assert(vim.deep_equal(got, want), 'publishes = ' .. vim.inspect(got))
elseif case == 'standalone' then
  vim.wait(500, function() return false end, 20)
  assert(#publishes() == 0, 'outside couch nothing publishes: ' .. vim.inspect(publishes()))
else
  error('unknown case ' .. case)
end

print('bang tag ' .. case .. ': ok')
vim.cmd('qa!')
