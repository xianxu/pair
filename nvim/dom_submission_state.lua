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

local function agent_text(pending, result)
  local prefix = pending.text .. (pending.text ~= '' and '\n\n' or '')
  local text = prefix .. 'DOM notes: read ' .. result.path .. ' (submission.json) for the selected notes, '
    .. 'recorded context and capture metadata. Read each referenced crop-N.png file beside the manifest '
    .. 'for that note\'s current pixels; unavailable crops are explicitly marked.'
  if type(result.how_to_read) == 'string' and result.how_to_read ~= '' then
    text = text .. '\n\n' .. result.how_to_read
  end
  return text
end

function M.step(state, event)
  local next = copy(state)
  if state.pending then next.pending = copy(state.pending) end

  next.receipts = copy(state.receipts)

  if event.kind == 'send' then
    next.epoch, next.polls = state.epoch + 1, 0
    next.pending = { body = event.body, text = event.text, input = event.input }
    next.phase = 'pinning'
    return next, { effect(next, 'pin') }
  end

  -- Confirmed-send receipts outlive draft epochs, but own no draft callbacks.
  if event.kind == 'acknowledged' then
    local receipt = state.receipts[event.id]
    if not receipt or receipt.epoch ~= event.epoch or receipt.polls ~= event.poll then return next, {} end
    if not event.error and event.result and event.result.state == 'resolved' then
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

  -- A callback from a timed-out/retried attempt cannot act on its successor.
  if event.epoch ~= state.epoch then return next, {} end
  local result = event.result

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
      return next, notify(next, 'send was not confirmed; edit freely and press the DOM chord again (duplicates are possible)')
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
    return next, {}
  end

  return next, {}
end

return M
