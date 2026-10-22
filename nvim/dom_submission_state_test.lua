local M = dofile('nvim/dom_submission_state.lua')
local function driver()
  local state = M.initial()
  local function step(kind, fields)
    local event = fields or {}
    event.kind = kind
    event.epoch = event.epoch or state.epoch
    if event.kind == 'acknowledged' and event.poll == nil and state.receipts[event.id] then event.poll = state.receipts[event.id].polls end
    local old, old_phase = state, state.phase
    local effects
    state, effects = M.step(state, event)
    assert(old.phase == old_phase, 'mutated previous state')
    return effects
  end
  return step, function() return state end
end
local function prepare(step, body, id)
  assert(step('send', {body=body,text=body,input=id})[1].kind=='pin')
  assert(step('pinned',{id=id,available=true})[1].op=='prepare')
end
local function ready(step, body)
  assert(step('prepared',{result={state='prepared',path='/bundle/submission.json',how_to_read='guide'}})[1].kind=='check_current')
  assert(step('current',{current=true})[1].op=='begin')
  assert(step('begun',{result={state='delivering'}})[1].kind=='check_current')
  local fx=step('current',{current=true})
  assert(fx[1].body==body and fx[1].text:find(body,1,true)==1)
  assert(fx[1].text:sub(-5)=='guide')
end
for _, failure in ipairs({{error='unexpected submission entry .DS_Store'}, {result={state='failed',error='capture failed'}}, {result={state='prepared',path='relative'}}}) do
  local step,state=driver()
  prepare(step,'lost original','old')
  step('prepared',failure)
  assert(state().pending==nil and state().phase=='idle','pre-dispatch failure retained draft lock')
  prepare(step,'edited current','new'); ready(step,'edited current')
  assert(state().pending.id=='new')
end

do
  local step,state=driver()
  prepare(step,'old','old'); ready(step,'old')
  local fx=step('sent',{sent=false})
  assert(state().pending==nil and fx[1].message:find('edit',1,true))
  prepare(step,'new','new'); ready(step,'new')
end

