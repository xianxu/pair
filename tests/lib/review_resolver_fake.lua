-- Owned observation fixtures only: model known loose refs, never interpret Git history.
local M = {}
local LIMIT, HISTORY = 8*1024*1024, 128
local function utf8_text(bytes)
  local i = 1
  while i <= #bytes do
    local a = bytes:byte(i)
    if a == 0 then return false end
    local extra, lo, hi = 0, 128, 191
    if a < 128 then
    elseif a >= 194 and a <= 223 then extra = 1
    elseif a >= 224 and a <= 239 then
      extra = 2
      if a == 224 then lo = 160 elseif a == 237 then hi = 159 end
    elseif a >= 240 and a <= 244 then
      extra = 3
      if a == 240 then lo = 144 elseif a == 244 then hi = 143 end
    else return false end
    for j=1,extra do
      local b = bytes:byte(i+j)
      if not b or b < (j==1 and lo or 128) or b > (j==1 and hi or 191) then return false end
    end
    i = i + extra + 1
  end
  return true
end
local function relative(path)
  if type(path)~='string' or path=='' or path:sub(1,1)=='/' or path:find('\0',1,true) then return false end
  for component in path:gmatch('[^/]+') do
    if component=='.' or component=='..' then return false end
  end
  return not path:find('//',1,true) and path:sub(-1)~='/'
end
function M.install(opts)
  assert(opts and type(opts.model_path)=='string' and opts.model_path:sub(1,1)=='/', 'absolute fixture model required')
  if _G.PairReviewResolverFake then _G.PairReviewResolverFake:close() end
  local fake = {closed=false}
  local totals, entries, pending = {calls=0, reads=0, waits=0}, {}, {}
  local function read(path)
    totals.reads = totals.reads + 1
    local info = vim.uv.fs_lstat(path)
    if not info or info.type~='file' then return nil, 'fixture file is absent or not regular' end
    if info.size > LIMIT then return nil, 'fixture read exceeds 8 MiB' end
    local fd, err = vim.uv.fs_open(path, 'r', 0)
    if not fd then return nil, err end
    local bytes
    bytes, err = vim.uv.fs_read(fd, LIMIT+1, 0)
    vim.uv.fs_close(fd)
    if not bytes then return nil, err end
    if #bytes > LIMIT then return nil, 'fixture read exceeds 8 MiB' end
    return bytes
  end
  local raw, err = read(opts.model_path)
  assert(raw, err)
  local model = vim.json.decode(raw)
  assert(type(model)=='table' and type(model.repo)=='string' and model.repo:sub(1,1)=='/'
    and vim.uv.fs_realpath(model.repo)==model.repo and type(model.by_ref)=='table', 'canonical fixture repository and ref model required')
  local function invalid(reason)
    return {status='invalid', repo=model.repo, diagnostic='fixture resolver: '..tostring(reason)}
  end
  local function capture(cmd)
    if cmd[5]~=model.repo then return invalid('repository is outside model') end
    local head, why = read(model.repo..'/.git/HEAD')
    if not head then return invalid(why) end
    local ref = head:match('^ref: ([^\r\n]+)\n?$')
    local known = ref and model.by_ref[ref]
    if not known or ref:sub(1,11)~='refs/heads/' or not relative(ref) then return invalid('unknown or detached HEAD') end
    local commit
    commit, why = read(model.repo..'/.git/'..ref)
    if not commit then return invalid(why) end
    commit = commit:gsub('\n$', '')
    if type(known)~='table' or known.repo~=model.repo or known.branch~=ref:sub(12)
      or known.status~='resolved' or commit~=known.head or not relative(known.file) then
      return invalid('unknown HEAD or inconsistent model')
    end
    local result = vim.deepcopy(known)
    result.snapshot = nil
    for i=6,#cmd do
      if cmd[i]=='--snapshot' then
        local body
        body, why = read(model.repo..'/'..known.file)
        if not body then return invalid(why) end
        if not utf8_text(body) then return invalid('snapshot is not UTF-8 text') end
        result.snapshot = body
      end
    end
    return result
  end
  function fake:counts() return vim.deepcopy(totals) end
  function fake:metadata() return vim.deepcopy(entries) end
  local native, wrapper = vim.system
  local function cancelled() return {code=143, signal=15, stdout='', stderr='fixture resolver cancelled'} end
  function fake:close()
    if self.closed then return end
    self.closed = true
    for process in pairs(pending) do
      process.callback, process.result, process.closed = nil, nil, true
    end
    pending = {}
    if vim.system==wrapper then vim.system = native end
  end
  wrapper = function(cmd, options, callback)
    if type(cmd)~='table' or cmd[2]~='review' or cmd[3]~='readiness' or cmd[4]~='--resolve' then
      return native(cmd, options, callback)
    end
    totals.calls = totals.calls + 1
    local entry = {id=totals.calls, mode=callback and 'callback' or 'wait', timeout=options and options.timeout}
    if #entries==HISTORY then table.remove(entries,1) end
    entries[#entries+1] = entry
    local result = {code=0, signal=0, stderr='', stdout=vim.json.encode(capture(cmd))..'\n'}
    local process = {result=result, callback=callback, closed=false}
    function process:wait(timeout)
      totals.waits = totals.waits + 1
      entry.wait_timeout = timeout
      self.closed = true
      return vim.deepcopy((fake.closed or self.killed) and cancelled() or self.result)
    end
    function process:kill(signal)
      self.killed, self.closed, self.callback, self.result = true, true, nil, nil
      entry.killed, entry.signal = true, signal
      pending[self] = nil
      return true
    end
    function process:is_closing() return self.closed or fake.closed end
    if callback then
      pending[process] = true
      vim.schedule(function()
        if fake.closed or process.killed or not process.callback then return end
        local deliver, value = process.callback, process.result
        process.callback, process.closed = nil, true
        pending[process] = nil
        deliver(vim.deepcopy(value))
      end)
    end
    return process
  end
  vim.system = wrapper
  _G.PairReviewResolverFake = fake
  return fake
end
return M
