local M = dofile('nvim/dom_submission_state.lua')

local function plain(value)
  assert(type(value) ~= 'function' and type(value) ~= 'userdata', 'model retained an effect capability')
  if type(value) == 'table' then for key, item in pairs(value) do plain(key); plain(item) end end
end

local function driver()
  local state = M.initial()
  local function step(kind, fields)
    local event = fields or {}
    event.kind = kind
    if event.epoch == nil then event.epoch = state.epoch end
    local old, pending = state, state.pending
    local phase, text = old.phase, pending and pending.agent_text
    local effects
    state, effects = M.step(state, event)
    plain(state); plain(effects)
    assert(old.phase == phase and (not pending or pending.agent_text == text), 'transition mutated its input')
    return effects
  end
  return step, function() return state end
end

local function preparing(step)
  assert(step('send', {body='draft', text='draft', input=1})[1].kind == 'pin')
  local fx = step('pinned', {id='bundle', available=true})
  assert(fx[1].kind == 'rpc' and fx[1].op == 'prepare')
end
local guide = 'Recording: /couch/webcast/recordings/rec\ncouch --broadcast-web-query /couch/webcast/recordings/rec notes'
local function ready(step)
  local fx = step('prepared', {result={state='prepared', path='/store/bundle/submission.json', how_to_read=guide}})
  assert(fx[1].kind == 'check_current')
  assert(step('current', {current=true})[1].op == 'begin')
  assert(step('begun', {result={state='delivering'}})[1].kind == 'check_current')
  fx = step('current', {current=true})
  assert(fx[1].kind == 'submit' and fx[1].body == 'draft')
  assert(fx[1].text:find('submission.json',1,true) and fx[1].text:find('crop-N.png',1,true))
  assert(fx[1].text:sub(-#guide) == guide, 'canonical guide must survive verbatim')
  return fx[1].text
end

do
  local step, state = driver()
  preparing(step)
  local frozen = ready(step)
  step('sent', {sent=false})
  local fx = step('send', {body='edited', text='edited', input=2})
  assert(fx[1].kind == 'notify' and state().pending.agent_text == frozen)
  assert(step('send', {body='draft', text='draft', input=3})[1].kind == 'check_current')
  step('current', {current=true}); step('begun', {result={state='delivering'}})
  fx = step('current', {current=true})
  assert(fx[1].text == frozen and fx[1].input == 3)
  fx = step('sent', {sent=true})
  assert(fx[1].kind == 'done' and fx[2].op == 'ack' and state().pending == nil)
  step('acknowledged', {error='lost acknowledgment'})
  fx = step('send', {body='next draft', text='next draft', input=4})
  assert(#fx == 1 and fx[1].op == 'ack', 'ack recovery must not inspect or consume next draft')
  fx = step('acknowledged', {result={state='resolved'}})
  assert(#fx == 0 and state().ack_id == nil and state().phase == 'idle')
end

do
  local step, state = driver()
  preparing(step)
  local epoch = state().epoch
  assert(step('send', {body='other', text='other', input=2})[1].kind == 'notify')
  local fx = step('prepared', {result={state='preparing'}})
  assert(fx[1].kind == 'defer_rpc' and fx[1].op == 'status')
  for _=1,359 do step('prepared', {result={state='preparing'}}) end
  assert(step('prepared', {result={state='preparing'}})[1].kind == 'notify')
  assert(state().pending.id == 'bundle' and state().phase == 'idle')
  step('send', {body='draft', text='draft', input=3})
  assert(#step('prepared', {epoch=epoch, result={state='failed'}}) == 0, 'stale response changed newer attempt')
  step('prepared', {result={state='failed', error='capture failed'}})
  assert(state().pending == nil and state().phase == 'idle')
end

do
  local step, state = driver()
  preparing(step)
  step('prepared', {result={state='prepared', path='/store/bundle/submission.json'}})
  assert(step('current', {current=false})[1].kind == 'notify')
  assert(state().pending and state().phase == 'idle')
  step('send', {body='draft',text='draft',input=2})
  step('current', {current=true})
  step('begun', {error='begin refused'})
  assert(state().pending and state().phase == 'idle')
end

do
  local step, state = driver()
  preparing(step)
  assert(step('prepared', {result={state='empty'}})[1].kind == 'check_current')
  assert(step('current', {current=true})[1].kind == 'ordinary')
  assert(step('ordinary_sent', {sent=true})[1].kind == 'done')
  assert(state().pending == nil and state().ack_id == nil)
end

do
  local step, state = driver()
  preparing(step); ready(step); step('sent', {sent=true})
  for _=1,360 do
    local fx=step('acknowledged',{result={state='dispatched'}})
    assert(fx[1].kind=='defer_rpc' and fx[1].op=='status')
  end
  assert(step('acknowledged',{result={state='dispatched'}})[1].kind=='notify')
  assert(state().ack_id=='bundle' and state().phase=='idle')
end

-- Every pre-send refusal leaves delivery effects absent. Transport uncertainty
-- retains the attempt; a definite capture failure alone permits a fresh one.
for _, response in ipairs({
  {error='socket unavailable'},
  {result={state='prepared',path='relative/submission.json'}},
  {result={state='delivering',path='/store/bundle/submission.json'}},
}) do
  local step, state = driver()
  preparing(step)
  local fx=step('prepared',response)
  assert(#fx==1 and fx[1].kind=='notify')
  assert(state().phase=='idle' and state().pending.id=='bundle' and state().ack_id==nil)
end

do
  local step, state = driver()
  step('send',{body='draft',text='draft',input=1})
  assert(step('pinned',{available=false,error='no agent'})[1].kind=='notify')
  assert(state().pending==nil and state().phase=='idle')
end

do
  local step, state = driver()
  step('send',{body='=== sticky',text='',input=1})
  step('pinned',{available=true,id='bundle'})
  step('prepared',{result={state='empty'}})
  assert(#step('current',{current=true})==0)
  assert(state().pending==nil and state().phase=='idle','empty comments must not send or consume draft')
end

print('dom_submission_state_test: passed')
