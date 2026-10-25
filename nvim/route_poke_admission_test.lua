local here = debug.getinfo(1, 'S').source:sub(2):match('(.*/)') or './'
local route = dofile(here .. 'workbench_route.lua')
local poke = dofile(here .. 'pair_poke.lua')
local trace = dofile(here .. 'zellij_trace.lua')
local state_path = assert(vim.env.PAIR_FAKE_ZELLIJ_STATE)
local function pane(id, command, tab, floating)
  return {id=id,is_plugin=false,is_floating=floating or false,is_fullscreen=false,is_focused=id==9,
    tab_id=tab or 2,tab_position=tab==0 and 0 or 1,terminal_command=command,title='cosmetic'}
end
local function rows()
  return {pane(4,'pair term'),pane(7,'pair wrap codex'),pane(42,'nvim -u /p/nvim/init.lua /d/draft.md'),
    pane(9,'nvim -u /p/nvim/review.lua /d/review.md',2,true),pane(77,'pair wrap claude',0)}
end
local function reset(ps,faults)
  local state={model={sessions={test={panes=ps or rows(),tab={tab_id=2},floating_visible=true}}},faults=faults or {},queries={},effects={},ordinal=0}
  vim.fn.writefile({vim.json.encode(state)},state_path)
end
local function state()return vim.json.decode(table.concat(vim.fn.readfile(state_path),'\n'))end
local notices={}
vim.notify=function(message)notices[#notices+1]=message end
vim.schedule=function(fn)fn()end
local function invoke(which,budget)
  if which=='route' then return route.route('PairConfirmQuit',true,budget) end
  if which=='return' then return route.return_to_draft(budget) end
  return poke.send('private body',budget)
end
local function count_queries(s)
  local n=0;for _,q in ipairs(s.queries)do if q.args[2]=='list-panes' then n=n+1 end end;return n
end
for _,which in ipairs({'route','poke','return'})do
  for _,mode in ipairs({'healthy','wrong-first','wrong-second','changed','ambiguous','source-missing','other-tab','source-plugin','source-template','target-template','duplicate-key','cosmetic'})do
    local ps=rows();local faults={};local ordinal=mode=='wrong-first' and 1 or 2
    if mode=='wrong-first' or mode=='wrong-second' then faults={{ordinal=ordinal,reply='{"tab_id":2}'}} end
    if mode=='changed' then local changed=rows();changed[which=='poke' and 2 or 3].id=43;faults={{ordinal=2,reply=vim.json.encode(changed)}} end
    if mode=='duplicate-key' then faults={{ordinal=1,reply='[{"id":9,"id":9}]'}} end
    if mode=='source-plugin' then ps[4].is_plugin=true end
    if mode=='source-template' or mode=='target-template' then local changed=rows();changed[mode=='source-template' and 4 or (which=='poke' and 2 or 3)].terminal_command=mode=='source-template' and 'nvim -u /p/nvim/review.lua /d/other.md' or (which=='poke' and 'pair wrap claude' or 'nvim -u /p/nvim/init.lua /d/other.md');faults={{ordinal=2,reply=vim.json.encode(changed)}} end
    if mode=='cosmetic' then local changed=rows();changed[2].title='new title';changed[3].title='new title';faults={{ordinal=2,reply=vim.json.encode(changed)}} end
    if mode=='ambiguous' then ps[#ps+1]=pane(44,which=='poke' and 'pair wrap claude' or 'nvim -u /p/nvim/init.lua /d/second.md') end
    if mode=='source-missing' then table.remove(ps,4)end
    if mode=='other-tab' then ps[which=='poke' and 2 or 3].tab_id=8 end
    reset(ps,faults);notices={}
    local result=invoke(which)
    local s=state();local healthy=mode=='healthy' or mode=='cosmetic'
    assert(result==healthy,which..' '..mode..' result='..tostring(result)..' notices='..vim.inspect(notices))
    assert(#s.effects==(healthy and (which=='route' and 5 or 2) or 0),which..' '..mode..' effects='..vim.inspect(s.effects))
    assert(count_queries(s)==((mode=='wrong-first' or mode=='ambiguous' or mode=='source-missing' or mode=='other-tab' or mode=='source-plugin' or mode=='duplicate-key') and 1 or 2),which..' '..mode..' query count')
    assert(healthy or #notices>0,which..' '..mode..' silent refusal')
    if healthy then
      for _,e in ipairs(s.effects)do assert(e.target==(which=='poke' and 'terminal_7' or (e.action=='hide-floating-panes' and 'tab_2' or 'terminal_42')),vim.inspect(e))end
      if which=='poke' then
        assert(s.effects[1].args[5]=='\27[200~private body\27[201~' and s.effects[2].args[5]=='Alt Enter')
        assert(s.model.sessions.test.panes[2].input=='\27[200~private body\27[201~' and vim.deep_equal(s.model.sessions.test.panes[2].keys,{'Alt Enter'}),'fake lost actual paste/chord semantics')
      end
    end
  end
  local effects=which=='route' and 5 or 2
  for fail=1,effects do
    reset(nil,{{ordinal=2+fail,error='mutation failed'}});notices={}
    assert(not invoke(which),which..' mutation failure accepted')
    local s=state();assert(#s.effects==fail-1 and #s.queries==2+fail,which..' continued after effect failure')
  end
end
-- Expiry is driven by an explicit parent clock, never elapsed-time sleeps.
for _,which in ipairs({'route','poke','return'})do
  local effects=which=='route' and 5 or 2
  for prefix=0,effects-1 do
    reset();local expired=false;local calls=0
    local parent={remaining=function()return expired and 0 or 4000 end,check=function()assert(not expired,'expired parent')end}
    _G.PairTestZellijExecutor=function(_,argv,opts)
      local result=vim.system(trace.command(argv),{text=true,timeout=opts.timeout_ms}):wait()
      calls=calls+1;if calls==2+prefix then expired=true end
      return result
    end
    assert(not invoke(which,parent),which..' expiry accepted')
    local s=state();assert(#s.effects==prefix and #s.queries==2+prefix,which..' expired effect prefix '..prefix)
    _G.PairTestZellijExecutor=nil
  end
end
-- Spawn exceptions remain visible refusals and cannot admit the next action.
for _,which in ipairs({'route','poke','return'})do
  local effects=which=='route' and 5 or 2
  for boundary=1,2+effects do
    reset();notices={};local calls=0
    _G.PairTestZellijExecutor=function(_,argv,opts)
      calls=calls+1;if calls==boundary then error('spawn exception')end
      return vim.system(trace.command(argv),{text=true,timeout=opts.timeout_ms}):wait()
    end
    local ok,accepted=pcall(invoke,which)
    _G.PairTestZellijExecutor=nil
    assert(ok and not accepted and #notices>0,which..' exception was not visibly refused at '..boundary)
    local prefix=math.max(0,boundary-3)
    assert(#state().effects==prefix and calls==boundary,which..' continued after exception')
  end
end
-- Local session/endpoint changes during the second native read revoke effects.
for _,which in ipairs({'route','poke','return'})do
  for _,key in ipairs({'ZELLIJ_PANE_ID','ZELLIJ_SESSION_NAME'})do
    reset();local original=vim.env[key];local calls=0
    _G.PairTestZellijExecutor=function(_,argv,opts)
      local result=vim.system(trace.command(argv),{text=true,timeout=opts.timeout_ms}):wait()
      calls=calls+1;if calls==2 then vim.env[key]='changed' end
      return result
    end
    local accepted=invoke(which)
    vim.env[key]=original;_G.PairTestZellijExecutor=nil
    assert(not accepted and #state().effects==0,which..' local context change granted authority')
  end
end
-- An ordinary return caps a looser parent rather than borrowing a new limit.
reset()
_G.PairTestZellijExecutor=function(_,argv,opts)
  if argv[3]=='list-panes' then assert(opts.timeout_ms<=5000,'ordinary return exceeded five-second total cap')end
  return vim.system(trace.command(argv),{text=true,timeout=opts.timeout_ms}):wait()
end
assert(route.return_to_draft({remaining=function()return 9000 end,check=function()end}))
_G.PairTestZellijExecutor=nil
for _,n in ipairs({4,8})do
  reset()
  for _=1,n do assert(route.route('PairConfirmQuit',false))end
  local s=state();assert(count_queries(s)==2*n and #s.effects==4*n,'route work grew beyond 2 queries per decision')
  reset()
  for _=1,n do assert(poke.send('private body'))end
  s=state();assert(count_queries(s)==2*n and #s.effects==2*n,'poke work grew beyond 2 queries per decision')
end
-- Restore's admitted context reuses the exact parent token without another pair.
reset();local expired=false
local parent={remaining=function()return expired and 0 or 4000 end,check=function()assert(not expired,'expired parent')end}
assert(route.return_to_draft(parent,{id=42,tab_id=2,tab_position=1,terminal_command='nvim -u /p/nvim/init.lua /d/draft.md'}))
assert(count_queries(state())==0 and #state().effects==2)
reset();expired=true
assert(not route.return_to_draft(parent,{id=42,tab_id=2,tab_position=1,terminal_command='nvim -u /p/nvim/init.lua /d/draft.md'}))
assert(#state().queries==0)
-- The trusted draft locator retains its zero-query route, including per-effect
-- expiry checks. A completed mutation prefix never permits later writes.
local locator=state_path..'.draft'
vim.fn.writefile({vim.json.encode({session='test',pane_id='42',pid=vim.fn.getpid()})},locator)
vim.env.PAIR_DRAFT_PANE_PATH=locator
for prefix=0,4 do
  reset();local calls=0;local expired=prefix==0
  local parent={remaining=function()return expired and 0 or 4000 end,check=function()assert(not expired,'expired cached route')end}
  _G.PairTestZellijExecutor=function(_,argv,opts)
    local result=vim.system(trace.command(argv),{text=true}):wait()
    calls=calls+1;if calls==prefix then expired=true end
    return result
  end
  assert(not route.route('PairConfirmQuit',true,parent))
  assert(count_queries(state())==0 and #state().effects==prefix,'cached route bypassed effect expiry')
  _G.PairTestZellijExecutor=nil
end
vim.env.PAIR_DRAFT_PANE_PATH=''
print('route_poke_admission_test ok')
vim.cmd('qa!')
