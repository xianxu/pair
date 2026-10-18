-- Stateful fixture model; native resolver execution is forbidden behind this seam.
local native = vim.system
local temp = vim.fn.system({'mktemp', '-d', (vim.env.TMPDIR or '/tmp') .. '/pair-review-fake.XXXXXX'}):gsub('\n$', '')
assert(vim.v.shell_error == 0 and temp:sub(1, 1) == '/', 'owned temporary root required')
temp = assert(vim.uv.fs_realpath(temp))
local fake
local function write(path, bytes)
  local file = assert(io.open(path, 'wb')); assert(file:write(bytes)); assert(file:close())
end
local function oversized(path)
  local file = assert(io.open(path, 'wb'))
  assert(file:seek('set', 8*1024*1024)); assert(file:write('x')); assert(file:close())
end
local function settle()
  local done = false
  vim.schedule(function() done = true end)
  assert(vim.wait(1000, function() return done end, 1), 'scheduled callbacks did not settle')
end
local ok, failure = xpcall(function()
  local repo = temp .. '/repo'
  vim.fn.mkdir(repo .. '/.git/refs/heads/review', 'p')
  local function select(ref, head)
    write(repo .. '/.git/HEAD', 'ref: ' .. ref .. '\n')
    write(repo .. '/.git/' .. ref, head .. '\n')
  end
  local a = {status='resolved', repo=repo, branch='review/a', head=string.rep('a', 40), file='a.md',
    latest_agent_body='A records', latest_agent_commit='agent-a', human_round=1, agent_round=2}
  local b = {status='resolved', repo=repo, branch='review/b', head=string.rep('b', 40), file='b.md',
    latest_agent_body='B records', latest_agent_commit='agent-b', human_round=3, agent_round=4}
  local model = temp .. '/model.json'
  write(model, vim.json.encode({repo=repo, by_ref={['refs/heads/review/a']=a, ['refs/heads/review/b']=b}}))
  write(repo .. '/a.md', 'A initial\n'); write(repo .. '/b.md', 'B initial\n')
  select('refs/heads/review/a', a.head)
  local delegated, sentinel = 0, {}
  local trap = function(cmd)
    assert(not (cmd[2]=='review' and cmd[3]=='readiness' and cmd[4]=='--resolve'), 'native resolver invoked')
    delegated = delegated + 1
    return sentinel
  end
  vim.system = trap
  local module = dofile('tests/lib/review_resolver_fake.lua')
  fake = module.install({model_path=model})
  assert(PairReviewResolverFake == fake)
  local function command(snapshot, dir)
    local cmd = {'pair', 'review', 'readiness', '--resolve', dir or repo}
    if snapshot then cmd[#cmd+1] = '--snapshot' end
    return cmd
  end
  local function query(snapshot, dir)
    local result = vim.system(command(snapshot, dir), {text=true}):wait(2500)
    assert(result.code == 0 and result.signal == 0 and result.stderr == '')
    return vim.json.decode(result.stdout)
  end
  assert(vim.deep_equal(query(), a), 'known A identity changed')
  select('refs/heads/review/b', b.head)
  assert(vim.deep_equal(query(), b), 'actual HEAD did not select B')
  select('refs/heads/review/a', string.rep('c', 40))
  assert(query().status == 'invalid', 'unknown commit was accepted')
  select('refs/heads/review/unknown', a.head)
  assert(query().status == 'invalid', 'unknown ref was accepted')
  write(repo .. '/.git/HEAD', a.head .. '\n')
  assert(query().status == 'invalid', 'detached HEAD was accepted')
  assert(query(false, temp).status == 'invalid', 'different repository was accepted')
  select('refs/heads/review/a', a.head)
  for _, bytes in ipairs({'plain\n', '\239\187\191first\r\nlast', '\239\187\191', '', 'no EOL', 'a\r\nb\r\n'}) do
    write(repo .. '/a.md', bytes)
    assert(query(true).snapshot == bytes, 'snapshot changed exact document bytes')
  end
  write(repo .. '/a.md', '\255')
  assert(query(true).status == 'invalid', 'invalid UTF-8 snapshot accepted')
  oversized(repo .. '/a.md')
  assert(query(true).status == 'invalid', 'oversized snapshot accepted')
  oversized(repo .. '/.git/HEAD')
  assert(query().status == 'invalid', 'oversized HEAD accepted')
  select('refs/heads/review/a', a.head)
  write(repo .. '/a.md', 'captured A\r\n')
  local delivered
  local process = vim.system(command(true), {text=true, timeout=2500}, function(result)
    delivered = vim.json.decode(result.stdout)
  end)
  assert(type(process.wait)=='function' and type(process.kill)=='function' and type(process.is_closing)=='function')
  assert(delivered == nil, 'callback was delivered inline')
  select('refs/heads/review/b', b.head); write(repo .. '/a.md', 'later body')
  settle()
  assert(delivered.branch=='review/a' and delivered.head==a.head and delivered.snapshot=='captured A\r\n',
    'async callback reread changed branch/body')
  assert(process:is_closing())
  local metadata = fake:metadata()
  assert(metadata[#metadata].timeout == 2500, 'async deadline option lost')
  local cancelled
  process = vim.system(command(), {timeout=2500}, function(value) cancelled = value end)
  process:kill(15); settle()
  assert(cancelled and cancelled.signal==15 and process:is_closing(), 'killed process lost its completion callback')
  assert(vim.system({'git', 'status'}, {}) == sentinel and delegated==1, 'unrelated command did not delegate')
  for _, n in ipairs({8, 16}) do
    local before = fake:counts()
    for _=1,n do assert(query().branch == 'review/b') end
    local after = fake:counts()
    assert(after.calls-before.calls == n and after.reads-before.reads == 2*n and after.waits-before.waits == n,
      'query work is not counted linear HEAD/ref reads and waits')
    metadata = fake:metadata()
    assert(metadata[#metadata].wait_timeout==2500, 'synchronous wait deadline lost')
  end
  for _=1,140 do query() end
  assert(#fake:metadata() == 128, 'metadata is not bounded')
  assert(not vim.json.encode(fake:metadata()):find('captured A', 1, true), 'metadata retained document bytes')
  metadata = fake:metadata(); metadata[1].timeout = 'tampered'
  assert(fake:metadata()[1].timeout ~= 'tampered', 'metadata exposes mutable records')
  local late = false
  vim.system(command(), {timeout=2500}, function() late = true end)
  fake:close(); settle()
  assert(not late and vim.system==trap, 'close did not cancel pending delivery and restore native owner')
  fake = module.install({model_path=model})
  local newer = function() end
  vim.system = newer; fake:close()
  assert(vim.system==newer, 'close replaced a newer wrapper')
  print('review-resolver-fake-test passed')
end, debug.traceback)
if fake then fake:close() end
vim.system = native
vim.fn.delete(temp, 'rf')
if not ok then error(failure, 0) end
