-- Effect adapter for the pure DOM submission transition model. Callbacks and
-- pane validators stay here; the model alone owns pending/busy/ack decisions.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local model = dofile(here .. 'dom_submission_state.lua')
local M = {}

function M.new(d)
  local state = model.initial()
  local contexts, targets, input = {}, {}, 0
  local dispatch

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
      local target, err = d.pin(id)
      if target then targets[id] = target end
      answer('pinned', { id = id, available = not not target, error = err })
    elseif effect.kind == 'rpc' or effect.kind == 'defer_rpc' then
      local function call()
        if effect.reply == 'acknowledged' then
          local receipt = state.receipts[effect.id]
          if not receipt or receipt.epoch ~= effect.epoch or receipt.polls ~= effect.poll then return end
        elseif state.epoch ~= effect.epoch then return end
        d.rpc(effect.op, effect.id, function(result, err)
          answer(effect.reply, { result = result, error = err })
        end)
      end
      if effect.kind == 'defer_rpc' then d.defer(call, 100) else call() end
    elseif effect.kind == 'check_current' then
      local context = contexts[effect.input]
      answer('current', { current = context ~= nil and context.current() })
    elseif effect.kind == 'submit' then
      answer('sent', { sent = d.submit(effect.body, effect.text,
        { id = effect.id, target = targets[effect.id] }) })
    elseif effect.kind == 'ordinary' then
      answer('ordinary_sent', { sent = d.ordinary(effect.body, effect.text) })
    elseif effect.kind == 'done' then
      contexts[effect.input].done()
    else
      error('unknown DOM submission effect: ' .. tostring(effect.kind))
    end
  end

  dispatch = function(event)
    local effects
    state, effects = model.step(state, event)
    for _, effect in ipairs(effects) do run(effect) end
    -- Receipt recovery carries only IDs; only the live attempt owns callbacks.
    for id in pairs(contexts) do
      if not state.pending or id ~= state.pending.input then contexts[id] = nil end
    end
    for id in pairs(targets) do
      if (not state.pending or id ~= state.pending.id) then targets[id] = nil end
    end
  end

  return { send = function(body, text, current, done)
    input = input + 1
    contexts[input] = { current = current, done = done }
    dispatch({ kind = 'send', body = body, text = text, input = input })
  end }
end

return M
