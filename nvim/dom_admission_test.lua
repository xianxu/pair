local transport=dofile('nvim/dom_transport.lua')
local send=dofile('nvim/draft_send.lua')
local function pane(id,command,tab)
 return {id=id,is_plugin=false,is_floating=false,is_focused=false,is_fullscreen=false,terminal_command=command,tab_id=tab or 2,tab_position=tab or 2}
end
local function fixture(reply)
 local rows={pane(1,'nvim -u /pair/nvim/init.lua'),pane(7,'pair wrap codex')}
 local reads,effects=0,{}
 local fake={env={ZELLIJ_PANE_ID='1'},json=vim.json,schedule=function(f)f()end}
 fake.system=function()local r={code=0,stdout=vim.json.encode({Submission={state='delivering'}})};return {wait=function()return r end}end
 local target=transport.new(fake,function(_,argv,opts)
  assert(argv[3]=='list-panes');reads=reads+1
  return {code=0,stdout=reply and reply(reads,rows) or vim.json.encode(rows)}
 end)
 return target,rows,function()return reads,effects end,function(label,argv)effects[#effects+1]=argv;return {code=0}end
end
for _,kind in ipairs({'wrong-kind','missing-tab','foreign-tab','duplicate-agent','changed-agent','changed-source'}) do
 local t,rows,counts,action=fixture(function(n,p)
  if kind=='wrong-kind' then return '{}' end
  if kind=='missing-tab' then p[2].tab_id=nil end
  if kind=='foreign-tab' then p[2].tab_id=3;p[2].tab_position=3 end
  if kind=='duplicate-agent' then p[3]=pane(8,'pair wrap claude') end
  if n==2 and kind=='changed-agent' then p[2].id=8 end
  if n==2 and kind=='changed-source' then p[1].tab_id=3;p[1].tab_position=3 end
  return vim.json.encode(p)
 end)
 local target=t.pin('id')
 if kind=='changed-agent' or kind=='changed-source' then
  assert(target,kind..' requires healthy pin');assert(not target.validate(),kind..' admitted changed context')
 else
  assert(not target,kind..' admitted malformed or ambiguous pin')
 end
 local n,effects=counts();assert(#effects==0,kind..' produced native effects')
 assert(n==((kind=='changed-agent' or kind=='changed-source') and 2 or 1),kind..' retried invalid evidence')
end

-- Preparation may outlive five seconds; admission starts only before write,
-- and its deadline cannot restart before submit.
do
 local original=vim.loop.hrtime
 local clock=0
 vim.loop.hrtime=function()return clock*1000000 end
 local ok,err=xpcall(function()
  local t,rows,counts=fixture()
  local parent={remaining=function()return 36000-clock end,check=function()assert(clock<36000,'transaction expired')end}
  local target=assert(t.pin('id',parent))
  clock=10000
  local calls=0
  local sent,phase=send.send('body',function()
   calls=calls+1;clock=15001;return {code=0}
  end,function()end,'start',target)
  assert(not sent and phase=='written' and calls==1,'submit must share first validation deadline')
  assert(counts()==2,'expired submit must not launch another native query')
 end,debug.traceback)
 vim.loop.hrtime=original
 assert(ok,err)
end

-- A rejected target cannot recover authority from a later good observation.
do
 local t,_,counts=fixture(function(n,rows)return n==2 and '{}' or vim.json.encode(rows)end)
 local target=assert(t.pin('id'))
 assert(not target.validate(),'first unknown should refuse')
 assert(not target.validate(),'later good evidence must not revive refused target')
 assert(counts()==2,'terminal refusal launched a replacement observation')
 local invoked=false
 assert(not target.effect(function()invoked=true;return {code=0}end) and not invoked,'refused target retained effect authority')
end

-- A pin owns the independently known native environment through preparation,
-- recipient/native reads and effects. Restoring it cannot revive a refused pin.
for _,key in ipairs({'ZELLIJ_PANE_ID','ZELLIJ_SESSION_NAME'}) do
 for _,where in ipairs({'before-validate','candidate-query','begin','pane-query','before-effect'}) do
  local env={ZELLIJ_PANE_ID='1',ZELLIJ_SESSION_NAME='test'}
  local original=env[key];local reads,begins,effects=0,0,0
  local rows={pane(1,'nvim -u /pair/nvim/init.lua'),pane(7,'pair wrap codex')}
  local v={env=env,json=vim.json,schedule=function(f)f()end}
  v.system=function()
   begins=begins+1
   return {wait=function()
    if where=='begin' then env[key]='changed' end
    return {code=0,stdout=vim.json.encode({Submission={state='delivering'}})}
   end}
  end
  local t=transport.new(v,function()
   reads=reads+1
   if (where=='candidate-query' and reads==1) or (where=='pane-query' and reads==2) then env[key]='changed' end
   return {code=0,stdout=vim.json.encode(rows)}
  end)
  local target=t.pin('id')
  if where=='candidate-query' then
   assert(not target,key..' candidate context change admitted pin')
  else
   assert(target)
   if where=='before-validate' then env[key]='changed' end
   local called,valid=pcall(target.validate)
   if where=='before-effect' then assert(called and valid);env[key]='changed'
   else assert(not called or not valid,key..' '..where..' validated changed context') end
   local result=target.effect(function()effects=effects+1;return {code=0}end)
   assert(not result and effects==0,key..' '..where..' reached native effect')
   local beforeReads,beforeBegins=reads,begins
   env[key]=original
   assert(not target.validate(),key..' '..where..' restored context revived pin')
   assert(reads==beforeReads and begins==beforeBegins,'terminal context refusal retried')
   if where=='before-validate' then assert(reads==1 and begins==0,'changed context reached recipient query')end
   if where=='begin' then assert(reads==1,'changed begin context reached pane query')end
  end
 end
end

print('dom_admission_test ok')
