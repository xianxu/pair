-- Effect adapter for the pure DOM submission transition model. Callbacks and
-- pane validators stay here; the model alone owns pending/busy/ack decisions.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local model = dofile(here .. 'dom_submission_state.lua')
local M = {}

function M.new(d)
  local state = model.initial()
  local contexts, targets, input = {}, {}, 0
  local dispatch
  local operation, cancel_timer
  local now = d.now or function() return os.clock() * 1000 end
  local function retire()
    if cancel_timer then pcall(cancel_timer); cancel_timer = nil end
  end

  local function run(effect)
    local function answer(kind, fields)
      local event = fields or {}
      event.kind, event.epoch = kind, effect.epoch
      event.id = event.id or effect.id
      event.poll = effect.poll
      dispatch(event)
    end
    if effect.kind == 'notify' then
      d.notify('DOM submission: ' .. tostring(effect.message))
    elseif effect.kind == 'pin' then
      local id = d.id()
      local target, err = d.pin(id, operation)
      if target then targets[id] = target end
      answer('pinned', { id = id, available = not not target, error = err })
    elseif effect.kind == 'rpc' or effect.kind == 'defer_rpc' then
      local function call()
        if effect.reply == 'acknowledged' then
          local receipt = state.receipts[effect.id]
          if not receipt or receipt.epoch ~= effect.epoch or receipt.polls ~= effect.poll then return end
        elseif state.epoch ~= effect.epoch or not state.pending then return end
        local ok, err = pcall(d.rpc, effect.op, effect.id, function(result, err)
          answer(effect.reply, { result = result, error = err })
        end, effect.reply ~= 'acknowledged' and operation or nil)
        if not ok then answer(effect.reply, {error=tostring(err)}) end
      end
      if effect.kind == 'defer_rpc' then d.defer(call, 100) else call() end
    elseif effect.kind == 'check_current' then
      local context = contexts[effect.input]
      answer('current', { current = context ~= nil and context.current() })
    elseif effect.kind == 'submit' then
      local sent, err = d.submit(effect.body, effect.text,
        { id = effect.id, target = targets[effect.id] }, operation)
      answer((state.epoch == effect.epoch and state.phase == 'sending') and 'sent' or 'late_sent', { sent = sent, error = err })
    elseif effect.kind == 'ordinary' then
      local sent, err = d.ordinary(effect.body, effect.text, operation)
      answer('ordinary_sent', { sent = sent, error = err })
    elseif effect.kind == 'done' then
      local context = contexts[effect.input]
      if context and context.current() then context.done() end
    else
      error('unknown DOM submission effect: ' .. tostring(effect.kind))
    end
  end

  local function apply(event)
    local before = state
    local effects
    state, effects = model.step(state, event)
    if state.pending and not before.pending then
      local epoch, deadline = state.epoch, now() + 36000
      local budget = { remaining = function()
        if not state.pending or state.epoch ~= epoch then return 0 end
        return math.max(0, math.floor(deadline - now()))
      end }
      budget.check = function()
        if budget.remaining() <= 0 then error('submission timed out', 0) end
      end
      operation = budget
      if d.arm then
        cancel_timer = d.arm(function() dispatch({kind='expired',epoch=epoch}) end, 36000)
      end
    elseif not state.pending and before.pending then
      retire()
    end
    if d.render and (before.message ~= state.message or before.pending ~= state.pending) then
      local owner = state.pending or before.pending
      local context = owner and contexts[owner.input]
      if context then d.render(state.message, state.pending ~= nil, context.view) end
    end
    for _, fx in ipairs(effects) do run(fx) end
    -- Receipt recovery carries only IDs; only the live attempt owns callbacks.
    for id in pairs(contexts) do
      if not state.pending or id ~= state.pending.input then contexts[id] = nil end
    end
    for id in pairs(targets) do
      if not state.pending or id ~= state.pending.id then targets[id] = nil end
    end
  end
  dispatch = function(event)
    local ok, err = pcall(apply, event)
    if not ok then
      -- Retire ownership even when an injected IO/UI effect raises.
      local pending = state.pending
      if pending and event.kind ~= 'acknowledged' and event.kind ~= 'late_sent' then
        local failure = {kind='failed',epoch=state.epoch,error=tostring(err)}
        local next = model.step(state, failure)
        state = next
        retire()
        local context = contexts[pending.input]
        if d.render and context then pcall(d.render, state.message, false, context.view) end
        contexts[pending.input], targets[pending.id or false] = nil, nil
      end
      pcall(d.notify, 'DOM submission: ' .. tostring(err))
    end
  end

  return { send = function(body, text, current, done, view)
    input = input + 1
    contexts[input] = { current = current, done = done, view = view }
    dispatch({ kind = 'send', body = body, text = text, input = input })
  end }
end

return M
