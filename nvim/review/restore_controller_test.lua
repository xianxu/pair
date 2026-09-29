local C = dofile('nvim/review/restore_controller.lua')
local tmp = vim.fn.tempname(); vim.fn.mkdir(tmp, 'p'); tmp=vim.uv.fs_realpath(tmp)
vim.fn.writefile({'A'}, tmp .. '/a.md'); vim.fn.writefile({'B'}, tmp .. '/b.md')
local a = {status='resolved',repo=tmp,branch='review/a',file='a.md',head='1'}
local b = {status='resolved',repo=tmp,branch='review/b',file='b.md',head='2'}
local current = a
local starts, stops, pending = 0, 0, nil
local ctl = C.new({ session='session', resolve=function() return current end,
  start=function() starts=starts+1 end, stop=function() stops=stops+1 end,
  pending=function() return pending end, reconstruct=function() end })
vim.cmd.edit(tmp .. '/a.md'); local abuf=vim.api.nvim_get_current_buf()
assert(not pcall(function() C.new({}):init(abuf,b,true) end),'startup must not bind B context to A buffer')
ctl:init(abuf, a)
local original_resolve=ctl.opts.resolve
ctl.opts.resolve=function(_,selected)
  if not selected then return {status='missing',repo=tmp,branch='review/a',head='empty-human-round'} end
  if selected.head~='empty-human-round' then return {status='invalid'} end
  return vim.tbl_extend('force',a,{head=selected.head})
end
assert(ctl:guard(abuf,true),'established active selection must survive an empty first human round')
ctl.opts.resolve=original_resolve
local old = ctl:context(abuf)
current=b
vim.api.nvim_buf_set_lines(abuf,0,-1,false,{'unsaved A'})
local req={token=ctl.token,session='session',identity=b}
assert(not ctl:request(req).ok and starts==0 and stops==0)
assert(vim.fn.readfile(tmp..'/a.md')[1]=='A')
vim.api.nvim_buf_set_lines(abuf,0,-1,false,{'A'}); vim.bo[abuf].modified=false
pending='agent working'; assert(not ctl:request(req).ok)
pending=nil
local r=ctl:request(req); assert(r.ok and not r.same, r.error)
assert(vim.api.nvim_buf_get_name(0)==tmp..'/b.md' and starts==1 and stops==1)
assert(not ctl:admit({context=old,records={}},vim.api.nvim_get_current_buf()))
local repeated=ctl:request(req); assert(repeated.ok and repeated.same and starts==1)
vim.fn.writefile({'A newer'},tmp..'/a.md'); current=a; req.identity=a
r=ctl:request(req); assert(r.ok, r.error)
assert(vim.api.nvim_get_current_buf()==abuf and vim.api.nvim_get_current_line()=='A newer')
assert(not vim.bo[abuf].modified)
assert(not ctl:request({token='wrong',session='session',identity=a}).ok)
current=b; req.identity=b
ctl.opts.start=function(_,file) if file==tmp..'/b.md' then error('injected activation failure') end end
local prior=ctl:context()
assert(not ctl:request(req).ok)
assert(ctl:context().activation==prior.activation and vim.api.nvim_get_current_buf()==abuf,'failed activation did not restore prior pane')
current=b; assert(not ctl:guard(abuf,true))
ctl:close(); vim.fn.delete(tmp,'rf')
print('restore_controller_test ok')
