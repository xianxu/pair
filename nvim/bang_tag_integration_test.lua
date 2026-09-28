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
local failed_submit = false
_G.PairTestZellijExecutor = function(label, argv)
  local kind = assert(label:match('draft%.send%.(.+)$'))
  if case == 'retry' and kind == 'submit' and not failed_submit then
    failed_submit = true
    return { code = 17 }
  end
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

local function wait_for_calls(count)
  assert(vim.wait(5000, function() return #publishes() >= count end, 20),
    'publishes never arrived: ' .. vim.inspect(publishes()))
end

if case == 'missing' or case == 'nonzero' or case == 'slow' or case == 'retry' then
  if case == 'missing' then
    -- Change PATH after init, keeping the real jobstart and its ENOENT behavior.
    vim.env.PATH = assert(os.getenv('PAIR_TEST_EMPTY_PATH'))
    assert(vim.fn.executable('couch') == 0, 'missing-publisher setup')
  end
  if case == 'retry' then
    assert(not send('! retry description'), 'failed agent submit must fail')
    assert(failed_submit and #dispatches == 0, 'dispatch failure was injected')
    vim.wait(300, function() return false end, 20)
    assert(#publishes() == 0, 'failed dispatch must not publish')
    assert(send('! retry description'), 'retry must succeed')
    assert(#dispatches == 1 and dispatches[1] == 'retry description', 'retry dispatches stripped text exactly once')
    wait_for_calls(1)
    vim.wait(300, function() return false end, 20)
    assert(vim.deep_equal(publishes(), {
      'scope=S1 tag=T1 --internal publish-description --description=retry description',
    }), 'retry must publish exactly once: ' .. vim.inspect(publishes()))
  else
    assert(send('! publisher boundary'), 'publisher failure/blocking must not fail send')
    assert(#dispatches == 1 and dispatches[1] == 'publisher boundary', 'agent receives stripped text')
    assert(appended[1] == '! publisher boundary', 'authored text is retained')
    if case == 'slow' then
      wait_for_calls(1)
      assert(vim.fn.filereadable(calls_path .. '.completed') == 0, 'submit returned before publisher completed')
      assert(vim.fn.filereadable(calls_path .. '.timed-out') == 0, 'submit must not wait for publisher timeout')
      vim.fn.writefile({}, calls_path .. '.release')
      assert(vim.wait(5000, function()
        return vim.fn.filereadable(calls_path .. '.completed') == 1
      end, 20), 'publisher did not complete after release')
    elseif case == 'nonzero' then
      wait_for_calls(1)
      vim.wait(300, function() return false end, 20)
    else
      assert(#publishes() == 0, 'missing executable cannot publish')
    end
    assert(send('next prompt'), 'publisher failure must not poison the next send')
    assert(#dispatches == 2 and dispatches[2] == 'next prompt', 'next prompt dispatches once')
  end
  print('bang tag ' .. case .. ': ok')
  vim.cmd('qa!')
  return
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
