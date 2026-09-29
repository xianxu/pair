package.path = './nvim/?.lua;' .. package.path
local ok, recovery = pcall(require, 'review.recovery')
assert(ok, 'recovery module must exist')
local uv = vim.uv
local root = vim.fn.tempname()
assert(uv.fs_mkdir(root, 448))
local ctx = { repo = '/canonical/repo', branch = 'review/a', file = 'doc.md', activation = 'one' }
local function buf(lines, eol)
  local b = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_buf_set_lines(b, 0, -1, false, lines)
  vim.bo[b].eol = eol
  return b
end
local function read(path)
  local fd = assert(uv.fs_open(path, 'r', 0))
  local data = assert(uv.fs_read(fd, assert(uv.fs_fstat(fd)).size, 0))
  uv.fs_close(fd)
  return data
end
local dir, b = root .. '/recovery', buf({ 'original', '', 'last' }, false)
local path = assert(recovery.save(dir, ctx, b, 'process-a'))
assert(uv.fs_stat(dir).mode % 512 == 448)
assert(uv.fs_stat(path).mode % 512 == 384)
local original = read(path)
assert(recovery.inspect(dir,ctx).path==path,'activation must discover available recovery without consuming it')
assert(read(path)==original)
local other = buf({ 'replacement' }, true)
assert(not recovery.save(dir, ctx, other, 'process-b'), 'prior owner must survive')
assert(read(path) == original)
local changed = vim.deepcopy(ctx)
changed.activation = 'new-process'
local restored = assert(recovery.restore(dir, changed, other))
assert(restored.path == path and vim.bo[other].modified and not vim.bo[other].eol)
assert(vim.deep_equal(vim.api.nvim_buf_get_lines(other, 0, -1, false), { 'original', '', 'last' }))
assert(uv.fs_stat(path), 'reading must not consume recovery')
vim.api.nvim_buf_set_lines(other, 0, -1, false, { 'recovered and edited' })
assert(recovery.saved(dir, ctx, other, 'process-b'))
assert(not uv.fs_stat(path))
assert(recovery.save(dir, ctx, b, 'process-a'))
assert(recovery.saved(dir, ctx, other, 'process-b') == false, 'unrelated buffer cannot clear recovery')
assert(recovery.saved(dir, ctx, other, 'process-a') == false, 'same process different buffer cannot clear recovery')
local old = read(path)
local rename = uv.fs_rename
uv.fs_rename = function() return nil, 'injected rename failure' end
vim.api.nvim_buf_set_lines(b, 0, -1, false, { 'new text' })
assert(not recovery.save(dir, ctx, b, 'process-a'))
uv.fs_rename = rename
assert(read(path) == old, 'atomic write failure must retain prior snapshot')
vim.api.nvim_buf_set_lines(b, 0, -1, false, { string.rep('x', 8 * 1024 * 1024) })
assert(not recovery.save(dir, ctx, b, 'process-a'))
assert(read(path) == old)
assert(recovery.discard(dir, ctx))
assert(uv.fs_symlink(root .. '/missing', path))
assert(not recovery.save(dir, ctx, other, 'process-b'))
assert(not recovery.restore(dir, ctx, other))
assert(not recovery.discard(dir, ctx))
assert(uv.fs_unlink(path))
local fd = assert(uv.fs_open(path, 'w', 384))
assert(uv.fs_write(fd, '{bad json', 0)); uv.fs_close(fd)
assert(not recovery.restore(dir, ctx, other))
assert(not recovery.save(dir, ctx, other, 'process-b'))
assert(recovery.discard(dir, ctx), 'explicit discard removes malformed regular snapshots')
assert(uv.fs_mkdir(path, 448))
assert(not recovery.save(dir, ctx, other, 'process-b'))
assert(uv.fs_rmdir(path))
assert(uv.fs_symlink(dir, root .. '/linked'))
assert(not recovery.save(root .. '/linked', ctx, other, 'process-b'))
for i = 1, 32 do
  local c = vim.deepcopy(ctx)
  c.file = 'doc' .. i
  assert(recovery.save(dir, c, other, 'process-b'))
end
assert(not recovery.save(dir, ctx, other, 'process-b'), 'capacity must refuse without eviction')
-- Separate Neovim processes prove disk recovery rather than an in-memory cache.
local script = root .. '/restart.lua'
vim.fn.writefile({
  "package.path = './nvim/?.lua;' .. package.path",
  "local r = require('review.recovery')",
  'local c = vim.json.decode(' .. string.format('%q', vim.json.encode(ctx)) .. ')' ,
  'local d = ' .. string.format('%q', root .. '/restart'),
  "local b = vim.api.nvim_create_buf(true, false)",
  "if vim.env.RECOVERY_WRITE == '1' then",
  "vim.api.nvim_buf_set_lines(b, 0, -1, false, {'durable', ''})",
  "vim.bo[b].eol = false; assert(r.save(d,c,b,'old-process'))",
  "else",
  "assert(not r.save(d,c,b,'new-process'))",
  "assert(r.restore(d,c,b)); assert(vim.deep_equal(vim.api.nvim_buf_get_lines(b,0,-1,false), {'durable',''})); assert(not vim.bo[b].eol)",
  "assert(r.saved(d,c,b,'new-process'))",
  'end',
}, script)
for _, write in ipairs({ '1', '0' }) do
  local result = vim.system({ vim.v.progpath, '-l', script }, { env = { RECOVERY_WRITE = write }, text = true }):wait()
  assert(result.code == 0, result.stderr)
end
-- Recovery carries write-affecting document options, independently of the
-- currently checked-out document's format. No-EOL survives an ordinary write.
local formatctx=vim.deepcopy(ctx); formatctx.file='format.md'
local source=buf({'recovered','tail'},false)
vim.bo[source].fileformat='dos'; vim.bo[source].fileencoding='utf-8'; vim.bo[source].bomb=true
local formatdir=root..'/formats'
local formatpath=assert(recovery.save(formatdir,formatctx,source,'format-owner'))
local destination=buf({'disk'},true)
vim.bo[destination].fileformat='unix'; vim.bo[destination].fileencoding='latin1'; vim.bo[destination].bomb=false
assert(recovery.restore(formatdir,formatctx,destination))
assert(vim.bo[destination].fileformat=='dos' and vim.bo[destination].fileencoding=='utf-8' and vim.bo[destination].bomb,
  'recovery must restore original byte-writing options')
local recoveredfile=root..'/recovered.md'
vim.api.nvim_buf_set_name(destination,recoveredfile)
vim.api.nvim_buf_call(destination,function() vim.cmd('silent write') end)
assert(read(recoveredfile)=='\239\187\191recovered\r\ntail','recovered byte format and no-EOL must survive write')
-- Old snapshots have no format metadata: retain destination interpretation,
-- while still honoring their explicit end-of-line flag.
local legacy=vim.json.decode(read(formatpath)); legacy.options=nil
local fd=assert(uv.fs_open(formatpath,'w',384)); assert(uv.fs_write(fd,vim.json.encode(legacy),0)); uv.fs_close(fd)
vim.bo[destination].fileformat='unix'; vim.bo[destination].fileencoding='utf-8'; vim.bo[destination].bomb=false
assert(recovery.restore(formatdir,formatctx,destination))
assert(vim.bo[destination].fileformat=='unix' and not vim.bo[destination].bomb,'legacy snapshots retain loaded document format')
vim.bo[source].fileencoding='utf-16le'
assert(not recovery.save(formatdir,formatctx,source,'format-owner'),'unsupported output encoding must fail closed')
vim.fn.delete(root, 'rf')
print('recovery tests passed')
