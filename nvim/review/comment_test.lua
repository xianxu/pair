local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local C=dofile(here..'comment.lua')
vim.notify=function() end
local b=vim.api.nvim_create_buf(true,false)
vim.api.nvim_set_current_buf(b)
local win=vim.api.nvim_get_current_win()
vim.wo.conceallevel=0;vim.wo.concealcursor='n'
vim.api.nvim_buf_set_lines(b,0,-1,false,{'🤖[old]{answer}[typing]','plain'})
C.attach(b)
assert(vim.wo.conceallevel==2)
vim.api.nvim_win_set_cursor(win,{1,7})
vim.api.nvim_exec_autocmds('CursorMoved',{buffer=b})
assert(vim.wo.concealcursor=='nvic')
local m=C.layout(b)[0][1]
local col=vim.api.nvim_win_get_cursor(win)[2]
for _,r in ipairs(m.hidden) do assert(col<r[1] or col>=r[2],'cursor still hidden') end
vim.api.nvim_win_set_cursor(win,{2,0})
vim.api.nvim_exec_autocmds('CursorMoved',{buffer=b})
assert(vim.wo.concealcursor=='n')
-- External option updates while not forcing survive leave/retarget.
vim.wo.concealcursor='i'
vim.api.nvim_win_set_cursor(win,{1,0})
vim.api.nvim_exec_autocmds('CursorMoved',{buffer=b})
assert(vim.wo.concealcursor=='nvic')
C.detach(b)
assert(vim.wo.concealcursor=='i' and vim.wo.conceallevel==0)
assert(vim.fn.maparg('<CR>','n',false,true).buffer~=1)
C.attach(b); C.attach(b)
local before=vim.api.nvim_buf_get_changedtick(b)
C.render(b); C.render(b)
assert(vim.api.nvim_buf_get_changedtick(b)==before,'render changes source')
-- Bounded projection falls back to raw and recovers after shrink.
local lines={};for i=1,1001 do lines[i]='🤖[x]{y}' end
vim.api.nvim_buf_set_lines(b,0,-1,false,lines)
C.render(b)
assert(next(C.layout(b))==nil,'large buffer was compacted')
vim.api.nvim_buf_set_lines(b,0,-1,false,{'🤖[x]{y}'})
C.render(b)
assert(C.layout(b)[0][1])
C.detach(b); C.detach(b)
print('ok comment attachment, cursor/options, idempotence, bounded projection')
