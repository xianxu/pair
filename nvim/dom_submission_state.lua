-- Pure authority for one draft's DOM submission. Values in, next state and
-- declared effects out: this module performs no RPC, delivery or callbacks.
local M = {}

function M.initial()
  return { phase = 'idle', epoch = 0, polls = 0, receipts = {} }
end

local function copy(value)
  local out = {}
  for key, item in pairs(value) do out[key] = item end
  return out
end

local function effect(state, kind, fields)
  local pending = state.pending or {}
  local out = { kind = kind, epoch = state.epoch, id = pending.id, input = pending.input }
  for key, value in pairs(fields or {}) do out[key] = value end
  return out
end

local function notify(state, message)
  return { effect(state, 'notify', { message = message }) }
end

local function rpc(state, op, reply, delayed)
  return { effect(state, delayed and 'defer_rpc' or 'rpc', { op = op, reply = reply }) }
end

local function whole_count(value, minimum, maximum)
  return type(value) == 'number' and value >= minimum and value <= maximum and value == math.floor(value)
end
local function sketch_counts(result)
  if type(result) ~= 'table' or not whole_count(result.sketches, 1, 8) then return nil end
  local text = string.format('%d sketch%s selected', result.sketches, result.sketches == 1 and '' or 'es')
  if whole_count(result.remaining, 1, 256) then text = text .. string.format(', %d more', result.remaining) end
  return text
end

local function agent_text(pending, result)
  local prefix = pending.text .. (pending.text ~= '' and '\n\n' or '')
  local text = prefix .. 'DOM evidence: read ' .. result.path .. ' (submission.json) for selected notes and sketches. '
    .. 'Read referenced crop-N.png files for saved draw-time note pixels. New sketch bundles use sketch-page.png: '
    .. 'one replay-rendered historical viewport at the latest selected finish, with numbered ink matching each sketch.index '
    .. 'and original coordinates/pressure in paths. Legacy bundles use sketch-N.png and image.plan.paths. '
    .. 'Read each item’s original page URL, literal element IDs/data-* and historical recording handles. '
    .. 'Apply the full-page and each note or sketch’s privacy decision before quoting text or sharing images; '
    .. 'unavailable evidence and replay limitations are explicit.'
  local counts = sketch_counts(result)
  if counts then text = text .. ' ' .. counts .. '.' end
  if type(result.how_to_read) == 'string' and result.how_to_read ~= '' then
    text = text .. '\n\n' .. result.how_to_read
  end
  return text
end

