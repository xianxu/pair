local case=assert(os.getenv('PAIR_TEST_DOM_CASE'))
local mode=case=='insert' and 'i' or 'n'
local body=(case=='empty' and '') or (case=='comments' and '=== keep') or 'draft'
local authored,sent,acks,preps={}, {},0,0
local composer=''
local ack_failed=false
local deferred
vim.notify=function()end
_G.PairTestSessionLogAppend=function(text)authored[#authored+1]=text;return true end
_G.PairTestSessionLogCommit=function()return true end
_G.PairTestDOMPin=function()return {pane_id=7,validate=function()return true end} end
_G.PairTestDOMRPC=function(op,id,cb)
 if op=='prepare' then
  preps=preps+1
  if case=='failure' then cb({state='failed',error='page changed'});return end
  if case=='changed' then deferred=cb;return end
  cb({state='prepared',path='/store/'..id..'/submission.json'})
 elseif op=='begin' then cb({state='delivering'})
 elseif op=='ack' then
  acks=acks+1
  if case=='ack' and not ack_failed then ack_failed=true;cb(nil,'lost');return end
  cb({state='resolved'})
 else error('unexpected '..op) end
end
_G.PairTestZellijExecutor=function(label,argv)
 assert(not label:find('focus',1,true),'DOM submission moved focus')
 assert(argv[4]=='--pane-id' and argv[5]=='7')
 if label=='draft.send.write-body' then composer=composer..assert(_G.PairDraftSend.unframe(argv[6]))
 elseif label=='draft.send.submit' then sent[#sent+1]=composer;composer=''
 else error('unexpected '..label) end
 return {code=0}
end
vim.api.nvim_buf_set_lines(0,0,-1,false,{body})
local map=vim.fn.maparg('<C-S-M-CR>',mode,false,true)
assert(type(map.callback)=='function','DOM chord absent in '..mode)
map.callback()
if case=='changed' then
 vim.api.nvim_buf_set_lines(0,0,-1,false,{'edited'})
 deferred({state='prepared',path='/store/bundle/submission.json'})
 assert(#sent==0 and #authored==0)
elseif case=='failure' then
 assert(#sent==0 and #authored==0 and vim.api.nvim_get_current_line()==body)
else
 assert(#sent==1 and #authored==1 and authored[1]==body)
 assert(sent[1]:find('/submission.json',1,true))
 assert(not sent[1]:find('=== keep',1,true))
 if case=='ack' then
  vim.api.nvim_buf_set_lines(0,0,-1,false,{'next draft'})
  map.callback()
  assert(#sent==1 and preps==1 and acks==2)
  assert(vim.api.nvim_get_current_line()=='next draft')
 end
 if case=='comments' then assert(vim.api.nvim_buf_get_lines(0,0,-1,false)[1]=='=== keep') end
end
vim.cmd('qall!')
