local C=dofile('nvim/review/restore_controller.lua')
local document=dofile('nvim/review/document_bytes.lua')
local root=vim.fn.tempname(); vim.fn.mkdir(root,'p'); root=vim.uv.fs_realpath(root)
local function write(path,bytes)
  local fd=assert(vim.uv.fs_open(path,'w',384)); assert(vim.uv.fs_write(fd,bytes,0)); assert(vim.uv.fs_close(fd))
end
local function read(path)
  local fd=assert(vim.uv.fs_open(path,'r',0)); local bytes=assert(vim.uv.fs_read(fd,assert(vim.uv.fs_fstat(fd)).size,0)); vim.uv.fs_close(fd); return bytes
end
write(root..'/base.md','base\n')
vim.cmd.edit(root..'/base.md')
local basebuf=vim.api.nvim_get_current_buf()
local observed={status='resolved',repo=root,branch='review/base',head='base',file='base.md'}
local ctl=C.new({session='test',resolve=function() return observed end})
ctl:init(basebuf,observed)
local cases={
  {name='crlf',bytes='first\r\nlast\r\n',lines={'first','last'},eol=true,format='dos'},
  {name='bom-crlf',bytes='\239\187\191first\r\nlast\r\n',lines={'first','last'},eol=true,format='dos',bomb=true},
  {name='bom-noeol',bytes='\239\187\191last',lines={'last'},eol=false,format='unix',bomb=true},
  {name='bom-empty',bytes='\239\187\191',lines={''},eol=false,format='unix',bomb=true},
  {name='lf',bytes='first\nlast\n',lines={'first','last'},eol=true,format='unix'},
  {name='empty',bytes='',lines={''},eol=false,format='unix'},
  {name='noeol',bytes='last',lines={'last'},eol=false,format='unix'},
  {name='crlf-noeol',bytes='first\r\nlast',lines={'first','last'},eol=false,format='dos'},
  {name='lone-cr',bytes='one\rtwo\r',lines={'one\rtwo\r'},eol=false,format='unix'},
  {name='mixed',bytes='one\r\ntwo\n',lines={'one\r','two'},eol=true,format='unix'},
  {name='trailing-cr',bytes='one\r\nlast\r',lines={'one','last\r'},eol=false,format='dos'},
}
local function activate(case,file,branch)
  observed={status='resolved',repo=root,branch='review/'..branch,head=branch,file=file}
  local reply=ctl:request({token=ctl.token,session='test',identity=observed})
  assert(reply.ok,reply.error)
  local buf=vim.api.nvim_get_current_buf()
  assert(vim.deep_equal(vim.api.nvim_buf_get_lines(buf,0,-1,false),case.lines),branch..': decoded lines differ')
  assert(vim.bo[buf].fileformat==case.format,branch..': fileformat differs')
  assert(vim.bo[buf].endofline==case.eol,branch..': endofline differs')
  assert(vim.bo[buf].bomb==(case.bomb or false),branch..': BOM differs')
  assert(vim.bo[buf].fileencoding=='utf-8',branch..': encoding differs')
  vim.cmd('silent write!')
  assert(read(root..'/'..file)==case.bytes,branch..': saved bytes differ')
end
for _,case in ipairs(cases) do
  assert(document.encode(assert(document.decode(case.bytes)).lines,assert(document.decode(case.bytes)))==case.bytes,case.name..': codec round trip differs')
  local file=case.name..'.md'
  write(root..'/'..file,case.bytes)
  activate(case,file,'new-'..case.name)
end
-- Branches reviewing the same retained pathname may change its line format.
-- Start with DOS, then LF, empty, and no-EOL to expose inherited-option bugs.
for _,case in ipairs(cases) do
  write(root..'/shared.md',case.bytes)
  activate(case,'shared.md','retained-'..case.name)
end
local previous=vim.api.nvim_get_current_buf()
for i,bytes in ipairs({'\255\254B\0','caf\233','\192\175','\237\160\128','\244\144\128\128','nul\0byte\n'}) do
  local file='unsupported'..i..'.md'; write(root..'/'..file,bytes)
  observed={status='resolved',repo=root,branch='review/unsupported'..i,head='invalid'..i,file=file}
  local reply=ctl:request({token=ctl.token,session='test',identity=observed})
  assert(not reply.ok and vim.api.nvim_get_current_buf()==previous,'unsupported encoding must refuse activation')
  assert(read(root..'/'..file)==bytes,'unsupported encoding refusal changed disk bytes')
end
ctl:close(); vim.fn.delete(root,'rf')
print('document byte activation tests passed')
