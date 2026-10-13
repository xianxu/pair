local module=dofile('nvim/dom_transport.lua')
local current={id=7,is_plugin=false,is_floating=false,terminal_command='pair wrap codex'}
local inventory={current}
local rpc_state='delivering'
local codes={}
local fake={env={},json={decode=function(s)if s=='panes' then return inventory end;return {Submission={state=rpc_state,id='id'}}end},schedule=function(fn)fn()end}
fake.system=function(argv,opts,cb)
 assert(argv[1]=='couch' and argv[2]=='--broadcast-web-submit' and argv[4]=='id')
 local r={code=table.remove(codes,1) or 0,stdout='rpc',stderr='refused'}
 if cb then cb(r)end
 return {wait=function()return r end}
end
local t=module.new(fake,function(label,argv)assert(argv[3]=='list-panes');return {code=0,stdout='panes'}end)
local target=assert(t.pin('id'));assert(target.pane_id==7 and target.validate())
current={id=8,is_plugin=false,is_floating=false,terminal_command='pair wrap codex'}
inventory={current}
assert(not target.validate())
inventory={tabs={{panes={current,{id=9,is_plugin=false,is_floating=false,terminal_command='pair wrap claude'}}}}}
assert(not t.pin('id'),'multiple agent panes must refuse to pin')
inventory={tabs={{panes={{id=4,is_plugin=false,is_floating=false,title='codex',terminal_command='pair term'},current}}}}
assert(t.pin('id').pane_id==8,'nested layout/right terminal confused agent identity')
inventory={}
assert(not t.pin('id'),'missing agent must refuse to pin')
rpc_state='failed';codes={1}
t.rpc('prepare','id',function(r,e)assert(r.state=='failed' and not e)end)
rpc_state='dispatched';codes={1}
t.rpc('ack','id',function(r,e)assert(not r and e=='refused')end)
-- Every subprocess of a DOM attempt consumes its remaining operation budget.
local remaining, spawned = 125, 0
local operation = { remaining = function() return remaining end }
operation.check = function() assert(remaining > 0, 'submission timed out') end
inventory={current}
rpc_state='delivering';codes={}
local original_system=fake.system
fake.system=function(argv,opts,cb)
 spawned=spawned+1
 assert(opts.timeout==125, 'RPC must use remaining operation budget')
 return original_system(argv,opts,cb)
end
local bounded=module.new(fake,function(_,_,opts)
 spawned=spawned+1
 assert(opts and opts.timeout_ms==125, 'pane queries must use remaining operation budget')
 return {code=0,stdout='panes'}
end)
local pinned=assert(bounded.pin('id',operation))
assert(pinned.validate())
bounded.rpc('prepare','id',function(r)assert(r)end,operation)
remaining=0
local before=spawned
assert(not pcall(function()bounded.pin('id',operation)end), 'expired pin must refuse')
assert(not pcall(function()pinned.validate()end), 'expired validation must refuse')
assert(not pcall(function()bounded.rpc('prepare','id',function()end,operation)end), 'expired RPC must refuse')
assert(spawned==before, 'expired operation spawned a process')
remaining=125
fake.system=function(_,_,cb)
 cb({code=124,stdout='',stderr=''})
end
bounded.rpc('prepare','id',function(r,e)
 assert(not r and e and e:find('timed out',1,true),'RPC deadline must name its failure')
end,operation)
print('dom_transport_test: passed')
