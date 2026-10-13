local module=dofile('nvim/dom_transport.lua')
local current={id=7,is_plugin=false,is_floating=false,terminal_command='pair wrap codex'}
local rpc_state='delivering'
local codes={}
local fake={env={},json={decode=function(s)if s=='panes' then return {current} end;return {Submission={state=rpc_state,id='id'}}end},schedule=function(fn)fn()end}
fake.system=function(argv,opts,cb)
 assert(argv[1]=='couch' and argv[2]=='--broadcast-web-submit' and argv[4]=='id')
 local r={code=table.remove(codes,1) or 0,stdout='rpc',stderr='refused'}
 if cb then cb(r)end
 return {wait=function()return r end}
end
local t=module.new(fake,function(label,argv)assert(argv[3]=='list-panes');return {code=0,stdout='panes'}end)
local target=assert(t.pin('id'));assert(target.pane_id==7 and target.validate())
current={id=8,is_plugin=false,is_floating=false,terminal_command='pair wrap codex'}
assert(not target.validate())
rpc_state='failed';codes={1}
t.rpc('prepare','id',function(r,e)assert(r.state=='failed' and not e)end)
rpc_state='dispatched';codes={1}
t.rpc('ack','id',function(r,e)assert(not r and e=='refused')end)
print('dom_transport_test: passed')
