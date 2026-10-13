-- Pure authority for one draft's DOM submission. Values in, next state and
-- declared effects out: this module performs no RPC, delivery or callbacks.
local M = {}

function M.initial()
  return { phase = 'idle', epoch = 0, polls = 0 }
end

local function copy(value)
  local out = {}
  for key, item in pairs(value) do out[key] = item end
  return out
end

local function effect(state, kind, fields)
  local pending = state.pending or {}
  local out = { kind = kind, epoch = state.epoch, id = pending.id or state.ack_id, input = pending.input }
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

  if event.kind == 'send' then
    if state.phase ~= 'idle' then return next, notify(next, 'already preparing or resolving') end
    if not state.ack_id and state.pending
        and (state.pending.body ~= event.body or state.pending.text ~= event.text) then
      return next, notify(next, 'retry the original draft before changing its submission')
    end
    next.epoch, next.polls = state.epoch + 1, 0
    if state.ack_id then
      next.phase = 'acknowledging'
      return next, rpc(next, 'ack', 'acknowledged')
    end
    if not state.pending then
      next.pending = { body = event.body, text = event.text, input = event.input }
      next.phase = 'pinning'
      return next, { effect(next, 'pin') }
    end
    next.pending.input = event.input
    if state.pending.agent_text then
      next.phase = 'checking'
      return next, { effect(next, 'check_current') }
    end
    next.phase = 'preparing'
    return next, rpc(next, 'prepare', 'prepared')
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
      next.phase = 'idle'
      return next, notify(next, event.error or 'preparation response missing')
    end
    if result.state == 'preparing' then
      if state.polls >= 360 then
        next.phase = 'idle'
        return next, notify(next, 'preparation timed out; retry to inspect the same submission')
      end
      next.polls = state.polls + 1
      return next, rpc(next, 'status', 'prepared', true)
    elseif result.state == 'prepared' then
      if type(result.path) ~= 'string' or result.path:sub(1, 1) ~= '/' then
        next.phase = 'idle'
        return next, notify(next, 'missing absolute bundle path')
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
    next.phase = 'idle'
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
        next.phase = 'idle'
        local message = state.phase == 'checking' and 'draft or view changed; return to the original draft to retry'
          or 'draft changed before delivery; retry the original draft'
        return next, notify(next, message)
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
      next.phase = 'idle'
      return next, notify(next, event.error or 'delivery intent was not confirmed')
    end
    next.phase = 'checking_after_begin'
    return next, { effect(next, 'check_current') }
  end

  if event.kind == 'sent' and state.phase == 'sending' then
    if not event.sent then next.phase = 'idle'; return next, {} end
    next.pending, next.ack_id, next.phase, next.polls = nil, state.pending.id, 'acknowledging', 0
    return next, {
      effect(next, 'done', { input = state.pending.input }),
      effect(next, 'rpc', { op = 'ack', reply = 'acknowledged' }),
    }
  end

  if event.kind == 'ordinary_sent' and state.phase == 'ordinary' then
    next.pending, next.phase = nil, 'idle'
    if event.sent then return next, { effect(next, 'done', { input = state.pending.input }) } end
    return next, {}
  end

  if event.kind == 'acknowledged' and state.phase == 'acknowledging' then
    if not event.error and result and result.state == 'resolved' then
      next.ack_id, next.phase = nil, 'idle'
      return next, {}
    end
    if not event.error and result and result.state == 'dispatched'
        and (not result.error or result.error == '') and state.polls < 360 then
      next.polls = state.polls + 1
      return next, rpc(next, 'status', 'acknowledged', true)
    end
    next.phase = 'idle'
    local message = event.error or (result and result.error ~= '' and result.error)
      or 'delivered; note resolution is pending (press the chord to retry acknowledgment)'
    return next, notify(next, message)
  end
  return next, {}
end

return M
