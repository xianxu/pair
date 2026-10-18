local path = vim.fn.getcwd() .. '/nvim/'
package.path = path .. '?.lua;' .. package.path
local m = require('zellij_observation')
local function absent(value, reason) assert(value == nil and type(reason) == 'string', tostring(reason)) end
local corpus=vim.json.decode(table.concat(vim.fn.readfile('tests/fixtures/zellij-replies.json'),'\n'))
for _,c in ipairs(corpus.cases) do
 local value,reason=m[c.kind](c.input,c.requirements)
 assert((value~=nil)==c.valid,c.name..': '..tostring(reason))
end
for _,raw in ipairs({
 '[{"id":1,"is_plugin":false,"\\u0069d":2}]',
 '[{"id":1,"is_plugin":false,"extension":{"a":1,"\\u0061":2}}]',
 '[{"id":9007199254740993,"is_plugin":false}]',
 '[{"id":9007199254740992,"is_plugin":false}]',
 '[{"id":1e0,"is_plugin":false}]',
 '[{"id":1,"is_plugin":false,"extension":{}}] trailing',
 '{"0":{}}','[[]]','null',
}) do absent(m.panes(raw)) end
assert(m.panes('[{"id":4294967295,"is_plugin":false}]')[1].id==4294967295)
assert(m.panes('[{"pane_id":"terminal_7","is_plugin":false}]')[1].id==7)
assert(m.panes('[{"id":1,"is_plugin":false,"terminal_command":null}]')[1].terminal_command=='')
assert(m.tab('{"active_swap_layout_name":null,"is_swap_layout_dirty":false}').active_swap_layout_name==vim.NIL)
local clock=0
local real=vim.loop.hrtime
vim.loop.hrtime=function()return clock*1000000 end
local b=m.budget()
assert(b.remaining()==5000)
local result={code=2,stdout='false',stderr='diagnostic',signal=0}
local calls=0
assert(m.query(b,'label',{'zellij'},function(label,argv,opts)
 calls=calls+1;assert(label=='label' and argv[1]=='zellij' and opts.timeout_ms==5000);return result
end)==result)
clock=5000
absent(m.query(b,'expired',{},function()error('spawned after expiry')end))
local invoked=false
absent(m.effect(b,function()invoked=true end));assert(not invoked)
clock=0;b=m.budget()
absent(m.query(b,'late',{},function()clock=5000;return result end))
clock=0
local cancelled=false
local parent={remaining=function()return 123 end,check=function()assert(not cancelled,'parent cancelled')end}
b=m.budget(parent);assert(b.remaining()==123)
cancelled=true;assert(not pcall(b.check))
absent(m.effect(b,function()error('cancelled parent dispatched effect')end))
cancelled=false
absent(m.query(b,'cancel-during-action',{},function()cancelled=true;return result end))
cancelled=false
assert(m.budget({remaining=function()return 10000 end,check=function()end}).remaining()==5000)
clock=25;assert(b.remaining()==123);clock=0
local function exercise(stop)
 clock=0;local reads,projects=0,0;local cancel=false
 local token=m.budget({remaining=function()return 5000 end,check=function()assert(not cancel,'cancelled')end})
 if stop=='before-read' then cancel=true end
 local value,reason=m.confirm(token,function()
  reads=reads+1
  if stop=='read'..reads then cancel=true end
  if stop=='unknown' then return nil,'unknown' end
  return {n=reads}
 end,function(raw)
  projects=projects+1
  if stop=='project'..projects then cancel=true end
  if stop=='invalid' then return nil,'invalid' end
  return {target=stop=='mismatch' and raw.n or 7}
 end)
 if stop=='healthy' then assert(value.target==7 and reads==2 and projects==2)
 else absent(value,reason) end
 if stop=='before-read' then assert(reads==0 and projects==0) end
 if stop=='unknown' then assert(reads==1 and projects==0) end
 if stop=='invalid' or stop=='project1' then assert(reads==1 and projects==1) end
 if stop=='read1' then assert(reads==1 and projects==0) end
 if stop=='read2' then assert(reads==2 and projects==1) end
