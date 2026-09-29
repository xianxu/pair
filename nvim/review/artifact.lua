-- File-generation receipts shared by ephemeral review response consumers.
-- Atomic producer replacements remain distinct even when bytes are identical.
local M={}
function M.read(path)
  if not path or path=='' then return nil end
  local kind=vim.uv.fs_lstat(path)
  if not kind or kind.type~='file' then return nil end
  local fd=vim.uv.fs_open(path,'r',0)
  if not fd then return nil end
  local stat=vim.uv.fs_fstat(fd)
  local data=stat and stat.type=='file' and vim.uv.fs_read(fd,stat.size,0) or nil
  vim.uv.fs_close(fd)
  return data,stat
end
function M.same(a,b)
  return a and b and a.dev==b.dev and a.ino==b.ino and a.size==b.size
    and a.mtime.sec==b.mtime.sec and a.mtime.nsec==b.mtime.nsec
    and a.ctime.sec==b.ctime.sec and a.ctime.nsec==b.ctime.nsec
end
function M.consume(path,data,generation)
  if not data or not generation then return false end
  local current,current_generation=M.read(path)
  if current~=data or not M.same(generation,current_generation)
    or not M.same(current_generation,vim.uv.fs_lstat(path)) then return false end
  return os.remove(path)
end
return M
