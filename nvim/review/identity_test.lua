local identity = dofile('nvim/review/identity.lua')

-- Empty native stderr is a real timeout result, not an actionable diagnostic.
for _, case in ipairs({
  {result={code=124, signal=9, stdout='', stderr=''}, reason='timed out'},
  {result={code=1, signal=0, stdout='', stderr=''}, reason='exit 1'},
  {result={code=1, signal=0, stdout=''}, reason='exit 1'},
  {result={code=1, signal=0, stdout='', stderr='specific Git failure'}, reason='specific Git failure'},
}) do
  local observed = identity.decode(case.result)
  assert(observed.status == 'invalid' and observed.snapshot == nil)
  assert(type(observed.diagnostic) == 'string' and observed.diagnostic:find(case.reason, 1, true),
    'failed resolver lost its cause: ' .. vim.inspect(observed))
end

local native = vim.system
local waits, callbacks = {}, {}
local result = {code=124, signal=9, stdout='', stderr=''}
vim.system = function(cmd, opts, callback)
  assert(cmd[2]=='review' and cmd[3]=='readiness' and cmd[4]=='--resolve')
  if callback then
    callbacks[#callbacks+1]={callback=callback, result=result, timeout=opts.timeout}
  end
  return {wait=function(_, timeout)
    waits[#waits+1]=timeout
    return result
  end}
end
local sync = identity.resolve('/repo')
assert(waits[1]==2500 and sync.status=='invalid' and sync.diagnostic:find('timed out',1,true))
local async
identity.resolve_async('/repo', nil, function(value) async=value end)
assert(async==nil and callbacks[1].timeout==2500)
callbacks[1].callback(callbacks[1].result)
assert(async.status=='invalid' and async.diagnostic:find('timed out',1,true))
vim.system = native
print('review identity: native failure diagnostics and deadlines preserved')