local function transition(state, event)
  local next = copy(state)
  if state.pending then next.pending = copy(state.pending) end

  next.receipts = copy(state.receipts)

  if event.kind == 'send' then
    if state.pending then
      next.message = 'already sending…'
      return next, {}
    end
    next.epoch, next.polls = state.epoch + 1, 0
    next.pending = { body = event.body, text = event.text, input = event.input }
    next.phase = 'pinning'
    return next, { effect(next, 'pin') }
  end

  -- Confirmed-send receipts outlive draft epochs, but own no draft callbacks.
  if event.kind == 'acknowledged' then
    local receipt = state.receipts[event.id]
    if not receipt or receipt.epoch ~= event.epoch or receipt.polls ~= event.poll then return next, {} end
    if not event.error and type(event.result) == 'table' and event.result.state == 'resolved' then
      next.receipts[event.id] = nil
      return next, {}
    end
    if receipt.polls < 360 then
      next.receipts[event.id] = { epoch = receipt.epoch, polls = receipt.polls + 1 }
      return next, {{ kind = 'defer_rpc', epoch = receipt.epoch, id = event.id,
        op = 'ack', reply = 'acknowledged', poll = receipt.polls + 1 }}
    end
    next.receipts[event.id] = nil
    return next, notify(next, 'sent; note resolution needs recovery: couch --broadcast-web-submit ack ' .. event.id)
  end

  -- A deadline releases the draft, but cannot erase a subsequently confirmed
  -- dispatch. Recover only its notes; never reuse its draft completion callback.
  if event.kind == 'late_sent' and event.sent and event.id then
    local count = 0
    for _ in pairs(state.receipts) do count = count + 1 end
    if count >= 64 then
      return next, notify(next, 'sent; note resolution needs recovery: couch --broadcast-web-submit ack ' .. event.id)
    end
    next.receipts[event.id] = { epoch = event.epoch, polls = 0 }
    return next, {{kind='rpc',id=event.id,epoch=event.epoch,op='ack',reply='acknowledged',poll=0}}
  end

  -- A callback from a timed-out/retried attempt cannot act on its successor.
  if event.epoch ~= state.epoch then return next, {} end
  if (event.kind == 'expired' or event.kind == 'failed') and state.pending then
    next.pending, next.phase = nil, 'idle'
    return next, notify(next, event.error or 'submission timed out; edit freely and press the DOM chord again (delivery may be uncertain)')
  end
  local result = event.result
  if type(result) == 'table' and type(result.slot) == 'string' and next.pending then next.pending.slot = result.slot end

  if event.kind == 'pinned' and state.phase == 'pinning' then
    if not event.available then
      next.pending, next.phase = nil, 'idle'
      return next, notify(next, event.error or 'agent pane unavailable')
    end
    next.pending.id, next.phase = event.id, 'preparing'
    return next, rpc(next, 'prepare', 'prepared')
  end

  if event.kind == 'prepared' and state.phase == 'preparing' then
    if event.error or not result then
      next.pending, next.phase = nil, 'idle'
      return next, notify(next, (event.error or 'preparation response missing') .. '; edit freely and press the DOM chord again')
    end
    if result.state == 'preparing' then
      if state.polls >= 360 then
        next.pending, next.phase = nil, 'idle'
        return next, notify(next, 'preparation timed out; edit freely and press the DOM chord again')
      end
      next.polls = state.polls + 1
      return next, rpc(next, 'status', 'prepared', true)
    elseif result.state == 'prepared' then
      if type(result.path) ~= 'string' or result.path:sub(1, 1) ~= '/' then
        next.pending, next.phase = nil, 'idle'
        return next, notify(next, 'missing absolute bundle path; press the DOM chord again')
      end
      next.pending.agent_text = agent_text(state.pending, result)
      next.phase = 'checking'
      return next, { effect(next, 'check_current') }
    elseif result.state == 'empty' then
      next.phase = 'checking_empty'
      return next, { effect(next, 'check_current') }
    elseif result.state == 'failed' then
      next.pending, next.phase = nil, 'idle'
      return next, notify(next, result.error or 'preparation failed')
    end
    next.pending, next.phase = nil, 'idle'
    return next, notify(next, 'existing submission is ' .. tostring(result.state)
      .. '; reconcile delivery at ' .. tostring(result.path))
  end

  if event.kind == 'current' then
    if state.phase == 'checking_empty' then
      if event.current and state.pending.text:match('%S') then
        next.phase = 'ordinary'
        return next, { effect(next, 'ordinary', { body = state.pending.body, text = state.pending.text }) }
      end
      next.pending, next.phase = nil, 'idle'
      return next, {}
    elseif state.phase == 'checking' or state.phase == 'checking_after_begin' then
      if not event.current then
        next.pending, next.phase = nil, 'idle'
        return next, notify(next, 'draft or view changed; press the DOM chord to send the current draft')
      end
      if state.phase == 'checking' then
        next.phase = 'beginning'
        return next, rpc(next, 'begin', 'begun')
      end
      next.phase = 'sending'
      return next, { effect(next, 'submit', { body = state.pending.body, text = state.pending.agent_text }) }
    end
  end

  if event.kind == 'begun' and state.phase == 'beginning' then
    if event.error or not result or result.state ~= 'delivering' then
      next.pending, next.phase = nil, 'idle'
      return next, notify(next, (event.error or 'delivery intent was not confirmed') .. '; edit freely and press the DOM chord again')
    end
    next.phase = 'checking_after_begin'
    return next, { effect(next, 'check_current') }
  end

  if event.kind == 'sent' and state.phase == 'sending' then
    next.pending, next.phase, next.polls = nil, 'idle', 0
    if not event.sent then
      return next, notify(next, (event.error or 'send was not confirmed') .. '; edit freely and press the DOM chord again (duplicates are possible)')
    end
    local id = state.pending.id
    local effects = { effect(next, 'done', { input = state.pending.input }) }
    local count = 0
    for _ in pairs(state.receipts) do count = count + 1 end
    if count >= 64 then
      effects[2] = effect(next, 'notify', { message = 'sent; note resolution needs recovery: couch --broadcast-web-submit ack ' .. id })
    else
      next.receipts[id] = { epoch = state.epoch, polls = 0 }
      effects[2] = { kind = 'rpc', id = id, epoch = state.epoch, op = 'ack', reply = 'acknowledged', poll = 0 }
    end
    return next, effects
  end

  if event.kind == 'ordinary_sent' and state.phase == 'ordinary' then
    next.pending, next.phase = nil, 'idle'
    if event.sent then return next, { effect(next, 'done', { input = state.pending.input }) } end
    return next, notify(next, event.error or 'send was not confirmed; edit freely and press the DOM chord again')
  end

  return next, {}
end

local function literal(value)
  return tostring(value):gsub('[%z\1-\31\127]', ' '):sub(1, 500)
end
function M.progress_text(progress, selection)
  local fallback = 'preparing submission…'
  local counts = sketch_counts(selection)
  local function finish(text)
    return counts and (text .. ' (' .. counts .. ')') or text
  end
  if type(progress) ~= 'table' then return finish(fallback) end
  local phase = progress.phase
  if phase == 'replay' then return finish('preparing submission: opening replay…') end
  if phase == 'context' then return finish('preparing submission: reading context…') end
  if phase == 'publishing' then return finish('preparing submission: saving bundle…') end
  if phase == 'rendering-sketches' and whole_count(progress.total, 1, 8) then
    return finish(string.format('preparing submission: rendering %d sketch%s…', progress.total, progress.total == 1 and '' or 'es'))
  end
  local i, n = progress.current, progress.total
  if type(i) ~= 'number' or type(n) ~= 'number' or i ~= math.floor(i) or n ~= math.floor(n)
    or i < 1 or n < i or n > 1000 then return finish(fallback) end
  if phase == 'settling' then return finish(string.format('waiting for the page to settle, crop %d/%d', i, n)) end
  if phase == 'capturing' then return finish(string.format('preparing submission: capturing crop %d/%d…', i, n)) end
  return finish(fallback)
end
function M.step(state, event)
  local next, effects = transition(state, event)
  if event.kind == 'send' and not state.pending then
    next.message = 'preparing submission…'
  elseif event.epoch == state.epoch and state.pending then
    if not next.pending then
      next.message = 'nothing to send'
      for _, fx in ipairs(effects) do
        if fx.kind == 'done' then next.message = 'sent'; break end
        if fx.kind == 'notify' then next.message = literal(fx.message) end
      end
    elseif next.phase == 'preparing' and type(event.result) == 'table' then
      next.message = M.progress_text(event.result.progress, event.result)
    elseif next.phase == 'sending' or next.phase == 'ordinary' then
      next.message = 'sending to ' .. literal(next.pending.slot or 'agent') .. '…'
    end
  end
  return next, effects
end

return M
