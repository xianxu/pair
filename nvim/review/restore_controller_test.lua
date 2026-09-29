local C = dofile('nvim/review/restore_controller.lua')
local tmp = vim.fn.tempname(); vim.fn.mkdir(tmp, 'p'); tmp=vim.uv.fs_realpath(tmp)
vim.fn.writefile({'A'}, tmp .. '/a.md'); vim.fn.writefile({'B'}, tmp .. '/b.md')
vim.fn.writefile({'unverified startup bytes'},tmp..'/initial.md')
vim.cmd.edit(tmp..'/initial.md');local initialbuf=vim.api.nvim_get_current_buf()
local initial={status='resolved',repo=tmp,branch='review/initial',file='initial.md',head='initial',snapshot='captured startup\n'}
local startup=C.new({});startup:init(initialbuf,initial,true)
assert(vim.api.nvim_get_current_line()=='captured startup','initial binding accepted pre-snapshot file bytes')
assert(not vim.bo[initialbuf].modified)
vim.api.nvim_buf_set_lines(initialbuf,0,-1,false,{'operator edit'})
assert(not pcall(function() C.new({}):init(initialbuf,initial,true) end),'startup overwrote modified operator buffer')
assert(vim.api.nvim_get_current_line()=='operator edit');vim.bo[initialbuf].modified=false;startup:close()

local a = {status='resolved',repo=tmp,branch='review/a',file='a.md',head='1',snapshot='A\n'}
local b = {status='resolved',repo=tmp,branch='review/b',file='b.md',head='2'}
local current = a
local starts, stops, pending = 0, 0, nil
local ctl = C.new({ session='session', resolve=function(_,_,snapshot)
    local value=vim.deepcopy(current)
    if snapshot then value.snapshot=table.concat(vim.fn.readfile(tmp..'/'..value.file,'b'),'\n') end
    return value
  end,
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
local notices={}; local original_notify=vim.notify
vim.notify=function(message) notices[#notices+1]=message end
ctl.opts.resolve=function() return {status='invalid',diagnostic='staged changes: commit or unstage before retrying'} end
assert(not ctl:guard(abuf))
assert(notices[1] and notices[1]:find('commit or unstage',1,true),'guard must retain actionable resolver diagnostic')
vim.notify=original_notify; ctl.opts.resolve=original_resolve
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
assert(ctl:context(abuf) and ctl:context(abuf).activation==old.activation,'retained buffer lost its ownership')
assert(not ctl:guard(abuf,true),'inactive review buffer may not write into another checkout')
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
ctl.opts.start=function()end
ctl.opts.resolve=function(_,_,snapshot)
  local value=vim.deepcopy(b)
  if snapshot then
    value.snapshot='B captured\n'
    vim.fn.writefile({'unverified later checkout'},tmp..'/b.md')
  end
  return value
end
req.identity=b
local captured=ctl:request(req);assert(captured.ok,captured.error)
assert(vim.api.nvim_get_current_line()=='B captured','activation used bytes read after identity-bound snapshot: '..vim.api.nvim_get_current_line())
ctl.opts.resolve=function()local value=vim.deepcopy(a);value.snapshot=nil;return value end
req.identity=a
assert(not ctl:request(req).ok,'activation accepted an identity without bounded captured bytes')
assert(vim.api.nvim_get_current_line()=='B captured','missing snapshot changed active document')
ctl:close(); vim.fn.delete(tmp,'rf')
print('restore_controller_test ok')
