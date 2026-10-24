-- Headless production DOM transport -> native CLI seam -> shared stateful fake.
local root=vim.fn.getcwd()
local transport=dofile(root..'/nvim/dom_transport.lua')
local delivery=dofile(root..'/nvim/draft_send.lua')
local trace=dofile(root..'/nvim/zellij_trace.lua')
local dir=vim.fn.system({'mktemp','-d',(vim.env.TMPDIR or '/tmp')..'/p647-dom-cli.XXXXXX'}):gsub('%s+$','')
assert(vim.v.shell_error==0 and dir~='','private fixture root')
local old={}
for _,key in ipairs({'PATH','PAIR_FAKE_ZELLIJ_STATE','ZELLIJ_SESSION_NAME','PAIR_ZELLIJ_TRACE','HOME','ZELLIJ_SOCKET_DIR'}) do old[key]=vim.env[key] end
local function pane(id,command,tab)
 return {id=id,is_plugin=false,is_floating=false,is_focused=false,is_fullscreen=false,
 terminal_command=command,tab_id=tab or 2,tab_position=tab or 2}
end
local function model()
 return {sessions={test={panes={pane(1,'nvim -u /pair/nvim/init.lua'),pane(7,'pair wrap codex'),pane(9,'pair wrap claude',3)},
 clients={},tab={tab_id=2,position=2,active_swap_layout_name=vim.NIL,is_swap_layout_dirty=false}}}}
end
local function save(value)vim.fn.writefile({vim.json.encode(value)},dir..'/state.json')end
local function load()return vim.json.decode(table.concat(vim.fn.readfile(dir..'/state.json'),'\n'))end
local ok,err=xpcall(function()
 vim.fn.mkdir(dir..'/bin','p')
 local built=vim.system({'bash',root..'/tests/lib/pair-test-binary.sh',dir..'/bin/pair'},{text=true}):wait()
 assert(built.code==0,'Pair helper build failed: '..(built.stderr or ''))
 vim.fn.writefile({'#!/bin/sh','exec '..vim.fn.shellescape(vim.fn.exepath('python3'))..' '..vim.fn.shellescape(root..'/tests/lib/fake-zellij.py')..' "$@"'},dir..'/bin/zellij')
 assert(vim.fn.setfperm(dir..'/bin/zellij','rwx------')==1)
 vim.env.PATH=dir..'/bin:'..old.PATH
 vim.env.PAIR_FAKE_ZELLIJ_STATE=dir..'/state.json';vim.env.ZELLIJ_SESSION_NAME='test'
 vim.env.PAIR_ZELLIJ_TRACE='0';vim.env.HOME=dir;vim.env.ZELLIJ_SOCKET_DIR=dir..'/sockets'
 for _,case in ipairs({'healthy','wrong-first','wrong-second','changed-second','changed-presubmit','expired-local-fence','recipient-gone','expiry-after-write','context-before-validate','context-begin','context-pane-query','context-local-fence'}) do
  local initial=model();local faults={}
  if case=='wrong-first' or case=='wrong-second' then faults={{ordinal=case=='wrong-first' and 1 or 2,reply='{}'}} end
  if case=='changed-second' or case=='changed-presubmit' then
   local changed=model();changed.sessions.test.panes[2].id=8
   faults={{ordinal=case=='changed-second' and 2 or 4,model=changed}}
  end
  save({model=initial,faults=faults,queries={},effects={}})
  local remaining=36000
  local operation={remaining=function()return remaining end,check=function()assert(remaining>0,'submission timed out')end}
  local calls=0
  local env=setmetatable({ZELLIJ_SESSION_NAME='test'},{__index=function(local_env,key)
   if key=='ZELLIJ_PANE_ID' then
    if case=='expired-local-fence' and calls==2 then remaining=0 end
    if case=='context-local-fence' and calls==2 then local_env.ZELLIJ_SESSION_NAME='changed' end
    return '1'
   end
  end})
  local v={env=env,json=vim.json,schedule=function(fn)fn()end,system=function()
   local r={code=0,stdout=vim.json.encode({Submission={state=case=='recipient-gone' and 'failed' or 'delivering'}})}
   return {wait=function()
    if case=='context-begin' then env.ZELLIJ_SESSION_NAME='changed' end
    return r
   end}
  end}
  local function action(label,argv,opts)
   local result=trace.action(label,argv,opts)
   if argv[3]=='list-panes' then
    calls=calls+1
    if case=='context-pane-query' and calls==2 then env.ZELLIJ_SESSION_NAME='changed' end
   end
   return result
  end
  local t=transport.new(v,action)
  local target,why=t.pin('id',operation)
  if case=='context-before-validate' then env.ZELLIJ_SESSION_NAME='changed' end
  local sent,phase,reason=false,'start',why
  if target then sent,phase,reason=delivery.send('private body',action,function()if case=='expiry-after-write' then remaining=0 end end,'start',target) end
  local state=load()
  local wantEffects=case=='healthy' and 2 or ((case=='changed-presubmit' or case=='expiry-after-write') and 1 or 0)
  assert(#state.effects==wantEffects,case..' effects='..vim.inspect(state.effects)..' reason='..tostring(reason))
  assert(sent==(case=='healthy'),case..' success='..tostring(sent)..' reason='..tostring(reason))
  assert(phase==(case=='healthy' and 'dispatched' or (wantEffects==1 and 'written' or 'start')),case..' phase='..phase)
  local expectedQueries=({healthy=3,['wrong-first']=1,['wrong-second']=2,['changed-second']=2,['changed-presubmit']=3,['expired-local-fence']=2,['recipient-gone']=1,['expiry-after-write']=2,['context-before-validate']=1,['context-begin']=1,['context-pane-query']=2,['context-local-fence']=2})[case]
  assert(calls==expectedQueries,case..' pane queries='..calls)
  if wantEffects>0 then
   assert(state.effects[1].target=='terminal_7' and state.effects[1].action=='write-chars',case..' wrong addressed write')
   if case~='changed-presubmit' then assert(state.model.sessions.test.panes[2].input==delivery.frame('private body'),case..' body changed') end
  end
  if sent then assert(state.effects[2].action=='send-keys' and state.effects[2].target=='terminal_7' and state.effects[2].args[5]=='Alt Enter','wrong submit') end
 end
 -- Work per repeated decision remains linear; foreign-tab agents stay irrelevant.
 for _,n in ipairs({2,4}) do
  save({model=model(),faults={},queries={},effects={}})
  local v={env={ZELLIJ_PANE_ID='1'},json=vim.json,schedule=function(fn)fn()end,
    system=function()return {wait=function()return {code=0,stdout=vim.json.encode({Submission={state='delivering'}})}end}end}
  local t=transport.new(v,trace.action)
  for _=1,n do
   local target=assert(t.pin('id'))
   assert(delivery.send('private body',trace.action,function()end,'start',target))
  end
  local state=load()
  assert(#state.queries==5*n and #state.effects==2*n,'nonlinear query/effect work')
  assert(state.model.sessions.test.panes[2].input==string.rep(delivery.frame('private body'),n),'repeated body bytes changed')
 end
end,debug.traceback)
for key,value in pairs(old)do vim.env[key]=value end
for _,key in ipairs({'PAIR_FAKE_ZELLIJ_STATE','PAIR_ZELLIJ_TRACE','ZELLIJ_SESSION_NAME','ZELLIJ_SOCKET_DIR'})do if old[key]==nil then vim.env[key]=nil end end
vim.fn.delete(dir,'rf')
assert(ok,err)
print('dom_admission_cli_test ok')
