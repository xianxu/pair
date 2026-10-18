-- Run: nvim --headless -u NONE -i NONE -l tests/review-resolver-native-deadline-test.lua
-- Native timeout conformance only. Ordering fixtures use their stateful double.
local identity = dofile('nvim/review/identity.lua')
local native = vim.system
local deadline_ms, teardown_ms = 2500, 5000
local hang_bound_ms = deadline_ms + teardown_ms
local owned, evidence = {}, {}

local function spawn(opts, callback)
  local output = ''
  opts.stdin = true -- Keep the owned pipe open: the shell's builtin read cannot finish.
  opts.text = true
  opts.stdout = function(err, data)
    assert(not err, tostring(err))
    if data then output = output .. data end
  end
  -- Both commands are shell builtins: no sleep, external program or grandchildren.
  local process = native({'/bin/sh', '-c', "printf 'ready\\n'; IFS= read -r token; exit 91"}, opts, callback)
  owned[#owned + 1] = process
  return process, function() return output end
end

local function check(arm, result, output)
  -- Async timeout includes child startup: a loaded host may expire it before
  -- the shell prints readiness. That is still the correct native outcome.
  assert(output == 'ready\n' or (arm == 'async' and output == ''), arm .. ': unexpected owned child output')
  assert(result.code == 124, arm .. ': expected native timeout code124, got ' .. vim.inspect(result))
  local refusal = identity.decode(result)
  assert(refusal.status == 'invalid', arm .. ': timed-out resolver was accepted')
  assert(type(refusal.diagnostic) == 'string' and refusal.diagnostic:find('timed out', 1, true),
    arm .. ': useful timeout refusal missing: ' .. vim.inspect(refusal))
  evidence[#evidence + 1] = {arm=arm, code=result.code, signal=result.signal,
    ready=output=='ready\n', stdout=output, stderr=result.stderr, diagnostic=refusal.diagnostic}
end

local ok, failure = xpcall(function()
  local process, output = spawn({})
  assert(vim.wait(hang_bound_ms, function() return output() == 'ready\n' end, 10),
    'synchronous child readiness exceeded hang bound')
  -- Same native synchronous deadline as shipped identity.resolve.
  check('sync', process:wait(2500), output())

  local result
  process, output = spawn({timeout=2500}, function(value) result = value end)
  -- This is a hang guard, not an elapsed-time performance assertion. The
  -- asynchronous process retains the shipped timeout option without enlargement.
  assert(vim.wait(hang_bound_ms, function() return result ~= nil end, 10),
    'asynchronous native timeout callback exceeded hang bound')
  check('async', result, output())
end, debug.traceback)

-- Always kill/wait every child, including readiness/assertion failures. Closing
-- stdin also prevents a surviving read if native termination itself fails.
local cleanup_failures = {}
for _, process in ipairs(owned) do
  pcall(function() process:kill(9) end)
  pcall(function() process:write(nil) end)
  local reaped, result = pcall(function() return process:wait(teardown_ms) end)
  local closing_ok, closing = pcall(function() return process:is_closing() end)
  if not reaped or not closing_ok or not closing then
    cleanup_failures[#cleanup_failures + 1] = tostring(result)
  end
end
print('native resolver deadline evidence: ' .. vim.json.encode(evidence))
if not ok then error(failure, 0) end
assert(#cleanup_failures == 0, 'owned child cleanup failed: ' .. table.concat(cleanup_failures, '; '))
print('review-resolver-native-deadline-test passed')