end
for _,stage in ipairs({'healthy','before-read','read1','read2','project1','project2','unknown','invalid','mismatch'})do exercise(stage)end
clock=0;b=m.budget()
local a,z=m.effect(b,function()clock=6000;return 'completed',false end)
assert(a=='completed' and z==false)
for _,n in ipairs({20,40})do
 clock=0;local reads,effects=0,0
 for _=1,n do
  local token=m.budget()
  assert(m.confirm(token,function()
   return m.query(token,'count',{},function()reads=reads+1;return {code=0,stdout='same'}end)
  end,function(raw)return {raw.stdout}end))
  assert(m.effect(token,function()effects=effects+1;return true end))
 end
 assert(reads==2*n and effects==n)
end
-- A token can expire between the preflight check and timeout collection.
clock=0
local throws_ok,unknown,spawn_reason=pcall(m.query,m.budget(),'throw',{},function()error('spawn failed')end)
assert(throws_ok and unknown==nil and spawn_reason:find('spawn failed',1,true),'query exception escaped nil/reason API')
local kept,kept_reason=m.query(m.budget(),'kept',{},function()return result,'kept reason'end)
assert(kept==result and kept_reason=='kept reason')
local effect_ok,effect_err=pcall(m.effect,m.budget(),function()error('dispatch uncertain')end)
assert(not effect_ok and effect_err:find('dispatch uncertain',1,true),'dispatched effect exception lost uncertainty')
local timeout_reads=0
absent(m.query({check=function()end,remaining=function()return 0 end},'zero',{},function()timeout_reads=timeout_reads+1;return result end))
assert(timeout_reads==0,'zero remaining timeout reached action')
local second
clock=0
absent(m.confirm(m.budget(),function()return result end,function()return false end))
assert(m.confirm(m.budget(),function()return 'raw'end,function()second={target=7};return second end)==second)
local returns={m.effect(m.budget(),function()return false,'kept',3 end)}
assert(returns[1]==false and returns[2]=='kept' and returns[3]==3)
-- Environment capture decorates existing tokens without changing a parent's
-- lifetime or introducing a new deadline. Reuse one snapshot across phases.
do
 local env={ZELLIJ_PANE_ID='1',ZELLIJ_SESSION_NAME='test'}
 local checks=0
 local token={remaining=function()return 12345 end,check=function()checks=checks+1 end}
 local original_check=token.check
 local guard=m.capture_environment(env)
 local wrapped=guard(token)
 assert(wrapped~=token and wrapped.remaining==token.remaining and wrapped.remaining()==12345)
 wrapped.check()
 env.ZELLIJ_SESSION_NAME='changed'
 assert(not pcall(wrapped.check),'changed environment retained authority')
 assert(token.check==original_check and pcall(token.check),'environment decorator mutated preparation token')
 assert(not pcall(guard({remaining=function()return 5000 end,check=function()end}).check),'phase transition recaptured changed environment')
end
-- Instruction work, rather than wall time, checks the duplicate-key scanner's
-- repeated extension traversal without exposing production instrumentation.
local function scanner_work(n)
 local fields={}
 for i=1,n do fields[i]='"extension'..i..'":{"value":null,"escaped\\u006bey":[]}' end
 local raw='[{"id":1,"is_plugin":false,'..table.concat(fields,',')..'}]'
 local work=0
 debug.sethook(function()work=work+1 end,'',100)
 local parsed,reason=m.panes(raw)
 debug.sethook()
 assert(parsed,reason)
 return work
end
local jit_enabled=jit.status()
jit.off();jit.flush()
local nwork,doublework=scanner_work(200),scanner_work(400)
if jit_enabled then jit.on() end
assert(doublework>nwork and doublework<nwork*2.5,'scanner work grew superlinearly: '..nwork..' / '..doublework)
vim.loop.hrtime=real
print('zellij observation tests passed')
