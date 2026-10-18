-- Test fixture only: retain a completed resolver callback, never its process.
-- Native commands, options, SystemObj methods and timeout semantics are untouched.
local M = {}
local LIMIT = 32
function M.install()
  if _G.PairReviewObservationGate then _G.PairReviewObservationGate:close() end
  local native = vim.system
  local entries, sequence, armed, closed = {}, 0, false, false
  local wrapper, teardown
  local gate = {}
  local function counts()
    local inflight, held = 0, 0
    for _, entry in ipairs(entries) do
      if entry.state == 'started' then inflight = inflight + 1 end
      if entry.state == 'result' then held = held + 1 end
    end
    return inflight, held
  end
  function gate:arm()
    local inflight, held = counts()
    if closed or inflight > 0 or held > 0 then return false end
    armed = true
    return true
  end
  function gate:disarm() armed = false end
  function gate:snapshot()
    local metadata = {}
    for _, entry in ipairs(entries) do
      metadata[#metadata + 1] = {id = entry.id, state = entry.state,
        code = entry.code, signal = entry.signal, stdout_bytes = entry.stdout_bytes,
        started_ns = entry.started_ns, completed_ns = entry.completed_ns,
        released_ns = entry.released_ns, timeout = entry.timeout}
    end
    local inflight, held = counts()
    return {armed = armed, entries = metadata, inflight = inflight, held = held}
  end
  function gate:release(id)
    if closed then return false end
    for _, entry in ipairs(entries) do
      if entry.id == id then
        if entry.state ~= 'result' then return false end
        local callback, result = entry.callback, entry.result
        entry.callback, entry.result, entry.state = nil, nil, 'released'
        entry.released_ns = vim.uv.hrtime()
        callback(result)
        return true
      end
    end
    return false
  end
  function gate:close()
    if closed then return end
    closed, armed = true, false
    for _, entry in ipairs(entries) do entry.callback, entry.result = nil, nil end
    entries = {}
    if vim.system == wrapper then vim.system = native end
    if teardown then vim.api.nvim_del_autocmd(teardown); teardown = nil end
  end
  wrapper = function(cmd, options, callback)
    if closed or type(callback) ~= 'function' or type(cmd) ~= 'table' or
      cmd[2] ~= 'review' or cmd[3] ~= 'readiness' or cmd[4] ~= '--resolve' then
      return native(cmd, options, callback)
    end
    local inflight, held = counts()
    assert(not armed or inflight + held == 0, 'observation gate already has unreleased work')
    if #entries == LIMIT then
      local released
      for i, entry in ipairs(entries) do
        if entry.state == 'released' then released = i; break end
      end
      assert(released, 'observation gate metadata capacity exhausted')
      table.remove(entries, released)
    end
    sequence = sequence + 1
    local entry = {id = sequence, state = 'started', callback = callback,
      started_ns = vim.uv.hrtime(), timeout = options and options.timeout}
    local hold = armed
    entries[#entries + 1] = entry
    local ok, process = pcall(native, cmd, options, function(result)
      if closed or entry.state ~= 'started' then return end
      entry.completed_ns = vim.uv.hrtime()
      entry.code, entry.signal = result.code, result.signal
      entry.stdout_bytes = type(result.stdout) == 'string' and #result.stdout or 0
      if hold then
        entry.state, entry.result = 'result', result
      else
        local deliver = entry.callback
        entry.state, entry.callback = 'released', nil
        entry.released_ns = vim.uv.hrtime()
        deliver(result)
      end
    end)
    if not ok then
      entry.state, entry.callback = 'released', nil
      error(process, 0)
    end
    return process
  end
  vim.system = wrapper
  teardown = vim.api.nvim_create_autocmd('VimLeavePre', {once = true, callback = function() gate:close() end})
  _G.PairReviewObservationGate = gate
  return gate
end
return M
