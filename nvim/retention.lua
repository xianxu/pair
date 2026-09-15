-- Managed editor access to Pair retention. The launcher advertises protocol
-- support so an updated config can coexist with an older launcher/binary.
local M = {}
local Guard = {}
Guard.__index = Guard

local function run(args)
  local argv = { 'pair', 'retention' }
  vim.list_extend(argv, args)
  local ok, process = pcall(vim.system, argv, { text = true })
  if not ok then return nil, tostring(process) end
  local result = process:wait(5000)
  if result.code ~= 0 then return nil, vim.trim(result.stderr or 'retention command failed') end
  return vim.trim(result.stdout or '')
end

function M.new(options, role)
  local self = setmetatable({ enabled = options.enabled, run = options.run or run,
    pid = options.pid or vim.fn.getpid() }, Guard)
  if self.enabled then
    local args = { 'register', '--pid', tostring(self.pid), '--role', role }
    if options.target and options.target ~= '' then
      vim.list_extend(args, { '--target', options.target })
    end
    local id, err = self.run(args)
    if not id or id == '' then return nil, err or 'missing process registration' end
    self.id = id
  end
  return self
end

function Guard:close()
  if not self.id then return true end
  if self.foreground_view then
    local id, err = self:begin(self.foreground_view)
    if not id then return false, err end
    local ok, failure = self:complete(id, true)
    if not ok then return false, failure end
    self.foreground_view = nil
  end
  local result, err = self.run({ 'release', '--id', self.id })
  if result == nil then return false, err end
  self.id = nil
  return true
end

function Guard:begin(target)
  if not self.enabled then return '' end
  if not self.id then return nil, 'editor storage registration is closed' end
  return self.run({ 'begin', '--pid', tostring(self.pid), '--target', target })
end

function Guard:complete(id, changed)
  if not self.enabled then return true end
  local action = changed == false and 'unchanged' or 'complete'
  local result, err = self.run({ action, '--id', id })
  return result ~= nil, err
end

-- Caller supplies actual IO so tests exercise ordering with a stateful command
-- backend. A failed write can be partial; leave the intent for recovery.
function Guard:write(path, content, read, write)
  if self.enabled and read() == content then return true end
  local id, err = self:begin(path)
  if id == nil then return false, err end
  local ok, result = pcall(write, content)
  if not ok or not result then return false, ok and 'content write failed' or result end
  return self:complete(id, true)
end

-- Append/delete preserve the caller's native IO operation (including O_APPEND).
function Guard:change(target, effect)
  local id, err = self:begin(target)
  if id == nil then return false, err end
  local ok, changed = pcall(effect)
  if not ok or not changed then return false, ok and 'content change failed' or changed end
  return self:complete(id, true)
end

function Guard:view(target, read)
  local id, err = self:begin(target)
  if id == nil then return nil, err end
  local ok, value = pcall(read)
  if not ok then return nil, value end
  local completed, failure = self:complete(id, value ~= nil)
  if not completed then return nil, failure end
  return value
end

function Guard:watch_writes(owns)
  if not self.enabled then return end
  local pending = {}
  vim.api.nvim_create_autocmd('BufWritePre', { callback = function(args)
    local path = vim.api.nvim_buf_get_name(args.buf)
    if not owns(path) then return end
    local content = table.concat(vim.api.nvim_buf_get_lines(args.buf, 0, -1, false), '\n')
    if vim.bo[args.buf].endofline or vim.bo[args.buf].fixendofline then content = content .. '\n' end
    local file = io.open(path, 'rb')
    local prior
    if file then prior = file:read('*a'); file:close() end
    if prior == content then return end
    local id, err = self:begin(path)
    if not id then error('pair: cannot protect write: ' .. tostring(err)) end
    pending[args.buf] = id
  end })
  vim.api.nvim_create_autocmd('BufWritePost', { callback = function(args)
    local id = pending[args.buf]
    if not id then return end
    local ok, err = self:complete(id, true)
    if not ok then error('pair: content saved; retention intent remains: ' .. tostring(err)) end
    pending[args.buf] = nil
  end })
end

function M.setup(role)
  local guard, err = M.new({ enabled = vim.env.PAIR_RETENTION_PROTOCOL == '1', target = role:match('viewer$') and vim.env.PAIR_RETENTION_TARGET or nil }, role)
  if not guard then
    vim.api.nvim_err_writeln('pair: cannot protect editor storage: ' .. tostring(err))
    vim.cmd('cquit 1')
    return nil
  end
  if guard.enabled then
    vim.api.nvim_create_autocmd('VimLeavePre', { once = true, callback = function() guard:close() end })
    local initial_read
    vim.api.nvim_create_autocmd('BufReadPre', { once = true, callback = function(args)
      local id, failure = guard:begin(args.file)
      if not id then error('pair: cannot protect initial read: ' .. tostring(failure)) end
      initial_read = { id = id, target = args.file }
    end })
    vim.api.nvim_create_autocmd('BufReadPost', { once = true, callback = function()
      if initial_read then
        local ok, failure = guard:complete(initial_read.id, true)
        if not ok then error('pair: cannot record initial read: ' .. tostring(failure)) end
        if role:match('viewer$') then guard.foreground_view = initial_read.target end
        initial_read = nil
      end
    end })
  end
  return guard
end

return M