-- Superseded callbacks cannot send or clear a newer draft; acknowledgments
-- remain independently addressable and never consume that draft.
do
  local step,state=driver()
  prepare(step,'first','one'); local first=state().epoch
  step('expired'); prepare(step,'second','two')
  assert(#step('prepared',{epoch=first,result={state='prepared',path='/old'}})==0)
  ready(step,'second'); local second=state().epoch
  local fx=step('sent',{sent=true})
  assert(fx[1].kind=='done' and fx[2].op=='ack' and fx[2].id=='two')
  prepare(step,'third','three'); local third=state().epoch
  fx=step('acknowledged',{epoch=second,id='two',error='lost ack'})
  assert(fx[1].kind=='defer_rpc' and fx[1].op=='ack' and fx[1].id=='two')
  assert(state().epoch==third and state().pending.body=='third')
  assert(#step('acknowledged',{epoch=second,id='two',result={state='resolved'}})==0)
  assert(state().receipts.two==nil and state().pending.body=='third')
  assert(#step('acknowledged',{epoch=second,id='two',result={state='dispatched'}})==0)
end

do
  local step,state=driver()
  prepare(step,'draft','bundle'); ready(step,'draft'); local epoch=state().epoch
  step('sent',{sent=true})
  for _=1,360 do
    local fx=step('acknowledged',{id='bundle',epoch=epoch,result={state='dispatched',error='busy'}})
    assert(fx[1].kind=='defer_rpc' and fx[1].op=='ack')
  end
  local fx=step('acknowledged',{id='bundle',epoch=epoch,error='unavailable'})
  assert(state().receipts.bundle==nil and fx[1].message:find('couch --broadcast-web-submit ack bundle',1,true))
end

do
  local step,state=driver()
  for i=1,65 do
    prepare(step,'draft','id'..i); ready(step,'draft')
    local fx=step('sent',{sent=true})
    assert(fx[1].kind=='done')
    if i==65 then assert(fx[2].kind=='notify' and fx[2].message:find('ack id65',1,true)) end
  end
  local count=0; for _ in pairs(state().receipts) do count=count+1 end
  assert(count==64,'unbounded acknowledgment receipts')
end

do
  local step,state=driver()
  prepare(step,'draft','one')
  step('prepared',{result={state='prepared',path='/bundle'}})
  step('current',{current=false})
  assert(state().pending==nil)
  prepare(step,'edited','two')
  step('prepared',{result={state='prepared',path='/bundle'}})
  step('current',{current=true}); step('begun',{error='lost begin'})
  assert(state().pending==nil)
end

do
  local step,state=driver()
  prepare(step,'draft','one')
  for _=1,360 do step('prepared',{result={state='preparing'}}) end
  step('prepared',{result={state='preparing'}})
  assert(state().pending==nil,'timed out preparation froze the draft')
  prepare(step,'draft','two'); step('prepared',{result={state='empty'}})
  assert(step('current',{current=true})[1].kind=='ordinary')
  assert(step('ordinary_sent',{sent=true})[1].kind=='done')
end
do
  local step,state=driver()
  prepare(step,'draft','bundle'); ready(step,'draft'); local epoch=state().epoch
  step('sent',{sent=true})
  step('acknowledged',{id='bundle',epoch=epoch,poll=0,error='lost'})
  assert(#step('acknowledged',{id='bundle',epoch=epoch,poll=0,error='duplicate'})==0,'duplicate ack callback forked recovery')
end
print('dom_submission_state_test: passed')

-- Active ownership cannot be replaced or extended by another chord.
do
 local step,state=driver();prepare(step,'one','one');local epoch=state().epoch
 step('send',{body='two',text='two',input='two'})
 assert(state().epoch==epoch and state().pending.body=='one','active send replaced owner')
 assert(state().message=='already sending…')
 step('prepared',{result={state='preparing',progress={phase='settling',current=2,total=3}}})
 assert(state().message=='waiting for the page to settle, crop 2/3')
 step('expired')
 assert(not state().pending and state().message:find('timed out',1,true))
end
assert(M.progress_text({phase='capturing',current=1,total=3})=='preparing submission: capturing crop 1/3…')
for _,p in ipairs({false,12,'bad',{phase='capturing',current=-1,total=3},{phase='settling',current=0/0,total=3},{phase='evil'}}) do
 assert(M.progress_text(p)=='preparing submission…')
end

do
 local step,state=driver();prepare(step,'Act on these','mixed')
 step('prepared',{result={state='preparing',sketches=2,remaining=4,progress={phase='rendering-sketches',total=2}}})
 assert(state().message:find('2 sketches',1,true) and state().message:find('4 more',1,true),'sketch preparation counts missing')
 step('prepared',{result={state='prepared',path='/bundle/submission.json',how_to_read='canonical guide',sketches=2,remaining=4}})
 step('current',{current=true});step('begun',{result={state='delivering'}})
 local effects=step('current',{current=true});local text=effects[1].text
 for _,part in ipairs({'Act on these','sketch-N.png','draw-time','replay-rendered','original','literal','privacy','2 sketches','4 more','canonical guide'})do
  assert(text:find(part,1,true),'mixed evidence handoff missing '..part)
 end
 assert(not text:find('current pixels',1,true),'historical evidence called current pixels')
end
assert(M.progress_text({phase='rendering-sketches',total=3}):find('3 sketches',1,true),'render total fallback missing')
for _,counts in ipairs({{sketches=0/0,remaining=4},{sketches=9,remaining=4},{sketches='2',remaining=4}}) do
 assert(M.progress_text(nil,counts)=='preparing submission…','invalid sketch count reached UI')
end
assert(M.progress_text(nil,{sketches=1,remaining=0}):find('1 sketch selected',1,true),'single sketch count grammar')
