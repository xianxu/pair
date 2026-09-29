-- Durable, bounded unsaved-text recovery. Callers own branch validation and must
-- call saved only after a successful write on the matching active branch.
local M = {}
local uv = vim.uv or vim.loop
local limit, capacity = 8 * 1024 * 1024, 32
local restored, written, sequence = {}, {}, 0
local function fail(message) error('review recovery: ' .. tostring(message), 0) end
local function must(value, err) if value == nil or value == false then fail(err) end; return value end
local function identity(context)
  if type(context) ~= 'table' then fail('invalid identity') end
  local parts = {}
  for _, key in ipairs({ 'repo', 'branch', 'file' }) do
    local value = context[key]
    if type(value) ~= 'string' or value == '' or value:find('\0', 1, true) then fail('invalid ' .. key) end
    parts[#parts + 1] = value
  end
  return vim.fn.sha256(vim.json.encode(parts))
end
local function private(stat, kind, mode)
  if stat.type ~= kind or stat.mode % 512 ~= mode or stat.uid ~= uv.getuid() then
    fail('storage must be an owned private ' .. kind .. ' (no symlinks)')
  end
end
local function location(dir, context, create)
  local key = identity(context)
  local stat, err, code = uv.fs_lstat(dir)
  if not stat and code ~= 'ENOENT' then fail(err) end
  if not stat and create then
    must(uv.fs_mkdir(dir, 448))
    stat = must(uv.fs_lstat(dir))
  end
  if stat then private(stat, 'directory', 448) end
  return dir .. '/' .. key .. '.json', stat ~= nil
end
local function read(path, context)
  local stat, err, code = uv.fs_lstat(path)
  if not stat then if code == 'ENOENT' then return nil end; fail(err) end
  private(stat, 'file', 384)
  if stat.size > limit then fail('snapshot exceeds 8 MiB') end
  local fd = must(uv.fs_open(path, 'r', 0))
  local inside = uv.fs_fstat(fd)
  if not inside or inside.ino ~= stat.ino or inside.dev ~= stat.dev then uv.fs_close(fd); fail('snapshot changed during read') end
  local bytes, readerr = uv.fs_read(fd, stat.size, 0)
  uv.fs_close(fd)
  must(bytes, readerr)
  local ok, snapshot = pcall(vim.json.decode, bytes)
  if not ok or type(snapshot) ~= 'table' or snapshot.version ~= 1 or type(snapshot.owner) ~= 'string'
    or type(snapshot.eol) ~= 'boolean' or type(snapshot.lines) ~= 'table' or #snapshot.lines == 0 then
    fail('invalid snapshot; preserve it for explicit discard')
  end
  for _, key in ipairs({ 'repo', 'branch', 'file' }) do
    if snapshot[key] ~= context[key] then fail('snapshot identity mismatch') end
  end
  for index, line in pairs(snapshot.lines) do
    if type(index) ~= 'number' or index < 1 or index > #snapshot.lines or index % 1 ~= 0
      or type(line) ~= 'string' or line:find('\n', 1, true) then fail('invalid snapshot lines') end
  end
  return snapshot, bytes
end
local function atomic(path, bytes)
  sequence = sequence + 1
  local temp = path .. '.tmp-' .. uv.os_getpid() .. '-' .. sequence
  local fd = must(uv.fs_open(temp, 'wx', 384))
  local ok, err = pcall(function()
    local offset = 0
    while offset < #bytes do
      local written = must(uv.fs_write(fd, bytes:sub(offset + 1), offset))
      if written == 0 then fail('short write') end
      offset = offset + written
    end
    must(uv.fs_fsync(fd))
  end)
  local closed, closeerr = uv.fs_close(fd)
  if not closed and ok then ok, err = false, closeerr end
  if ok then ok, err = uv.fs_rename(temp, path) end
  if not ok then uv.fs_unlink(temp); fail(err) end
end
local function guarded(fn)
  return function(...)
    local ok, result = pcall(fn, ...)
    if not ok then return nil, tostring(result) end
    return result
  end
end
M.save = guarded(function(dir, context, buf, owner)
  if type(owner) ~= 'string' or owner == '' then fail('owner required') end
  local path = location(dir, context, true)
  local previous, previous_bytes = read(path, context)
  local admitted = restored[path]
  if previous and previous.owner ~= owner and not (admitted and admitted.buf == buf and admitted.bytes == previous_bytes) then
    fail('unconsumed prior-process snapshot; use :PairReviewRecover or :PairReviewDiscardRecovery')
  end
  if not previous then
    local scan = must(uv.fs_scandir(dir))
    local count = 0
    while true do
      local name = uv.fs_scandir_next(scan)
      if not name then break end
      -- Count every entry: unknown files cannot be used to evade bounded storage.
      count = count + 1
    end
    if count >= capacity then fail('32 snapshot capacity reached; recover or discard an existing snapshot') end
  end
  local snapshot = { version = 1, repo = context.repo, branch = context.branch, file = context.file,
    owner = owner, lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false), eol = vim.bo[buf].eol }
  local bytes = vim.json.encode(snapshot)
  if #bytes > limit then fail('snapshot exceeds 8 MiB; preserve text before quitting') end
  atomic(path, bytes)
  written[path] = { buf = buf, owner = owner, bytes = bytes }
  if admitted and admitted.buf == buf then admitted.bytes = bytes end
  return path
end)
M.inspect = guarded(function(dir, context)
  local path,exists=location(dir,context,false)
  if not exists then return nil end
  local snapshot=read(path,context)
  if snapshot then snapshot.path=path end
  return snapshot
end)
M.restore = guarded(function(dir, context, buf)
  local path, exists = location(dir, context, false)
  if not exists then return nil end
  local snapshot, bytes = read(path, context)
  if not snapshot then return nil end
  -- API replacement retains the buffer's undo tree; never reload or edit!.
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, snapshot.lines)
  vim.bo[buf].eol = snapshot.eol
  vim.bo[buf].modified = true
  restored[path] = { buf = buf, bytes = bytes }
  snapshot.path = path
  return snapshot
end)
M.discard = guarded(function(dir, context)
  local path, exists = location(dir, context, false)
  if exists then
    local stat, err, code = uv.fs_lstat(path)
    if stat then private(stat, 'file', 384); must(uv.fs_unlink(path))
    elseif code ~= 'ENOENT' then fail(err) end
  end
  restored[path], written[path] = nil, nil
  return true
end)
M.saved = guarded(function(dir, context, buf, owner)
  local path, exists = location(dir, context, false)
  if not exists then return false end
  local snapshot, bytes = read(path, context)
  if not snapshot then return false end
  local admitted = restored[path]
  local own = written[path]
  if not (own and own.buf == buf and own.owner == owner and own.bytes == bytes)
    and not (admitted and admitted.buf == buf and admitted.bytes == bytes) then return false end
  must(uv.fs_unlink(path))
  restored[path], written[path] = nil, nil
  return true
end)
return M
