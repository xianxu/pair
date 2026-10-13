local case=assert(os.getenv('PAIR_TEST_DOM_CASE'))
local mode=case=='insert' and 'i' or 'n'
local body=(case=='empty' and '') or (case=='comments' and '=== keep') or 'draft'
local authored,sent,acks,preps={}, {},0,0
local composer=''
local ids={}
local write_failed=false
local ack_failed=false
local deferred
local guide='Recording: /store/webcast/recordings/recording-1\ncouch --broadcast-web-query /store/webcast/recordings/recording-1 notes'
vim.notify=function()end
_G.PairTestSessionLogAppend=function(text)
 assert(not vim.bo.modifiable and _G.PairStatusline():find('sending to',1,true),'append preceded sending status')
 authored[#authored+1]=text;return true end
_G.PairTestSessionLogCommit=function()return true end
_G.PairTestDOMPin=function()
 assert(not vim.bo.modifiable and _G.PairStatusline():find('preparing submission',1,true),'pin preceded visible lock/status')
 return {pane_id=7,validate=function()return true end} end
_G.PairTestDOMRPC=function(op,id,cb)
 if op=='prepare' then
  preps=preps+1;ids[#ids+1]=id
  if case=='no_notes' then cb({state='empty'});return end
  if case=='pre_error' and preps==1 then cb(nil,'unexpected submission entry .DS_Store');return end
  if case=='failure' then cb({state='failed',error='page changed'});return end
  if case=='changed' or case=='progress' or case=='timeout' then deferred=cb;return end
  cb({state='prepared',slot='pair:6',path='/store/'..id..'/submission.json',how_to_read=guide})
 elseif op=='begin' then cb({state='delivering'})
 elseif op=='ack' then
  acks=acks+1
  if case=='ack' and not ack_failed then ack_failed=true;cb(nil,'lost');return end
  cb({state='resolved'})
 else error('unexpected '..op) end
end
_G.PairTestZellijExecutor=function(label,argv)
 if case=='no_notes' and label:find('focus',1,true) then return {code=0} end
 assert(not label:find('focus',1,true),'DOM submission moved focus')
 if case~='no_notes' then assert(argv[4]=='--pane-id' and argv[5]=='7') end
 if label=='draft.send.write-body' then
  composer=composer..assert(_G.PairDraftSend.unframe(argv[case=='no_notes' and 4 or 6]))
  if (case=='uncertain' or case=='no_notes') and not write_failed then write_failed=true;return {code=17}end
 elseif label=='draft.send.submit' then sent[#sent+1]=composer;composer=''
 else error('unexpected '..label) end
 return {code=0}
end
vim.api.nvim_buf_set_lines(0,0,-1,false,{body})
if case=='queue' then
 vim.fn.maparg('<M-Right>','n',false,true).callback()
 body='queued'
 assert(vim.api.nvim_get_current_line()==body)
end
local deadline
_G.PairTestDOMArm=function(fn)deadline=fn;return function()end end
local map=vim.fn.maparg('<C-S-M-CR>',mode,false,true)
assert(type(map.callback)=='function','DOM chord absent in '..mode)
map.callback()
if case=='pre_error' or case=='uncertain' or case=='no_notes' then
 assert(#sent==0 and vim.api.nvim_get_current_line()==body)
 vim.api.nvim_buf_set_lines(0,0,-1,false,{'edited current'})
 map.callback()
 assert(#sent==1 and authored[#authored]=='edited current' and preps==2 and ids[1]~=ids[2])
 assert(sent[1]:find('edited current',1,true) and acks==(case=='no_notes' and 0 or 1))
elseif case=='changed' or case=='progress' or case=='timeout' then
 assert(not vim.bo.modifiable,'draft not locked during preparation')
 assert(_G.PairStatusline():find('preparing submission',1,true),'progress missing')
 assert(_G.PairDOMStatus.buf==vim.api.nvim_get_current_buf())
 local ok=pcall(vim.api.nvim_buf_set_lines,0,0,-1,false,{'edited'})
 assert(not ok,'editable during active operation')
 map.callback();assert(preps==1 and _G.PairStatusline():find('already sending',1,true))
 if case=='timeout' then deadline() else deferred({state='failed',error='fixture capture failure 50%'}) end
 assert(vim.bo.modifiable,'failure retained lock')
 assert(_G.PairStatusline():find(case=='timeout' and 'timed out' or 'fixture capture failure',1,true))
 vim.api.nvim_buf_set_lines(0,0,-1,false,{'edited'})
 deferred({state='prepared',path='/store/bundle/submission.json'})
 assert(#sent==0 and #authored==0)
 map.callback()
 deferred({state='prepared',path='/store/new/submission.json',how_to_read=guide})
 assert(#sent==1 and authored[1]=='edited' and ids[1]~=ids[2])
elseif case=='failure' then
 assert(#sent==0 and #authored==0 and vim.api.nvim_get_current_line()==body)
else
 assert(#sent==1 and #authored==1 and authored[1]==body)
 assert(sent[1]:find('/submission.json',1,true))
 assert(sent[1]:find('crop-N.png',1,true) and sent[1]:sub(-#guide)==guide,'agent prompt lost canonical recording guide')
 assert(not sent[1]:find('=== keep',1,true))
 if case=='ack' then
  vim.api.nvim_buf_set_lines(0,0,-1,false,{'next draft'})
  map.callback()
  assert(#sent==2 and preps==2 and authored[2]=='next draft' and ids[1]~=ids[2])
  assert(vim.wait(2000,function()return acks>=3 end,10),'old acknowledgment was lost')
  assert(vim.api.nvim_get_current_line()~='next draft')
 end
 if case=='queue' then
  local path=assert(os.getenv('PAIR_QUEUE_DIR'))
  assert(vim.fn.filereadable(path..'/500000.md')==0,'sent queue item not consumed')
  local files=vim.fn.readdir(path)
  assert(#files==1 and table.concat(vim.fn.readfile(path..'/'..files[1]),'\n')=='draft','star draft not preserved')
 end
 if case=='comments' then assert(vim.api.nvim_buf_get_lines(0,0,-1,false)[1]=='=== keep') end
end
vim.cmd('qall!')
