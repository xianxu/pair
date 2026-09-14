local here = debug.getinfo(1, 'S').source:sub(2):match('(.*/)') or './'
local retention = dofile(here .. 'retention.lua')
local state = { ids = 0, intents = {}, leases = {}, uses = 0, fail = false }
local function command(args)
  if state.fail then return nil, 'metadata unavailable' end
  local action = args[1]
  if action == 'register' then state.leases.lease = true; return 'lease' end
  if action == 'release' then state.leases[args[3]] = nil; return '' end
  if action == 'begin' then
    assert(state.leases.lease, 'effect without live lease')
    state.ids = state.ids + 1
    local id = tostring(state.ids)
    state.intents[id] = true
    return id
  end
  if action == 'complete' or action == 'unchanged' then
    assert(state.intents[args[3]], 'missing intent')
    state.intents[args[3]] = nil
    if action == 'complete' then state.uses = state.uses + 1 end
    return ''
  end
  error('unexpected action ' .. action)
end
local guard = assert(retention.new({ enabled = true, run = command, pid = 123 }, 'draft-editor'))
local body = 'old'
assert(guard:write('draft', 'new', function() return body end, function(value)
  assert(next(state.intents), 'write preceded intent')
  body = value; return true
end))
assert(body == 'new' and state.uses == 1 and not next(state.intents))
assert(guard:write('draft', 'new', function() return body end, function() error('unchanged write') end))
assert(state.uses == 1)
state.fail = true
assert(not guard:write('draft', 'later', function() return body end, function() error('unprotected write') end))
state.fail = false
assert(not guard:write('draft', 'later', function() return body end, function(value) body = value; return false end))
assert(next(state.intents), 'indeterminate write lost protection')
assert(guard:close())
assert(not next(state.leases))
local disabled = assert(retention.new({ enabled = false }, 'draft-editor'))
assert(disabled:write('draft', 'x', function() return '' end, function() return true end))
print('retention: intent ordering, unchanged writes, failure protection passed')

local native = assert(retention.new({ enabled = true, run = command, pid = 123 }, 'draft-editor'))
local tmp = vim.fn.tempname()
vim.fn.writefile({ 'original' }, tmp)
tmp = vim.uv.fs_realpath(tmp)
native:watch_writes(function(path) return path == tmp end)
vim.cmd('edit ' .. vim.fn.fnameescape(tmp))
vim.api.nvim_buf_set_lines(0, 0, -1, false, { 'replacement' })
state.fail = true
local wrote = pcall(vim.cmd, 'write')
assert(not wrote, 'metadata failure did not abort native write')
assert(vim.fn.readfile(tmp)[1] == 'original', 'native write escaped durable intent')
state.fail = false
vim.cmd('write')
assert(vim.fn.readfile(tmp)[1] == 'replacement')
os.remove(tmp)
native:close()
